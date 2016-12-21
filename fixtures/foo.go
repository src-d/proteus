package foo

import (
	"net/url"
	"time"
)

// Foo ...
//proteus:generate
type Foo struct {
	Bar
	IntList   []int
	IntArray  [8]int
	Map       map[string]*Qux
	Timestamp time.Time
	External  url.URL
	Duration  time.Duration
	Aliased   IntList
}

// IntList ...
//proteus:generate
type IntList []int

// Qux ...
type Qux struct {
	A int
	B int
}
