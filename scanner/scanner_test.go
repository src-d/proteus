package scanner

import (
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
