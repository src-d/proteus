package resolver

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/src-d/proteus/report"
	"github.com/src-d/proteus/scanner"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const project = "github.com/src-d/proteus"

func TestPackagesEnums(t *testing.T) {
	packages := []*scanner.Package{
		{
			Path: "foo",
			Enums: []*scanner.Enum{
				enum("Foo", "A", "B", "C"),
				enum("Bar", "D", "E"),
			},
		},
		{
			Path: "bar",
			Enums: []*scanner.Enum{
				enum("Cmp", "Lt", "Eq", "Gt"),
			},
		},
	}

	enumSet := packagesEnums(packages)
	require.Equal(t, 3, len(enumSet), "enums size")
	assertStrSet(t, enumSet, "bar.Cmp", "foo.Bar", "foo.Foo")
}

func TestGetPackagesInfo(t *testing.T) {
	packages := []*scanner.Package{
		{
			Path: "foo",
			Aliases: map[string]scanner.Type{
				"foo.Foo": scanner.NewBasic("int"),
				"foo.Bar": scanner.NewBasic("int"),
				"foo.Baz": scanner.NewBasic("int"),
			},
			Enums: []*scanner.Enum{
				enum("Foo", "A", "B", "C"),
				enum("Bar", "D", "E"),
			},
		},
		{
			Path: "bar",
			Aliases: map[string]scanner.Type{
				"bar.Cmp": scanner.NewBasic("int"),
				"bar.Qux": scanner.NewBasic("int"),
			},
			Enums: []*scanner.Enum{
				enum("Cmp", "Lt", "Eq", "Gt"),
			},
		},
	}

	info := getPackagesInfo(packages)
	require.Equal(t, 2, len(info.packages))
	assertStrSet(t, info.packages, "bar", "foo")

	require.Equal(t, 2, len(info.aliases))
	_, ok := info.aliases["foo.Baz"]
	require.True(t, ok)

	_, ok = info.aliases["bar.Qux"]
	require.True(t, ok)
}

func TestResolver(t *testing.T) {
	suite.Run(t, new(ResolverSuite))
}

type ResolverSuite struct {
	suite.Suite
	r *Resolver
}

func (s *ResolverSuite) SetupSuite() {
	s.r = New()
}

func (s *ResolverSuite) TestIsCustomType() {
	cases := []struct {
		path   string
		name   string
		result bool
	}{
		{"foo.bar/baz/bar", "Baz", false},
		{"net/url", "URL", false},
		{"time", "Time", true},
		{"time", "Duration", true},
	}

	for _, c := range cases {
		s.Equal(c.result, s.r.isCustomType(&scanner.Named{nil, c.path, c.name}), "%s.%s", c.path, c.name)
	}
}

func (s *ResolverSuite) TestAliasToRepeatedFieldWarning() {
	report.TestMode()

	aliasOf := scanner.NewNamed("", "alias")
	aliasOf.SetRepeated(true)
	info := &packagesInfo{
		aliases: map[string]scanner.Type{"named": aliasOf},
	}
	typ := scanner.NewNamed("", "named")
	s.Nil(s.r.resolveType(typ, info))
	s.Len(report.MessageStack(), 1, "it contains one message")

	report.EndTestMode()
}

func (s *ResolverSuite) TestResolve() {
	sc, err := scanner.New(projectPath("fixtures"), projectPath("fixtures/subpkg"))
	s.Nil(err)
	pkgs, err := sc.Scan()
	s.Nil(err)

	s.Equal(3, len(pkgs[1].Structs), "num of structs in pkg")
	s.r.Resolve(pkgs)

	pkg := pkgs[0]
	s.assertStruct(pkg.Structs[0], "Bar", "Bar", "Baz")
	s.assertStruct(pkg.Structs[1], "Foo", "Bar", "Baz", "IntList", "IntArray", "Map", "Timestamp", "Duration", "Aliased")
	// Qux is not opted-in, but is required by Foo, so should be here
	s.assertStruct(pkg.Structs[2], "Qux", "A", "B")

	s.Equal(0, len(pkg.Funcs), "num of funcs in pkg")

	foo := pkg.Structs[1]
	aliasedType := foo.Fields[len(foo.Fields)-1].Type
	// Change when repeated aliases are supported
	s.False(aliasedType.IsRepeated(), "Aliased type should not be repeated")
	aliasType, ok := aliasedType.(*scanner.Alias)
	s.True(ok, "Aliased should be a alias type")

	aliasTypeType, ok := aliasType.Type.(*scanner.Named)
	s.True(ok, "Alias is a named type")
	s.Equal("MyInt", aliasTypeType.Name, "alias name")

	basic, ok := aliasType.Underlying.(*scanner.Basic)
	s.True(ok, "Aliased type is basic")
	s.Equal("int", basic.Name)

	s.Equal(1, len(pkgs[1].Structs), "a struct of subpkg should have been removed")
	s.Equal(4, len(pkgs[1].Funcs), "num of funcs in subpkg")

	s.Equal(&scanner.Func{
		Name: "Generated",
		Input: []scanner.Type{
			scanner.NewBasic("string"),
		},
		Output: []scanner.Type{
			scanner.NewBasic("bool"),
			scanner.NewNamed("", "error"),
		},
	}, findFuncByName("Generated", pkgs[1].Funcs))

	s.Equal(&scanner.Func{
		Name: "GeneratedMethod",
		Input: []scanner.Type{
			scanner.NewBasic("int32"),
		},
		Output: []scanner.Type{
			nullable(scanner.NewNamed(projectPath("fixtures/subpkg"), "Point")),
		},
		Receiver: scanner.NewNamed(projectPath("fixtures/subpkg"), "Point"),
	}, findFuncByName("GeneratedMethod", pkgs[1].Funcs))

	s.Equal(&scanner.Func{
		Name: "GeneratedMethodOnPointer",
		Input: []scanner.Type{
			scanner.NewBasic("bool"),
		},
		Output: []scanner.Type{
			nullable(scanner.NewNamed(projectPath("fixtures/subpkg"), "Point")),
		},
		Receiver: nullable(scanner.NewNamed(projectPath("fixtures/subpkg"), "Point")),
	}, findFuncByName("GeneratedMethodOnPointer", pkgs[1].Funcs))

	s.Equal(&scanner.Func{
		Name:  "Name",
		Input: []scanner.Type{},
		Output: []scanner.Type{
			scanner.NewBasic("string"),
		},
		Receiver: nullable(scanner.NewNamed(projectPath("fixtures/subpkg"), "MyContainer")),
	}, findFuncByName("Name", pkgs[1].Funcs))
}

func (s *ResolverSuite) assertStruct(st *scanner.Struct, name string, fields ...string) {
	s.Equal(name, st.Name, "struct name")
	s.Equal(len(fields), len(st.Fields), "should have same struct fields")
	for _, f := range fields {
		s.True(st.HasField(f), "should have struct field %q", f)
	}
}

func assertStrSet(t *testing.T, set map[string]struct{}, expected ...string) {
	var vals []string
	for v := range set {
		vals = append(vals, v)
	}
	sort.Strings(vals)
	require.Equal(t, expected, vals)
}

func enum(name string, values ...string) *scanner.Enum {
	return &scanner.Enum{
		Name:   name,
		Values: values,
	}
}

func projectPath(pkg string) string {
	return filepath.Join(project, pkg)
}

func findFuncByName(name string, fns []*scanner.Func) *scanner.Func {
	for _, f := range fns {
		if f.Name == name {
			return f
		}
	}

	return nil
}

func nullable(t scanner.Type) scanner.Type {
	t.SetNullable(true)
	return t
}
