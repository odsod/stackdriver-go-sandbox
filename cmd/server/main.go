package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"os"

	"cloud.google.com/go/logging"
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
	ctx := context.Background()

	// Check credentials file
	if _, err := os.Stat(*credentialsFile); err != nil {
		panic(errors.Wrapf(err, "credentials file not found: %v", *credentialsFile))
	}

	// Init logging
	loggingClient, err := logging.NewClient(
		ctx, *projectID, option.WithCredentialsFile(*credentialsFile))
	logger := loggingClient.Logger("server").StandardLogger(logging.Info)

	// Init monitoring
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
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		panic(errors.Wrap(err, "failed to register metric views for gRPC server"))
	}

	// Start server
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		panic(errors.Wrap(err, "failed to start listener"))
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	sandboxpb.RegisterSandboxServer(grpcServer, &sandboxServer{logger: logger})
	grpcServer.Serve(lis)
}

type sandboxServer struct {
	logger *log.Logger
}

func (s *sandboxServer) Ping(_ context.Context, req *sandboxpb.PingRequest) (*sandboxpb.PingResponse, error) {
	log.Printf("Got request: %v", req)
	s.logger.Printf("Got request: %v", req)
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
