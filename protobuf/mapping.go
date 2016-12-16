package protobuf

// ProtobufType represents a protobuf type. It can optionally have a
// package and it may require an import to work.
type ProtobufType struct {
	Package string
	Basic   bool
	Name    string
	Import  string
}

// Type returns the type representation of the protobuf type.
func (t *ProtobufType) Type() Type {
	if t.Basic {
		return NewBasic(t.Name)
	}
	return NewNamed(t.Package, t.Name)
}

// TypeMappings is a mapping between Go types and protobuf types.
// The names of the Go types can have packages. For example: "time.Time" is a
// valid name. "foo.bar/baz.Qux" is a valid type name as well.
type TypeMappings map[string]*ProtobufType

var defaultMappings = TypeMappings{
	"float64": &ProtobufType{Name: "double", Basic: true},
	"float32": &ProtobufType{Name: "float", Basic: true},
	"int32":   &ProtobufType{Name: "int32", Basic: true},
	"int64":   &ProtobufType{Name: "int64", Basic: true},
	"uint32":  &ProtobufType{Name: "uint32", Basic: true},
	"uint64":  &ProtobufType{Name: "uint64", Basic: true},
	"bool":    &ProtobufType{Name: "bool", Basic: true},
	"string":  &ProtobufType{Name: "string", Basic: true},
	"uint8":   &ProtobufType{Name: "uint32", Basic: true},
	"int8":    &ProtobufType{Name: "int32", Basic: true},
	"byte":    &ProtobufType{Name: "uint32", Basic: true},
	"uint16":  &ProtobufType{Name: "uint32", Basic: true},
	"int16":   &ProtobufType{Name: "int32", Basic: true},
	"int":     &ProtobufType{Name: "int32", Basic: true},
	"uint":    &ProtobufType{Name: "uint32", Basic: true},
	"uintptr": &ProtobufType{Name: "uint64", Basic: true},
	"rune":    &ProtobufType{Name: "int32", Basic: true},
	"time.Time": &ProtobufType{
		Name:    "Timestamp",
		Package: "google.protobuf",
		Import:  "google/protobuf/timestamp.proto",
	},
	"time.Duration": &ProtobufType{Name: "int64", Basic: true},
}
