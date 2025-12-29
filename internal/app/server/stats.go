package server

import engine2 "github.com/tcb-odds/matching-engine/internal/app/engine"

// PairStats represents statistics for a single trading pair
type PairStats struct {
	BuyOrders   int `json:"buyOrders"`
	SellOrders  int `json:"sellOrders"`
	TotalOrders int `json:"totalOrders"`
}

// Stats represents overall matching engine statistics
type Stats struct {
	TotalPairs      int                  `json:"totalPairs"`
	TotalOrders     int                  `json:"totalOrders"`
	TotalBuyOrders  int                  `json:"totalBuyOrders"`
	TotalSellOrders int                  `json:"totalSellOrders"`
	Subscribers     int                  `json:"subscribers"`
	PairStats       map[string]PairStats `json:"pairStats"`
}

// GetStats returns statistics about the matching engine
func (e *Engine) GetStats() interface{} {
	stats := &Stats{
		TotalPairs:      len(e.book),
		TotalOrders:     0,
		TotalBuyOrders:  0,
		TotalSellOrders: 0,
		Subscribers:     e.subscriptionManager.Count(),
		PairStats:       make(map[string]PairStats),
	}

	for pair, orderBook := range e.book {
		buyCount := 0
		sellCount := 0

		if orderBook.BuyTree.Root != nil {
			orderBook.BuyTree.Root.InOrderTraverse(func(i float64) {
				node := orderBook.BuyTree.Root.SearchSubTree(i)
				node.Data.(*engine2.OrderType).Tree.Root.InOrderTraverse(func(j float64) {
					subNode := node.Data.(*engine2.OrderType).Tree.Root.SearchSubTree(j)
					buyCount += len(subNode.Data.(*engine2.OrderNode).Orders)
				})
			})
		}

		if orderBook.SellTree.Root != nil {
			orderBook.SellTree.Root.InOrderTraverse(func(i float64) {
				node := orderBook.SellTree.Root.SearchSubTree(i)
				node.Data.(*engine2.OrderType).Tree.Root.InOrderTraverse(func(j float64) {
					subNode := node.Data.(*engine2.OrderType).Tree.Root.SearchSubTree(j)
					sellCount += len(subNode.Data.(*engine2.OrderNode).Orders)
				})
			})
		}

		pairStat := PairStats{
			BuyOrders:   buyCount,
			SellOrders:  sellCount,
			TotalOrders: buyCount + sellCount,
		}

		stats.PairStats[pair] = pairStat
		stats.TotalBuyOrders += buyCount
		stats.TotalSellOrders += sellCount
		stats.TotalOrders += buyCount + sellCount
	}

	return stats
}

// EraseOrders clears all orders from a specific pair or all pairs
func (e *Engine) EraseOrders(pair string) error {
	if pair != "" {
		if orderBook, ok := e.book[pair]; ok {
			orderBook.Clear()
			delete(e.book, pair)
			return nil
		}
		return nil
	}

	for pairName := range e.book {
		delete(e.book, pairName)
	}
	return nil
}
