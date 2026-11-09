package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchSearXNG(t *testing.T) {
	var gotQuery, gotFormat string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery, gotFormat = r.URL.Query().Get("q"), r.URL.Query().Get("format")
		if r.URL.Path != "/search" {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(`{"results":[{"title":"Go docs","url":"https://go.dev/doc","content":"Documentation   for Go"},{"title":"bad","url":"javascript:alert(1)"},{"title":"Two","url":"https://example.com/2"}]}`))
	}))
	defer s.Close()
	r := Search(context.Background(), s.URL, json.RawMessage(`{"query":"go generics","limit":5}`), s.Client())
	if r.Error || gotQuery != "go generics" || gotFormat != "json" {
		t.Fatal(r.Content, gotQuery, gotFormat)
	}
	if !strings.Contains(r.Content, `"Documentation for Go"`) || strings.Contains(r.Content, "javascript:") || !strings.Contains(r.Content, "example.com/2") {
		t.Fatal(r.Content)
	}
	one := Search(context.Background(), s.URL, json.RawMessage(`{"query":"go","limit":1}`), s.Client())
	if strings.Contains(one.Content, "example.com/2") {
		t.Fatal("limit ignored")
	}
	redirect := httptest.NewServer(http.RedirectHandler("https://elsewhere.example/search", http.StatusFound))
	defer redirect.Close()
	if r := Search(context.Background(), redirect.URL, json.RawMessage(`{"query":"x"}`), redirect.Client()); !r.Error {
		t.Fatal("redirect followed")
	}
	for _, bad := range []string{`{"query":""}`, `{"query":"x","limit":11}`, `{"query":"x","extra":1}`} {
		if _, err := Prepare(s.URL, json.RawMessage(bad)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
	if preview, _ := Prepare(s.URL, json.RawMessage(`{"query":"go"}`)); !strings.Contains(preview, "for: go") {
		t.Fatal(preview)
	}
	for _, bad := range []string{"http://search.example.com", "https://u:p@x.example", "https://x.example?q=1", "ftp://x"} {
		if CheckURL(bad) == nil {
			t.Fatal("accepted", bad)
		}
	}
}
