package source

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// ErrTooManyPackages is returned when there is more than one package in a
// directory where there should only be one Go package.
var ErrTooManyPackages = errors.New("more than one package found in a directory")

// PackageAST returs the AST of the package at the given path.
func PackageAST(path string) (pkg *ast.Package, err error) {
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
