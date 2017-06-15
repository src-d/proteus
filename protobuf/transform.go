package protobuf

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"gopkg.in/src-d/proteus.v1/report"
	"gopkg.in/src-d/proteus.v1/scanner"
)

// Transformer is in charge of converting scanned Go entities to protobuf
// entities as well as mapping between Go and Protobuf types.
// Take into account that custom mappings are used first to check for the
// corresponding type mapping, and then the default mappings to give the user
// ability to override any kind of type.
type Transformer struct {
	mappings  TypeMappings
	structSet TypeSet
	enumSet   TypeSet
}

// NewTransformer creates a new transformer instance.
func NewTransformer() *Transformer {
	return &Transformer{
		mappings: make(TypeMappings),
	}
}

// SetMappings will set the custom mappings of the transformer. If nil is
// provided, the change will be ignored.
func (t *Transformer) SetMappings(m TypeMappings) {
	if m == nil {
		return
	}
	t.mappings = m
}

// SetStructSet sets the passed TypeSet as a known list of structs.
func (t *Transformer) SetStructSet(ts TypeSet) {
	t.structSet = ts
}

// IsStruct checks if the given pkg path and name is a known struct.
func (t *Transformer) IsStruct(pkg, name string) bool {
	return t.structSet.Contains(pkg, name)
}

// IsEnum checks if the given pkg path and name is a known enum.
func (t *Transformer) IsEnum(pkg, name string) bool {
	return t.enumSet.Contains(pkg, name)
}

// SetEnumSet sets the passed TypeSet as a known list of enums.
func (t *Transformer) SetEnumSet(ts TypeSet) {
	t.enumSet = ts
}

// Transform converts a scanned package to a protobuf package.
func (t *Transformer) Transform(p *scanner.Package) *Package {
	pkg := &Package{
		Name:    toProtobufPkg(p.Path),
		Path:    p.Path,
		Imports: []string{"github.com/gogo/protobuf/gogoproto/gogo.proto"},
		Options: t.defaultOptionsForPackage(p),
	}

	for _, s := range p.Structs {
		msg := t.transformStruct(pkg, s)
		pkg.Messages = append(pkg.Messages, msg)
	}

	for _, e := range p.Enums {
		enum := t.transformEnum(e)
		pkg.Enums = append(pkg.Enums, enum)
	}

	names := buildNameSet(p)
	for _, f := range p.Funcs {
		rpc := t.transformFunc(pkg, f, names)
		if rpc != nil {
			pkg.RPCs = append(pkg.RPCs, rpc)
		}
	}

	return pkg
}

func (t *Transformer) transformFunc(pkg *Package, f *scanner.Func, names nameSet) *RPC {
	var (
		name         = f.Name
		receiverName string
	)

	if f.Receiver != nil {
		n, ok := f.Receiver.(*scanner.Named)
		if !ok {
			report.Warn("invalid receiver type for func %s", f.Name)
			return nil
		}

		name = fmt.Sprintf("%s_%s", n.Name, name)
		receiverName = n.Name
	}

	input, hasCtx := removeFirstCtx(f.Input)
	output, hasError := removeLastError(f.Output)
	rpc := &RPC{
		Docs:       f.Doc,
		Name:       name,
		Recv:       receiverName,
		Method:     f.Name,
		HasCtx:     hasCtx,
		HasError:   hasError,
		IsVariadic: f.IsVariadic,
		Input:      t.transformInputTypes(pkg, input, names, name),
		Output:     t.transformOutputTypes(pkg, output, names, name),
	}
	if rpc.Input == nil || rpc.Output == nil {
		return nil
	}

	return rpc
}

func (t *Transformer) transformInputTypes(pkg *Package, types []scanner.Type, names nameSet, name string) Type {
	return t.transformTypeList(pkg, types, names, name, "Request", "arg")
}

func (t *Transformer) transformOutputTypes(pkg *Package, types []scanner.Type, names nameSet, name string) Type {
	return t.transformTypeList(pkg, types, names, name, "Response", "result")
}

func (t *Transformer) transformTypeList(pkg *Package, types []scanner.Type, names nameSet, name, msgNameSuffix, msgFieldPrefix string) Type {
	// the type list should be wrapped in a separate message if:
	// - there is more than one element
	// - there is one element and it is repeated, as this is not supported in protobuf
	// - there is one element and it is not a message, as protobuf expects messages as input/output
	if len(types) != 1 || types[0].IsRepeated() || !isNamed(types[0]) {
		msgName := name + msgNameSuffix
		if _, ok := names[msgName]; ok {
			report.Warn("tried to register message %s, but there is already a message with that name. RPC %s will not be generated", msgName, name)
			return nil
		}

		msg := t.createMessageFromTypes(pkg, msgName, types, msgFieldPrefix)
		pkg.Messages = append(pkg.Messages, msg)
		return NewGeneratedNamed(toProtobufPkg(pkg.Path), msgName)
	}

	return t.transformType(pkg, types[0], &Message{}, &Field{})
}

func (t *Transformer) createMessageFromTypes(pkg *Package, name string, types []scanner.Type, fieldPrefix string) *Message {
	msg := &Message{Name: name}
	for i, typ := range types {
		f := t.transformField(pkg, msg, &scanner.Field{
			Name: fmt.Sprintf("%s%d", capitalize(fieldPrefix), i+1),
			Type: typ,
		}, i+1)
		if f != nil {
			msg.Fields = append(msg.Fields, f)
		}
	}
	return msg
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:len(s)]
}

func (t *Transformer) transformEnum(e *scanner.Enum) *Enum {
	enum := &Enum{
		Docs:    e.Doc,
		Name:    e.Name,
		Options: t.defaultOptionsForScannedEnum(e),
	}

	for i, v := range e.Values {
		enum.Values = append(enum.Values, &EnumValue{
			Docs:  v.Doc,
			Name:  toUpperSnakeCase(v.Name),
			Value: uint(i),
			Options: Options{
				"(gogoproto.enumvalue_customname)": NewStringValue(v.Name),
			},
		})
	}
	return enum
}

func (t *Transformer) defaultOptionsForScannedEnum(e *scanner.Enum) (opts Options) {
	opts = Options{
		"(gogoproto.enumdecl)":            NewLiteralValue("false"),
		"(gogoproto.goproto_enum_prefix)": NewLiteralValue("false"),
	}

	if e.IsStringer {
		opts["(gogoproto.goproto_enum_stringer)"] = NewLiteralValue("false")
	}

	return
}

func (t *Transformer) transformStruct(pkg *Package, s *scanner.Struct) *Message {
	msg := &Message{
		Docs:    s.Doc,
		Name:    s.Name,
		Options: t.defaultOptionsForScannedMessage(s),
	}

	for i, f := range s.Fields {
		field := t.transformField(pkg, msg, f, i+1)
		if field == nil {
			msg.Reserve(uint(i) + 1)
			report.Warn("field %q of struct %q has an invalid type, ignoring field but reserving its position", f.Name, s.Name)
		} else {
			msg.Fields = append(msg.Fields, field)
		}
	}

	return msg
}

func (t *Transformer) defaultOptionsForScannedMessage(s *scanner.Struct) (opts Options) {
	opts = Options{
		"(gogoproto.typedecl)":        NewLiteralValue("false"),
		"(gogoproto.goproto_getters)": NewLiteralValue("false"),
	}

	if s.IsStringer {
		opts["(gogoproto.goproto_stringer)"] = NewLiteralValue("false")
	}

	return
}

func (t *Transformer) transformField(pkg *Package, msg *Message, field *scanner.Field, pos int) *Field {
	var (
		typ      Type
		repeated = field.Type.IsRepeated()
	)

	f := &Field{
		Docs:     field.Doc,
		Name:     toLowerSnakeCase(field.Name),
		Options:  t.defaultOptionsForStructField(field),
		Pos:      pos,
		Repeated: repeated,
	}

	// []byte is the only repeated type that maps to
	// a non-repeated type in protobuf, so we handle
	// it a bit differently.
	if isByteSlice(field.Type) {
		typ = NewBasic("bytes")
		f.Repeated = false
	} else {
		typ = t.transformType(pkg, field.Type, msg, f)
		if typ == nil {
			return nil
		}
	}

	f.Type = typ

	return f
}

func (t *Transformer) defaultOptionsForStructField(field *scanner.Field) Options {
	opts := make(Options)
	if generator.CamelCase(toLowerSnakeCase(field.Name)) != field.Name {
		opts["(gogoproto.customname)"] = NewStringValue(field.Name)
	}

	if t.needsNotNullableOption(field.Type) {
		opts["(gogoproto.nullable)"] = NewLiteralValue("false")
	}

	return opts
}

func (t *Transformer) needsNotNullableOption(typ scanner.Type) bool {
	isNullable := typ.IsNullable()

	switch ty := typ.(type) {
	case *scanner.Named:
		return !isNullable && !t.IsEnum(ty.Path, ty.Name)
	case *scanner.Alias:
		return t.needsNotNullableOption(ty.Underlying)
	case *scanner.Map:
		return t.needsNotNullableOption(ty.Value)
	}

	return false
}

func (t *Transformer) transformType(pkg *Package, typ scanner.Type, msg *Message, field *Field) Type {
	if isError(typ) {
		report.Error("error type is not supported")
		return nil
	}

	switch ty := typ.(type) {
	case *scanner.Named:
		protoType := t.findMapping(ty.String())
		if protoType != nil {
			pkg.Import(protoType)
			protoType.Decorate(pkg, msg, field)
			n := protoType.Type()
			n.SetSource(ty)
			return n
		}

		pkg.ImportFromPath(ty.Path)
		n := NewNamed(toProtobufPkg(ty.Path), ty.Name)
		n.SetSource(ty)
		return n
	case *scanner.Basic:
		protoType := t.findMapping(ty.Name)
		if protoType != nil {
			pkg.Import(protoType)
			protoType.Decorate(pkg, msg, field)
			b := protoType.Type()
			b.SetSource(ty)
			return b
		}

		report.Warn("basic type %q is not defined in the mappings, ignoring", ty.Name)
	case *scanner.Map:
		m := NewMap(
			t.transformType(pkg, ty.Key, msg, field),
			t.transformType(pkg, ty.Value, msg, field),
		)
		m.SetSource(ty)
		return m
	case *scanner.Alias:
		n := NewAlias(
			t.transformType(pkg, ty.Type, msg, field),
			t.transformType(pkg, ty.Underlying, msg, field),
		)
		n.SetSource(ty)
		if field.Options == nil {
			field.Options = make(Options)
		}

		// Repeated types cannot use casttype :(
		if !ty.IsRepeated() {
			field.Options["(gogoproto.casttype)"] = NewStringValue(castType(pkg, n.Type))
		}
		return n
	}

	return nil
}

func castType(pkg *Package, typ Type) string {
	switch t := typ.Source().(type) {
	case *scanner.Named:
		if pkg.Path == t.Path {
			return t.Name
		}
		return t.TypeString()
	}
	return typ.Source().TypeString()
}

func (t *Transformer) findMapping(name string) *ProtoType {
	typ := t.mappings[name]
	if typ == nil {
		typ = DefaultMappings[name]
	}

	if typ != nil && typ.Warn != "" {
		report.Warn(typ.Warn, name)
	}

	return typ
}

func removeFirstCtx(types []scanner.Type) ([]scanner.Type, bool) {
	if len(types) > 0 {
		first := types[0]
		if isCtx(first) {
			return types[1:], true
		}
	}

	return types, false
}

func removeLastError(types []scanner.Type) ([]scanner.Type, bool) {
	if len(types) > 0 {
		ln := len(types)
		last := types[ln-1]
		if isError(last) {
			return types[:ln-1], true
		}
	}

	return types, false
}

func isNamed(typ scanner.Type) bool {
	_, ok := typ.(*scanner.Named)
	return ok
}

func isCtx(typ scanner.Type) bool {
	if ctx, ok := typ.(*scanner.Named); ok {
		return ctx.Path == "context" && ctx.Name == "Context"
	}
	return false
}

func isError(typ scanner.Type) bool {
	if err, ok := typ.(*scanner.Named); ok {
		return err.Path == "" && err.Name == "error"
	}
	return false
}

func isByteSlice(typ scanner.Type) bool {
	if t, ok := typ.(*scanner.Basic); ok && typ.IsRepeated() {
		return t.Name == "byte"
	}
	return false
}

func toProtobufPkg(path string) string {
	pkg := strings.Map(func(r rune) rune {
		if r == '/' || r == '.' {
			return '.'
		}

		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}

		return ' '
	}, path)
	pkg = strings.Replace(pkg, " ", "", -1)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	pkg, _, _ = transform.String(t, pkg)
	return pkg
}

func toLowerSnakeCase(s string) string {
	var buf bytes.Buffer
	var lastWasUpper bool
	for i, r := range s {
		if unicode.IsUpper(r) && i != 0 && !lastWasUpper {
			buf.WriteRune('_')
		}
		lastWasUpper = unicode.IsUpper(r)
		buf.WriteRune(unicode.ToLower(r))
	}
	return buf.String()
}

func toUpperSnakeCase(s string) string {
	return strings.ToUpper(toLowerSnakeCase(s))
}

func (t *Transformer) defaultOptionsForPackage(p *scanner.Package) Options {
	return Options{
		"go_package":                 NewStringValue(p.Name),
		"(gogoproto.sizer_all)":      NewLiteralValue("false"),
		"(gogoproto.protosizer_all)": NewLiteralValue("true"),
	}
}

type nameSet map[string]struct{}

func buildNameSet(pkg *scanner.Package) nameSet {
	l := make(nameSet)

	for _, e := range pkg.Enums {
		l[e.Name] = struct{}{}
	}

	for _, s := range pkg.Structs {
		l[s.Name] = struct{}{}
	}

	return l
}
