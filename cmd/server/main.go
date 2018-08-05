package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/odsod/stackdriver-go-sandbox/api/sandbox"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	projectID       = flag.String("projectID", "", "")
	credentialsFile = flag.String("credentialsFile", "", "")
)

func main() {
	flag.Parse()
	stackdriverExporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: *projectID,
		MonitoringClientOptions: []option.ClientOption{
			option.WithCredentialsFile(*credentialsFile),
		},
		TraceClientOptions: []option.ClientOption{
			option.WithCredentialsFile(*credentialsFile),
		},
	})
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize Stackdriver exporter"))
	}
	view.RegisterExporter(stackdriverExporter)
	view.SetReportingPeriod(10 * time.Second)
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		panic(errors.Wrap(err, "failed to register metric views for gRPC server"))
	}
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(errors.Wrap(err, "failed to start listener"))
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
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
