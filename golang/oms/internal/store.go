package internal

import (
	"fmt"
	"sync"
)

// Store is a thread-safe in-memory order store
type Store struct {
	mu     sync.RWMutex
	orders map[string]*Order
	seq    int
}

// New creates a new empty Store
func NewStore() *Store {
	return &Store{
		orders: make(map[string]*Order),
	}
}

// Save inserts a new order and returns its assigned ID
func (s *Store) Save(order *Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	order.ID = fmt.Sprintf("ORD-%04d", s.seq)
	s.orders[order.ID] = order
}

// Get retrieves an order by ID; returns nil if not found
func (s *Store) Get(id string) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

// UpdateStatus sets a new status on an existing order
func (s *Store) UpdateStatus(id, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return false
	}
	o.Status = status
	return true
}
