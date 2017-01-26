package proteus

import (
	"github.com/src-d/proteus/protobuf"
	"github.com/src-d/proteus/resolver"
	"github.com/src-d/proteus/rpc"
	"github.com/src-d/proteus/scanner"
)

// Options are all the available options to configure proto generation.
type Options struct {
	BasePath string
	Packages []string
}

type generator func(*scanner.Package, *protobuf.Package) error

func transformToProtobuf(packages []string, generate generator) error {
	scanner, err := scanner.New(packages...)
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
	for _, p := range pkgs {
		pkg := t.Transform(p)
		if err := generate(p, pkg); err != nil {
			return err
		}
	}

	return nil
}

// GenerateProtos generates proto files for the given options.
func GenerateProtos(options Options) error {
	g := protobuf.NewGenerator(options.BasePath)
	return transformToProtobuf(options.Packages, func(_ *scanner.Package, pkg *protobuf.Package) error {
		return g.Generate(pkg)
	})
}

// GenerateRPCServer generates the gRPC server implementation of the given
// packages.
func GenerateRPCServer(packages []string) error {
	g := rpc.NewGenerator()
	return transformToProtobuf(packages, func(p *scanner.Package, pkg *protobuf.Package) error {
		return g.Generate(pkg, p.Path)
	})
}
