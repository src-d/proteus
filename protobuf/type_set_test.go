package protobuf

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeSet_Add(t *testing.T) {
	ts := NewTypeSet()
	assert.Equal(t, 0, ts.Len())

	res := ts.Add("github.com/src-d/proteus/protobuf", "TypeSet")
	assert.True(t, res, "element was added")
	assert.Equal(t, 1, ts.Len(), "contains one element")

	res = ts.Add("github.com/src-d/proteus/protobuf", "Transformer")
	assert.True(t, res, "another element in the same package can be added")
	assert.Equal(t, 2, ts.Len(), "contains two elements")

	res = ts.Add("github.com/src-d/proteus/protobuf", "TypeSet")
	assert.False(t, res, "adding an element twice returns false")
	assert.Equal(t, 2, ts.Len(), "there is no new element")

	res = ts.Add("github.com/src-d/proteus/resolver", "Resolver")
	assert.True(t, res, "adding an element in a new package")
	assert.Equal(t, 3, ts.Len(), "a new element was added")
}

func ExampleTypeSet() {
	ts := NewTypeSet()

	// Returns whether the item was added or not. If false, it means the item was
	// already there.
	res := ts.Add("github.com/src-d/proteus/protobuf", "TypeSet")
	fmt.Println(res)

	res = ts.Add("github.com/src-d/proteus/protobuf", "TypeSet")
	fmt.Println(res)
	fmt.Println(ts.Len())

	fmt.Println(ts.Contains("github.com/src-d/proteus/protobuf", "TypeSet"))
	fmt.Println(ts.Len())
	// Output: true
	// false
	// 1
	// true
	// 1
}

func TestTypeSet_Contains(t *testing.T) {
	ts := NewTypeSet()
	assert.Equal(t, 0, ts.Len())

	res := ts.Add("github.com/src-d/proteus/protobuf", "TypeSet")
	assert.True(t, res, "element was added")
	res = ts.Add("github.com/src-d/proteus/protobuf", "Type")
	assert.True(t, res, "second element was added")
	res = ts.Add("github.com/src-d/proteus/resolver", "Resolver")
	assert.True(t, res, "adding an element in a new package")

	assert.True(t, ts.Contains("github.com/src-d/proteus/protobuf", "Type"), "contains protobuf.Type")
	assert.True(t, ts.Contains("github.com/src-d/proteus/protobuf", "TypeSet"), "contains protobuf.TypeSet")
	assert.True(t, ts.Contains("github.com/src-d/proteus/resolver", "Resolver"), "contains resolver.Resolver")

	assert.False(t, ts.Contains("github.com/src-d/proteus/protobuf", "NotType"), "does not contain protobuf.NotType")
	assert.False(t, ts.Contains("github.com/src-d/proteus/resolver", "NotType"), "does not contain resolver.NotType")
	assert.False(t, ts.Contains("github.com/src-d/proteus/notpackage", "NotType"), "does not contain notpackage.NotType")
}
