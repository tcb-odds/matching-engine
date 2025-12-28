package main

import (
	"fmt"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tcb-odds/matching-engine/internal/app/server"
	"github.com/tcb-odds/matching-engine/internal/initializers"
	"github.com/tcb-odds/matching-engine/internal/modules/diagnostic"
	"github.com/tcb-odds/matching-engine/internal/shared/config"
	engineGrpc "github.com/tcb-odds/matching-engine/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	grpcPort = ":9000"
	httpPort = ":5001"
)

func main() {
	fmt.Printf("%s server starting...\n", config.AppName)

	// Initialize Gin engine
	engine := gin.Default()

	// Initialize modules (HTTP endpoints)
	initializers.InitModules(engine)

	// Start HTTP server in background
	go func() {
		fmt.Printf("HTTP server listening on %s\n", httpPort)
		if err := engine.Run(httpPort); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Create subscription manager
	subscriptionManager := server.NewSubscriptionManager()

	// Setup gRPC server
	gs := grpc.NewServer()
	cs := server.NewEngine(subscriptionManager)
	engineGrpc.RegisterEngineServer(gs, cs)
	reflection.Register(gs)

	// Set stats provider for diagnostic endpoints
	diagnostic.SetStatsProvider(cs)

	// Start gRPC server
	l, err := net.Listen("tcp", grpcPort)
	if err != nil {
		e := fmt.Errorf("Unable to listen server, err: %v", err)
		fmt.Println(e)
		os.Exit(1)
	}
	fmt.Printf("gRPC server listening to %s\n", grpcPort)
	fmt.Printf("Subscription manager initialized\n")
	gs.Serve(l)
}
