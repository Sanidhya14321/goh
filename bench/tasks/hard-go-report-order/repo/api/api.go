package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"example.com/invsvc/model"
	"example.com/invsvc/store"
)

// Page is one page of items. Next is the SKU to continue after ("" at end).
type Page struct {
	Items []model.Item `json:"items"`
	Next  string       `json:"next"`
}

// ListItems pages through items in SKU order: ?after=SKU&limit=N. The cursor
// is the last SKU returned, so deleting items between requests never skips
// or repeats an item.
func ListItems(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}
		after := r.URL.Query().Get("after")
		var page Page
		for _, it := range s.List() {
			if after != "" && it.SKU <= after {
				continue
			}
			if len(page.Items) == limit {
				page.Next = page.Items[len(page.Items)-1].SKU
				break
			}
			page.Items = append(page.Items, it)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	}
}
