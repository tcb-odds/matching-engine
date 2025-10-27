package main

import (
	"fmt"
	"net"
	"os"

	"github.com/tcb-odds/matching-engine/internal/app/server"
	engineGrpc "github.com/tcb-odds/matching-engine/pkg/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":9000"
)

func main() {
	// Create subscription manager
	subscriptionManager := server.NewSubscriptionManager()

	gs := grpc.NewServer()
	cs := server.NewEngine(subscriptionManager)
	engineGrpc.RegisterEngineServer(gs, cs)

	reflection.Register(gs)

	l, err := net.Listen("tcp", port)
	if err != nil {
		e := fmt.Errorf("Unable to listen server, err: %v", err)
		fmt.Println(e)
		os.Exit(1)
	}
	fmt.Printf("grpc server listening to %s\n", port)
	fmt.Printf("Subscription manager initialized\n")
	gs.Serve(l)
}
