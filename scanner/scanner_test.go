package scanner

import (
	"fmt"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var gopath = os.Getenv("GOPATH")

const project = "gopkg.in/src-d/proteus.v1"

func projectPath(pkg string) string {
	return filepath.Join(gopath, "src", project, pkg)
}

func Test_unexistingPackage(t *testing.T) {
	_, err := New("github.com/src-d/nonexistingprojectforsure")
	require.NotNil(t, err)
}

func TestScanType(t *testing.T) {
	cases := []struct {
		name     string
		typ      types.Type
		expected Type
	}{
		{
			"named type",
			newNamedWithUnderlying("/foo/bar", "Bar", nil),
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
			nullable(NewBasic("int")),
		},
		{
			"named behind a pointer",
			types.NewPointer(newNamedWithUnderlying("/foo/bar", "Bar", nil)),
			nullable(NewNamed("/foo/bar", "Bar")),
		},
		{
			"map of basic and named",
			types.NewMap(
				types.Typ[types.Int],
				newNamedWithUnderlying("/foo/bar", "Bar", nil),
			),
			NewMap(
				NewBasic("int"),
				NewNamed("/foo/bar", "Bar"),
			),
		},
		{
			"array of pointers",
			types.NewArray(types.NewPointer(types.Typ[types.Int]), 8),
			nullable(repeated(NewBasic("int"))),
		},
		{
			"slice of pointers",
			types.NewSlice(types.NewPointer(types.Typ[types.Int])),
			nullable(repeated(NewBasic("int"))),
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
					{Name: "Foo", Type: NewBasic("int")},
					{Name: "Bar", Type: NewBasic("string")},
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
					{Name: "Foo", Type: NewBasic("int")},
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
					{Name: "Foo", Type: NewBasic("int")},
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
					{Name: "Foo", Type: NewBasic("int")},
				},
			},
		},
		{
			"embedded struct",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						newNamedWithUnderlying("/foo", "Foo", types.NewStruct(
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
					{Name: "Foo", Type: NewBasic("int")},
					{Name: "Bar", Type: NewBasic("string")},
					{Name: "Baz", Type: NewBasic("uint64")},
				},
			},
		},
		{
			"embedded struct with repeated field",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						newNamedWithUnderlying("/foo", "Foo", types.NewStruct(
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
					{Name: "Foo", Type: NewBasic("int")},
					{Name: "Bar", Type: NewBasic("string")},
				},
			},
		},
		{
			"embedded pointer to struct",
			types.NewStruct(
				[]*types.Var{
					mkField("Foo",
						types.NewPointer(
							newNamedWithUnderlying("/foo", "Foo", types.NewStruct(
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
					{Name: "Foo", Type: NewBasic("int")},
					{Name: "Bar", Type: NewBasic("string")},
					{Name: "Baz", Type: NewBasic("uint64")},
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
					{Name: "Baz", Type: NewBasic("uint64")},
				},
			},
		},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, scanStruct(&Struct{}, c.elem), c.name)
	}
}

func TestScannerScanFunc(t *testing.T) {
	cases := []struct {
		name      string
		signature *types.Signature
		expected  *Func
	}{
		{
			"empty",
			types.NewSignature(
				nil,
				types.NewTuple(),
				types.NewTuple(),
				false,
			),
			&Func{
				Input:  make([]Type, 0),
				Output: make([]Type, 0),
			},
		},
		{
			"with receiver",
			types.NewSignature(
				mkParam("p", types.Typ[types.Int32]),
				types.NewTuple(),
				types.NewTuple(),
				false,
			),
			&Func{
				Receiver: NewBasic("int32"),
				Input:    make([]Type, 0),
				Output:   make([]Type, 0),
			},
		},
		{
			"with params",
			types.NewSignature(
				nil,
				types.NewTuple(
					mkParam("a", types.Typ[types.Int32]),
					mkParam("b", types.Typ[types.String]),
				),
				types.NewTuple(),
				false,
			),
			&Func{
				Input:  []Type{NewBasic("int32"), NewBasic("string")},
				Output: make([]Type, 0),
			},
		},
		{
			"with result",
			types.NewSignature(
				nil,
				types.NewTuple(),
				types.NewTuple(mkParam("a", types.Typ[types.String])),
				false,
			),
			&Func{
				Input:  make([]Type, 0),
				Output: []Type{NewBasic("string")},
			},
		},
		{
			"with everything",
			types.NewSignature(
				mkParam("a", types.Typ[types.Bool]),
				types.NewTuple(mkParam("b", types.Typ[types.Int32]), mkParam("c", types.Typ[types.String])),
				types.NewTuple(mkParam("d", types.Typ[types.Float32])),
				false,
			),
			&Func{
				Receiver: NewBasic("bool"),
				Input:    []Type{NewBasic("int32"), NewBasic("string")},
				Output:   []Type{NewBasic("float32")},
			},
		},
		{
			"variadic",
			types.NewSignature(
				nil,
				types.NewTuple(mkParam("a", types.NewSlice(types.Typ[types.Int32]))),
				types.NewTuple(),
				true,
			),
			&Func{
				Input:      []Type{repeated(NewBasic("int32"))},
				Output:     make([]Type, 0),
				IsVariadic: true,
			},
		},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, scanFunc(&Func{}, c.signature), c.name)
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

	require.Equal(5, len(pkg.Structs), "pkg")
	assertStruct(t, findStructByName("Bar", pkg.Structs), "Bar", true, "Bar", "Baz")
	assertStruct(t, findStructByName("Foo", pkg.Structs), "Foo", true, "Bar", "Baz", "IntList", "IntArray", "Map", "AliasedMap", "Timestamp", "External", "Duration", "Aliased")
	assertStruct(t, findStructByName("Qux", pkg.Structs), "Qux", false, "A", "B")
	assertStruct(t, findStructByName("Saz", pkg.Structs), "Saz", true, "Point", "Foo")
	assertStruct(t, findStructByName("Jur", pkg.Structs), "Jur", false, "A")

	require.Equal(3, len(subpkg.Structs), "subpkg")
	assertStruct(t, findStructByName("MyContainer", subpkg.Structs), "MyContainer", false)
	assertStruct(t, findStructByName("NotGenerated", subpkg.Structs), "NotGenerated", false)
	assertStruct(t, findStructByName("Point", subpkg.Structs), "Point", true, "X", "Y")

	_, ok := pkg.Aliases[fmt.Sprintf("%s.%s", projectPath("fixtures"), "Baz")]
	require.False(ok, "Baz should not be an alias anymore")

	require.Equal(1, len(pkg.Enums), "pkg enums")
	require.Equal("Baz", pkg.Enums[0].Name)

	assertEnumValues(t, pkg.Enums[0].Values, "ABaz", "BBaz", "CBaz", "DBaz")

	require.Equal(0, len(pkg.Funcs), "pkg funcs")
	require.Equal(4, len(subpkg.Funcs), "subpkg funcs")
	assertFunc(t, findFuncByName("Generated", subpkg.Funcs), "Generated", "", []string{"string"}, []string{"bool", "error"}, false)
	assertFunc(t, findFuncByName("GeneratedMethod", subpkg.Funcs), "GeneratedMethod", "Point", []string{"int32"}, []string{"Point"}, false)
	assertFunc(t, findFuncByName("GeneratedMethodOnPointer", subpkg.Funcs), "GeneratedMethodOnPointer", "Point", []string{"bool"}, []string{"Point"}, false)
	assertFunc(t, findFuncByName("Name", subpkg.Funcs), "Name", "MyContainer", []string{}, []string{"string"}, false)
}

func assertEnumValues(t *testing.T, values []*EnumValue, expected ...string) {
	require := require.New(t)
	require.Len(values, len(expected), "expected same enum values")
	for i := range values {
		require.Equal(expected[i], values[i].Name, "expected same enum value name")
		require.Equal(fmt.Sprintf("%s ...", values[i].Name), strings.TrimSpace(strings.Join(values[i].Doc, "\n")))
	}

}

func findFuncByName(name string, fns []*Func) *Func {
	for _, f := range fns {
		if f.Name == name {
			return f
		}
	}

	return nil
}

func findStructByName(name string, coll []*Struct) *Struct {
	for _, s := range coll {
		if s.Name == name {
			return s
		}
	}

	return nil
}

func assertStruct(t *testing.T, s *Struct, name string, generate bool, fields ...string) {
	require.Equal(
		t,
		name,
		s.Name,
		"struct name",
	)
	require.Equal(t, generate, s.Generate, "struct has been asked to be generated (before resolver)")

	require.Equal(t, len(fields), len(s.Fields), "should have same struct fields")
	for _, f := range fields {
		require.True(t, s.HasField(f), "should have struct field %q", f)
	}

	doc := strings.TrimSpace(strings.Join(s.Doc, "\n"))
	require.True(t, strings.HasPrefix(doc, s.Name), "Doc for %s starts with its name", s.Name)
}

func assertFunc(t *testing.T, fn *Func, name string, recv string, input []string, result []string, variadic bool) {
	require.Equal(t, name, fn.Name, "func name")

	if fn.Receiver != nil {
		require.Equal(t, recv, typeFrom(fn.Receiver), "receiver")
	}

	for idx, in := range fn.Input {
		require.Equal(t, input[idx], typeFrom(in), fmt.Sprintf("input %d", idx))
	}

	for idx, out := range fn.Output {
		require.Equal(t, result[idx], typeFrom(out), fmt.Sprintf("output %d", idx))
	}

	require.Equal(t, variadic, fn.IsVariadic, "is variadic")
	require.Equal(t, fmt.Sprintf("%s ...", fn.Name), strings.TrimSpace(strings.Join(fn.Doc, "\n")))
}

func typeFrom(t Type) string {
	switch t.(type) {
	case *Named:
		return t.(*Named).Name
	case *Basic:
		return t.(*Basic).Name
	}

	return ""
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

func mkParam(name string, typ types.Type) *types.Var {
	return types.NewParam(
		token.NoPos,
		types.NewPackage("/foo", "mock"),
		name,
		typ,
	)
}

func repeated(t Type) Type {
	t.SetRepeated(true)
	return t
}

func nullable(t Type) Type {
	t.SetNullable(true)
	return t
}

func newNamedWithUnderlying(path, name string, underlying types.Type) types.Type {
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
