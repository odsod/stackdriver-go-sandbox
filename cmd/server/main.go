package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/odsod/stackdriver-go-sandbox/api/sandbox"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(errors.Wrap(err, "failed to start listener"))
	}
	grpcServer := grpc.NewServer()
	sandboxpb.RegisterSandboxServer(grpcServer, &sandboxServer{})
	grpcServer.Serve(lis)
}

type sandboxServer struct{}

func (s *sandboxServer) Ping(_ context.Context, req *sandboxpb.PingRequest) (*sandboxpb.PingResponse, error) {
	log.Printf("Got request: %v", req)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	switch rand.Intn(10) {
	case 0:
		return nil, status.Error(codes.InvalidArgument, "argument error")
	case 1:
		return nil, status.Error(codes.FailedPrecondition, "precondition error")
	case 2:
		return nil, status.Error(codes.Unavailable, "rate-limited")
	default:
		return &sandboxpb.PingResponse{Msg: "pong"}, nil
	}
}
