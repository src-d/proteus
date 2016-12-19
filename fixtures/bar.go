package foo

import "github.com/src-d/proteus/fixtures/subpkg"

// Bar ...
type Bar struct {
	Bar uint64
	Baz Baz
}

// Baz ...
type Baz byte

// Saz ...
type Saz struct {
	Point subpkg.Point
	Foo   float64
}

const (
	// ABaz ...
	ABaz Baz = iota
	// BBaz ...
	BBaz
	// CBaz ...
	CBaz
	// DBaz ...
	DBaz
)
