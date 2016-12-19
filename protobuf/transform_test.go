package protobuf

import (
	"path/filepath"
	"testing"

	"github.com/src-d/proteus/resolver"
	"github.com/src-d/proteus/scanner"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestToLowerSnakeCase(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"fooBarBaz", "foo_bar_baz"},
		{"FooBarBaz", "foo_bar_baz"},
		{"foo1barBaz", "foo1bar_baz"},
		{"fooBAR", "foo_bar"},
		{"FBar", "fbar"},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, toLowerSnakeCase(c.input))
	}
}

func TestToUpperSnakeCase(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"FooBarBaz", "FOO_BAR_BAZ"},
		{"fooBarBaz", "FOO_BAR_BAZ"},
		{"foo1barBaz", "FOO1BAR_BAZ"},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, toUpperSnakeCase(c.input))
	}
}

func TestIsByteSlice(t *testing.T) {
	cases := []struct {
		t      scanner.Type
		result bool
	}{
		{scanner.NewBasic("byte"), false},
		{repeated(scanner.NewBasic("byte")), true},
		{scanner.NewNamed("foo", "Bar"), false},
	}

	for _, c := range cases {
		require.Equal(t, c.result, isByteSlice(c.t))
	}
}

func TestToProtobufPkg(t *testing.T) {
	cases := []struct {
		path string
		pkg  string
	}{
		{"foo", "foo"},
		{"net/url", "net.url"},
		{"github.com/foo/bar", "github.com.foo.bar"},
		{"github.cóm/fòo/bar", "github.com.foo.bar"},
		{"gopkg.in/go-foo/foo.v1", "gopkg.in.gofoo.foo.v1"},
	}

	for _, c := range cases {
		require.Equal(t, c.pkg, toProtobufPkg(c.path))
	}
}

type TransformerSuite struct {
	suite.Suite
	t *Transformer
}

func (s *TransformerSuite) SetupTest() {
	s.t = NewTransformer()
	s.t.SetMappings(TypeMappings{
		"url.URL":       &ProtobufType{Name: "string", Basic: true},
		"time.Duration": &ProtobufType{Name: "uint64", Basic: true},
	})
	s.t.SetMappings(nil)
	s.NotNil(s.t.mappings)
}

func (s *TransformerSuite) TestFindMapping() {
	cases := []struct {
		name         string
		protobufType string
		isNil        bool
	}{
		{"foo.Bar", "", true},
		{"url.URL", "string", false},
		{"time.Duration", "uint64", false},
		{"time.Time", "Timestamp", false},
	}

	for _, c := range cases {
		t := s.t.findMapping(c.name)
		if c.isNil {
			s.Nil(t)
		} else {
			s.NotNil(t)
			s.Equal(c.protobufType, t.Name)
		}
	}
}

func (s *TransformerSuite) TestTransformType() {
	cases := []struct {
		typ      scanner.Type
		expected Type
		imported string
	}{
		{
			scanner.NewNamed("time", "Time"),
			NewNamed("google.protobuf", "Timestamp"),
			"google/protobuf/timestamp.proto",
		},
		{
			scanner.NewNamed("foo", "Bar"),
			NewNamed("foo", "Bar"),
			"foo/generated.proto",
		},
		{
			scanner.NewBasic("string"),
			NewBasic("string"),
			"",
		},
		{
			scanner.NewMap(
				scanner.NewBasic("string"),
				scanner.NewBasic("int64"),
			),
			NewMap(NewBasic("string"), NewBasic("int64")),
			"",
		},
		{
			repeated(scanner.NewBasic("int")),
			NewBasic("int32"),
			"",
		},
	}

	for _, c := range cases {
		var pkg Package
		t := s.t.transformType(&pkg, c.typ)
		s.Equal(c.expected, t)

		if c.imported != "" {
			s.Equal(1, len(pkg.Imports))
			s.Equal(c.imported, pkg.Imports[0])
		}
	}
}

func (s *TransformerSuite) TestTransformField() {
	cases := []struct {
		name     string
		typ      scanner.Type
		expected *Field
	}{
		{
			"Foo",
			scanner.NewBasic("int"),
			&Field{Name: "foo", Type: NewBasic("int32"), Nullable: true},
		},
		{
			"Bar",
			repeated(scanner.NewBasic("byte")),
			&Field{Name: "bar", Type: NewBasic("bytes"), Nullable: true},
		},
		{
			"BazBar",
			repeated(scanner.NewBasic("int")),
			&Field{Name: "baz_bar", Type: NewBasic("int32"), Repeated: true, Nullable: true},
		},
		{
			"Invalid",
			scanner.NewBasic("complex64"),
			nil,
		},
	}

	for _, c := range cases {
		f := s.t.transformField(&Package{}, &scanner.Field{
			Name: c.name,
			Type: c.typ,
		}, 0)
		s.Equal(c.expected, f, c.name)
	}
}

func (s *TransformerSuite) TestTransformStruct() {
	st := &scanner.Struct{
		Name: "Foo",
		Fields: []*scanner.Field{
			&scanner.Field{
				Name: "Invalid",
				Type: scanner.NewBasic("complex64"),
			},
			&scanner.Field{
				Name: "Bar",
				Type: scanner.NewBasic("string"),
			},
		},
	}

	msg := s.t.transformStruct(&Package{}, st)
	s.Equal("Foo", msg.Name)
	s.Equal(1, len(msg.Fields), "should have one field")
	s.Equal(2, msg.Fields[0].Pos)
	s.Equal(1, len(msg.Reserved), "should have reserved field")
	s.Equal(1, msg.Reserved[0])
}

func (s *TransformerSuite) TestTransformEnum() {
	enum := s.t.transformEnum(&scanner.Enum{
		Name:   "Foo",
		Values: []string{"Foo", "Bar", "BarBaz"},
	})

	s.Equal("Foo", enum.Name)
	s.Equal(3, len(enum.Values), "should have same number of values")
	s.assertEnumVal(enum.Values[0], "FOO", 0)
	s.assertEnumVal(enum.Values[1], "BAR", 1)
	s.assertEnumVal(enum.Values[2], "BAR_BAZ", 2)
}

func (s *TransformerSuite) TestTransform() {
	pkgs := s.fixtures()
	pkg := s.t.Transform(pkgs[0])

	s.Equal("github.com.srcd.proteus.fixtures", pkg.Name)
	s.Equal("github.com/src-d/proteus/fixtures", pkg.Path)
	s.Equal([]string{
		"google/protobuf/timestamp.proto",
		"github.com/src-d/proteus/fixtures/subpkg/generated.proto",
	}, pkg.Imports)
	s.Equal(1, len(pkg.Enums))
	s.Equal(4, len(pkg.Messages))

	pkg = s.t.Transform(pkgs[1])
	s.Equal("github.com.srcd.proteus.fixtures.subpkg", pkg.Name)
	s.Equal("github.com/src-d/proteus/fixtures/subpkg", pkg.Path)
	s.Equal([]string(nil), pkg.Imports)
	s.Equal(0, len(pkg.Enums))
	s.Equal(1, len(pkg.Messages))
}

func (s *TransformerSuite) fixtures() []*scanner.Package {
	sc, err := scanner.New(projectPath("fixtures"), projectPath("fixtures/subpkg"))
	s.Nil(err)
	pkgs, err := sc.Scan()
	s.Nil(err)
	resolver.New().Resolve(resolver.Packages(pkgs))
	return pkgs
}

func (s *TransformerSuite) assertEnumVal(v *EnumValue, name string, val uint) {
	s.Equal(name, v.Name)
	s.Equal(val, v.Value)
}

func TestTransformer(t *testing.T) {
	suite.Run(t, new(TransformerSuite))
}

func repeated(t scanner.Type) scanner.Type {
	t.SetRepeated(true)
	return t
}

const project = "github.com/src-d/proteus"

func projectPath(pkg string) string {
	return filepath.Join(project, pkg)
}
