package protobuf

import (
	"testing"

	"gopkg.in/src-d/proteus.v1/scanner"

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

	pkg.Import(&ProtoType{})
	require.Equal(0, len(pkg.Imports))

	pkg.Import(&ProtoType{Import: "foo"})
	require.Equal(1, len(pkg.Imports))

	pkg.Import(&ProtoType{Import: "foo"})
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

func TestIsNullable(t *testing.T) {
	var typ Type
	nullableSource := nullable(scanner.NewBasic("string"))
	require.True(t, nullableSource.IsNullable(), "nullable source is nullable")
	notNullableSource := scanner.NewNamed("foo", "Baz")
	require.False(t, notNullableSource.IsNullable(), "not nullable source is not nullable")
	nullableType := NewNamed("nullable", "Type")
	nullableType.SetSource(nullableSource)
	require.True(t, nullableType.IsNullable(), "nullable type is nullable")
	notNullableType := NewNamed("notnullable", "Type")
	notNullableType.SetSource(notNullableSource)
	require.False(t, notNullableType.IsNullable(), "not nullable type is not nullable")

	// Named
	typ = NewNamed("foo", "Bar")
	require.True(t, typ.IsNullable(), "Named without Source is nullable")
	typ.SetSource(nullableSource)
	require.True(t, typ.IsNullable(), "Named with nullable Source is nullable")
	typ.SetSource(notNullableSource)
	require.False(t, typ.IsNullable(), "Named with not nullable Source is not nullable")

	// Alias
	typ = NewAlias(nullableType, nullableType)
	require.True(t, typ.IsNullable(), "nullable Type and nullable Underlying without Source makes alias nullable")
	typ = NewAlias(nullableType, notNullableType)
	require.True(t, typ.IsNullable(), "nullable Type and not nullable Underlying without Source makes alias nullable")
	typ = NewAlias(notNullableType, nullableType)
	require.False(t, typ.IsNullable(), "not nullable Type and nullable Underlying without Source makes alias not nullable")
	typ = NewAlias(notNullableType, notNullableType)
	require.False(t, typ.IsNullable(), "not nullable Type and not nullable Underlying without Source makes alias not nullable")

	typ = NewAlias(nullableType, nullableType)
	typ.SetSource(nullableSource)
	require.True(t, typ.IsNullable(), "nullable Type and nullable Underlying with nullable Source makes alias nullable")
	typ.SetSource(notNullableSource)
	require.False(t, typ.IsNullable(), "nullable Type and nullable Underlying with not nullable Source makes alias not nullable")
	typ = NewAlias(notNullableType, nullableType)
	typ.SetSource(nullableSource)
	require.True(t, typ.IsNullable(), "not nullable Type and nullable Underlying with nullable Source makes alias nullable")
	typ = NewAlias(notNullableType, nullableType)
	typ.SetSource(notNullableSource)
	require.False(t, typ.IsNullable(), "not nullable Type and nullable Underlying with not nullable Source makes alias not nullable")

	// Basic
	typ = NewBasic("string")
	require.False(t, typ.IsNullable(), "Basic types are never nullable")

	// Map
	typ = NewMap(notNullableType, nullableType)
	require.True(t, typ.IsNullable(), "map<notNullable>*Nullable is nullable")
	typ = NewMap(notNullableType, notNullableType)
	require.False(t, typ.IsNullable(), "map<notNullable>NotNullable is not nullable")
}
