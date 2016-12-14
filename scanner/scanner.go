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
	"strings"
	"sync"

	"github.com/src-d/protogo/report"
)

// Package holds information about a single Go package and
// a reference of all defined structs and type aliases.
// A Package is only safe to use once it is resolved.
type Package struct {
	Resolved bool
	Path     string
	Name     string
	Structs  []*Struct
	Enums    []*Enum
	Aliases  map[string]Type
	values   map[string][]string
}

// Type is the common interface for all possible types supported in protogo.
// Type is neither a representation of a Go type nor a representation of a
// protobuf type. Is an intermediate representation to ease future steps in
// the conversion from Go to protobuf.
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

// Map is a map type with a key and a value type.
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
	Name   string
	Values []string
}

// Struct represents a Go struct with its name and fields.
type Struct struct {
	Name   string
	Fields []*Field
}

func (s *Struct) HasField(name string) bool {
	for _, f := range s.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
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
	var (
		pkgs   = make([]*Package, len(s.paths))
		errors []error
		mut    sync.Mutex
		wg     = new(sync.WaitGroup)
	)

	wg.Add(len(s.paths))
	for i, p := range s.paths {
		go func(p string, i int) {
			defer wg.Done()

			pkg, err := s.scanPackage(p)
			mut.Lock()
			defer mut.Unlock()
			if err != nil {
				errors = append(errors, fmt.Errorf("error scanning package %q: %s", p, err))
			} else {
				pkgs[i] = pkg
			}
		}(p, i)
	}

	wg.Wait()
	if len(errors) > 0 {
		var lines []string
		for _, err := range errors {
			lines = append(lines, err.Error())
		}
		return nil, fmt.Errorf(strings.Join(lines, "\n"))
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

		st := processStruct(&Struct{Name: o.Name()}, s)
		p.Structs = append(p.Structs, st)
		return
	}

	p.Aliases[objName(n.Obj())] = processType(n.Underlying())
}

func processType(typ types.Type) (t Type) {
	switch u := typ.(type) {
	case *types.Named:
		t = NewNamed(
			u.Obj().Pkg().Path(),
			u.Obj().Name(),
		)
	case *types.Basic:
		t = NewBasic(u.Name())
	case *types.Slice:
		t = processType(u.Elem())
		t.SetRepeated(true)
	case *types.Array:
		t = processType(u.Elem())
		t.SetRepeated(true)
	case *types.Pointer:
		t = processType(u.Elem())
	case *types.Map:
		key := processType(u.Key())
		val := processType(u.Elem())
		t = NewMap(key, val)
	default:
		report.Warn("ignoring type %s", typ.String())
		return nil
	}

	return
}

func (p *Package) processEnumValue(name string, named *types.Named) {
	typ := objName(named.Obj())
	p.values[typ] = append(p.values[typ], name)
}

func processStruct(s *Struct, elem *types.Struct) *Struct {
	for i := 0; i < elem.NumFields(); i++ {
		v := elem.Field(i)
		tags := findProtoTags(elem.Tag(i))

		if isIgnoredField(v, tags) {
			continue
		}

		// TODO: It has not been decided yet what exact behaviour
		// is the intended when a struct overrides a field from
		// a previously embedded type. For now, the field is just
		// completely ignored and a warning is printed to give
		// feedback to the user.
		if s.HasField(v.Name()) {
			report.Warn("struct %q already has a field %q", s.Name, v.Name())
			continue
		}

		if v.Anonymous() {
			embedded := findStruct(v.Type())
			if embedded == nil {
				report.Warn("field %q with type %q is not a valid embedded type", v.Name(), v.Type())
			} else {
				s = processStruct(s, embedded)
			}
			continue
		}

		f := &Field{
			Name: v.Name(),
			Type: processType(v.Type()),
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

func (p *Package) collectEnums() {
	for k := range p.Aliases {
		if vals, ok := p.values[k]; ok {
			idx := strings.LastIndex(k, ".")
			name := k[idx+1:]

			p.Enums = append(p.Enums, &Enum{
				Name:   name,
				Values: vals,
			})

			delete(p.Aliases, k)
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

	pkg.collectEnums()
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

func objName(obj types.Object) string {
	return fmt.Sprintf("%s.%s", obj.Pkg().Path(), obj.Name())
}
