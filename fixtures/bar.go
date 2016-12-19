// +build ignore
package foo

import "github.com/src-d/proteus/fixtures/subpkg"

type Bar struct {
	Bar uint64
	Baz Baz
}

type Baz byte

type Saz struct {
	Point subpkg.Point
	Foo   float64
}

const (
	ABaz Baz = iota
	BBaz
	CBaz
	DBaz
)
