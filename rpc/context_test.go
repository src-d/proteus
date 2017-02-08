package rpc

import (
	"fmt"
	"testing"

	"github.com/src-d/proteus/protobuf"
	"github.com/stretchr/testify/assert"

	"gopkg.in/src-d/go-parse-utils.v1"
)

func TestContext_isNameDefined(t *testing.T) {
	pkg, err := parseutil.NewImporter().Import("github.com/src-d/proteus/fixtures")
	if err != nil {
		assert.Fail(t, fmt.Sprintf("could not import project fixtures: %v", err))
	}
	ctx := &context{pkg: pkg}

	assert.True(t, ctx.isNameDefined("Foo"), "Generator is defined")
	assert.False(t, ctx.isNameDefined("supercalifragilisticexpialidocious"), "this pakage has something to say")
}

func TestContext_findMessage(t *testing.T) {
	ctx := &context{
		proto: &protobuf.Package{
			Messages: []*protobuf.Message{
				&protobuf.Message{Name: "a"},
			},
		},
	}

	var msg *protobuf.Message
	msg = ctx.findMessage("a")
	assert.NotNil(t, msg)

	msg = ctx.findMessage("nothere")
	assert.Nil(t, msg)
}

func TestContext_addImport(t *testing.T) {
	ctx := &context{}

	assert.Equal(t, 0, len(ctx.imports))
	ctx.addImport("my-path")
	assert.Equal(t, 1, len(ctx.imports))

	ctx.addImport("other-path")
	assert.Equal(t, 2, len(ctx.imports), "adds other element")

	ctx.addImport("my-path")
	assert.Equal(t, 2, len(ctx.imports), "does not addd the same element twice")
}
