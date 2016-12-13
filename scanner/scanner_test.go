package scanner

import (
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var gopath = os.Getenv("GOPATH")

const project = "github.com/src-d/protogo"

func TestGetSourceFiles(t *testing.T) {
	paths, err := getSourceFiles(projectPath("fixtures/scanner"))
	require.Nil(t, err)
	expected := []string{
		projectPath("fixtures/scanner/bar.go"),
		projectPath("fixtures/scanner/foo.go"),
	}

	require.Equal(t, expected, paths)
}

func projectPath(pkg string) string {
	return filepath.Join(gopath, "src", project, pkg)
}

func TestParseSourceFiles(t *testing.T) {
	paths := []string{
		projectPath("fixtures/scanner/bar.go"),
		projectPath("fixtures/scanner/foo.go"),
	}

	pkg, err := parseSourceFiles(projectPath("fixtures/scanner"), paths)
	require.Nil(t, err)

	require.Equal(t, "foo", pkg.Name())
}

func TestProcessType(t *testing.T) {
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
		require.Equal(t, c.expected, processType(c.typ), c.name)
	}
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
