package scanner

import (
	"fmt"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var gopath = os.Getenv("GOPATH")

const project = "github.com/src-d/proteus"

func projectPath(pkg string) string {
	return filepath.Join(gopath, "src", project, pkg)
}

func TestScanType(t *testing.T) {
	cases := []struct {
		name     string
		typ      types.Type
		expected Type
	}{
		{
			"named type",
			newNamed("/foo/bar", "Bar", nil),
			NewNamed("/foo/bar", "Bar"),
		},
		{
			"basic type",
			types.Typ[types.Int],
			NewBasic("int"),
		},
		{
			"basic array",
			types.NewArray(types.Typ[types.Int], 8),
			repeated(NewBasic("int")),
		},
		{
			"basic slice",
			types.NewSlice(types.Typ[types.Int]),
			repeated(NewBasic("int")),
		},
		{
			"basic behind a pointer",
			types.NewPointer(types.Typ[types.Int]),
			NewBasic("int"),
		},
		{
			"named behind a pointer",
			types.NewPointer(newNamed("/foo/bar", "Bar", nil)),
			NewNamed("/foo/bar", "Bar"),
		},
		{
			"map of basic and named",
			types.NewMap(
				types.Typ[types.Int],
				newNamed("/foo/bar", "Bar", nil),
			),
			NewMap(
				NewBasic("int"),
				NewNamed("/foo/bar", "Bar"),
			),
		},
		{
			"struct",
			types.NewStruct(nil, nil),
			nil,
		},
		{
			"interface",
			types.NewInterface(nil, nil),
			nil,
		},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, scanType(c.typ), c.name)
	}
}

func TestScanStruct(t *testing.T) {
	cases := []struct {
		name     string
		elem     *types.Struct
		expected *Struct
	}{
		{
			"simple struct",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo", types.Typ[types.Int], false),
					mkField("Bar", types.Typ[types.String], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
					{"Bar", NewBasic("string")},
				},
			},
		},
		{
			"struct with non exported field",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo", types.Typ[types.Int], false),
					mkField("bar", types.Typ[types.String], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
				},
			},
		},
		{
			"struct with ignore tag",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo", types.Typ[types.Int], false),
					mkField("Bar", types.Typ[types.String], false),
				},
				[]string{"", `proteus:"-"`},
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
				},
			},
		},
		{
			"struct with unsupported type",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo", types.Typ[types.Int], false),
					mkField("Bar", types.NewStruct(nil, nil), false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
				},
			},
		},
		{
			"embedded struct",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						newNamed("/foo", "Foo", types.NewStruct(
							[]*types.Var{
								mkField("Foo", types.Typ[types.Int], false),
								mkField("Bar", types.Typ[types.String], false),
							},
							nil,
						),
						),
						true,
					),
					mkField("Baz", types.Typ[types.Uint64], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
					{"Bar", NewBasic("string")},
					{"Baz", NewBasic("uint64")},
				},
			},
		},
		{
			"embedded struct with repeated field",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						newNamed("/foo", "Foo", types.NewStruct(
							[]*types.Var{
								mkField("Foo", types.Typ[types.Int], false),
								mkField("Bar", types.Typ[types.String], false),
							},
							nil,
						),
						),
						true,
					),
					mkField("Bar", types.Typ[types.Uint64], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
					{"Bar", NewBasic("string")},
				},
			},
		},
		{
			"embedded pointer to struct",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						types.NewPointer(
							newNamed("/foo", "Foo", types.NewStruct(
								[]*types.Var{
									mkField("Foo", types.Typ[types.Int], false),
									mkField("Bar", types.Typ[types.String], false),
								},
								nil,
							),
							),
						),
						true,
					),
					mkField("Baz", types.Typ[types.Uint64], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Foo", NewBasic("int")},
					{"Bar", NewBasic("string")},
					{"Baz", NewBasic("uint64")},
				},
			},
		},
		{
			"invalid embedded type",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo", types.Typ[types.Int], true),
					mkField("Baz", types.Typ[types.Uint64], false),
				},
				nil,
			),
			&Struct{
				Fields: []*Field{
					{"Baz", NewBasic("uint64")},
				},
			},
		},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, scanStruct(&Struct{}, c.elem), c.name)
	}
}

func TestScannerNotDir(t *testing.T) {
	require := require.New(t)

	scanner, err := New(projectPkg("fixtures/foo.go"))
	require.Nil(scanner)
	require.NotNil(err)
}

const errFile = `package barbaz

import "bar/baz"

type Bar struct {
	baz.Foo
}
`

func TestScannerError(t *testing.T) {
	require := require.New(t)

	require.Nil(os.MkdirAll(absPath("fixtures/error"), 0777))
	require.Nil(ioutil.WriteFile(absPath("fixtures/error/foo.go"), []byte(errFile), 0777))

	scanner, err := New(projectPkg("fixtures/error"))
	require.Nil(err)

	_, err = scanner.Scan()
	require.NotNil(err)

	require.Nil(os.RemoveAll(absPath("fixtures/error")))
}

func TestScanner(t *testing.T) {
	require := require.New(t)

	scanner, err := New(projectPkg("fixtures"), projectPkg("fixtures/subpkg"))
	require.Nil(err)

	pkgs, err := scanner.Scan()
	require.Nil(err)
	require.Equal(2, len(pkgs), "scan packages")

	pkg := pkgs[0]
	subpkg := pkgs[1]

	require.Equal(4, len(pkg.Structs), "pkg")
	assertStruct(t, pkg.Structs[0], "Bar", true, "Bar", "Baz")
	assertStruct(t, pkg.Structs[1], "Foo", true, "Bar", "Baz", "IntList", "IntArray", "Map", "Timestamp", "External", "Duration", "Aliased")
	assertStruct(t, pkg.Structs[2], "Qux", false, "A", "B")
	assertStruct(t, pkg.Structs[3], "Saz", true, "Point", "Foo")

	require.Equal(2, len(subpkg.Structs), "subpkg")
	assertStruct(t, subpkg.Structs[0], "NotGenerated", false)
	assertStruct(t, subpkg.Structs[1], "Point", true, "X", "Y")

	_, ok := pkg.Aliases[fmt.Sprintf("%s.%s", projectPath("fixtures"), "Baz")]
	require.False(ok, "Baz should not be an alias anymore")

	require.Equal(1, len(pkg.Enums), "pkg enums")
	require.Equal("Baz", pkg.Enums[0].Name)

	require.Equal(
		[]string{"ABaz", "BBaz", "CBaz", "DBaz"},
		pkg.Enums[0].Values,
		"enum values",
	)
}

func assertStruct(t *testing.T, s *Struct, name string, generate bool, fields ...string) {
	require.Equal(
		t,
		name,
		s.Name,
		"struct name",
	)
	require.Equal(t, generate, s.Generate, "struct should be generated")

	require.Equal(t, len(fields), len(s.Fields), "should have same struct fields")
	for _, f := range fields {
		require.True(t, s.HasField(f), "should have struct field %q", f)
	}
}

func mkField(name string, typ types.Type, anon bool) *types.Var {
	return types.NewField(
		token.NoPos,
		types.NewPackage("/foo", "mock"),
		name,
		typ,
		anon,
	)
}

func repeated(t Type) Type {
	t.SetRepeated(true)
	return t
}

func newNamed(path, name string, underlying types.Type) types.Type {
	obj := types.NewTypeName(
		token.NoPos,
		types.NewPackage(path, "mock"),
		name,
		underlying,
	)
	return types.NewNamed(obj, underlying, nil)
}

func projectPkg(pkg string) string {
	return filepath.Join(project, pkg)
}

func absPath(path string) string {
	return filepath.Join(goPath, "src", project, path)
}
