package protobuf

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/proteus.v1/report"
	"gopkg.in/src-d/proteus.v1/resolver"
	"gopkg.in/src-d/proteus.v1/scanner"
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
	report.TestMode()
	s.t = NewTransformer()
	s.t.SetMappings(TypeMappings{
		"url.URL":       &ProtoType{Name: "string", Basic: true},
		"time.Duration": &ProtoType{Name: "uint64", Basic: true},
	})
	s.t.SetMappings(nil)
	s.NotNil(s.t.mappings)
}

func (s *TransformerSuite) TearDownTest() {
	report.EndTestMode()
}

func (s *TransformerSuite) TestIsEnum() {
	ts := NewTypeSet()
	ts.Add("paquete", "Tipo")
	ts.Add("package", "Type")
	s.t.SetEnumSet(ts)

	s.True(s.t.IsEnum("paquete", "Tipo"), "paquete.Tipo is an enum")
	s.True(s.t.IsEnum("package", "Type"), "package.Type is an enum")
	s.False(s.t.IsEnum("package", "Tipo"), "package.Tipo is not an enum")
	s.False(s.t.IsEnum("paquete", "Type"), "paquete.Type is not an enum")
}

func (s *TransformerSuite) TestIsStruct() {
	ts := NewTypeSet()
	ts.Add("paquete", "Tipo")
	ts.Add("package", "Type")
	s.t.SetStructSet(ts)

	s.True(s.t.IsStruct("paquete", "Tipo"), "paquete.Tipo is an enum")
	s.True(s.t.IsStruct("package", "Type"), "package.Type is an enum")
	s.False(s.t.IsStruct("package", "Tipo"), "package.Tipo is not an enum")
	s.False(s.t.IsStruct("paquete", "Type"), "paquete.Type is not an enum")
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

func (s *TransformerSuite) TestFindMappingWithWarn() {
	s.t.SetMappings(TypeMappings{
		"url.URL": &ProtoType{Name: "string", Basic: true},
		"int16":   &ProtoType{Name: "int32", Basic: true, Warn: "%s upgraded to int32"},
	})
	cases := []struct {
		name string
		typ  string
		warn string
	}{
		{
			"Without Warning",
			"url.URL",
			"",
		},
		{
			"With Warning",
			"int16",
			"upgraded to int32",
		},
	}

	for _, c := range cases {
		_ = s.t.findMapping(c.typ)
		stack := report.MessageStack()
		if c.warn == "" {
			s.Empty(stack)
		} else {
			s.NotEmpty(
				stack,
				fmt.Sprintf("stack empty in %s", c.name),
			)
			s.True(
				strings.HasSuffix(stack[len(stack)-1], c.warn),
				fmt.Sprintf("last message does not match for %s:\nExpected '%s' to end with '%s'", c.name, c.warn, stack[len(stack)-1]),
			)
		}
	}
}

func (s *TransformerSuite) TestMappingDecorators() {
	s.t.SetMappings(TypeMappings{
		"int": &ProtoType{
			Name:  "int64",
			Basic: true,
			Decorators: NewDecorators(
				func(p *Package, m *Message, f *Field) {
					f.Options["greeting"] = NewStringValue("hola")
				},
			),
		},
	})

	f := s.t.transformField(&Package{}, &Message{}, &scanner.Field{
		Name: "MyField",
		Type: scanner.NewBasic("int"),
	}, 1)

	s.Equal(NewStringValue("hola"), f.Options["greeting"], "option was added")
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
			NewBasic("int64"),
			"",
		},
		{
			scanner.NewAlias(
				scanner.NewNamed("foo", "Bar"),
				scanner.NewBasic("string"),
			),
			NewAlias(
				NewNamed("foo", "Bar"),
				NewBasic("string"),
			),
			"foo/generated.proto",
		},
	}

	for _, c := range cases {
		var pkg Package
		t := s.t.transformType(&pkg, c.typ, &Message{}, &Field{})
		s.assertType(c.expected, t, "type")
		s.assertSource(t, c.typ)

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
			&Field{
				Name:    "foo",
				Type:    NewBasic("int64"),
				Options: Options{},
			},
		},
		{
			"Bar",
			repeated(scanner.NewBasic("byte")),
			&Field{
				Name:    "bar",
				Type:    NewBasic("bytes"),
				Options: Options{},
			},
		},
		{
			"BazBar",
			repeated(scanner.NewBasic("int")),
			&Field{
				Name:     "baz_bar",
				Type:     NewBasic("int64"),
				Repeated: true,
				Options:  Options{},
			},
		},
		{
			"CustomID",
			scanner.NewBasic("int"),
			&Field{
				Name: "custom_id",
				Type: NewBasic("int64"),
				Options: Options{
					"(gogoproto.customname)": NewStringValue("CustomID"),
				},
			},
		},
		{
			"NullableType",
			nullable(scanner.NewNamed("my/pckg", "hello")),
			&Field{
				Name:    "nullable_type",
				Type:    NewNamed("my.pckg", "hello"),
				Options: Options{},
			},
		},
		{
			"NonNullableType",
			scanner.NewNamed("my/pckg", "hello"),
			&Field{
				Name: "non_nullable_type",
				Type: NewNamed("my.pckg", "hello"),
				Options: Options{
					"(gogoproto.nullable)": NewLiteralValue("false"),
				},
			},
		},
		{
			"Invalid",
			scanner.NewBasic("complex64"),
			nil,
		},
		{
			"MyEnum",
			scanner.NewNamed("my/pckg", "MyEnum"),
			&Field{
				Name:    "my_enum",
				Type:    NewNamed("my.pckg", "MyEnum"),
				Options: Options{},
			},
		},
		{
			"MyAlias",
			scanner.NewAlias(
				scanner.NewNamed("my/pckg", "MyAlias"),
				scanner.NewBasic("string"),
			),
			&Field{
				Name: "my_alias",
				Type: NewAlias(
					NewNamed("my.pckg", "MyAlias"),
					NewBasic("string"),
				),
				Options: Options{
					"(gogoproto.casttype)": NewStringValue("my/pckg.MyAlias"),
				},
			},
		},
		{
			"MyRepeatedAlias",
			scanner.NewAlias(
				scanner.NewNamed("my/pckg", "MyRepeatedAlias"),
				repeated(scanner.NewBasic("string")),
			),
			&Field{
				Name: "my_repeated_alias",
				Type: NewAlias(
					NewNamed("my.pckg", "MyRepeatedAlias"),
					NewBasic("string"),
				),
				Options: Options{},
			},
		},
	}

	ts := NewTypeSet()
	ts.Add("my/pckg", "MyEnum")
	s.t.SetEnumSet(ts)

	for _, c := range cases {
		f := s.t.transformField(&Package{}, &Message{}, &scanner.Field{
			Name: c.name,
			Type: c.typ,
		}, 0)
		if c.expected == nil {
			s.Nil(f, c.name)
		} else {
			s.Equal(c.expected.Name, f.Name, fmt.Sprintf("Name in %s", c.name))
			s.assertType(c.expected.Type, f.Type, c.name)
			s.Equal(c.expected.Options, f.Options, fmt.Sprintf("Options in %s", c.name))
		}
	}
}

func (s *TransformerSuite) TestTransformStruct() {
	st := &scanner.Struct{
		Docs: mkDocs("fancy struct"),
		Name: "Foo",
		Fields: []*scanner.Field{
			{
				Docs: mkDocs("fancy invalid"),
				Name: "Invalid",
				Type: scanner.NewBasic("complex64"),
			},
			{
				Docs: mkDocs("fancy bar"),
				Name: "Bar",
				Type: scanner.NewBasic("string"),
			},
		},
	}

	msg := s.t.transformStruct(&Package{}, st)
	s.Equal("fancy struct", strings.Join(msg.Docs, "\n"))
	s.Equal("Foo", msg.Name)
	s.Equal(1, len(msg.Fields), "should have one field")
	s.Equal("fancy bar", strings.Join(msg.Fields[0].Docs, "\n"))
	s.Equal(2, msg.Fields[0].Pos)
	s.Equal(0, len(msg.Fields[0].Options))
	s.Equal(1, len(msg.Reserved), "should have reserved field")
	s.Equal(uint(1), msg.Reserved[0])
	s.Equal(NewLiteralValue("false"), msg.Options["(gogoproto.typedecl)"], "should drop declaration by default")
	s.Equal(NewLiteralValue("false"), msg.Options["(gogoproto.typedecl)"], "should drop declaration by default")
	s.NotContains(msg.Options, "(gogoproto.goproto_stringer)", "not contains goproto_stringer")
}

func (s *TransformerSuite) TestTransformStructIsStringer() {
	st := &scanner.Struct{
		Name: "Foo",
		Fields: []*scanner.Field{
			{
				Name: "Invalid",
				Type: scanner.NewBasic("complex64"),
			},
			{
				Name: "Bar",
				Type: scanner.NewBasic("string"),
			},
		},
		IsStringer: true,
	}

	msg := s.t.transformStruct(&Package{}, st)
	s.Equal("Foo", msg.Name)
	s.Equal(1, len(msg.Fields), "should have one field")
	s.Equal(2, msg.Fields[0].Pos)
	s.Equal(0, len(msg.Fields[0].Options))
	s.Equal(1, len(msg.Reserved), "should have reserved field")
	s.Equal(uint(1), msg.Reserved[0])
	s.Equal(NewLiteralValue("false"), msg.Options["(gogoproto.typedecl)"], "should drop declaration by default")
	s.Contains(msg.Options, "(gogoproto.goproto_stringer)", "contains goproto_stringer")
	s.Equal(NewLiteralValue("false"), msg.Options["(gogoproto.goproto_stringer)"], "goproto_stringer is false")
}

func (s *TransformerSuite) TestTransformFuncMultiple() {
	fn := &scanner.Func{
		Name: "DoFoo",
		Input: []scanner.Type{
			scanner.NewNamed("foo", "Bar"),
			scanner.NewBasic("int"),
		},
		Output: []scanner.Type{
			scanner.NewNamed("foo", "Foo"),
			scanner.NewBasic("bool"),
			scanner.NewNamed("", "error"),
		},
	}
	pkg := &Package{Path: "baz"}
	rpc := s.t.transformFunc(pkg, fn, nameSet{})

	s.NotNil(rpc)
	s.Equal(fn.Name, rpc.Name)
	s.assertType(NewGeneratedNamed("baz", "DoFooRequest"), rpc.Input, "rpc input")
	s.assertType(NewGeneratedNamed("baz", "DoFooResponse"), rpc.Output, "rpc output")

	s.Equal(2, len(pkg.Messages), "two messages should have been created")
	msg := pkg.Messages[0]
	s.Equal("DoFooRequest", msg.Name)
	s.Equal(2, len(msg.Fields), "DoFooRequest should have same fields as args")
	s.assertField(msg.Fields[0], "arg1", NewNamed("foo", "Bar"))
	s.assertField(msg.Fields[1], "arg2", NewBasic("int64"))

	msg = pkg.Messages[1]
	s.Equal("DoFooResponse", msg.Name)
	s.Equal(2, len(msg.Fields), "DoFooResponse should have same results as return args")
	s.assertField(msg.Fields[0], "result1", NewNamed("foo", "Foo"))
	s.assertField(msg.Fields[1], "result2", NewBasic("bool"))
}

func (s *TransformerSuite) TestTransformFuncInputRegistered() {
	fn := &scanner.Func{
		Name: "DoFoo",
		Input: []scanner.Type{
			scanner.NewNamed("foo", "Bar"),
			scanner.NewBasic("int"),
		},
		Output: []scanner.Type{
			scanner.NewNamed("foo", "Foo"),
			scanner.NewBasic("bool"),
			scanner.NewNamed("", "error"),
		},
	}
	rpc := s.t.transformFunc(&Package{}, fn, nameSet{"DoFooRequest": struct{}{}})

	s.Nil(rpc)
}

func (s *TransformerSuite) TestTransformFuncOutputRegistered() {
	fn := &scanner.Func{
		Name: "DoFoo",
		Input: []scanner.Type{
			scanner.NewNamed("foo", "Bar"),
			scanner.NewBasic("int"),
		},
		Output: []scanner.Type{
			scanner.NewNamed("foo", "Foo"),
			scanner.NewBasic("bool"),
			scanner.NewNamed("", "error"),
		},
	}
	rpc := s.t.transformFunc(&Package{}, fn, nameSet{"DoFooResponse": struct{}{}})

	s.Nil(rpc)
}

func (s *TransformerSuite) TestTransformFuncEmpty() {
	fn := &scanner.Func{Name: "DoFoo"}
	pkg := &Package{Path: "baz"}
	rpc := s.t.transformFunc(pkg, fn, nameSet{})

	s.NotNil(rpc)
	s.Equal(fn.Name, rpc.Name)
	s.assertType(NewGeneratedNamed("baz", "DoFooRequest"), rpc.Input, "rpc input")
	s.assertType(NewGeneratedNamed("baz", "DoFooResponse"), rpc.Output, "rpc output")
	s.Equal(2, len(pkg.Messages), "two messages should have been created")
	msg := pkg.Messages[0]
	s.Equal("DoFooRequest", msg.Name)
	s.Equal(0, len(msg.Fields), "DoFooRequest should have no args")

	msg = pkg.Messages[1]
	s.Equal("DoFooResponse", msg.Name)
	s.Equal(0, len(msg.Fields), "DoFooResponse should have no results")
}

func (s *TransformerSuite) TestTransformFunc1BasicArg() {
	fn := &scanner.Func{
		Name: "DoFoo",
		Input: []scanner.Type{
			scanner.NewBasic("int"),
		},
		Output: []scanner.Type{
			scanner.NewBasic("bool"),
			scanner.NewNamed("", "error"),
		},
	}
	pkg := new(Package)
	rpc := s.t.transformFunc(pkg, fn, nameSet{})

	s.NotNil(rpc)
	s.Equal(fn.Name, rpc.Name)
	s.assertType(NewGeneratedNamed("", "DoFooRequest"), rpc.Input, "rpc input")
	s.assertType(NewGeneratedNamed("", "DoFooResponse"), rpc.Output, "rpc output")

	s.Equal(2, len(pkg.Messages), "two messages should have been created")
	msg := pkg.Messages[0]
	s.Equal("DoFooRequest", msg.Name)
	s.Equal(1, len(msg.Fields), "DoFooRequest should have same fields as args")
	s.assertField(msg.Fields[0], "arg1", NewBasic("int64"))

	msg = pkg.Messages[1]
	s.Equal("DoFooResponse", msg.Name)
	s.Equal(1, len(msg.Fields), "DoFooResponse should have same results as return args")
	s.assertField(msg.Fields[0], "result1", NewBasic("bool"))
}

func (s *TransformerSuite) TestTransformFunc1NamedArg() {
	fn := &scanner.Func{
		Name: "DoFoo",
		Input: []scanner.Type{
			scanner.NewNamed("foo", "Foo"),
		},
		Output: []scanner.Type{
			scanner.NewNamed("foo", "Bar"),
			scanner.NewNamed("", "error"),
		},
	}
	rpc := s.t.transformFunc(new(Package), fn, nameSet{})

	s.NotNil(rpc)
	s.Equal(fn.Name, rpc.Name)
	s.assertType(NewNamed("foo", "Foo"), rpc.Input, "transform func 1")
	s.assertSource(rpc.Input, fn.Input[0])
	s.assertType(NewNamed("foo", "Bar"), rpc.Output, "transform func 1")
	s.assertSource(rpc.Output, fn.Output[0])
}

func (s *TransformerSuite) TestTransformFuncReceiver() {
	fn := &scanner.Func{
		Name:     "DoFoo",
		Receiver: scanner.NewNamed("foo", "Fooer"),
	}
	rpc := s.t.transformFunc(new(Package), fn, nameSet{})
	s.NotNil(rpc)
	s.Equal("Fooer_DoFoo", rpc.Name)
}

func (s *TransformerSuite) TestTransformFuncComments() {
	fn := &scanner.Func{
		Docs:     mkDocs("fooo bar"),
		Name:     "DoFoo",
		Receiver: scanner.NewNamed("foo", "Fooer"),
	}
	rpc := s.t.transformFunc(new(Package), fn, nameSet{})
	s.NotNil(rpc)
	s.Equal("Fooer_DoFoo", rpc.Name)
	s.Equal("fooo bar", strings.Join(rpc.Docs, "\n"))
}

func (s *TransformerSuite) TestTransformFuncReceiverInvalid() {
	fn := &scanner.Func{
		Name:     "DoFoo",
		Receiver: scanner.NewBasic("int"),
	}
	rpc := s.t.transformFunc(new(Package), fn, nameSet{})
	s.Nil(rpc)
}

func (s *TransformerSuite) TestTransformFuncRepeatedSingle() {
	fn := &scanner.Func{
		Name:       "DoFoo",
		IsVariadic: true,
		Input: []scanner.Type{
			repeated(scanner.NewBasic("int")),
		},
		Output: []scanner.Type{
			repeated(scanner.NewBasic("bool")),
			scanner.NewNamed("", "error"),
		},
	}
	pkg := new(Package)
	rpc := s.t.transformFunc(pkg, fn, nameSet{})

	s.NotNil(rpc)
	s.Equal(fn.Name, rpc.Name)
	s.assertType(NewGeneratedNamed("", "DoFooRequest"), rpc.Input, "rpc input")
	s.assertType(NewGeneratedNamed("", "DoFooResponse"), rpc.Output, "rpc output")

	s.Equal(2, len(pkg.Messages), "two messages should have been created")
	msg := pkg.Messages[0]
	s.Equal("DoFooRequest", msg.Name)
	s.NotContains(msg.Options, "(gogoproto.typedecl)", "should not have the option typedecl")
	s.Equal(1, len(msg.Fields), "DoFooRequest should have same fields as args")
	s.assertField(msg.Fields[0], "arg1", NewBasic("int64"))
	s.True(msg.Fields[0].Repeated, "field should be repeated")

	msg = pkg.Messages[1]
	s.Equal("DoFooResponse", msg.Name)
	s.Equal(1, len(msg.Fields), "DoFooResponse should have same results as return args")
	s.assertField(msg.Fields[0], "result1", NewBasic("bool"))
	s.True(msg.Fields[0].Repeated, "field should be repeated")
}

func (s *TransformerSuite) TestTransformEnum() {
	enum := s.t.transformEnum(&scanner.Enum{
		Docs: mkDocs("foo bar baz"),
		Name: "Foo",
		Values: []*scanner.EnumValue{
			mkEnumVal("fooo bar", "Foo"),
			mkEnumVal("baaar bar", "Bar"),
			mkEnumVal("barbaz bar", "BarBaz"),
		},
	})

	s.Equal("Foo", enum.Name)
	s.Equal("foo bar baz", strings.Join(enum.Docs, "\n"))
	s.Equal(3, len(enum.Values), "should have same number of values")
	s.assertEnumVal(enum.Values[0], "FOO", 0, "fooo bar")
	s.assertEnumVal(enum.Values[1], "BAR", 1, "baaar bar")
	s.assertEnumVal(enum.Values[2], "BAR_BAZ", 2, "barbaz bar")
	s.Equal(NewLiteralValue("false"), enum.Options["(gogoproto.enumdecl)"], "should drop declaration by default")
	s.NotContains(enum.Options, "(gogoproto.goproto_enum_stringer)", "not contains goproto_stringer")
}

func (s *TransformerSuite) TestTransformEnumIsStringer() {
	enum := s.t.transformEnum(&scanner.Enum{
		Name: "Foo",
		Values: []*scanner.EnumValue{
			mkEnumVal("fooo bar", "Foo"),
			mkEnumVal("baaar bar", "Bar"),
			mkEnumVal("barbaz bar", "BarBaz"),
		},
		IsStringer: true,
	})

	s.Equal("Foo", enum.Name)
	s.Equal(3, len(enum.Values), "should have same number of values")
	s.assertEnumVal(enum.Values[0], "FOO", 0, "fooo bar")
	s.assertEnumVal(enum.Values[1], "BAR", 1, "baaar bar")
	s.assertEnumVal(enum.Values[2], "BAR_BAZ", 2, "barbaz bar")
	s.Equal(NewLiteralValue("false"), enum.Options["(gogoproto.enumdecl)"], "should drop declaration by default")
	s.Contains(enum.Options, "(gogoproto.goproto_enum_stringer)", "contains goproto_stringer")
	s.Equal(NewLiteralValue("false"), enum.Options["(gogoproto.goproto_enum_stringer)"], "should drop declaration by default")
}

func (s *TransformerSuite) TestTransform() {
	pkgs := s.fixtures()
	pkg := s.t.Transform(pkgs[0])

	s.Equal("gopkg.in.srcd.proteus.v1.fixtures", pkg.Name)
	s.Equal("gopkg.in/src-d/proteus.v1/fixtures", pkg.Path)
	s.Equal(NewStringValue("foo"), pkg.Options["go_package"])
	s.Equal([]string{
		"github.com/gogo/protobuf/gogoproto/gogo.proto",
		"google/protobuf/timestamp.proto",
		"gopkg.in/src-d/proteus.v1/fixtures/subpkg/generated.proto",
	}, pkg.Imports)
	s.Equal(1, len(pkg.Enums))
	s.Equal(4, len(pkg.Messages))
	s.Equal(0, len(pkg.RPCs))

	pkg = s.t.Transform(pkgs[1])
	s.Equal("gopkg.in.srcd.proteus.v1.fixtures.subpkg", pkg.Name)
	s.Equal("gopkg.in/src-d/proteus.v1/fixtures/subpkg", pkg.Path)
	s.Equal(NewStringValue("subpkg"), pkg.Options["go_package"])
	s.Equal([]string{"github.com/gogo/protobuf/gogoproto/gogo.proto"}, pkg.Imports)
	s.Equal(0, len(pkg.Enums))

	var msgs = []string{
		"GeneratedRequest",
		"GeneratedResponse",
		"MyContainer_NameRequest",
		"MyContainer_NameResponse",
		"Point",
		"Point_GeneratedMethodOnPointerRequest",
		"Point_GeneratedMethodRequest",
	}
	s.Equal(len(msgs), len(pkg.Messages))
	for _, m := range pkg.Messages {
		s.True(hasString(m.Name, msgs), fmt.Sprintf("should have message %s", m.Name))
	}

	s.Equal(4, len(pkg.RPCs))
}

func hasString(str string, coll []string) bool {
	for _, s := range coll {
		if s == str {
			return true
		}
	}
	return false
}

func (s *TransformerSuite) fixtures() []*scanner.Package {
	sc, err := scanner.New(projectPath("fixtures"), projectPath("fixtures/subpkg"))
	s.Nil(err)
	pkgs, err := sc.Scan()
	s.Nil(err)
	resolver.New().Resolve(pkgs)
	return pkgs
}

func (s *TransformerSuite) assertType(expected, actual Type, name string) {
	switch e := expected.(type) {
	case *Basic:
		a, ok := actual.(*Basic)
		if !ok {
			s.Fail(fmt.Sprintf("expected type Basic but found %T for case %s", actual, name))
		}

		s.Equal(e.Name, a.Name, fmt.Sprintf("both are basic types: %s", name))
		return
	case *Named:
		a, ok := actual.(*Named)
		if !ok {
			s.Fail(fmt.Sprintf("expected type Named but found %T for case %s", actual, name))
		}

		s.Equal(e.Package, a.Package, fmt.Sprintf("package for %s", name))
		s.Equal(e.Name, a.Name, fmt.Sprintf("name for %s", name))
		return
	case *Map:
		a, ok := actual.(*Map)
		if !ok {
			s.Fail(fmt.Sprintf("expected type Map but found %T for case %s", actual, name))
		}

		s.assertType(e.Key, a.Key, fmt.Sprintf("key for %s", name))
		s.assertType(e.Value, a.Value, fmt.Sprintf("value for %s", name))
		return
	}
}

func (s *TransformerSuite) assertField(f *Field, name string, typ Type) {
	s.Equal(f.Name, name)
	s.assertType(f.Type, typ, name)
}

// assertFieldToField compares name, type and options of the given field
func (s *TransformerSuite) assertFieldToField(expected, actual *Field, name string) {
	s.Equal(expected.Name, actual.Name, fmt.Sprintf("Name in %s", name))
	s.assertType(expected.Type, actual.Type, fmt.Sprintf("Type in %s", name))
	s.Equal(expected.Repeated, actual.Repeated, fmt.Sprintf("Repeated in %s", name))
	s.Equal(expected.Options, actual.Options, fmt.Sprintf("Options in %s", name))
}

func (s *TransformerSuite) assertEnumVal(v *EnumValue, name string, val uint, doc string) {
	s.Equal(name, v.Name)
	s.Equal(val, v.Value)
	s.Equal(doc, strings.Join(v.Docs, "\n"))
}

func (s *TransformerSuite) assertSource(typ Type, src scanner.Type) {
	switch t := typ.(type) {
	case *Named:
		s.Equal(src, t.Source())
		return
	case *Map:
		s.Equal(src, t.Source())
		return
	}
}

func TestTransformer(t *testing.T) {
	suite.Run(t, new(TransformerSuite))
}

func repeated(t scanner.Type) scanner.Type {
	t.SetRepeated(true)
	return t
}

func nullable(t scanner.Type) scanner.Type {
	t.SetNullable(true)
	return t
}

func mkEnumVal(doc, name string) *scanner.EnumValue {
	return &scanner.EnumValue{
		mkDocs(doc),
		name,
	}
}

func mkDocs(doc ...string) scanner.Docs {
	return scanner.Docs{
		Doc: doc,
	}
}

const project = "gopkg.in/src-d/proteus.v1"

func projectPath(pkg string) string {
	return filepath.Join(project, pkg)
}
