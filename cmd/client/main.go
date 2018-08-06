package main

import (
	"context"
	"flag"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/odsod/stackdriver-go-sandbox/api/sandbox"
	"github.com/odsod/stackdriver-go-sandbox/internal/zapextra"
	"github.com/odsod/stackdriver-go-sandbox/internal/zapgcp"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	// Init GCP logger
	loggingClient, err := logging.NewClient(
		ctx, *projectID, option.WithCredentialsFile(*credentialsFile))
	gcpLogger := loggingClient.Logger("server")

	// Init zap GCP logging
	zapGCPCore := zapgcp.NewCore(
		zap.InfoLevel,
		zapcore.NewConsoleEncoder(zapgcp.NewEncoderConfig()),
		gcpLogger,
		zapgcp.FileAndFunctionSourceLocator)

	// Init zap console logging
	zapConsoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stderr),
		zap.InfoLevel)

	// Create zap logger
	logger := zap.New(zapcore.NewTee(zapConsoleCore, zapGCPCore))
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize logging"))
	}
	logger.Info("Starting up: Logging initialized")

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
		logger.Panic("Failed to initialize Stackdriver exporter", zap.Error(err))
	}
	view.RegisterExporter(stackdriverExporter)
	view.SetReportingPeriod(10 * time.Second)
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		logger.Panic("Failed to register metric views for gRPC server", zap.Error(err))
	}

	// Connect gRPC client
	conn, err := grpc.Dial(
		":3000",
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithInsecure())
	if err != nil {
		logger.Panic("Failed to bind listener", zap.Error(err))
	}
	defer conn.Close()
	client := sandboxpb.NewSandboxClient(conn)

	// Send requests
	for {
		ctx, cancel := context.WithTimeout(ctx, 850*time.Millisecond)
		req := &sandboxpb.PingRequest{Msg: "ping"}
		response, err := client.Ping(ctx, req)
		cancel()
		if err != nil {
			logger.Error("Request failed", zapextra.Proto("request", req), zap.Error(err))
			continue
		}
		logger.Info("Got response", zapextra.Proto("response", response))
	}
}
