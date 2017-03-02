package protobuf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/src-d/proteus.v1/report"
)

// Generator is in charge of generating the .proto files and write them
// to disk in a file at the given path.
type Generator struct {
	basePath string
}

// NewGenerator creates a new Generator with the given base path.
func NewGenerator(basePath string) *Generator {
	return &Generator{basePath}
}

// Generate generates the proto3 .proto file of the given package and
// writes it to disk.
func (g *Generator) Generate(pkg *Package) error {
	var buf bytes.Buffer
	buf.WriteString(`syntax = "proto3";` + "\n")

	writePackageData(&buf, pkg)
	if len(pkg.Options) > 0 {
		writeOptions(&buf, pkg.Options, false)
		buf.WriteRune('\n')
	}

	for _, msg := range pkg.Messages {
		writeMessage(&buf, msg)
		buf.WriteRune('\n')
	}

	for _, enum := range pkg.Enums {
		writeEnum(&buf, enum)
		buf.WriteRune('\n')
	}

	if len(pkg.RPCs) > 0 {
		writeService(&buf, pkg)
	}

	return g.writeFile(pkg.Path, buf.Bytes())
}

func (g *Generator) writeFile(path string, data []byte) error {
	path = filepath.Join(g.basePath, path)
	fi, err := os.Stat(g.basePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path, fi.Mode()); err != nil {
		return err
	}

	file := filepath.Join(path, "generated.proto")
	if err := ioutil.WriteFile(file, data, fi.Mode()); err != nil {
		return err
	}

	report.Info("Generated proto: %s", file)
	return nil
}

func writePackageData(buf *bytes.Buffer, pkg *Package) {
	buf.WriteString(fmt.Sprintf("package %s;\n", pkg.Name))

	if len(pkg.Imports) > 0 {
		buf.WriteRune('\n')

		for _, i := range pkg.Imports {
			buf.WriteString(fmt.Sprintf(`import "%s";`, i))
			buf.WriteRune('\n')
		}
	}
	buf.WriteRune('\n')
}

func writeMessage(buf *bytes.Buffer, msg *Message) {
	writeDocs(buf, msg.Docs, false)
	buf.WriteString(fmt.Sprintf("message %s {\n", msg.Name))
	writeOptions(buf, msg.Options, true)

	if len(msg.Reserved) > 0 {
		buf.WriteString("\treserved ")

		for i, p := range msg.Reserved {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprint(p))
		}

		buf.WriteString(";\n")
	}

	for _, f := range msg.Fields {
		writeDocs(buf, f.Docs, true)
		buf.WriteRune('\t')
		if f.Repeated {
			buf.WriteString("repeated ")
		}

		buf.WriteString(f.Type.String())

		buf.WriteString(fmt.Sprintf(" %s = %d", f.Name, f.Pos))
		if len(f.Options) > 0 {
			buf.WriteRune(' ')
			writeFieldOptions(buf, f.Options)
		}
		buf.WriteString(";\n")
	}

	buf.WriteString("}\n")
}

func writeEnum(buf *bytes.Buffer, enum *Enum) {
	writeDocs(buf, enum.Docs, false)
	buf.WriteString(fmt.Sprintf("enum %s {\n", enum.Name))
	writeOptions(buf, enum.Options, true)

	for _, v := range enum.Values {
		writeDocs(buf, v.Docs, true)
		buf.WriteString(fmt.Sprintf("\t%s = %d", v.Name, v.Value))
		if len(v.Options) > 0 {
			buf.WriteRune(' ')
			writeFieldOptions(buf, v.Options)
		}
		buf.WriteString(";\n")
	}

	buf.WriteString("}\n")
}

func writeOptions(buf *bytes.Buffer, options Options, indent bool) {
	for _, opt := range options.Sorted() {
		if indent {
			buf.WriteRune('\t')
		}
		buf.WriteString(fmt.Sprintf("option %s = %s;\n", opt.Name, opt.Value))
	}
}

func writeFieldOptions(buf *bytes.Buffer, options Options) {
	buf.WriteRune('[')
	for i, opt := range options.Sorted() {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s = %s", opt.Name, opt.Value))
	}
	buf.WriteRune(']')
}

func writeDocs(buf *bytes.Buffer, docs []string, indent bool) {
	for _, d := range docs {
		if indent {
			buf.WriteRune('\t')
		}
		buf.WriteString("// ")
		buf.WriteString(d)
		buf.WriteRune('\n')
	}
}

func writeService(buf *bytes.Buffer, pkg *Package) {
	buf.WriteString(fmt.Sprintf("service %s {\n", pkg.ServiceName()))
	for _, rpc := range pkg.RPCs {
		writeDocs(buf, rpc.Docs, true)
		buf.WriteString(fmt.Sprintf(
			"\trpc %s (%s) returns (%s);\n",
			rpc.Name,
			rpc.Input,
			rpc.Output,
		))
	}
	buf.WriteString("}\n\n")
}
