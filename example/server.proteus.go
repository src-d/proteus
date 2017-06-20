package example

import (
	xcontext "golang.org/x/net/context"
	"gopkg.in/src-d/proteus.v1/example/categories"
)

type exampleServiceServer struct {
}

func NewExampleServiceServer() *exampleServiceServer {
	return &exampleServiceServer{}
}
func (s *exampleServiceServer) GetAlphaTime(ctx xcontext.Context, in *GetAlphaTimeRequest) (result *MyTime, err error) {
	result = new(MyTime)
	aux := GetAlphaTime()
	result = &aux
	return
}
func (s *exampleServiceServer) GetDurationForLength(ctx xcontext.Context, in *GetDurationForLengthRequest) (result *MyDuration, err error) {
	result = new(MyDuration)
	result = GetDurationForLength(in.Arg1)
	return
}
func (s *exampleServiceServer) GetDurationForLengthCtx(ctx xcontext.Context, in *GetDurationForLengthCtxRequest) (result *MyDuration, err error) {
	result = new(MyDuration)
	result, err = GetDurationForLengthCtx(ctx, in.Arg1)
	return
}
func (s *exampleServiceServer) GetOmegaTime(ctx xcontext.Context, in *GetOmegaTimeRequest) (result *MyTime, err error) {
	result = new(MyTime)
	result, err = GetOmegaTime()
	return
}
func (s *exampleServiceServer) GetPhone(ctx xcontext.Context, in *GetPhoneRequest) (result *Product, err error) {
	result = new(Product)
	result = GetPhone()
	return
}
func (s *exampleServiceServer) RandomCategory(ctx xcontext.Context, in *RandomCategoryRequest) (result *categories.CategoryOptions, err error) {
	result = new(categories.CategoryOptions)
	aux := RandomCategory()
	result = &aux
	return
}
func (s *exampleServiceServer) RandomNumber(ctx xcontext.Context, in *RandomNumberRequest) (result *RandomNumberResponse, err error) {
	result = new(RandomNumberResponse)
	result.Result1 = RandomNumber(in.Arg1, in.Arg2)
	return
}
