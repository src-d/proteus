package protobuf

import (
	"fmt"
	"path/filepath"
)

// Package represents an unique .proto file with its own package definition.
type Package struct {
	Name     string
	Path     string
	Imports  []string
	Options  Options
	Messages []*Message
	Enums    []*Enum
}

// Import tries to import the given protobuf type to the current package.
// If the type requires no import at all, nothing will be done.
func (p *Package) Import(typ *ProtobufType) {
	if typ.Import != "" && !p.isImported(typ.Import) {
		p.Imports = append(p.Imports, typ.Import)
	}
}

// ImportFromPath adds a new import from a Go path.
func (p *Package) ImportFromPath(path string) {
	file := filepath.Join(path, "generated.proto")
	if path != p.Path && !p.isImported(file) {
		p.Imports = append(p.Imports, filepath.Join(path, "generated.proto"))
	}
}

func (p *Package) isImported(file string) bool {
	for _, i := range p.Imports {
		if i == file {
			return true
		}
	}
	return false
}

// Message is the representation of a Protobuf message.
type Message struct {
	Name     string
	Reserved []int
	Options  Options
	Fields   []*Field
}

// Reserve reserves a position in the message.
func (m *Message) Reserve(pos int) {
	m.Reserved = append(m.Reserved, pos)
}

// Fields is the representation of a protobuf message field.
type Field struct {
	Name     string
	Pos      int
	Repeated bool
	Nullable bool
	Type     Type
	Options  Options
}

// Options are the set of options given to a field, message or enum value.
type Options map[string]OptionValue

// OptionValue is the common interface for the value of an option, which can be
// a literal value (a number, true, etc) or a string value ("foo").
type OptionValue interface {
	fmt.Stringer
	isOptionValue()
}

// LiteralValue is a literal option value like true, false or a number.
type LiteralValue string

func (LiteralValue) isOptionValue() {}
func (v LiteralValue) String() string {
	return string(v)
}

// StringValue is a string option value.
type StringValue string

func (StringValue) isOptionValue() {}
func (v StringValue) String() string {
	return fmt.Sprintf("%q", v)
}

// Type is the common interface of all possible types, which are named types,
// maps and basic types.
type Type interface {
	isType()
}

// Named is a type which has a name and is defined somewhere else, maybe even
// in another package.
type Named struct {
	Package string
	Name    string
}

func NewNamed(pkg, name string) *Named {
	return &Named{pkg, name}
}

// Basic is one of the basic types of protobuf.
type Basic string

func NewBasic(name string) *Basic {
	b := Basic(name)
	return &b
}

// Map is a key-value map type.
type Map struct {
	Key   Type
	Value Type
}

func NewMap(k, v Type) *Map {
	return &Map{k, v}
}

func (*Named) isType() {}
func (*Basic) isType() {}
func (*Map) isType()   {}

// Enum is the representation of a protobuf enumeration.
type Enum struct {
	Name    string
	Options Options
	Values  EnumValues
}

// EnumValues is a collction of enumeration values.
type EnumValues []*EnumValue

func (v *EnumValues) Add(name string, val uint, options Options) {
	*v = EnumValues(append(*v, &EnumValue{name, val, options}))
}

// EnumValue is a single value in an enumeration.
type EnumValue struct {
	Name    string
	Value   uint
	Options Options
}
