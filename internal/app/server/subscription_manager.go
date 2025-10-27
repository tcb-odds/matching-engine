package server

import (
	"context"
	"sync"

	"github.com/google/uuid"
	proto "github.com/tcb-odds/matching-engine/pkg/proto"
)

// Subscriber represents a single subscriber to order updates
type Subscriber struct {
	ID         string
	Channel    chan *proto.OrderUpdateEvent
	Context    context.Context
	CancelFunc context.CancelFunc
}

// SubscriptionManager manages all active subscriptions
type SubscriptionManager struct {
	subscribers map[string]*Subscriber // key: subscriberID
	mu          sync.RWMutex
}

// NewSubscriptionManager creates a new SubscriptionManager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[string]*Subscriber),
	}
}

// Subscribe creates a new subscriber and returns it
func (sm *SubscriptionManager) Subscribe(ctx context.Context) *Subscriber {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := uuid.New().String()
	ctx, cancel := context.WithCancel(ctx)

	sub := &Subscriber{
		ID:         id,
		Channel:    make(chan *proto.OrderUpdateEvent, 100),
		Context:    ctx,
		CancelFunc: cancel,
	}

	sm.subscribers[id] = sub
	return sub
}

// Unsubscribe removes a subscriber and cleans up resources
func (sm *SubscriptionManager) Unsubscribe(subscriberID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sub, ok := sm.subscribers[subscriberID]; ok {
		sub.CancelFunc()
		close(sub.Channel)
		delete(sm.subscribers, subscriberID)
	}
}

// Broadcast sends an event to all subscribers
func (sm *SubscriptionManager) Broadcast(event *proto.OrderUpdateEvent) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, sub := range sm.subscribers {
		select {
		case sub.Channel <- event:
			// Event sent successfully
		case <-sub.Context.Done():
			// Subscriber disconnected, skip
		default:
			// Channel is full, skip this event for this subscriber
			// Could log this if needed
		}
	}
}

// Count returns the number of active subscribers
func (sm *SubscriptionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.subscribers)
}

// Shutdown gracefully closes all subscriptions
func (sm *SubscriptionManager) Shutdown() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, sub := range sm.subscribers {
		sub.CancelFunc()
		close(sub.Channel)
		delete(sm.subscribers, id)
	}
}
