package events

import "sync"

// Event is a change notification.
type Event struct {
	Kind string // "updated", "deleted"
	SKU  string
}

// Bus delivers events to subscribers in subscription order.
type Bus struct {
	mu   sync.Mutex
	subs []func(Event)
}

// Subscribe registers fn for all later events.
func (b *Bus) Subscribe(fn func(Event)) {
	b.mu.Lock()
	b.subs = append(b.subs, fn)
	b.mu.Unlock()
}

// Publish delivers e to every subscriber. Subscribers may publish again.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	subs := append([]func(Event){}, b.subs...)
	b.mu.Unlock()
	for _, fn := range subs {
		fn(e)
	}
}
