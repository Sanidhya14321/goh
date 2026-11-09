package store

import (
	"errors"
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestApplyIsAtomic(t *testing.T) {
	bus := &events.Bus{}
	var seen []string
	bus.Subscribe(func(e events.Event) { seen = append(seen, e.SKU) })
	s := New(bus)
	s.Put(model.Item{SKU: "A", Quantity: 1})
	seen = nil
	err := s.Apply([]func(map[string]model.Item) error{
		func(m map[string]model.Item) error { it := m["A"]; it.Quantity = 99; m["A"] = it; return nil },
		func(m map[string]model.Item) error { m["B"] = model.Item{SKU: "B"}; return nil },
		func(map[string]model.Item) error { return errors.New("bad row") },
	})
	if err == nil {
		t.Fatal("error lost")
	}
	if it, _ := s.Get("A"); it.Quantity != 1 {
		t.Fatalf("partial update applied: %+v", it)
	}
	if _, err := s.Get("B"); err == nil {
		t.Fatal("partial insert applied")
	}
	if len(seen) != 0 {
		t.Fatalf("events published for a failed import: %v", seen)
	}
	if err := s.Apply([]func(map[string]model.Item) error{func(m map[string]model.Item) error { m["C"] = model.Item{SKU: "C"}; return nil }}); err != nil || len(seen) != 1 || seen[0] != "C" {
		t.Fatalf("successful apply: %v %v", err, seen)
	}
}
