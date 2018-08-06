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
	"github.com/odsod/stackdriver-go-sandbox/internal/zapgithub"
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
	commitHash      = flag.String("commitHash", "", "")
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
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize GCP logging client"))
	}
	gcpLogger := loggingClient.Logger("server")

	// Init GitHub source locator
	var sourceLocator zapgcp.SourceLocator
	var callerEncoder zapcore.CallerEncoder
	if *commitHash != "" {
		sourceLocator = zapgcp.NewGitHubSourceLocator(*commitHash)
		callerEncoder = zapgithub.GitHubCallerEncoder(*commitHash)
	} else {
		sourceLocator = zapgcp.FileAndFunctionSourceLocator
		callerEncoder = zapcore.ShortCallerEncoder
	}

	// Init zap GCP logging
	zapGCPCore := zapgcp.NewLoggingCore(
		zap.InfoLevel,
		zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey:     "+", // anything non-"" includes message
			StacktraceKey:  "+", // anything non-"" includes stack trace
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		gcpLogger,
		sourceLocator)

	// Init zap console logging
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeCaller = callerEncoder
	zapConsoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.Lock(os.Stderr),
		zap.InfoLevel)

	// Create zap logger
	logger := zap.New(zapcore.NewTee(zapConsoleCore, zapGCPCore)).WithOptions(zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel))
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
			err = errors.Wrap(err, "request failed")
			logger.Error("Request failed", zapextra.Proto("request", req), zap.Error(err))
			continue
		}
		logger.Info("Got response", zapextra.Proto("response", response))
	}
}
