package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	proto "github.com/tcb-odds/matching-engine/pkg/proto"
)

func main() {
	// Get server address from environment or use default
	serverAddr := "localhost:9000"
	if addr := os.Getenv("MATCHING_ENGINE_HOST"); addr != "" {
		serverAddr = addr
	}

	fmt.Printf("Connecting to matching engine at %s...\n", serverAddr)

	// Connect to the gRPC server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewEngineClient(conn)

	// Subscribe to all order updates
	fmt.Println("Subscribing to order updates...")
	stream, err := client.SubscribeToOrderUpdates(context.Background(), &proto.SubscribeRequest{})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Listen for events in a goroutine
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("Stream error: %v", err)
				return
			}

			// Print the event
			fmt.Printf("\nEVENT RECEIVED:\n")
			fmt.Printf("  Type: %s\n", event.EventType)
			fmt.Printf("  Pair: %s\n", event.Pair)
			fmt.Printf("  Timestamp: %d\n", event.Timestamp)
			fmt.Printf("  Order ID: %s\n", event.Order.ID)
			fmt.Printf("  Order Type: %s\n", event.Order.Type)
			fmt.Printf("  Amount: %s\n", event.Order.Amount)
			fmt.Printf("  Price: %s\n", event.Order.Price)
			fmt.Printf("  Filled Amount: %s\n", event.Order.FilledAmount)

			if event.MatchedWith != nil {
				fmt.Printf("  Matched with Order ID: %s\n", event.MatchedWith.ID)
				fmt.Printf("  Trade Amount: %s\n", event.TradeAmount)
				fmt.Printf("  Execution Price: %s\n", event.ExecutionPrice)
			}
			fmt.Println("-------------------------------------")
		}
	}()

	// Simulate placing some orders to generate events
	fmt.Println("\nPlacing test orders...\n")

	// Order 1: Buy limit order
	time.Sleep(1 * time.Second)
	fmt.Println("Placing Buy Order...")
	_, err = client.Process(context.Background(), &proto.Order{
		ID:     "order-buy-1",
		Type:   proto.Side_buy,
		Amount: "10",
		Price:  "50000",
		Pair:   "BTC/USDT",
	})
	if err != nil {
		log.Printf("Error placing buy order: %v", err)
	}

	// Order 2: Sell limit order that matches
	time.Sleep(2 * time.Second)
	fmt.Println("Placing Sell Order (should match)...")
	_, err = client.Process(context.Background(), &proto.Order{
		ID:     "order-sell-1",
		Type:   proto.Side_sell,
		Amount: "5",
		Price:  "49000",
		Pair:   "BTC/USDT",
	})
	if err != nil {
		log.Printf("Error placing sell order: %v", err)
	}

	// Order 3: Cancel an order
	time.Sleep(2 * time.Second)
	fmt.Println("Cancelling remaining buy order...")
	_, err = client.Cancel(context.Background(), &proto.Order{
		ID:   "order-buy-1",
		Pair: "BTC/USDT",
	})
	if err != nil {
		log.Printf("Error cancelling order: %v", err)
	}

	// Keep listening for a bit
	time.Sleep(3 * time.Second)
	fmt.Println("\nDemo completed!")
}
