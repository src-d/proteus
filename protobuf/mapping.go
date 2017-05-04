package protobuf

import (
	"fmt"
	"sort"
	"strings"
)

// ProtoType represents a protobuf type. It can optionally have a
// package and it may require an import to work.
type ProtoType struct {
	Package string
	Basic   bool
	Name    string
	Import  string
	// GoImport represents the go package to import to use this type.
	GoImport string
	// Decorators define a set of function to apply to each field, message and
	// package that contain a field with this type.
	Decorators Decorators
	// Warn contains the warning message to show if this mapping happens. This string
	// is passed to fmt.Sprintf with the original type parameter.
	// For example, if a mapping is defined for Go type "A" to become "B" in
	// protobuf and the warning message "%s becomes B", then the reported message
	// will be "A becomes B"
	Warn string
}

func (pt *ProtoType) Decorate(p *Package, m *Message, f *Field) {
	pt.Decorators.Run(p, m, f)
}

// Type returns the type representation of the protobuf type.
func (t *ProtoType) Type() Type {
	if t.Basic {
		return NewBasic(t.Name)
	}
	return NewNamed(t.Package, t.Name)
}

// Decorator is run when a type is resolved. The decorator gets the package, message
// and field this type was resolved for.
// In some corner cases, a type might need to be resolved not for used in a field. In
// such cases, an empty structure is passed for each argument that does not exists.
type Decorator func(*Package, *Message, *Field)

// Decorators is a collection of Decorator that simplifies running them all with a
// given set of options.
type Decorators []Decorator

func NewDecorators(fns ...Decorator) Decorators {
	var decorators Decorators

	for _, fn := range fns {
		decorators = append(decorators, Decorator(fn))
	}

	return decorators
}

func (d Decorator) Run(p *Package, m *Message, f *Field) {
	d(p, m, f)
}

func (ds Decorators) Run(p *Package, m *Message, f *Field) {
	for _, d := range ds {
		d.Run(p, m, f)
	}
}

func CastToBasicType(basic string) Decorators {
	return NewDecorators(
		func(p *Package, m *Message, f *Field) {
			if f.Options == nil {
				f.Options = make(Options)
			}

			f.Options["(gogoproto.casttype)"] = NewStringValue(basic)
		},
	)
}

// TypeMappings is a mapping between Go types and protobuf types.
// The names of the Go types can have packages. For example: "time.Time" is a
// valid name. "foo.bar/baz.Qux" is a valid type name as well.
type TypeMappings map[string]*ProtoType

var DefaultMappings = TypeMappings{
	"float64": &ProtoType{Name: "double", Basic: true},
	"float32": &ProtoType{Name: "float", Basic: true},
	"int32":   &ProtoType{Name: "int32", Basic: true},
	"int64":   &ProtoType{Name: "int64", Basic: true},
	"uint32":  &ProtoType{Name: "uint32", Basic: true},
	"uint64":  &ProtoType{Name: "uint64", Basic: true},
	"bool":    &ProtoType{Name: "bool", Basic: true},
	"string":  &ProtoType{Name: "string", Basic: true},
	"uint8": &ProtoType{
		Name:       "uint32",
		Basic:      true,
		Warn:       "type %s was upgraded to uint32",
		Decorators: CastToBasicType("uint8"),
	},
	"int8": &ProtoType{
		Name:       "int32",
		Basic:      true,
		Warn:       "type %s was upgraded to int32",
		Decorators: CastToBasicType("int8"),
	},
	"byte": &ProtoType{
		Name:       "uint32",
		Basic:      true,
		Warn:       "type %s was upgraded to uint32",
		Decorators: CastToBasicType("byte"),
	},
	"uint16": &ProtoType{
		Name:       "uint32",
		Basic:      true,
		Warn:       "type %s was upgraded to uint32",
		Decorators: CastToBasicType("uint16"),
	},
	"int16": &ProtoType{
		Name:       "int32",
		Basic:      true,
		Warn:       "type %s was upgraded to int32",
		Decorators: CastToBasicType("int16"),
	},
	"int": &ProtoType{
		Name:       "int64",
		Basic:      true,
		Warn:       "type %s was upgraded to int64",
		Decorators: CastToBasicType("int"),
	},
	"uint": &ProtoType{
		Name:       "uint64",
		Basic:      true,
		Warn:       "type %s was upgraded to uint64",
		Decorators: CastToBasicType("uint"),
	},
	"uintptr": &ProtoType{
		Name:       "uint64",
		Basic:      true,
		Decorators: CastToBasicType("uintptr"),
	},
	"rune": &ProtoType{
		Name:       "int32",
		Basic:      true,
		Decorators: CastToBasicType("rune"),
	},
	"time.Time": &ProtoType{
		Name:     "Timestamp",
		Package:  "google.protobuf",
		Import:   "google/protobuf/timestamp.proto",
		GoImport: "github.com/gogo/protobuf/types",
		Decorators: NewDecorators(
			func(p *Package, m *Message, f *Field) {
				if f.Options == nil {
					f.Options = make(Options)
				}
				f.Options["(gogoproto.stdtime)"] = NewLiteralValue("true")
				f.Options["(gogoproto.nullable)"] = NewLiteralValue("false")
			},
		),
	},
	"time.Duration": &ProtoType{
		Name:     "Duration",
		Package:  "google.protobuf",
		Import:   "google/protobuf/duration.proto",
		GoImport: "github.com/gogo/protobuf/types",
		Decorators: NewDecorators(
			func(p *Package, m *Message, f *Field) {
				if f.Options == nil {
					f.Options = make(Options)
				}
				f.Options["(gogoproto.stdduration)"] = NewLiteralValue("true")
				f.Options["(gogoproto.nullable)"] = NewLiteralValue("false")
			},
		),
	},
}

// ToGoOutPath returns the set of import mappings for the --go_out family of options.
// For more info see src-d/proteus#41
func (t TypeMappings) ToGoOutPath() string {
	var strs []string
	var keys []string
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		value := t[k]
		if value.Import != "" && value.GoImport != "" {
			strs = append(strs, fmt.Sprintf("M%s=%s", value.Import, value.GoImport))
		}
	}

	return strings.Join(strs, ",")
}
