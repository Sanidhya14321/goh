package service

import (
	"testing"
	"time"

	"example.com/invsvc/cache"
	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestPriceUpdatesImmediately(t *testing.T) {
	now := time.Unix(0, 0)
	s := New(&events.Bus{}, cache.NewTTL(time.Minute, func() time.Time { return now }))
	s.Store.Put(model.Item{SKU: "X", Price: 100})
	if v, _ := s.PriceText("X"); v != "1.00" {
		t.Fatal(v)
	}
	s.Store.Put(model.Item{SKU: "X", Price: 250})
	if v, _ := s.PriceText("X"); v != "2.50" {
		t.Fatalf("stale price after update: %s", v)
	}
	s.Store.Delete("X")
	if _, err := s.PriceText("X"); err == nil {
		t.Fatal("deleted item still priced")
	}
}
