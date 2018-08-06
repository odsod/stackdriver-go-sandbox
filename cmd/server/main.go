package main

import (
	"context"
	"flag"
	"math/rand"
	"net"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	logger.Info("Initializing monitoring...")
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

	// Start server
	logger.Info("Binding listener...", zap.Int("port", 3000))
	lis, err := net.Listen("tcp", ":3000")
	if err != nil {
		logger.Panic("Failed to bind listener", zap.Error(err))
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	sandboxpb.RegisterSandboxServer(grpcServer, &sandboxServer{
		logger: logger,
	})
	logger.Info("Starting server...")
	grpcServer.Serve(lis)
}

type sandboxServer struct {
	logger *zap.Logger
}

func (s *sandboxServer) Ping(_ context.Context, req *sandboxpb.PingRequest) (*sandboxpb.PingResponse, error) {
	s.logger.Info("Got request", zapextra.Proto("request", req))
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
