package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseType(t *testing.T) {
	typ := newBaseType()
	name := "BaseType"

	assertRepeatable(t, typ, name)
	assertNullable(t, typ, name)
	assert.Panics(t, func() { typ.TypeString() }, "does not implement TypeString")
	assert.Panics(t, func() { typ.String() }, "does not implement String")
	assert.Panics(t, func() { typ.UnqualifiedName() }, "does not implement UnqualifiedName")
}

func TestBasic(t *testing.T) {
	typ := NewBasic("basic")
	name := "Basic"

	assertRepeatable(t, typ, name)

	assert.True(t, typ.IsNullable(), "Basic is nullable by default")
	typ.SetNullable(false)
	assert.True(t, typ.IsNullable(), "Basic cannot be set not nullable")

	assert.Equal(t, "basic", typ.String(), "Basic.String returns type's name")
	assert.Equal(t, "basic", typ.TypeString(), "Basic.TypeString returns type's name")
	assert.Equal(t, "basic", typ.UnqualifiedName(), "Basic.UnqualifiedName returns type's name")
}

func TestNamed_withPath(t *testing.T) {
	typ := NewNamed("pkg", "name")
	name := "Named (with path)"

	assertRepeatable(t, typ, name)
	assertNullable(t, typ, name)

	assert.Equal(t, "pkg.name", typ.String(), "Named.String() (with path) returns type's name with path")
	assert.Equal(t, "pkg.name", typ.TypeString(), "Named.TypeString() (with path) returns type's name with path")
	assert.Equal(t, "name", typ.UnqualifiedName(), "Named.UnqualifiedName() (with path) returns type's name without path")
}

func TestNamed_withoutPath(t *testing.T) {
	typ := NewNamed("", "name")
	name := "Named (without path)"

	assertRepeatable(t, typ, name)
	assertNullable(t, typ, name)

	assert.Equal(t, "name", typ.String(), "Named.String() (with path) returns type's name without path")
	assert.Equal(t, "name", typ.TypeString(), "Named.TypeString() (with path) returns type's name without path")
	assert.Equal(t, "name", typ.UnqualifiedName(), "Named.UnqualifiedName() (with path) returns type's name without path")
}

func TestAlias_IsNullable(t *testing.T) {
	typ := NewAlias(newBaseType(), newBaseType()).(*Alias)

	assert.False(t, typ.IsNullable(), "Alias is not nullable if neither the type nor the underlying is nullable")
	typ.Type.SetNullable(true)
	typ.Underlying.SetNullable(false)
	assert.True(t, typ.IsNullable(), "Alias is nullable if the type is nullable but the underlying is not")
	typ.Type.SetNullable(false)
	typ.Underlying.SetNullable(true)
	assert.True(t, typ.IsNullable(), "Alias is nullable if the type is not nullable but the underlying is")
	typ.Type.SetNullable(true)
	typ.Underlying.SetNullable(true)
	assert.True(t, typ.IsNullable(), "Alias is nullable if both the type and the underlying are")
}

func TestAlias_IsRepeated(t *testing.T) {
	typ := NewAlias(newBaseType(), newBaseType()).(*Alias)

	assert.False(t, typ.IsRepeated(), "Alias is not repeated if neither the type nor the underlying is repeated")

	typ.Type.SetRepeated(true)
	typ.Underlying.SetRepeated(false)
	assert.True(t, typ.IsRepeated(), "Alias is repeated if the type is repeated but the underlying is not")
	typ.Type.SetRepeated(false)
	typ.Underlying.SetRepeated(true)
	assert.True(t, typ.IsRepeated(), "Alias is repeated if the type is not repeated but the underlying is")
	typ.Type.SetRepeated(true)
	typ.Underlying.SetRepeated(true)
	assert.True(t, typ.IsRepeated(), "Alias is repeated if both the type and the underlying are")
}

func TestAlias_stringMethods(t *testing.T) {
	typ := NewAlias(NewNamed("", "Aliasing"), NewBasic("string"))

	assert.Equal(t, "type Aliasing string", typ.String(), "Alias.String returns a type declaration for the alias")
	assert.Equal(t, "Aliasing", typ.TypeString(), "Alias.TypeString returns the type string for the alias")
	assert.Equal(t, "Aliasing", typ.UnqualifiedName(), "Alias.UnqualifiedName returns the unqualified name of the alias")
}

func TestMap(t *testing.T) {
	typ := NewMap(NewBasic("string"), NewBasic("int"))
	name := "Map"

	assertRepeatable(t, typ, name)
	assertNullable(t, typ, name)
	assert.Equal(t, "map[string]int", typ.String(), "Map.String returns a map signature")
	assert.Equal(t, "map[string]int", typ.TypeString(), "Map.TypeString returns a map signature")
	assert.Equal(t, "map[string]int", typ.UnqualifiedName(), "Map.UnqualifiedName returns a map signature")
}

// assertRepeatable asserts a type respond as expected to IsRepeated and SetRepeated.
func assertRepeatable(t *testing.T, typ Type, name string) {
	assert.False(t, typ.IsRepeated(), "%s is not repeated by default", name)
	typ.SetRepeated(true)
	assert.True(t, typ.IsRepeated(), "%s can be set as repeated", name)
	typ.SetRepeated(false)
	assert.False(t, typ.IsRepeated(), "%s can be set as not repeated", name)
}

// assertNullable asserts a type responds as expected to IsNullable and SetNullable.
func assertNullable(t *testing.T, typ Type, name string) {
	assert.False(t, typ.IsNullable(), "%s is not nullable by default", name)
	typ.SetNullable(true)
	assert.True(t, typ.IsNullable(), "%s can be set as nullable", name)
	typ.SetNullable(false)
	assert.False(t, typ.IsNullable(), "%s can be set as not nullable", name)
}
