package main

import (
	"fmt"

	"github.com/src-d/proteus/example"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func requestRandomCategories(client example.ExampleServiceClient, num int) {
	for i := 0; i < num; i++ {
		fmt.Printf("Category %02d:", i+1)
		cat, err := client.RandomCategory(context.Background(), &example.RandomCategoryRequest{})
		switch {
		case err != nil:
			fmt.Printf(" errored %s", err)
		case cat.CanBuy && cat.ShowPrices:
			fmt.Print(" can buy and shows prices")
		case cat.CanBuy:
			fmt.Print(" can buy but does not show prices")
		case cat.ShowPrices:
			fmt.Print(" cannot buy but shows prices")
		default:
			fmt.Print(" cannot do anything")
		}
		fmt.Println()
	}
}

func requestRandomNumbers(client example.ExampleServiceClient, num int, mean, std float64) {
	for i := 0; i < num; i++ {
		fmt.Printf("Random number %03d with mean %f and std %f: ", i+1, mean, std)
		num, err := client.RandomNumber(context.Background(), &example.RandomNumberRequest{
			Arg1: mean,
			Arg2: std,
		})
		if err != nil {
			fmt.Printf("errored %s", err)
		} else {
			fmt.Print(num.Result1)
		}
		fmt.Println()
	}
}

func main() {
	conn, err := grpc.Dial("localhost:8001", grpc.WithInsecure())
	if err != nil {
		grpclog.Fatalf("could not connect to server: %s", err)
	}
	defer conn.Close()

	client := example.NewExampleServiceClient(conn)

	requestRandomCategories(client, 10)
	requestRandomNumbers(client, 100, 5, 3.98)
}
