package example

import (
	"golang.org/x/net/context"
)

type exampleServiceServer struct {
}

func NewExampleServiceServer() *exampleServiceServer {
	return &exampleServiceServer{}
}
func (s *exampleServiceServer) RandomCategory(ctx context.Context, in *RandomCategoryRequest) (result *CategoryOptions, err error) {
	result = new(CategoryOptions)
	aux := RandomCategory()
	result = &aux
	return
}
func (s *exampleServiceServer) RandomNumber(ctx context.Context, in *RandomNumberRequest) (result *RandomNumberResponse, err error) {
	result = new(RandomNumberResponse)
	result.Result1 = RandomNumber(in.Arg1, in.Arg2)
	return
}
