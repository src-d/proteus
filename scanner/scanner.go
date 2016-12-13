package scanner

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
)

// Package holds information about a single Go package and
// a reference of all defined structs and type aliases.
// A Package is only safe to use once it is resolved.
type Package struct {
	Resolved bool
	Path     string
	Name     string
	Structs  []*Struct
	Aliases  map[string]Type
	values   map[string][]string
}

// Type is the common interface for all possible protobuf types supported in protogo
// which are Map, Enum, Named and Basic.
// All types can be nullable (or not) or repeated (or not).
type Type interface {
	SetRepeated(bool)
	SetNullable(bool)
	IsRepeated() bool
	IsNullable() bool
}

// BaseType contains the common fields for all the types.
type BaseType struct {
	Nullable bool
	Repeated bool
}

func newBaseType() *BaseType {
	return &BaseType{
		Nullable: true,
		Repeated: false,
	}
}

func (t *BaseType) IsRepeated() bool   { return t.Repeated }
func (t *BaseType) IsNullable() bool   { return t.Nullable }
func (t *BaseType) SetRepeated(v bool) { t.Repeated = v }
func (t *BaseType) SetNullable(v bool) { t.Nullable = v }

// Basic is a basic type, which only is identified by its name.
type Basic struct {
	*BaseType
	Name string
}

func NewBasic(name string) Type {
	return &Basic{
		newBaseType(),
		name,
	}
}

// Named is non-basic type identified by a name on some package.
type Named struct {
	*BaseType
	Path string
	Name string
}

func NewNamed(path, name string) Type {
	return &Named{
		newBaseType(),
		path,
		name,
	}
}

func (n *Named) FullName() string {
	return fmt.Sprintf("%s.%s", n.Path, n.Name)
}

type Map struct {
	*BaseType
	Key   Type
	Value Type
}

func NewMap(key, val Type) Type {
	return &Map{
		newBaseType(),
		key,
		val,
	}
}

// Enum consists of a list of possible values.
type Enum struct {
	*BaseType
	Values []string
}

func NewEnum(values ...string) Type {
	return &Enum{
		newBaseType(),
		values,
	}
}

// Struct represents a Go struct with its name and fields.
type Struct struct {
	Name   string
	Fields []*Field
}

// Field contains name and type of a struct field.
type Field struct {
	Name string
	Type Type
}

// Scanner scans paths looking for Go source files to parse
// and extract types and structs from.
type Scanner struct {
	paths []string
}

// New creates a new Scanner that will look for types and structs
// only in the given paths.
func New(paths ...string) (*Scanner, error) {
	for _, p := range paths {
		fi, err := os.Stat(p)
		switch {
		case err != nil:
			return nil, err
		case !fi.IsDir():
			return nil, fmt.Errorf("path is not directory: %s", p)
		}
	}

	return &Scanner{paths: paths}, nil
}

// Scan retrieves the scanned packages containing the extracted
// go types and structs.
func (s *Scanner) Scan() ([]*Package, error) {
	var pkgs []*Package
	for _, p := range s.paths {
		pkg, err := s.scanPackage(p)
		if err != nil {
			return nil, fmt.Errorf("error scanning package %q: %s", p, err)
		}

		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func (s *Scanner) scanPackage(path string) (*Package, error) {
	files, err := getSourceFiles(path)
	if err != nil {
		return nil, err
	}

	gopkg, err := parseSourceFiles(path, files)
	if err != nil {
		return nil, err
	}

	return buildPackage(gopkg)
}

func (p *Package) processObject(o types.Object) {
	n, ok := o.Type().(*types.Named)
	if !ok || !o.Exported() {
		return
	}

	switch o.(type) {
	case *types.Var, *types.Const:
		if _, ok := n.Underlying().(*types.Basic); ok {
			p.processEnumValue(o.Name(), n)
		}
		return
	}

	if s, ok := n.Underlying().(*types.Struct); ok {

		st := p.processStruct(&Struct{Name: o.Name()}, s)
		p.Structs = append(p.Structs, st)
		return
	}

	name := fmt.Sprintf("%s.%s", n.Obj().Pkg().Path(), n.Obj().Name())
	p.Aliases[name] = p.processType(n.Underlying())
}

func (p *Package) processType(typ types.Type) (t Type) {
	switch u := typ.(type) {
	case *types.Named:
		t = NewNamed(
			u.Obj().Pkg().Path(),
			u.Obj().Name(),
		)
	case *types.Basic:
		t = NewBasic(u.Name())
	case *types.Slice:
		t = p.processType(u.Elem())
		t.SetRepeated(true)
	case *types.Array:
		t = p.processType(u.Elem())
		t.SetRepeated(true)
	case *types.Pointer:
		t = p.processType(u.Elem())
	case *types.Map:
		key := p.processType(u.Key())
		val := p.processType(u.Elem())
		t = NewMap(key, val)
	default:
		fmt.Printf("ignoring type %s\n", typ.String())
		return nil
	}

	return
}

func (p *Package) processEnumValue(name string, named *types.Named) {
	typ := fmt.Sprintf("%s.%s", named.Obj().Pkg().Path(), named.Obj().Name())
	p.values[typ] = append(p.values[typ], name)
}

func (p *Package) processStruct(s *Struct, elem *types.Struct) *Struct {
	for i := 0; i < elem.NumFields(); i++ {
		v := elem.Field(i)
		tags := findProtoTags(elem.Tag(i))
		if isIgnoredField(v, tags) {
			continue
		}

		if v.Anonymous() {
			embedded := findStruct(v.Type())
			if embedded != nil {
				s = p.processStruct(s, embedded)
			}
			continue
		}

		f := &Field{
			Name: v.Name(),
			Type: p.processType(v.Type()),
		}
		if f.Type == nil {
			continue
		}

		s.Fields = append(s.Fields, f)
	}

	return s
}

func findStruct(t types.Type) *types.Struct {
	switch elem := t.(type) {
	case *types.Pointer:
		return findStruct(elem.Elem())
	case *types.Named:
		return findStruct(elem.Underlying())
	case *types.Struct:
		return elem
	default:
		return nil
	}
}

func (p *Package) checkEnums() {
	for k := range p.Aliases {
		if vals, ok := p.values[k]; ok {
			p.Aliases[k] = NewEnum(vals...)
		}
	}
}

func isIgnoredField(f *types.Var, tags []string) bool {
	return !f.Exported() || (len(tags) > 0 && tags[0] == "-")
}

func buildPackage(gopkg *types.Package) (*Package, error) {
	objs := objectsInScope(gopkg.Scope())

	pkg := &Package{
		Path:    gopkg.Path(),
		Name:    gopkg.Name(),
		values:  make(map[string][]string),
		Aliases: make(map[string]Type),
	}

	for _, o := range objs {
		pkg.processObject(o)
	}

	pkg.checkEnums()
	return pkg, nil
}

func objectsInScope(scope *types.Scope) (objs []types.Object) {
	for _, n := range scope.Names() {
		objs = append(objs, scope.Lookup(n))
	}
	return
}

func getSourceFiles(path string) ([]string, error) {
	pkg, err := build.ImportDir(path, 0)
	if err != nil {
		return nil, err
	}

	var filenames []string
	filenames = append(filenames, pkg.GoFiles...)
	filenames = append(filenames, pkg.CgoFiles...)

	if len(filenames) == 0 {
		return nil, fmt.Errorf("no go source files in path: %s", path)
	}

	var paths []string
	for _, f := range filenames {
		paths = append(paths, filepath.Join(path, f))
	}

	return paths, nil
}

func parseSourceFiles(root string, paths []string) (*types.Package, error) {
	var files []*ast.File
	fs := token.NewFileSet()
	for _, p := range paths {
		f, err := parser.ParseFile(fs, p, nil, 0)
		if err != nil {
			return nil, err
		}

		files = append(files, f)
	}

	config := types.Config{Importer: importer.For("gc", nil)}

	return config.Check(root, fs, files, new(types.Info))
}
