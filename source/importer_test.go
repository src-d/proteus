package source

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const project = "github.com/src-d/proteus"

func projectPath(pkg string) string {
	return filepath.Join(goPath, "src", project, pkg)
}

func TestGetSourceFiles(t *testing.T) {
	_, paths, err := NewImporter().getSourceFiles(filepath.Join(project, "fixtures"), goPath)
	require.Nil(t, err)
	expected := []string{
		projectPath("fixtures/bar.go"),
		projectPath("fixtures/foo.go"),
	}

	require.Equal(t, expected, paths)
}

func TestParseSourceFiles(t *testing.T) {
	paths := []string{
		projectPath("fixtures/bar.go"),
		projectPath("fixtures/foo.go"),
	}

	pkg, err := NewImporter().parseSourceFiles(projectPath("fixtures"), paths)
	require.Nil(t, err)

	require.Equal(t, "foo", pkg.Name())
}
