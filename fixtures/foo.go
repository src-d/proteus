// +build ignore
package foo

import (
	"net/url"
	"time"
)

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

type IntList []int

type Qux struct {
	A int
	B int
}
