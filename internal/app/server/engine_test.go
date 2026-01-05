package server

import (
	"context"
	"sync"
	"testing"
	"time"

	engine2 "github.com/tcb-odds/matching-engine/internal/app/engine"
	engine3 "github.com/tcb-odds/matching-engine/pkg/proto"
)

// eventCollector subscribes to events and collects them for testing
type eventCollector struct {
	mu     sync.Mutex
	events []*engine3.OrderUpdateEvent
	ctx    context.Context
	cancel context.CancelFunc
}

func newEventCollector(sm *SubscriptionManager) *eventCollector {
	ctx, cancel := context.WithCancel(context.Background())
	collector := &eventCollector{
		events: make([]*engine3.OrderUpdateEvent, 0),
		ctx:    ctx,
		cancel: cancel,
	}

	// Subscribe to events
	sub := sm.Subscribe(ctx)

	// Start listening in a goroutine
	go func() {
		for {
			select {
			case event, ok := <-sub.Channel:
				if !ok {
					return
				}
				collector.mu.Lock()
				collector.events = append(collector.events, event)
				collector.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	return collector
}

func (ec *eventCollector) GetEvents() []*engine3.OrderUpdateEvent {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	result := make([]*engine3.OrderUpdateEvent, len(ec.events))
	copy(result, ec.events)
	return result
}

func (ec *eventCollector) ClearEvents() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.events = make([]*engine3.OrderUpdateEvent, 0)
}

func (ec *eventCollector) Close() {
	ec.cancel()
}

// newTestEngine creates an Engine with event collection capability
func newTestEngine() (*Engine, *eventCollector) {
	sm := NewSubscriptionManager()
	collector := newEventCollector(sm)

	eng := &Engine{
		book:                map[string]*engine2.OrderBook{},
		subscriptionManager: sm,
	}

	return eng, collector
}

// TestProcessMarket_MatchedWithOrderID tests that the MatchedWith field
// is correctly set to the counterparty order ID in matched events
func TestProcessMarket_MatchedWithOrderID(t *testing.T) {
	eng, collector := newTestEngine()
	defer collector.Close()

	pair := "BTC/USDT"
	ctx := context.Background()

	// Create sell orders in the book first
	sellOrder1 := &engine3.Order{
		ID:     "1",
		Type:   engine3.Side_sell,
		Amount: "0.02",
		Price:  "3",
		Pair:   pair,
	}

	sellOrder2 := &engine3.Order{
		ID:     "2",
		Type:   engine3.Side_sell,
		Amount: "0.03",
		Price:  "4",
		Pair:   pair,
	}

	// Add sell orders to the book
	_, err := eng.Process(ctx, sellOrder1)
	if err != nil {
		t.Fatalf("Failed to process sell order 1: %v", err)
	}

	_, err = eng.Process(ctx, sellOrder2)
	if err != nil {
		t.Fatalf("Failed to process sell order 2: %v", err)
	}

	// Clear events from adding orders to the book
	collector.ClearEvents()

	// Now create a market buy order that will match both sell orders
	marketBuyOrder := &engine3.Order{
		ID:     "3",
		Type:   engine3.Side_buy,
		Amount: "0.05",
		Price:  "0", // Market order (price doesn't matter)
		Pair:   pair,
	}

	_, err = eng.ProcessMarket(ctx, marketBuyOrder)
	if err != nil {
		t.Fatalf("Failed to process market buy order: %v", err)
	}

	// Give a small delay for events to be broadcasted
	time.Sleep(10 * time.Millisecond)

	// Get all broadcasted events
	events := collector.GetEvents()

	// We should have 3 matched events:
	// 1. Sell order 1 matched
	// 2. Sell order 2 matched
	// 3. Buy order 3 matched
	if len(events) != 3 {
		t.Fatalf("Expected 3 matched events, got %d", len(events))
	}

	// Verify all events are "matched" type
	for i, event := range events {
		if event.EventType != "matched" {
			t.Errorf("Event %d: expected type 'matched', got '%s'", i, event.EventType)
		}
	}

	// Event 1: Sell order 1 matched with Buy order 3
	event1 := events[0]
	if event1.Order.ID != "1" {
		t.Errorf("Event 1: expected order ID '1', got '%s'", event1.Order.ID)
	}
	if event1.Order.Type != engine3.Side_sell {
		t.Errorf("Event 1: expected order type 'sell', got '%v'", event1.Order.Type)
	}
	if event1.MatchedWith == nil {
		t.Errorf("Event 1: MatchedWith should not be nil")
	} else if event1.MatchedWith.ID != "3" {
		t.Errorf("Event 1: expected MatchedWith ID '3' (buy order), got '%s'", event1.MatchedWith.ID)
	}
	if event1.TradeAmount != "0.02" {
		t.Errorf("Event 1: expected trade amount '0.02', got '%s'", event1.TradeAmount)
	}
	if event1.ExecutionPrice != "3" {
		t.Errorf("Event 1: expected execution price '3', got '%s'", event1.ExecutionPrice)
	}

	// Event 2: Sell order 2 matched with Buy order 3
	event2 := events[1]
	if event2.Order.ID != "2" {
		t.Errorf("Event 2: expected order ID '2', got '%s'", event2.Order.ID)
	}
	if event2.Order.Type != engine3.Side_sell {
		t.Errorf("Event 2: expected order type 'sell', got '%v'", event2.Order.Type)
	}
	if event2.MatchedWith == nil {
		t.Errorf("Event 2: MatchedWith should not be nil")
	} else if event2.MatchedWith.ID != "3" {
		t.Errorf("Event 2: expected MatchedWith ID '3' (buy order), got '%s'", event2.MatchedWith.ID)
	}
	if event2.TradeAmount != "0.03" {
		t.Errorf("Event 2: expected trade amount '0.03', got '%s'", event2.TradeAmount)
	}
	if event2.ExecutionPrice != "4" {
		t.Errorf("Event 2: expected execution price '4', got '%s'", event2.ExecutionPrice)
	}

	// Event 3: Buy order 3 matched (should have matched with sell orders)
	event3 := events[2]
	if event3.Order.ID != "3" {
		t.Errorf("Event 3: expected order ID '3', got '%s'", event3.Order.ID)
	}
	if event3.Order.Type != engine3.Side_buy {
		t.Errorf("Event 3: expected order type 'buy', got '%v'", event3.Order.Type)
	}

	// This is the key assertion: the buy order's MatchedWith should not be nil
	// and should contain information about one of the sell orders it matched with
	if event3.MatchedWith == nil {
		t.Errorf("Event 3: MatchedWith should not be nil for the buy order")
	}
}

// TestProcess_MatchedWithOrderID tests the same for limit orders
func TestProcess_MatchedWithOrderID(t *testing.T) {
	eng, collector := newTestEngine()
	defer collector.Close()

	pair := "BTC/USDT"
	ctx := context.Background()

	// Create a buy order in the book first
	buyOrder := &engine3.Order{
		ID:     "1",
		Type:   engine3.Side_buy,
		Amount: "0.05",
		Price:  "100",
		Pair:   pair,
	}

	_, err := eng.Process(ctx, buyOrder)
	if err != nil {
		t.Fatalf("Failed to process buy order: %v", err)
	}

	// Clear events from adding the order to the book
	collector.ClearEvents()

	// Now create a sell order that will match the buy order
	sellOrder := &engine3.Order{
		ID:     "2",
		Type:   engine3.Side_sell,
		Amount: "0.05",
		Price:  "100",
		Pair:   pair,
	}

	_, err = eng.Process(ctx, sellOrder)
	if err != nil {
		t.Fatalf("Failed to process sell order: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Get all broadcasted events
	events := collector.GetEvents()

	// We should have 2 matched events (buy and sell both fully filled)
	if len(events) != 2 {
		t.Fatalf("Expected 2 matched events, got %d", len(events))
	}

	// Event 1: Buy order 1 matched with Sell order 2
	event1 := events[0]
	if event1.Order.ID != "1" {
		t.Errorf("Event 1: expected order ID '1', got '%s'", event1.Order.ID)
	}
	if event1.MatchedWith == nil {
		t.Errorf("Event 1: MatchedWith should not be nil")
	} else if event1.MatchedWith.ID != "2" {
		t.Errorf("Event 1: expected MatchedWith ID '2', got '%s'", event1.MatchedWith.ID)
	}

	// Event 2: Sell order 2 matched with Buy order 1
	event2 := events[1]
	if event2.Order.ID != "2" {
		t.Errorf("Event 2: expected order ID '2', got '%s'", event2.Order.ID)
	}

	// This should not be nil
	if event2.MatchedWith == nil {
		t.Errorf("Event 2: MatchedWith should not be nil")
	}
}
