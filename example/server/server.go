package main

import (
	"net"

	"github.com/src-d/proteus/example"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func main() {
	lis, err := net.Listen("tcp", "localhost:8001")
	if err != nil {
		grpclog.Fatalf("could not open port 8001")
	}

	grpcServer := grpc.NewServer()
	example.RegisterExampleServiceServer(grpcServer, example.NewExampleServiceServer())
	grpcServer.Serve(lis)
}
