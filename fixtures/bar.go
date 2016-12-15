// +build ignore
package foo

type Bar struct {
	Bar uint64
	Baz Baz
}

type Baz byte

const (
	ABaz Baz = iota
	BBaz
	CBaz
	DBaz
)
