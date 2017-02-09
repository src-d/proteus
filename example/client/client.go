package client

import (
	"github.com/src-d/proteus/example"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client struct {
	example.ExampleServiceClient
	Conn *grpc.ClientConn
}

func (c *Client) Close() {
	c.Conn.Close()
}

func (c *Client) RequestRandomNumber(mean, std float64) (float64, error) {
	res, err := c.RandomNumber(context.Background(), &example.RandomNumberRequest{
		Arg1: mean,
		Arg2: std,
	})
	if err != nil {
		return 0, err
	}
	return res.Result1, nil
}

func (c *Client) RequestAlphaTime() (*example.MyTime, error) {
	return c.GetAlphaTime(context.Background(), &example.GetAlphaTimeRequest{})
}

func (c *Client) RequestOmegaTime() (*example.MyTime, error) {
	return c.GetOmegaTime(context.Background(), &example.GetOmegaTimeRequest{})
}

func (c *Client) RequestRandomCategory() (*example.CategoryOptions, error) {
	return c.RandomCategory(context.Background(), &example.RandomCategoryRequest{})
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial("localhost:8001", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Client{
		ExampleServiceClient: example.NewExampleServiceClient(conn),
		Conn:                 conn,
	}, nil
}
