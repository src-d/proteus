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
		},
	}
}

// Resolve checks the types of all the packages passed in a global manner.
// Also, it sets to `true` the `Resolved` field of the package, meaning that
// they can be safely used after it.
func (r *Resolver) Resolve(pkgs []*scanner.Package) {
	info := Packages(pkgs).Info()

	for _, p := range pkgs {
		r.resolvePackage(p, info)
	}
}

func (r *Resolver) isCustomType(n *scanner.Named) bool {
	_, ok := r.customTypes[n.String()]
	return ok
}

func (r *Resolver) resolvePackage(p *scanner.Package, info *PackagesInfo) {
	for _, s := range p.Structs {
		s.Fields = r.resolveStructFields(s.Fields, info)
	}
	p.Resolved = true
}

func (r *Resolver) resolveStructFields(fields []*scanner.Field, info *PackagesInfo) []*scanner.Field {
	var result = make([]*scanner.Field, 0, len(fields))

	for _, f := range fields {
		if typ := r.resolveType(f.Type, info); typ != nil {
			f.Type = typ
			result = append(result, f)
		}
	}

	return result
}

func (r *Resolver) resolveType(typ scanner.Type, info *PackagesInfo) (result scanner.Type) {
	switch t := typ.(type) {
	case *scanner.Named:
		if r.isCustomType(t) {
			return t
		}

		if _, ok := info.Packages[t.Path]; !ok {
			report.Warn("type %q of package %s will be ignored because it was not present on the scan path", t.Name, t.Path)
			return nil
		}

		alias := info.AliasOf(t)
		if alias != nil {
			return alias
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

// Packages is a collection of scanned packages.
type Packages []*scanner.Package

// Info retrieves some information about a list of packages like the
// aliases in all of them combined and the paths of all the packages.
// Note that enums are removed from the aliases as we do not want to
// think of them as aliases but as named types instead.
func (pkgs Packages) Info() *PackagesInfo {
	result := &PackagesInfo{
		Aliases:  make(map[string]scanner.Type),
		Packages: make(map[string]struct{}),
	}
	enums := pkgs.Enums()

	for _, p := range pkgs {
		result.Packages[p.Path] = struct{}{}
		for n, t := range p.Aliases {
			if _, ok := enums[n]; !ok {
				result.Aliases[n] = t
			}
		}
	}

	return result
}

// Enums returns a set with all the enums in all packages.
func (pkgs Packages) Enums() map[string]struct{} {
	result := make(map[string]struct{})

	for _, p := range pkgs {
		for _, e := range p.Enums {
			result[fmt.Sprintf("%s.%s", p.Path, e.Name)] = struct{}{}
		}
	}

	return result
}

// PackagesInfo contains information about a collection of packages.
type PackagesInfo struct {
	Aliases  map[string]scanner.Type
	Packages map[string]struct{}
}

// AliasOf returns the alias of a given named type or nil if there is
// no alias for that type.
func (i *PackagesInfo) AliasOf(named *scanner.Named) scanner.Type {
	alias, ok := i.Aliases[named.String()]
	if !ok {
		return nil
	}
	return alias
}
