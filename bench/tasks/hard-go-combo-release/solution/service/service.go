package service

import (
	"example.com/invsvc/cache"
	"example.com/invsvc/events"
	"example.com/invsvc/store"
)

// Service wires the store, price cache and events together.
type Service struct {
	Store *store.Store
	Cache *cache.TTL
	Bus   *events.Bus
}

// New builds a service. Cached price strings are invalidated whenever an
// item is updated or deleted, so callers never see a stale price.
func New(bus *events.Bus, c *cache.TTL) *Service {
	s := &Service{Store: store.New(bus), Cache: c, Bus: bus}
	bus.Subscribe(func(e events.Event) { c.Invalidate(e.SKU) })
	return s
}

// PriceText returns the item's price as text, using the cache.
func (s *Service) PriceText(sku string) (string, error) {
	if v, ok := s.Cache.Get(sku); ok {
		return v, nil
	}
	it, err := s.Store.Get(sku)
	if err != nil {
		return "", err
	}
	v := it.Price.String()
	s.Cache.Set(sku, v)
	return v, nil
}
