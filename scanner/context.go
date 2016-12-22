package scanner

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// context holds all the scanning context of a single package. Contains all
// the enum values we find during the scan as well as some info extracted
// from the AST that will be needed throughout the process of scanning.
type context struct {
	// types holds the type declarations indexed by the type name. The TypeSpec
	// is guaranteed to include the comments, if any, even though they were on
	// the GenDecl.
	types map[string]*ast.TypeSpec
	// consts holds the const objects indexed by the const name. We store an
	// object instead of a ValueSpec because the iota of the const is not
	// available there.
	consts map[string]*ast.Object
	// enumValues contains all the values found until a point in time.
	// It is indexed by qualified type name e.g: time.Time
	enumValues map[string][]string
}

// ErrTooManyPackages is returned when there is more than one package in a
// directory where there should only be one Go package.
var ErrTooManyPackages = errors.New("more than one package found in a directory")

func buildPackageAST(path string) (pkg *ast.Package, err error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, filepath.Join(goSrc, path), nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		return nil, ErrTooManyPackages
	}

	for _, p := range pkgs {
		pkg = p
	}

	return
}

func newContext(path string) (*context, error) {
	pkg, err := buildPackageAST(path)
	if err != nil {
		return nil, err
	}

	return &context{
		types:      findPkgTypes(pkg),
		consts:     findObjectsOfType(pkg, ast.Con),
		enumValues: make(map[string][]string),
	}, nil
}

func findPkgTypes(pkg *ast.Package) map[string]*ast.TypeSpec {
	f := ast.MergePackageFiles(pkg, 0)

	var types = make(map[string]*ast.TypeSpec)
	for _, d := range f.Decls {
		decl := d.(*ast.GenDecl)
		if decl.Tok == token.TYPE {
			for _, s := range decl.Specs {
				spec := s.(*ast.TypeSpec)
				if spec.Doc == nil {
					spec.Doc = decl.Doc
				}
				types[spec.Name.Name] = spec
			}
		}
	}

	return types
}

func findObjectsOfType(pkg *ast.Package, kind ast.ObjKind) map[string]*ast.Object {
	var objects = make(map[string]*ast.Object)

	for _, f := range pkg.Files {
		for k, o := range f.Scope.Objects {
			if o.Kind == kind {
				objects[k] = o
			}
		}
	}

	return objects
}

const genComment = `//proteus:generate`

func (ctx *context) shouldGenerateType(name string) bool {
	if typ, ok := ctx.types[name]; ok && typ.Doc != nil {
		for _, l := range typ.Doc.List {
			if strings.HasPrefix(l.Text, genComment) {
				return true
			}
		}
	}
	return false
}
