package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	engine2 "github.com/tcb-odds/matching-engine/internal/app/engine"
	"github.com/tcb-odds/matching-engine/internal/app/util"
	engine3 "github.com/tcb-odds/matching-engine/pkg/proto"
)

// Engine ...
type Engine struct {
	engine3.UnimplementedEngineServer
	book                map[string]*engine2.OrderBook
	subscriptionManager *SubscriptionManager
}

// NewEngine returns Engine object
func NewEngine(subscriptionManager *SubscriptionManager) *Engine {
	return &Engine{
		book:                map[string]*engine2.OrderBook{},
		subscriptionManager: subscriptionManager,
	}
}

// convertOrderToProto converts internal Order to proto Order
func convertOrderToProto(order *engine2.Order, pair string) *engine3.Order {
	return &engine3.Order{
		ID:           order.ID,
		Type:         engine3.Side(engine3.Side_value[order.Type.String()]),
		Amount:       order.Amount.String(),
		Price:        order.Price.String(),
		Pair:         pair,
		FilledAmount: order.FilledAmount.String(),
	}
}

// Process implements EngineServer interface
func (e *Engine) Process(ctx context.Context, req *engine3.Order) (*engine3.OutputOrders, error) {
	bigZero, _ := util.NewDecimalFromString("0.0")
	orderString := fmt.Sprintf("{\"id\":\"%s\", \"type\": \"%s\", \"amount\": \"%s\", \"price\": \"%s\" }", req.GetID(), req.GetType(), req.GetAmount(), req.GetPrice())

	var order engine2.Order
	// decode the message
	fmt.Println("Orderstring =: ", orderString)
	err := order.FromJSON([]byte(orderString))
	if err != nil {
		fmt.Println("JSON Parse Error =: ", err)
		return nil, err
	}

	if order.Amount.Cmp(bigZero) <= 0 {
		fmt.Println("Invalid order amount")
		return nil, errors.New("Order amount should be greater than zero")
	}

	if order.Price.Cmp(bigZero) <= 0 {
		fmt.Println("Invalid order price")
		return nil, errors.New("Order price should be greater than zero")
	}

	if req.GetPair() == "" {
		fmt.Println("Invalid pair")
		return nil, errors.New("Invalid pair")
	}

	var pairBook *engine2.OrderBook
	if val, ok := e.book[req.GetPair()]; ok {
		pairBook = val
	} else {
		pairBook = engine2.NewOrderBook()
		e.book[req.GetPair()] = pairBook
	}

	ordersProcessed, partialOrder := pairBook.Process(order)

	// Notify subscribers about matched orders
	incomingOrderID := req.GetID()
	var lastMatchedBookOrder *engine2.Order // Track last book order for partial fill event

	if len(ordersProcessed) > 0 {
		for i := 0; i < len(ordersProcessed); i++ {
			processedOrder := ordersProcessed[i]

			event := &engine3.OrderUpdateEvent{
				EventType:      "matched",
				Timestamp:      time.Now().Unix(),
				Pair:           req.GetPair(),
				Order:          convertOrderToProto(processedOrder, req.GetPair()),
				ExecutionPrice: processedOrder.Price.String(),
			}

			// Find the counterparty order
			// Orders are returned in pairs when both are fully filled: [bookOrder, incomingOrder]
			// or individually when only one is filled
			if processedOrder.ID == incomingOrderID {
				// This is the incoming order - find its counterparty (the previous book order)
				if i > 0 && ordersProcessed[i-1].ID != incomingOrderID {
					event.MatchedWith = convertOrderToProto(ordersProcessed[i-1], req.GetPair())
					event.TradeAmount = processedOrder.FilledAmount.String()
				}
			} else {
				// This is a book order - track it for potential partial fill event
				lastMatchedBookOrder = processedOrder

				// This is a book order - its counterparty is the incoming order
				if i+1 < len(ordersProcessed) && ordersProcessed[i+1].ID == incomingOrderID {
					event.MatchedWith = convertOrderToProto(ordersProcessed[i+1], req.GetPair())
					event.TradeAmount = processedOrder.FilledAmount.String()
				} else {
					// The incoming order is not in this batch (partial fill case)
					// Create a minimal order representation for the matched_with field
					event.MatchedWith = &engine3.Order{
						ID:   incomingOrderID,
						Type: engine3.Side(engine3.Side_value[order.Type.String()]),
					}
					event.TradeAmount = processedOrder.FilledAmount.String()
				}
			}

			e.subscriptionManager.Broadcast(event)
		}
	}

	// Notify about partially filled order
	if partialOrder != nil {
		event := &engine3.OrderUpdateEvent{
			EventType: "partially_filled",
			Timestamp: time.Now().Unix(),
			Pair:      req.GetPair(),
			Order:     convertOrderToProto(partialOrder, req.GetPair()),
		}

		// Set MatchedWith for partial fill event
		if partialOrder.ID == incomingOrderID {
			// Incoming order was partially filled - matched with last book order
			if lastMatchedBookOrder != nil {
				event.MatchedWith = convertOrderToProto(lastMatchedBookOrder, req.GetPair())
			}
		} else {
			// Book order was partially filled - matched with incoming order
			event.MatchedWith = &engine3.Order{
				ID:   incomingOrderID,
				Type: engine3.Side(engine3.Side_value[order.Type.String()]),
			}
		}
		event.TradeAmount = partialOrder.FilledAmount.String()

		e.subscriptionManager.Broadcast(event)
	}

	ordersProcessedString, err := json.Marshal(ordersProcessed)

	// if order.Type.String() == "sell" {
	fmt.Println("pair:", req.GetPair())
	fmt.Println(pairBook)
	// }

	if err != nil {
		fmt.Println("Marshal error", err)
		return nil, err
	}

	if partialOrder != nil {
		var partialOrderString []byte
		partialOrderString, err = json.Marshal(partialOrder)
		if err != nil {
			fmt.Println("partialOrderString Marshal error", err)
			return nil, err
		}
		return &engine3.OutputOrders{OrdersProcessed: string(ordersProcessedString), PartialOrder: string(partialOrderString)}, nil
	}
	return &engine3.OutputOrders{OrdersProcessed: string(ordersProcessedString), PartialOrder: "null"}, nil
}

// Cancel implements EngineServer interface
func (e *Engine) Cancel(ctx context.Context, req *engine3.Order) (*engine3.Order, error) {
	order := &engine2.Order{ID: req.GetID()}

	if order.ID == "" {
		fmt.Println("Invalid JSON")
		return nil, errors.New("Invalid JSON")
	}

	if req.GetPair() == "" {
		fmt.Println("Invalid pair")
		return nil, errors.New("Invalid pair")
	}

	var pairBook *engine2.OrderBook
	if val, ok := e.book[req.GetPair()]; ok {
		pairBook = val
	} else {
		pairBook = engine2.NewOrderBook()
		e.book[req.GetPair()] = pairBook
	}

	order = pairBook.CancelOrder(order.ID)

	fmt.Println("pair:", req.GetPair())
	fmt.Println(pairBook)

	if order == nil {
		return nil, errors.New("NoOrderPresent")
	}

	// Notify subscribers about cancelled order
	event := &engine3.OrderUpdateEvent{
		EventType: "cancelled",
		Timestamp: time.Now().Unix(),
		Pair:      req.GetPair(),
		Order:     convertOrderToProto(order, req.GetPair()),
	}
	e.subscriptionManager.Broadcast(event)

	orderEngine := &engine3.Order{}

	orderEngine.ID = order.ID
	orderEngine.Amount = order.Amount.String()
	orderEngine.Price = order.Price.String()
	orderEngine.Type = engine3.Side(engine3.Side_value[order.Type.String()])

	return orderEngine, nil
}

// ProcessMarket implements EngineServer interface
func (e *Engine) ProcessMarket(ctx context.Context, req *engine3.Order) (*engine3.OutputOrders, error) {
	bigZero, _ := util.NewDecimalFromString("0.0")
	orderString := fmt.Sprintf("{\"id\":\"%s\", \"type\": \"%s\", \"amount\": \"%s\", \"price\": \"%s\" }", req.GetID(), req.GetType(), req.GetAmount(), req.GetPrice())

	var order engine2.Order
	// decode the message
	// fmt.Println("Orderstring =: ", orderString)
	err := order.FromJSON([]byte(orderString))
	if err != nil {
		fmt.Println("JSON Parse Error =: ", err)
		return nil, err
	}

	if order.Amount.Cmp(bigZero) == 0 {
		fmt.Println("Invalid JSON")
		return nil, errors.New("Invalid JSON")
	}

	if req.GetPair() == "" {
		fmt.Println("Invalid pair")
		return nil, errors.New("Invalid pair")
	}

	var pairBook *engine2.OrderBook
	if val, ok := e.book[req.GetPair()]; ok {
		pairBook = val
	} else {
		pairBook = engine2.NewOrderBook()
		e.book[req.GetPair()] = pairBook
	}

	ordersProcessed, partialOrder := pairBook.ProcessMarket(order)

	// Notify subscribers about matched orders (market orders)
	incomingOrderID := req.GetID()
	var lastMatchedBookOrder *engine2.Order // Track last book order for partial fill event

	if len(ordersProcessed) > 0 {
		for i := 0; i < len(ordersProcessed); i++ {
			processedOrder := ordersProcessed[i]

			event := &engine3.OrderUpdateEvent{
				EventType:      "matched",
				Timestamp:      time.Now().Unix(),
				Pair:           req.GetPair(),
				Order:          convertOrderToProto(processedOrder, req.GetPair()),
				ExecutionPrice: processedOrder.Price.String(),
			}

			// Find the counterparty order
			// Orders are returned in pairs when both are fully filled: [bookOrder, incomingOrder]
			// or individually when only one is filled
			if processedOrder.ID == incomingOrderID {
				// This is the incoming order - find its counterparty (the previous book order)
				if i > 0 && ordersProcessed[i-1].ID != incomingOrderID {
					event.MatchedWith = convertOrderToProto(ordersProcessed[i-1], req.GetPair())
					event.TradeAmount = processedOrder.FilledAmount.String()
				}
			} else {
				// This is a book order - track it for potential partial fill event
				lastMatchedBookOrder = processedOrder

				// This is a book order - its counterparty is the incoming order
				if i+1 < len(ordersProcessed) && ordersProcessed[i+1].ID == incomingOrderID {
					event.MatchedWith = convertOrderToProto(ordersProcessed[i+1], req.GetPair())
					event.TradeAmount = processedOrder.FilledAmount.String()
				} else {
					// The incoming order is not in this batch (partial fill case)
					// Create a minimal order representation for the matched_with field
					event.MatchedWith = &engine3.Order{
						ID:   incomingOrderID,
						Type: engine3.Side(engine3.Side_value[order.Type.String()]),
					}
					event.TradeAmount = processedOrder.FilledAmount.String()
				}
			}

			e.subscriptionManager.Broadcast(event)
		}
	}

	// Notify about partially filled order
	if partialOrder != nil {
		event := &engine3.OrderUpdateEvent{
			EventType: "partially_filled",
			Timestamp: time.Now().Unix(),
			Pair:      req.GetPair(),
			Order:     convertOrderToProto(partialOrder, req.GetPair()),
		}

		// Set MatchedWith for partial fill event
		if partialOrder.ID == incomingOrderID {
			// Incoming order was partially filled - matched with last book order
			if lastMatchedBookOrder != nil {
				event.MatchedWith = convertOrderToProto(lastMatchedBookOrder, req.GetPair())
			}
		} else {
			// Book order was partially filled - matched with incoming order
			event.MatchedWith = &engine3.Order{
				ID:   incomingOrderID,
				Type: engine3.Side(engine3.Side_value[order.Type.String()]),
			}
		}
		event.TradeAmount = partialOrder.FilledAmount.String()

		e.subscriptionManager.Broadcast(event)
	}

	ordersProcessedString, err := json.Marshal(ordersProcessed)

	// if order.Type.String() == "sell" {
	fmt.Println("pair:", req.GetPair())
	fmt.Println(pairBook)
	// }

	if err != nil {
		return nil, err
	}

	if partialOrder != nil {
		var partialOrderString []byte
		partialOrderString, err = json.Marshal(partialOrder)
		return &engine3.OutputOrders{OrdersProcessed: string(ordersProcessedString), PartialOrder: string(partialOrderString)}, nil
	}
	return &engine3.OutputOrders{OrdersProcessed: string(ordersProcessedString), PartialOrder: "null"}, nil
}

// FetchBook implements EngineServer interface
func (e *Engine) FetchBook(ctx context.Context, req *engine3.BookInput) (*engine3.BookOutput, error) {
	if req.GetPair() == "" {
		fmt.Println("Invalid pair")
		return nil, errors.New("Invalid pair")
	}

	var pairBook *engine2.OrderBook
	if val, ok := e.book[req.GetPair()]; ok {
		pairBook = val
	} else {
		pairBook = engine2.NewOrderBook()
		e.book[req.GetPair()] = pairBook
	}

	fmt.Println(pairBook)
	book := pairBook.GetOrders(req.GetLimit())

	result := &engine3.BookOutput{Buys: []*engine3.BookArray{}, Sells: []*engine3.BookArray{}}

	for _, buy := range book.Buys {
		arr := &engine3.BookArray{PriceAmount: []string{}}

		bodyBytes, err := json.Marshal(buy)
		if err != nil {
			fmt.Println("1", err)
			return &engine3.BookOutput{Buys: []*engine3.BookArray{}, Sells: []*engine3.BookArray{}}, nil
		}

		err = json.Unmarshal(bodyBytes, &arr.PriceAmount)
		if err != nil {
			fmt.Println("2", err)
			return &engine3.BookOutput{Buys: []*engine3.BookArray{}, Sells: []*engine3.BookArray{}}, nil
		}

		result.Buys = append(result.Buys, arr)
	}

	for _, sell := range book.Sells {
		arr := &engine3.BookArray{PriceAmount: []string{}}

		bodyBytes, err := json.Marshal(sell)
		if err != nil {
			fmt.Println("json.Marshal Error", err)
			return &engine3.BookOutput{Buys: []*engine3.BookArray{}, Sells: []*engine3.BookArray{}}, nil
		}

		err = json.Unmarshal(bodyBytes, &arr.PriceAmount)
		if err != nil {
			fmt.Println("json.Unmarshal Error", err)
			return &engine3.BookOutput{Buys: []*engine3.BookArray{}, Sells: []*engine3.BookArray{}}, nil
		}

		result.Sells = append(result.Sells, arr)
	}
	return result, nil
}

// SubscribeToOrderUpdates implements EngineServer interface for streaming order updates
func (e *Engine) SubscribeToOrderUpdates(req *engine3.SubscribeRequest, stream engine3.Engine_SubscribeToOrderUpdatesServer) error {
	// Create subscriber
	subscriber := e.subscriptionManager.Subscribe(stream.Context())
	defer e.subscriptionManager.Unsubscribe(subscriber.ID)

	fmt.Printf("New subscriber connected: %s (total: %d)\n", subscriber.ID, e.subscriptionManager.Count())

	// Listen to channel and send updates
	for {
		select {
		case event, ok := <-subscriber.Channel:
			if !ok {
				fmt.Printf("Subscriber channel closed: %s\n", subscriber.ID)
				return nil
			}
			if err := stream.Send(event); err != nil {
				fmt.Printf("Error sending to subscriber %s: %v\n", subscriber.ID, err)
				return err
			}
		case <-stream.Context().Done():
			fmt.Printf("Subscriber disconnected: %s (total: %d)\n", subscriber.ID, e.subscriptionManager.Count()-1)
			return nil
		}
	}
}
