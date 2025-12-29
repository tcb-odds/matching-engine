package main

import (
	"fmt"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tcb-odds/matching-engine/internal/app/server"
	"github.com/tcb-odds/matching-engine/internal/initializers"
	"github.com/tcb-odds/matching-engine/internal/modules/diagnostic"
	"github.com/tcb-odds/matching-engine/internal/shared/config"
	"github.com/tcb-odds/matching-engine/internal/shared/info"
	engineGrpc "github.com/tcb-odds/matching-engine/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	grpcPort = ":9000"
	httpPort = ":5001"
)

func main() {
	info.PrintAppInfo(config.AppName, config.AppVersion)

	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: No .env file found or error loading it")
	}

	fmt.Printf("%s server starting...\n", config.AppName)

	engine := gin.Default()
	initializers.InitModules(engine)

	go func() {
		fmt.Printf("HTTP server listening on %s\n", httpPort)
		if err := engine.Run(httpPort); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	subscriptionManager := server.NewSubscriptionManager()

	gs := grpc.NewServer()
	cs := server.NewEngine(subscriptionManager)
	engineGrpc.RegisterEngineServer(gs, cs)
	reflection.Register(gs)

	diagnostic.SetStatsProvider(cs)

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
