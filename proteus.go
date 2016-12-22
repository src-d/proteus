package proteus

import (
	"github.com/src-d/proteus/protobuf"
	"github.com/src-d/proteus/resolver"
	"github.com/src-d/proteus/scanner"
)

// Options are all the available options to configure proto generation.
type Options struct {
	BasePath string
	Packages []string
}

// GenerateProtos generates proto files for the given options.
func GenerateProtos(options Options) error {
	scanner, err := scanner.New(options.Packages...)
	if err != nil {
		return err
	}

	pkgs, err := scanner.Scan()
	if err != nil {
		return err
	}

	r := resolver.New()
	r.Resolve(pkgs)

	t := protobuf.NewTransformer()
	g := protobuf.NewGenerator(options.BasePath)
	for _, p := range pkgs {
		pkg := t.Transform(p)
		if err := g.Generate(pkg); err != nil {
			return err
		}
	}

	return nil
}
