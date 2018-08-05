package main

import (
	"context"
	"flag"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/odsod/stackdriver-go-sandbox/api/sandbox"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
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

	// Init zap logging
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize logging"))
	}

	// Init Stackdriver logging
	loggingClient, err := logging.NewClient(
		ctx, *projectID, option.WithCredentialsFile(*credentialsFile))
	stackdriverLogger := loggingClient.Logger("client").StandardLogger(logging.Info)

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

	// Connect gRPC client
	conn, err := grpc.Dial(
		":3000",
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithInsecure())
	if err != nil {
		panic(errors.Wrap(err, "failed to connect"))
	}
	defer conn.Close()
	client := sandboxpb.NewSandboxClient(conn)

	// Send requests
	for {
		ctx, cancel := context.WithTimeout(ctx, 850*time.Millisecond)
		response, err := client.Ping(ctx, &sandboxpb.PingRequest{Msg: "ping"})
		cancel()
		if err != nil {
			zapLogger.Error("Request failed", zap.Error(err))
			stackdriverLogger.Printf("Got error: %v", err)
			continue
		}
		zapLogger.Info("Got response", zap.Stringer("response", response))
		stackdriverLogger.Printf("Got response: %v", response)
	}
}
