package subpkg

// Point ...
//proteus:generate
type Point struct {
	X int
	Y int
}

func (p *Point) Dist(p2 Point) float64 {
	return .0
}

// NotGenerated ...
type NotGenerated struct{}

func Foo(a int) (float64, error) {
	return float64(a), nil
}

//proteus:generate
func Generated(a string) (bool, error) {
	return len(a) > 0, nil
}

//proteus:generate
func (p Point) GeneratedMethod(a int32) *Point {
	return &p
}

//proteus:generate
func (p *Point) GeneratedMethodOnPointer(a bool) *Point {
	return p
}

type MyContainer struct {
	name string
}

//proteus:generate
func (c *MyContainer) Name() string {
	return c.Name()
}
