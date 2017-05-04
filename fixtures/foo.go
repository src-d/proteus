package foo

import (
	"net/url"
	"time"
)

// Foo ...
//proteus:generate
type Foo struct {
	Bar
	IntList    []int
	IntArray   [8]int
	Map        map[string]*Qux
	AliasedMap MyMap
	Timestamp  time.Time
	External   url.URL
	Duration   time.Duration
	Aliased    MyInt
}

// IntList ...
//proteus:generate
type MyInt int

// MyIntMap
type MyMap map[string]Jur

// Jur should be generated since MyMap is referencing it and MyMap is used in Foo
type Jur struct {
	A string
}

// Qux ...
type Qux struct {
	A int
	B int
}
