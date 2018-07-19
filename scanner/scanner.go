package scanner

import (
	"errors"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/src-d/proteus.v1/report"

	"gopkg.in/src-d/go-parse-utils.v1"
)

var goPath = os.Getenv("GOPATH")

// Scanner scans packages looking for Go source files to parse
// and extract types and structs from.
type Scanner struct {
	packages []string
	importer *parseutil.Importer
}

// ErrNoGoPathSet is the error returned when the GOPATH variable is not
// set.
var ErrNoGoPathSet = errors.New("GOPATH environment variable is not set")

// New creates a new Scanner that will look for types and structs
// only in the given packages.
func New(packages ...string) (*Scanner, error) {
	if goPath == "" {
		return nil, ErrNoGoPathSet
	}

	for _, pkg := range packages {
		p := filepath.Join(goPath, "src", pkg)
		fi, err := os.Stat(p)
		switch {
		case err != nil:
			return nil, err
		case !fi.IsDir():
			return nil, fmt.Errorf("path is not directory: %s", p)
		}
	}

	return &Scanner{
		packages: packages,
		importer: parseutil.NewImporter(),
	}, nil
}

// Scan retrieves the scanned packages containing the extracted
// go types and structs.
func (s *Scanner) Scan() ([]*Package, error) {
	var (
		pkgs   = make([]*Package, len(s.packages))
		errors errorList
		mut    sync.Mutex
		wg     = new(sync.WaitGroup)
	)

	wg.Add(len(s.packages))
	for i, p := range s.packages {
		go func(p string, i int) {
			defer wg.Done()

			pkg, err := s.scanPackage(p)
			mut.Lock()
			defer mut.Unlock()
			if err != nil {
				errors.add(fmt.Errorf("error scanning package %q: %s", p, err))
				return
			}

			pkgs[i] = pkg
		}(p, i)
	}

	wg.Wait()
	if len(errors) > 0 {
		return nil, errors.err()
	}

	return pkgs, nil
}

func (s *Scanner) scanPackage(p string) (*Package, error) {
	pkg, err := s.importer.ImportWithFilters(
		p,
		parseutil.FileFilters{
			func(pkg, file string, typ parseutil.FileType) bool {
				return !strings.HasSuffix(file, ".pb.go")
			},
			func(pkg, file string, typ parseutil.FileType) bool {
				return !strings.HasSuffix(file, ".proteus.go")
			},
		},
	)
	if err != nil {
		return nil, err
	}

	ctx, err := newContext(p)
	if err != nil {
		return nil, err
	}

	return buildPackage(ctx, pkg)
}

func buildPackage(ctx *context, gopkg *types.Package) (*Package, error) {
	objs := objectsInScope(gopkg.Scope())

	pkg := &Package{
		Path:    removeGoPath(gopkg),
		Name:    gopkg.Name(),
		Aliases: make(map[string]Type),
	}

	for _, o := range objs {
		if err := pkg.scanObject(ctx, o); err != nil {
			return nil, err
		}
	}

	pkg.collectEnums(ctx)
	return pkg, nil
}

func (p *Package) scanObject(ctx *context, o types.Object) error {
	if !o.Exported() {
		return nil
	}

	switch t := o.Type().(type) {
	case *types.Named:
		hasStringMethod, err := isStringer(t)
		if err != nil {
			return err
		}
		switch o.(type) {
		case *types.Const:
			if _, ok := t.Underlying().(*types.Basic); ok {
				scanEnumValue(ctx, o.Name(), t, hasStringMethod)
			}
		case *types.TypeName:
			if s, ok := t.Underlying().(*types.Struct); ok {
				st := scanStruct(
					&Struct{
						Name:       o.Name(),
						Generate:   ctx.shouldGenerateType(o.Name()),
						IsStringer: hasStringMethod,
					},
					s,
				)
				ctx.trySetDocs(o.Name(), st)
				p.Structs = append(p.Structs, st)
				return nil
			}

			p.Aliases[objName(t.Obj())] = scanType(t.Underlying())
		}
	case *types.Signature:
		if ctx.shouldGenerateFunc(nameForFunc(o)) {
			fn := scanFunc(&Func{Name: o.Name()}, t)
			ctx.trySetDocs(nameForFunc(o), fn)
			p.Funcs = append(p.Funcs, fn)
		}
	}

	return nil
}

func isStringer(t *types.Named) (bool, error) {
	for i := 0; i < t.NumMethods(); i++ {
		m := t.Method(i)
		if m.Name() != "String" {
			continue
		}

		sign := m.Type().(*types.Signature)
		if sign.Params().Len() != 0 {
			return false, fmt.Errorf("type %s implements a String method that does not satisfy fmt.Stringer (wrong number of parameters)", t.Obj().Name())
		}

		results := sign.Results()
		if results == nil || results.Len() != 1 {
			return false, fmt.Errorf("type %s implements a String method that does not satisfy fmt.Stringer (wrong number of results)", t.Obj().Name())
		}

		if returnType, ok := results.At(0).Type().(*types.Basic); ok {
			if returnType.Name() == "string" {
				return true, nil
			}
			return false, fmt.Errorf("type %s implements a String method that does not satisfy fmt.Stringer (wrong type of result)", t.Obj().Name())
		}
	}

	return false, nil
}

func nameForFunc(o types.Object) (name string) {
	s := o.Type().(*types.Signature)

	if s.Recv() != nil {
		name = nameForType(s.Recv().Type()) + "."
	}

	name = name + o.Name()

	return
}

func nameForType(o types.Type) (name string) {
	name = o.String()
	i := strings.LastIndex(name, ".")
	name = name[i+1 : len(name)]

	return
}

func scanType(typ types.Type) (t Type) {
	switch u := typ.(type) {
	case *types.Basic:
		t = NewBasic(u.Name())
	case *types.Named:
		t = NewNamed(
			removeGoPath(u.Obj().Pkg()),
			u.Obj().Name(),
		)
	case *types.Slice:
		t = scanType(u.Elem())
		t.SetRepeated(true)
	case *types.Array:
		t = scanType(u.Elem())
		t.SetRepeated(true)
	case *types.Pointer:
		t = scanType(u.Elem())
		t.SetNullable(true)
	case *types.Map:
		key := scanType(u.Key())
		val := scanType(u.Elem())
		if val == nil {
			report.Warn("ignoring map with value type %s", typ.String())
			return nil
		}
		t = NewMap(key, val)
	default:
		report.Warn("ignoring type %s", typ.String())
		return nil
	}

	return
}

func scanEnumValue(ctx *context, name string, named *types.Named, hasStringMethod bool) {
	typ := objName(named.Obj())
	ctx.enumValues[typ] = append(ctx.enumValues[typ], name)
	ctx.enumWithString = append(ctx.enumWithString, typ)
}

func scanStruct(s *Struct, elem *types.Struct) *Struct {
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
				s = scanStruct(s, embedded)
			}
			continue
		}

		f := &Field{
			Name: v.Name(),
			Type: scanType(v.Type()),
		}
		if f.Type == nil {
			continue
		}

		s.Fields = append(s.Fields, f)
	}

	return s
}

func scanFunc(fn *Func, signature *types.Signature) *Func {
	if signature.Recv() != nil {
		fn.Receiver = scanType(signature.Recv().Type())
	}
	fn.Input = scanTuple(signature.Params())
	fn.Output = scanTuple(signature.Results())
	fn.IsVariadic = signature.Variadic()

	return fn
}

func scanTuple(tuple *types.Tuple) []Type {
	result := make([]Type, 0, tuple.Len())

	for i := 0; i < tuple.Len(); i++ {
		result = append(result, scanType(tuple.At(i).Type()))
	}

	return result
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

// newEnum creates a new enum with the given name.
// The values are looked up in the ast package and only if they are constants
// they will be added as enum values.
// All values are guaranteed to be sorted by their iota.
func newEnum(ctx *context, name string, vals []string, hasStringMethod bool) *Enum {
	enum := &Enum{Name: name, IsStringer: hasStringMethod}
	ctx.trySetDocs(name, enum)
	var values enumValues
	for _, v := range vals {
		if obj, ok := ctx.consts[v]; ok {
			values = append(values, enumValue{
				name: v,
				pos:  uint(obj.Data.(int)),
			})
		}
	}

	sort.Stable(values)

	for _, v := range values {
		val := &EnumValue{Name: v.name}
		ctx.trySetDocs(v.name, val)
		enum.Values = append(enum.Values, val)
	}

	return enum
}

type enumValue struct {
	name string
	pos  uint
}

type enumValues []enumValue

func (v enumValues) Swap(i, j int) {
	v[j], v[i] = v[i], v[j]
}

func (v enumValues) Len() int {
	return len(v)
}

func (v enumValues) Less(i, j int) bool {
	return v[i].pos < v[j].pos
}

func isIgnoredField(f *types.Var, tags []string) bool {
	return !f.Exported() || (len(tags) > 0 && tags[0] == "-")
}

func objectsInScope(scope *types.Scope) (objs []types.Object) {
	for _, n := range scope.Names() {
		obj := scope.Lookup(n)
		objs = append(objs, obj)

		typ := obj.Type()

		if _, ok := typ.Underlying().(*types.Struct); ok {
			// Only need to extract methods for the pointer type since it contains
			// the methods for the non-pointer type as well.
			objs = append(objs, methodsForType(types.NewPointer(typ))...)
		}
	}
	return
}

func methodsForType(typ types.Type) (objs []types.Object) {
	methods := types.NewMethodSet(typ)

	for i := 0; i < methods.Len(); i++ {
		objs = append(objs, methods.At(i).Obj())
	}

	return
}

func objName(obj types.Object) string {
	return fmt.Sprintf("%s.%s", removeGoPath(obj.Pkg()), obj.Name())
}

func removeGoPath(pkg *types.Package) string {
	// error is a type.Named whose package is nil.
	if pkg == nil {
		return ""
	} else {
		return strings.Replace(pkg.Path(), filepath.Join(goPath, "src")+"/", "", -1)
	}
}

type errorList []error

func (l *errorList) add(err error) {
	*l = append(*l, err)
}

func (l errorList) err() error {
	var lines []string
	for _, err := range l {
		lines = append(lines, err.Error())
	}
	return errors.New(strings.Join(lines, "\n"))
}
