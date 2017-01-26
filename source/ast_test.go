package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackageAST(t *testing.T) {
	pkg, err := PackageAST("github.com/src-d/proteus/fixtures")
	require.Nil(t, err)
	require.Equal(t, "foo", pkg.Name)
}
