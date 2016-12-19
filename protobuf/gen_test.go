package protobuf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestGenerator(t *testing.T) {
	suite.Run(t, new(GenSuite))
}

type GenSuite struct {
	suite.Suite
	path string
	g    *Generator
	buf  *bytes.Buffer
}

func (s *GenSuite) SetupTest() {
	var err error
	s.path, err = ioutil.TempDir("", "proteus")
	s.Nil(err)

	s.buf = bytes.NewBuffer(nil)
	s.g = NewGenerator(s.path)
}

func (s *GenSuite) TearDownTest() {
	s.Nil(os.RemoveAll(s.path))
}

const expectedOptions = `	option bar = true;
	option foo = "bar";
`

func (s *GenSuite) TestWriteOptions() {
	writeOptions(s.buf, Options{
		"foo": NewStringValue("bar"),
		"bar": NewLiteralValue("true"),
	})

	s.Equal(expectedOptions, s.buf.String())
}

const expectedEnum = `enum PonyRace {
	option is_cute = true;
	PINK_CUTIE = 0;
	RED_FURY = 1;
}
`

var mockEnum = &Enum{
	Name: "PonyRace",
	Options: Options{
		"is_cute": NewLiteralValue("true"),
	},
	Values: EnumValues{
		&EnumValue{Name: "PINK_CUTIE", Value: 0},
		&EnumValue{Name: "RED_FURY", Value: 1},
	},
}

func (s *GenSuite) TestWriteEnum() {
	writeEnum(s.buf, mockEnum)
	s.Equal(expectedEnum, s.buf.String())
}

const expectedMsg = `message Pony {
	option is_cute = true;
	string name = 1;
	google.protobuf.Timestamp born_at = 2;
	foo.bar.PonyRace race = 3;
	repeated string nick_names = 4;
}
`

var mockMsg = &Message{
	Name: "Pony",
	Options: Options{
		"is_cute": NewLiteralValue("true"),
	},
	Fields: []*Field{
		&Field{
			Name: "name",
			Type: NewBasic("string"),
			Pos:  1,
		},
		&Field{
			Name: "born_at",
			Type: NewNamed("google.protobuf", "Timestamp"),
			Pos:  2,
		},
		&Field{
			Name: "race",
			Type: NewNamed("foo.bar", "PonyRace"),
			Pos:  3,
		},
		&Field{
			Name:     "nick_names",
			Repeated: true,
			Type:     NewBasic("string"),
			Pos:      4,
		},
	},
}

func (s *GenSuite) TestWriteMessage() {
	writeMessage(s.buf, mockMsg)
	s.Equal(expectedMsg, s.buf.String())
}

var expectedProto = fmt.Sprintf(`syntax = "proto3";
package foo.bar;

import "google/protobuf/timestamp.proto";

%s
%s
`, expectedMsg, expectedEnum)

func (s *GenSuite) TestGenerate() {
	err := s.g.Generate(&Package{
		Name:     "foo.bar",
		Imports:  []string{"google/protobuf/timestamp.proto"},
		Messages: []*Message{mockMsg},
		Enums:    []*Enum{mockEnum},
	})
	s.Nil(err)

	bytes, err := ioutil.ReadFile(filepath.Join(s.path, "generated.proto"))
	s.Nil(err)

	s.Equal(expectedProto, string(bytes))
}
