package proteus

import (
	"gopkg.in/src-d/proteus.v1/protobuf"
	"gopkg.in/src-d/proteus.v1/resolver"
	"gopkg.in/src-d/proteus.v1/rpc"
	"gopkg.in/src-d/proteus.v1/scanner"
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
	t.SetStructSet(createStructTypeSet(pkgs))
	t.SetEnumSet(createEnumTypeSet(pkgs))
	for _, p := range pkgs {
		pkg := t.Transform(p)
		if err := generate(p, pkg); err != nil {
			return err
		}
	}

	return nil
}

func createStructTypeSet(pkgs []*scanner.Package) protobuf.TypeSet {
	ts := protobuf.NewTypeSet()
	for _, p := range pkgs {
		for _, s := range p.Structs {
			ts.Add(p.Path, s.Name)
		}
	}
	return ts
}

func createEnumTypeSet(pkgs []*scanner.Package) protobuf.TypeSet {
	ts := protobuf.NewTypeSet()
	for _, p := range pkgs {
		for _, e := range p.Enums {
			ts.Add(p.Path, e.Name)
		}
	}
	return ts
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
