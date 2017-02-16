package protobuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_timeTimeDecorator(t *testing.T) {
	f := new(Field)
	(*DefaultMappings["time.Time"]).Decorators.Run(&Package{}, &Message{}, f)

	assert.Equal(t, NewLiteralValue("true"), f.Options["(gogoproto.stdtime)"])
}

func Test_timeDurationDecorator(t *testing.T) {
	f := new(Field)
	(*DefaultMappings["time.Duration"]).Decorators.Run(&Package{}, &Message{}, f)

	assert.Equal(t, NewLiteralValue("true"), f.Options["(gogoproto.stdduration)"])
}

func TestToGoOutPath(t *testing.T) {
	// Empty case
	assert.Equal(t, "", TypeMappings{}.ToGoOutPath())

	// Only one and invalid
	assert.Equal(t, "", TypeMappings{
		"a": &ProtoType{},
	}.ToGoOutPath()) // No Immport and no GoImport
	assert.Equal(t, "", TypeMappings{
		"a": &ProtoType{GoImport: "github.com/src-d/proteus"},
	}.ToGoOutPath()) // No Immport
	assert.Equal(t, "", TypeMappings{
		"a": &ProtoType{Import: "src-d/proteus"},
	}.ToGoOutPath()) // No GoImmport

	// Only one and valid
	assert.Equal(t, "Mimport=goimport", TypeMappings{
		"a": &ProtoType{Import: "import", GoImport: "goimport"},
	}.ToGoOutPath())

	// Several
	assert.Equal(t, "Ma=1,Mb=2,Mc=3", TypeMappings{
		"typA": &ProtoType{Import: "a", GoImport: "1"},
		"typB": &ProtoType{Import: "b", GoImport: "2"},
		"typC": &ProtoType{Import: "c", GoImport: "3"},
	}.ToGoOutPath())
}
