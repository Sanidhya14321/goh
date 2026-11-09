package store

import (
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
)

func TestPutGet(t *testing.T) {
	s := New(&events.Bus{})
	s.Put(model.Item{SKU: "A1", Name: "Bolt", Price: 125})
	if it, err := s.Get("A1"); err != nil || it.Price != 125 {
		t.Fatal(it, err)
	}
}
