package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"example.com/invsvc/events"
	"example.com/invsvc/model"
	"example.com/invsvc/store"
)

func get(t *testing.T, s *store.Store, after string, limit int) Page {
	rec := httptest.NewRecorder()
	ListItems(s)(rec, httptest.NewRequest("GET", fmt.Sprintf("/items?after=%s&limit=%d", after, limit), nil))
	var p Page
	json.Unmarshal(rec.Body.Bytes(), &p)
	return p
}

func TestPagingVisitsEveryItemOnce(t *testing.T) {
	s := store.New(&events.Bus{})
	for i := 0; i < 7; i++ {
		s.Put(model.Item{SKU: fmt.Sprintf("S%02d", i)})
	}
	seen := map[string]int{}
	after := ""
	for pages := 0; pages < 10; pages++ {
		p := get(t, s, after, 3)
		for _, it := range p.Items {
			seen[it.SKU]++
		}
		if pages == 0 {
			s.Delete("S01") // already returned
			s.Delete("S04") // not yet returned
		}
		if p.Next == "" {
			break
		}
		after = p.Next
	}
	for i := 0; i < 7; i++ {
		sku := fmt.Sprintf("S%02d", i)
		want := 1
		if sku == "S04" {
			want = 0
		}
		if seen[sku] != want {
			t.Fatalf("%s seen %d times: %v", sku, seen[sku], seen)
		}
	}
}

func TestLastPageHasNoNext(t *testing.T) {
	s := store.New(&events.Bus{})
	for i := 0; i < 3; i++ {
		s.Put(model.Item{SKU: fmt.Sprintf("S%d", i)})
	}
	if p := get(t, s, "", 3); len(p.Items) != 3 || p.Next != "" {
		t.Fatalf("%+v", p)
	}
}
