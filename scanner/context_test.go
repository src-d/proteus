package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var goSrc = filepath.Join(os.Getenv("GOPATH"), "src")
var projectDir = filepath.Join(goSrc, "gopkg.in/src-d/proteus.v1")

func TestNewContext_error(t *testing.T) {
	createDirWithMultipleFiles("erroring")
	defer removeDir("erroring")
	_, err := newContext("gopkg.in/src-d/proteus.v1/fixtures/erroring/multiple")
	assert.NotNil(t, err)
}

func createDirWithMultipleFiles(pkg string) error {
	path := filepath.Join(projectDir, pkg)
	os.Mkdir(path, os.ModeDir)

	f, err := os.Create(filepath.Join(path, "foo.go"))
	if err != nil {
		return err
	}
	f.Write([]byte("package foo"))
	f.Close()

	f, err = os.Create(filepath.Join(path, "bar.go"))
	if err != nil {
		return err
	}
	f.Write([]byte("package bar"))
	f.Close()

	return nil
}

func removeDir(pkg string) {
	os.RemoveAll(filepath.Join(projectDir, pkg))
}
