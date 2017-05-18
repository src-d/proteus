package example

import (
	"fmt"
	"time"

	"gopkg.in/src-d/proteus.v1/example/categories"
)

//go:generate proteus -p gopkg.in/src-d/proteus.v1/example -f $GOPATH/src/gopkg.in/src-d/proteus.v1/example/protos

//proteus:generate
type Product struct {
	Model

	Name  string
	Price Prices

	Tags              Tags
	CategoryID        int64
	PrimaryCategoryID int8
	// Category will not be generated because we explicitly said so.
	Category Category `proteus:"-"`
}

type Prices map[string]Price

type Price struct {
	Currency string
	Amount   int64
}

//proteus:generate
type Category struct {
	Model

	Name    string
	Type    Type
	Color   Color
	Options categories.CategoryOptions
}

func (c *Category) String() string {
	return c.Name
}

type Tags []string

// Type will be transformed into an enum.
//proteus:generate
type Type byte

const (
	Public Type = iota
	Private
	Custom
)

func (t Type) String() string {
	switch t {
	case Public:
		return "Public"
	case Private:
		return "Private"
	case Custom:
		return "Custom"
	}
	return "UnknownType"
}

// Color does not have proteus:generate in a comment, so all fields of
// type Color will be treated as just string, not as an enum.
type Color string

const (
	Blue   Color = "blue"
	Red    Color = "red"
	Yellow Color = "yellow"
)

// Model is not marked for generation, so it won't be generated.
type Model struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

// User will not be generated.
type User struct {
	Model

	Username string
	Password string
	Email    string
}

type MyTime struct {
	Time time.Time
	Name string
}

//proteus:generate
type MyDuration struct {
	Duration time.Duration
	Name     string
}

//proteus:generate
func RandomNumber(mean, std float64) float64 {
	// Related documentation: https://xkcd.com/221/
	return 4*std + mean // 4 was chosen using the XKCD RNG
}

//proteus:generate
func RandomCategory() categories.CategoryOptions {
	return categories.CategoryOptions{
		ShowPrices: RandomBool(),
		CanBuy:     RandomBool(),
	}
}

//proteus:generate
func GetAlphaTime() MyTime {
	return MyTime{Time: time.Unix(0, 0), Name: "alpha"}
}

//proteus:generate
func GetOmegaTime() (*MyTime, error) {
	t, err := time.Parse("Jan 2, 2006 at 3:04pm", "Dec 12, 2012 at 10:30am")
	if err != nil {
		return nil, err
	}

	return &MyTime{Time: t, Name: "omega"}, nil
}

//proteus:generate
func GetDurationForLength(meters int64) *MyDuration {
	return &MyDuration{
		Duration: time.Second * time.Duration(meters/299792458),
		Name:     fmt.Sprintf("The light takes this duration to travel %dm", meters),
	}
}

//proteus:generate
func GetPhone() *Product {
	return &Product{
		Name:       "MiPhone",
		Price:      map[string]Price{"EUR": Price{"EUR", 12300}},
		Tags:       Tags{"cool", "mi", "phone"},
		CategoryID: 1,
		Category:   Category{},
	}
}

func RandomBool() bool {
	return true // Truly random. Selected by flipping a coin... once.
}
