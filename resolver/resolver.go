package resolver

import (
	"fmt"

	"github.com/src-d/proteus/report"
	"github.com/src-d/proteus/scanner"
)

// Resolver has the responsibility of checking the types of all the packages
// scanned globally and exclude struct fields with types not included in any
// of the scanned packages and replacing some aliases to other types with their
// actual type.
// Consider the type `type IntList []int` on the field `Foo`, the type of that
// field would be changed from a named `IntList` type to a repeated basic
// type `int`.
type Resolver struct {
	customTypes map[string]struct{}
}

// New creates a new Resolver with the default custom types registered.
// These are time.Time and time.Duration. Those types will be considered correct
// even though their packages are not in any of the packages given.
func New() *Resolver {
	return &Resolver{
		customTypes: map[string]struct{}{
			"time.Time":     {},
			"time.Duration": {},
			"error":         {},
		},
	}
}

// Resolve checks the types of all the packages passed in a global manner.
// Also, it sets to `true` the `Resolved` field of the package, meaning that
// they can be safely used after it.
func (r *Resolver) Resolve(pkgs []*scanner.Package) {
	info := getPackagesInfo(pkgs)

	for _, p := range pkgs {
		r.resolvePackage(p, info)
	}
}

func (r *Resolver) isCustomType(n *scanner.Named) bool {
	_, ok := r.customTypes[n.String()]
	return ok
}

func (r *Resolver) resolvePackage(p *scanner.Package, info *packagesInfo) {
	for _, s := range p.Structs {
		r.resolveStruct(s, info)
	}

	var funcs = make([]*scanner.Func, 0, len(p.Funcs))
	for _, f := range p.Funcs {
		if r.resolveFunc(f, info) {
			funcs = append(funcs, f)
		} else {
			report.Warn("func %s had an unresolvable type and it will not be generated", f.Name)
		}
	}
	p.Funcs = funcs

	r.removeUnmarkedStructs(p, info)
	p.Resolved = true
}

func (r *Resolver) resolveFunc(f *scanner.Func, info *packagesInfo) bool {
	f.Input = r.resolveTypeList(f.Input, info)
	if f.Input == nil {
		return false
	}

	f.Output = r.resolveTypeList(f.Output, info)
	if f.Output == nil {
		return false
	}

	return true
}

func (r *Resolver) resolveTypeList(types []scanner.Type, info *packagesInfo) []scanner.Type {
	var result = make([]scanner.Type, 0, len(types))
	for _, t := range types {
		typ := r.resolveType(t, info)
		if typ == nil {
			return nil
		}
		result = append(result, typ)
	}
	return result
}

func (r *Resolver) removeUnmarkedStructs(p *scanner.Package, info *packagesInfo) {
	var structs []*scanner.Struct
	for _, s := range p.Structs {
		name := fmt.Sprintf("%s.%s", p.Path, s.Name)
		if info.isStructMarked(name) {
			structs = append(structs, s)
		}
	}
	p.Structs = structs
}

func (r *Resolver) resolveStruct(s *scanner.Struct, info *packagesInfo) {
	var result = make([]*scanner.Field, 0, len(s.Fields))

	for _, f := range s.Fields {
		if typ := r.resolveType(f.Type, info); typ != nil {
			f.Type = typ
			result = append(result, f)
		}
	}

	s.Fields = result
}

func (r *Resolver) resolveType(typ scanner.Type, info *packagesInfo) (result scanner.Type) {
	switch t := typ.(type) {
	case *scanner.Named:
		if r.isCustomType(t) {
			return t
		}

		if !info.hasPackage(t.Path) {
			report.Warn("type %q of package %s will be ignored because it was not present on the scan path", t.Name, t.Path)
			return nil
		}

		alias := info.aliasOf(t)
		if alias != nil {
			if alias.IsRepeated() {
				report.Warn(
					"type %q of package %s is an alias for %s that is marked as repeated. Alias for repeated fields are not currently supported, this field will be ignored.",
					t.Name,
					t.Path,
					alias.String(),
				)
				return nil
			}
			return scanner.NewAlias(t, alias)
		}

		if info.isStruct(t.String()) {
			info.markStruct(t.String())
		}

		result = t
	case *scanner.Basic:
		result = t
	case *scanner.Map:
		t.Key = r.resolveType(t.Key, info)
		t.Value = r.resolveType(t.Value, info)
		result = t
	}

	return
}

// getPackagesInfo retrieves some information about a list of packages like the
// aliases in all of them combined and the paths of all the packages.
// Note that enums are removed from the aliases as we do not want to
// think of them as aliases but as named types instead.
func getPackagesInfo(pkgs []*scanner.Package) *packagesInfo {
	result := &packagesInfo{
		aliases:  make(map[string]scanner.Type),
		packages: make(map[string]struct{}),
		structs:  make(map[string]bool),
	}
	enums := packagesEnums(pkgs)

	for _, p := range pkgs {
		result.packages[p.Path] = struct{}{}
		for n, t := range p.Aliases {
			if _, ok := enums[n]; !ok {
				result.aliases[n] = t
			}
		}

		for _, s := range p.Structs {
			result.structs[fmt.Sprintf("%s.%s", p.Path, s.Name)] = s.Generate
		}
	}

	return result
}

// packagesEnums returns a set with all the enums in all packages.
func packagesEnums(pkgs []*scanner.Package) map[string]struct{} {
	result := make(map[string]struct{})

	for _, p := range pkgs {
		for _, e := range p.Enums {
			result[fmt.Sprintf("%s.%s", p.Path, e.Name)] = struct{}{}
		}
	}

	return result
}

// packagesInfo contains information about a collection of packages.
type packagesInfo struct {
	aliases  map[string]scanner.Type
	packages map[string]struct{}
	structs  map[string]bool
}

// aliasOf returns the alias of a given named type or nil if there is
// no alias for that type.
func (i *packagesInfo) aliasOf(named *scanner.Named) scanner.Type {
	alias, ok := i.aliases[named.String()]
	if !ok {
		return nil
	}
	return alias
}

func (i *packagesInfo) isStruct(name string) bool {
	_, ok := i.structs[name]
	return ok
}

func (i *packagesInfo) markStruct(name string) {
	i.structs[name] = true
}

func (i *packagesInfo) isStructMarked(name string) bool {
	return i.structs[name]
}

func (i *packagesInfo) hasPackage(path string) bool {
	_, ok := i.packages[path]
	return ok
}
