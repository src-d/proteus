package server

import (
	"net"

	"github.com/src-d/proteus/example"

	"google.golang.org/grpc"
)

// NewServer returns a new server serving in the given address
func NewServer(addr string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	example.RegisterExampleServiceServer(grpcServer, example.NewExampleServiceServer())
	go grpcServer.Serve(lis)
	return grpcServer, nil
}
