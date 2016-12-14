// +build ignore
package foo

type Foo struct {
	Bar
	IntList  []int
	IntArray [8]int
	Map      map[string]*Qux
}

type Qux struct {
	A int
	B int
}
