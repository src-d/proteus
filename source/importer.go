package source

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	goPath = os.Getenv("GOPATH")
	goSrc  = filepath.Join(goPath, "src")
)

// Importer is an implementation of `types.Importer` and `types.ImporterFrom`
// that builds actual source files and not the compiled objects in the pkg
// directory.
// It is safe to use it concurrently as far as the underlying default importer
// is thread safe.
// A package is cached after building it the first time.
type Importer struct {
	mut   sync.RWMutex
	cache map[string]*types.Package

	defaultImporter types.Importer
}

// NewImporter creates a new Importer instance with the default importer of
// the runtime assigned as the underlying importer.
func NewImporter() *Importer {
	return &Importer{
		cache:           make(map[string]*types.Package),
		defaultImporter: importer.Default(),
	}
}

// Import returns the imported package for the given import
// path, or an error if the package couldn't be imported.
// Two calls to Import with the same path return the same
// package.
func (i *Importer) Import(path string) (*types.Package, error) {
	return i.ImportFrom(path, goSrc, 0)
}

// ImportFrom returns the imported package for the given import
// path when imported by the package in srcDir, or an error
// if the package couldn't be imported. The mode value must
// be 0; it is reserved for future use.
// Two calls to ImportFrom with the same path and srcDir return
// the same package.
func (i *Importer) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	i.mut.Lock()
	if pkg, ok := i.cache[path]; ok {
		i.mut.Unlock()
		return pkg, nil
	}
	i.mut.Unlock()

	root, files, err := i.getSourceFiles(path, srcDir)
	if err != nil {
		return nil, err
	}

	// If it's not on the GOPATH use the default importer instead
	if !strings.HasPrefix(root, goPath) {
		imp, ok := i.defaultImporter.(types.ImporterFrom)
		if ok {
			return imp.ImportFrom(path, srcDir, mode)
		}
		return imp.Import(path)
	}

	pkg, err := i.parseSourceFiles(root, files)
	if err != nil {
		return nil, err
	}

	i.mut.Lock()
	i.cache[path] = pkg
	i.mut.Unlock()
	return pkg, nil
}

func (i *Importer) getSourceFiles(path, srcDir string) (string, []string, error) {
	pkg, err := build.Import(path, srcDir, 0)
	if err != nil {
		return "", nil, err
	}

	var filenames []string
	filenames = append(filenames, pkg.GoFiles...)
	filenames = append(filenames, pkg.CgoFiles...)

	if len(filenames) == 0 {
		return "", nil, fmt.Errorf("no go source files in path: %s", path)
	}

	var paths []string
	for _, f := range filenames {
		paths = append(paths, filepath.Join(pkg.Dir, f))
	}

	return pkg.Dir, paths, nil
}

func (i *Importer) parseSourceFiles(root string, paths []string) (*types.Package, error) {
	var files []*ast.File
	fs := token.NewFileSet()
	for _, p := range paths {
		f, err := parser.ParseFile(fs, p, nil, 0)
		if err != nil {
			return nil, err
		}

		files = append(files, f)
	}

	config := types.Config{
		FakeImportC: true,
		Importer:    i,
	}

	return config.Check(root, fs, files, nil)
}
