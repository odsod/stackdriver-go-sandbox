package main

import (
	"context"
	"log"
	"time"

	"github.com/odsod/stackdriver-go-sandbox/api/sandbox"
	"github.com/odsod/stackdriver-go-sandbox/internal/ocextra"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
)

func main() {
	view.RegisterExporter(&ocextra.PrintExporter{})
	view.SetReportingPeriod(time.Second)
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		panic(errors.Wrap(err, "failed to register metric views for gRPC server"))
	}
	ctx := context.Background()
	conn, err := grpc.Dial(
		":3000",
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithInsecure())
	if err != nil {
		panic(errors.Wrap(err, "failed to connect"))
	}
	defer conn.Close()
	client := sandboxpb.NewSandboxClient(conn)
	for {
		ctx, cancel := context.WithTimeout(ctx, 850*time.Millisecond)
		response, err := client.Ping(ctx, &sandboxpb.PingRequest{Msg: "ping"})
		cancel()
		if err != nil {
			log.Printf("Got error: %+v", err)
			continue
		}
		log.Printf("Got response: %v", response)
	}
}
