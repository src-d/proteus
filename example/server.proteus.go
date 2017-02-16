package example

import (
	"golang.org/x/net/context"
)

type exampleServiceServer struct {
}

func NewExampleServiceServer() *exampleServiceServer {
	return &exampleServiceServer{}
}
func (s *exampleServiceServer) GetAlphaTime(ctx context.Context, in *GetAlphaTimeRequest) (result *MyTime, err error) {
	result = new(MyTime)
	aux := GetAlphaTime()
	result = &aux
	return
}
func (s *exampleServiceServer) GetOmegaTime(ctx context.Context, in *GetOmegaTimeRequest) (result *MyTime, err error) {
	result = new(MyTime)
	result, err = GetOmegaTime()
	return
}
func (s *exampleServiceServer) GetPhone(ctx context.Context, in *GetPhoneRequest) (result *Product, err error) {
	result = new(Product)
	result = GetPhone()
	return
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
