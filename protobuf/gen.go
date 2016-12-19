package protobuf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/src-d/proteus/report"
)

type Generator struct {
	basePath string
}

func NewGenerator(basePath string) *Generator {
	return &Generator{basePath}
}

func (g *Generator) Generate(pkg *Package) error {
	var buf bytes.Buffer
	buf.WriteString(`syntax = "proto3";` + "\n")

	writePackageData(&buf, pkg)
	for _, msg := range pkg.Messages {
		writeMessage(&buf, msg)
		buf.WriteRune('\n')
	}

	for _, enum := range pkg.Enums {
		writeEnum(&buf, enum)
		buf.WriteRune('\n')
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
	buf.WriteString(fmt.Sprintf("message %s {\n", msg.Name))
	writeOptions(buf, msg.Options)

	// TODO: Write reserved fields

	for _, f := range msg.Fields {
		buf.WriteRune('\t')
		if f.Repeated {
			buf.WriteString("repeated ")
		}

		buf.WriteString(f.Type.String())

		// TODO: Write field options
		buf.WriteString(fmt.Sprintf(" %s = %d;\n", f.Name, f.Pos))
	}

	buf.WriteString("}\n")
}

func writeEnum(buf *bytes.Buffer, enum *Enum) {
	buf.WriteString(fmt.Sprintf("enum %s {\n", enum.Name))
	writeOptions(buf, enum.Options)

	for _, v := range enum.Values {
		// TODO: Write enum value options
		buf.WriteString(fmt.Sprintf("\t%s = %d;\n", v.Name, v.Value))
	}

	buf.WriteString("}\n")
}

func writeOptions(buf *bytes.Buffer, options Options) {
	for _, opt := range options.Sorted() {
		fmt.Println(opt)
		buf.WriteString(fmt.Sprintf("\toption %s = %s;\n", opt.Name, opt.Value))
	}
}
