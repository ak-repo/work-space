package stream

import "sync"

type Broker struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func New() *Broker { return &Broker{clients: map[chan []byte]struct{}{}} }
func (b *Broker) Subscribe() chan []byte {
	ch := make(chan []byte, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}
func (b *Broker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}
func (b *Broker) Publish(v []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- v:
		default:
		}
	}
}
