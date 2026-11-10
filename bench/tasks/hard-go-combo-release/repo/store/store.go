package store

import (
	"errors"
	"sort"
	"sync"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

var ErrNotFound = errors.New("item not found")

// Store is an in-memory item store that publishes change events.
type Store struct {
	mu    sync.RWMutex
	items map[string]model.Item
	bus   *events.Bus
}

func New(bus *events.Bus) *Store { return &Store{items: map[string]model.Item{}, bus: bus} }

func (s *Store) Get(sku string) (model.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	it, ok := s.items[sku]
	if !ok {
		return model.Item{}, ErrNotFound
	}
	return it, nil
}

func (s *Store) Put(it model.Item) {
	s.mu.Lock()
	s.items[it.SKU] = it
	s.mu.Unlock()
	s.bus.Publish(events.Event{Kind: "updated", SKU: it.SKU})
}

func (s *Store) Delete(sku string) {
	s.mu.Lock()
	delete(s.items, sku)
	s.mu.Unlock()
	s.bus.Publish(events.Event{Kind: "deleted", SKU: sku})
}

// List returns items sorted by SKU.
func (s *Store) List() []model.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Item, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

// Apply runs all updates atomically: if any update fails, none is applied
// and no events are published.
func (s *Store) Apply(updates []func(map[string]model.Item) error) error {
	s.mu.Lock()
	draft := make(map[string]model.Item, len(s.items))
	for k, v := range s.items {
		draft[k] = v
	}
	for _, u := range updates {
		if err := u(draft); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	var changed []string
	for k, v := range draft {
		if old, ok := s.items[k]; !ok || old != v {
			changed = append(changed, k)
		}
	}
	s.items = draft
	s.mu.Unlock()
	sort.Strings(changed)
	for _, k := range changed {
		s.bus.Publish(events.Event{Kind: "updated", SKU: k})
	}
	return nil
}
