package proteus

import (
	"fmt"

	"github.com/src-d/proteus/example/client"
	"github.com/src-d/proteus/example/server"
)

func ExampleProteus() {
	addr := "localhost:8001"
	s, err := server.NewServer(addr)
	if err != nil {
		panic(fmt.Sprintf("could not open server: %s", err))
	}
	defer s.Stop()
	c, err := client.NewClient(addr)
	if err != nil {
		panic(fmt.Sprintf("could not open client: %s", err))
	}
	defer c.Close()

	n, err := c.RequestRandomNumber(0, 1)
	if err != nil {
		panic(fmt.Sprintf("could not receive random number: %s", err))
	}
	fmt.Println(n)

	cat, err := c.RequestRandomCategory()
	if err != nil {
		panic(fmt.Sprintf("could not receive category: %s", err))
	}
	fmt.Println(cat.CanBuy)
	fmt.Println(cat.ShowPrices)
	// Output: 4
	// true
	// true
}
