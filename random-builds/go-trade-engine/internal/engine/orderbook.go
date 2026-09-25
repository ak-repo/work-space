package engine

import (
	"time"

	"github.com/ak-repo/go-trade-engine/internal/models"
	"github.com/google/btree"
)

type OrderItem struct {
	*models.Order
}

func (a OrderItem) Less(b btree.Item) bool {
	other := b.(OrderItem)
	if a.Price == other.Price {
		return a.Timestamp < other.Timestamp
	}

	return a.Price < other.Price
}

type OrderBook struct {
	Pair string
	Bids *btree.BTree
	Asks *btree.BTree

	Orders map[string]*models.Order
}

func NewOrderBook(pair string) *OrderBook {
	return &OrderBook{
		Pair:   pair,
		Bids:   btree.New(2),
		Asks:   btree.New(2),
		Orders: make(map[string]*models.Order),
	}
}

func (ob *OrderBook) ProcessLimitOrder(order *models.Order) ([]models.Trade, error) {
	trades := []models.Trade{}

	if order.Side == models.Buy {
		trades = ob.matchBuy(order)
	} else {
		trades = ob.matchSell(order)
	}

	return trades, nil
}

func (ob *OrderBook) matchBuy(order *models.Order) []models.Trade {
	trades := []models.Trade{}


	ob.Asks.Ascend(func(item btree.Item) bool {
		ask := item.(OrderItem).Order

		if ask.Price > order.Price {
			return false // No match possible, lowest ask is too expensive
		}

		// Determine trade amount
		matchAmount := min(order.Amount, ask.Amount)

		trades = append(trades, models.Trade{
			MakerOrderID: ask.ID,
			TakerOrderID: order.ID,
			Price:        ask.Price,
			Amount:       matchAmount,
			Pair:         ob.Pair,
			Timestamp:    time.Now().UnixNano(),
		})

		order.Amount -= matchAmount
		ask.Amount -= matchAmount

		// If ask is filled, remove it
		if ask.Amount == 0 {
			ob.Asks.Delete(item)
			delete(ob.Orders, ask.ID)
		}

		return order.Amount > 0 // Continue if buy order not filled
	})

	// If buy order still has remaining amount, add to Bids
	if order.Amount > 0 {
		ob.Bids.ReplaceOrInsert(OrderItem{order})
		ob.Orders[order.ID] = order
	}

	return trades
}

func (ob *OrderBook) matchSell(order *models.Order) []models.Trade {
	trades := []models.Trade{}

	// Iterate through Bids (Highest price first - Descend)
	ob.Bids.Descend(func(item btree.Item) bool {
		bid := item.(OrderItem).Order

		if bid.Price < order.Price {
			return false // Highest bid is too low
		}

		matchAmount := min(order.Amount, bid.Amount)

		trades = append(trades, models.Trade{
			MakerOrderID: bid.ID,
			TakerOrderID: order.ID,
			Price:        bid.Price,
			Amount:       matchAmount,
			Pair:         ob.Pair,
			Timestamp:    time.Now().UnixNano(),
		})

		order.Amount -= matchAmount
		bid.Amount -= matchAmount

		if bid.Amount == 0 {
			ob.Bids.Delete(item)
			delete(ob.Orders, bid.ID)
		}

		return order.Amount > 0
	})

	if order.Amount > 0 {
		ob.Asks.ReplaceOrInsert(OrderItem{order})
		ob.Orders[order.ID] = order
	}

	return trades
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
