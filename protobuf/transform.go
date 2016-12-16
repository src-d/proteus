package protobuf

import (
	"bytes"
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
		Name: toProtobufPkg(p.Path),
		Path: p.Path,
	}

	for _, s := range p.Structs {
		msg := t.transformStruct(pkg, s)
		pkg.Messages = append(pkg.Messages, msg)
	}

	for _, e := range p.Enums {
		enum := t.transformEnum(e)
		pkg.Enums = append(pkg.Enums, enum)
	}

	return pkg
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
			msg.Reserve(i + 1)
			report.Warn("field %q of struct %q has an invalid type, ignoring field but reserving its position", f.Name, s.Name)
			continue
		}

		msg.Fields = append(msg.Fields, field)
	}

	return msg
}

func (t *Transformer) transformField(pkg *Package, field *scanner.Field, pos int) *Field {
	var typ Type
	var repeated = field.Type.IsRepeated()

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
		Nullable: field.Type.IsNullable(),
	}
}

func (t *Transformer) transformType(pkg *Package, typ scanner.Type) Type {
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
	case *scanner.Map:
		return NewMap(
			t.transformType(pkg, ty.Key),
			t.transformType(pkg, ty.Value),
		)
	}

	// TODO: only way to reach this is with a basic type that is not defined in the
	// mappings. Probably should panic?
	return nil
}

func (t *Transformer) findMapping(name string) *ProtobufType {
	typ := t.mappings[name]
	if typ == nil {
		typ = defaultMappings[name]
	}

	return typ
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
	for i, r := range s {
		if unicode.IsUpper(r) && i != 0 {
			buf.WriteRune('_')
		}
		buf.WriteRune(unicode.ToLower(r))
	}
	return buf.String()
}

func toUpperSnakeCase(s string) string {
	return strings.ToUpper(toLowerSnakeCase(s))
}
