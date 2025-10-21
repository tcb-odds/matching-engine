package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	engine2 "github.com/Pantelwar/matching-engine/internal/app/engine"
	engine3 "github.com/Pantelwar/matching-engine/internal/app/engineGrpc"
	"github.com/Pantelwar/matching-engine/internal/app/util"
)

// Engine ...
type Engine struct {
	book map[string]*engine2.OrderBook
}

// NewEngine returns Engine object
func NewEngine() *Engine {
	return &Engine{book: map[string]*engine2.OrderBook{}}
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

	if order.Amount.Cmp(bigZero) == 0 || order.Price.Cmp(bigZero) == 0 {
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

	ordersProcessed, partialOrder := pairBook.Process(order)

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
		return nil, errors.New("Invalid pair")
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
