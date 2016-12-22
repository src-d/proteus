package scanner

import (
	"fmt"
	"strings"
)

// Package holds information about a single Go package and
// a reference of all defined structs and type aliases.
// A Package is only safe to use once it is resolved.
type Package struct {
	Resolved bool
	Path     string
	Name     string
	Structs  []*Struct
	Enums    []*Enum
	Aliases  map[string]Type
}

// collectEnums finds the enum values collected during the scan and generates
// the corresponding enum types, removing them as aliases from the package.
func (p *Package) collectEnums(ctx *context) {
	for k := range p.Aliases {
		if vals, ok := ctx.enumValues[k]; ok {
			idx := strings.LastIndex(k, ".")
			name := k[idx+1:]
			if !ctx.shouldGenerateType(name) {
				continue
			}

			p.Enums = append(p.Enums, newEnum(ctx, name, vals))
			delete(p.Aliases, k)
		}
	}
}

// Type is the common interface for all possible types supported in protogo.
// Type is neither a representation of a Go type nor a representation of a
// protobuf type. Is an intermediate representation to ease future steps in
// the conversion from Go to protobuf.
// All types can be repeated (or not).
type Type interface {
	SetRepeated(bool)
	IsRepeated() bool
}

// BaseType contains the common fields for all the types.
type BaseType struct {
	Repeated bool
}

func newBaseType() *BaseType {
	return &BaseType{
		Repeated: false,
	}
}

// IsRepeated reports wether the type is repeated or not.
func (t *BaseType) IsRepeated() bool { return t.Repeated }

// SetRepeated sets the type as repeated or not repeated.
func (t *BaseType) SetRepeated(v bool) { t.Repeated = v }

// Basic is a basic type, which only is identified by its name.
type Basic struct {
	*BaseType
	Name string
}

// NewBasic creates a new basic type given its name.
func NewBasic(name string) Type {
	return &Basic{
		newBaseType(),
		name,
	}
}

// Named is non-basic type identified by a name on some package.
type Named struct {
	*BaseType
	Path string
	Name string
}

func (n Named) String() string {
	return fmt.Sprintf("%s.%s", n.Path, n.Name)
}

// NewNamed creates a new named type given its package path and name.
func NewNamed(path, name string) Type {
	return &Named{
		newBaseType(),
		path,
		name,
	}
}

// Map is a map type with a key and a value type.
type Map struct {
	*BaseType
	Key   Type
	Value Type
}

// NewMap creates a new map type with the given key and value types.
func NewMap(key, val Type) Type {
	return &Map{
		newBaseType(),
		key,
		val,
	}
}

// Enum consists of a list of possible values.
type Enum struct {
	Name   string
	Values []string
}

// Struct represents a Go struct with its name and fields.
// All structs
type Struct struct {
	Generate bool
	Name     string
	Fields   []*Field
}

// HasField reports wether a struct has a given field name.
func (s *Struct) HasField(name string) bool {
	for _, f := range s.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// Field contains name and type of a struct field.
type Field struct {
	Name string
	Type Type
}
