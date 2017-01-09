package protobuf

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/src-d/proteus/report"
	"github.com/src-d/proteus/scanner"
)

// Transformer is in charge of converting scanned Go entities to protobuf
// entities as well as mapping between Go and Protobuf types.
// Take into account that custom mappings are used first to check for the
// corresponding type mapping, and then the default mappings to give the user
// ability to override any kind of type.
type Transformer struct {
	mappings TypeMappings
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

// Transform converts a scanned package to a protobuf package.
func (t *Transformer) Transform(p *scanner.Package) *Package {
	pkg := &Package{
		Name:    toProtobufPkg(p.Path),
		Path:    p.Path,
		Options: defaultOptionsForPackage(p),
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

	output, hasError := removeLastError(f.Output)
	rpc := &RPC{
		Name:       name,
		Recv:       receiverName,
		Method:     f.Name,
		HasError:   hasError,
		IsVariadic: f.IsVariadic,
		Input:      t.transformInputTypes(pkg, f.Input, names, name),
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

	return t.transformType(pkg, types[0])
}

func (t *Transformer) createMessageFromTypes(pkg *Package, name string, types []scanner.Type, fieldPrefix string) *Message {
	msg := &Message{Name: name}
	for i, typ := range types {
		f := t.transformField(pkg, &scanner.Field{
			Name: fmt.Sprintf("%s%d", fieldPrefix, i+1),
			Type: typ,
		}, i+1)
		msg.Fields = append(msg.Fields, f)
	}
	return msg
}

func (t *Transformer) transformEnum(e *scanner.Enum) *Enum {
	enum := &Enum{Name: e.Name}
	for i, v := range e.Values {
		enum.Values.Add(toUpperSnakeCase(v), uint(i), nil)
	}
	return enum
}

func (t *Transformer) transformStruct(pkg *Package, s *scanner.Struct) *Message {
	msg := &Message{Name: s.Name}

	for i, f := range s.Fields {
		field := t.transformField(pkg, f, i+1)
		if field == nil {
			msg.Reserve(uint(i) + 1)
			report.Warn("field %q of struct %q has an invalid type, ignoring field but reserving its position", f.Name, s.Name)
			continue
		}

		msg.Fields = append(msg.Fields, field)
	}

	return msg
}

func (t *Transformer) transformField(pkg *Package, field *scanner.Field, pos int) *Field {
	var (
		typ      Type
		repeated = field.Type.IsRepeated()
	)

	// []byte is the only repeated type that maps to
	// a non-repeated type in protobuf, so we handle
	// it a bit differently.
	if isByteSlice(field.Type) {
		typ = NewBasic("bytes")
		repeated = false
	} else {
		typ = t.transformType(pkg, field.Type)
		if typ == nil {
			return nil
		}
	}

	return &Field{
		Name:     toLowerSnakeCase(field.Name),
		Pos:      pos,
		Type:     typ,
		Repeated: repeated,
	}
}

func (t *Transformer) transformType(pkg *Package, typ scanner.Type) Type {
	if isError(typ) {
		report.Error("error type is not supported")
		return nil
	}

	switch ty := typ.(type) {
	case *scanner.Named:
		protoType := t.findMapping(ty.String())
		if protoType != nil {
			pkg.Import(protoType)
			return protoType.Type()
		}

		pkg.ImportFromPath(ty.Path)
		return NewNamed(toProtobufPkg(ty.Path), ty.Name)
	case *scanner.Basic:
		protoType := t.findMapping(ty.Name)
		if protoType != nil {
			pkg.Import(protoType)
			return protoType.Type()
		}

		report.Warn("basic type %q is not defined in the mappings, ignoring", ty.Name)
	case *scanner.Map:
		return NewMap(
			t.transformType(pkg, ty.Key),
			t.transformType(pkg, ty.Value),
		)
	}

	return nil
}

func (t *Transformer) findMapping(name string) *ProtoType {
	typ := t.mappings[name]
	if typ == nil {
		typ = defaultMappings[name]
	}

	return typ
}

func removeLastError(types []scanner.Type) ([]scanner.Type, bool) {
	if len(types) > 0 {
		last := types[len(types)-1]
		if isError(last) {
			return types[:len(types)-1], true
		}
	}

	return types, false
}

func isNamed(typ scanner.Type) bool {
	_, ok := typ.(*scanner.Named)
	return ok
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

func defaultOptionsForPackage(p *scanner.Package) Options {
	return Options{
		"go_package": NewStringValue(p.Name),
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
