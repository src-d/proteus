package protobuf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReserve(t *testing.T) {
	msg := new(Message)
	msg.Reserve(1)
	msg.Reserve(1)
	require.Equal(t, []uint{1}, msg.Reserved)
}

func TestImport(t *testing.T) {
	pkg := new(Package)
	require := require.New(t)

	pkg.Import(&ProtobufType{})
	require.Equal(0, len(pkg.Imports))

	pkg.Import(&ProtobufType{Import: "foo"})
	require.Equal(1, len(pkg.Imports))

	pkg.Import(&ProtobufType{Import: "foo"})
	require.Equal(1, len(pkg.Imports))
}

func TestImportFromPath(t *testing.T) {
	pkg := &Package{Path: "foo"}
	require := require.New(t)

	pkg.ImportFromPath("foo")
	require.Equal(0, len(pkg.Imports))

	pkg.ImportFromPath("bar")
	require.Equal(1, len(pkg.Imports))
	require.Equal("bar/generated.proto", pkg.Imports[0])
}

func TestTypesString(t *testing.T) {
	require.Equal(t, "int32", NewBasic("int32").String())
	require.Equal(t, "foo.Bar", NewNamed("foo", "Bar").String())
	require.Equal(t, "map<string, int32>", NewMap(
		NewBasic("string"),
		NewBasic("int32"),
	).String())
}

func TestOptionsString(t *testing.T) {
	require.Equal(t, "foo", NewLiteralValue("foo").String())
	require.Equal(t, `"bar"`, NewStringValue("bar").String())
}
