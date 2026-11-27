package webfetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const shop = `<!doctype html><html><head><title>Shop</title><script>var x="<a href=/js>"</script></head><body>
<nav><a href="/home">Home</a></nav>
<main><div class="product card" id="p1" data-sku="A1"><h2>Lamp</h2><span class="price">$20</span><a href="/p/1">More</a></div>
<div class="product card" id="p2" data-sku="B2"><h2>Desk</h2><span class="price">$90</span><a href="/p/2">More</a>
<div class="product"><span class="price">$5</span></div></div>
<ul><li><a href="https://other.example/x">Elsewhere</a></li><li><a href="javascript:alert(1)">Bad</a></li><li><a href="/p/1">Dup</a></li></ul></main>
</body></html>`

func fetchJSON(t *testing.T, server *httptest.Server, args string) map[string]any {
	t.Helper()
	res := NewAllowingPrivate().Fetch(context.Background(), json.RawMessage(fmt.Sprintf(args, server.URL)))
	if res.Error {
		t.Fatalf("%s: %s", args, res.Content)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(res.Content), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWebfetchSelectorAndLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data.json" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"a":1}`)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, shop)
	}))
	defer server.Close()

	out := fetchJSON(t, server, `{"url":"%s/","selector":"main > .product.card"}`)
	content := out["content"].(string)
	if out["matches"].(float64) != 2 || !strings.Contains(content, "Lamp") || !strings.Contains(content, "Desk") || strings.Contains(content, "Home") {
		t.Fatalf("selector: %v", out)
	}
	out = fetchJSON(t, server, `{"url":"%s/","selector":"div[data-sku=B2] .price","format":"text"}`)
	if out["matches"].(float64) != 2 || !strings.Contains(out["content"].(string), "$90") || strings.Contains(out["content"].(string), "$20") {
		t.Fatalf("attribute and descendant: %v", out)
	}
	out = fetchJSON(t, server, `{"url":"%s/","selector":"#p1 h2, #p2 > h2","format":"html"}`)
	if c := out["content"].(string); c != "<h2>Lamp</h2>\n\n---\n\n<h2>Desk</h2>" {
		t.Fatalf("html of matches: %q", c)
	}
	out = fetchJSON(t, server, `{"url":"%s/","selector":"table"}`)
	if out["matches"].(float64) != 0 || out["content"] != "" {
		t.Fatalf("no match: %v", out)
	}

	out = fetchJSON(t, server, `{"url":"%s/","links":true}`)
	links := out["links"].([]any)
	var urls []string
	for _, l := range links {
		urls = append(urls, l.(map[string]any)["url"].(string))
	}
	want := []string{server.URL + "/home", server.URL + "/p/1", server.URL + "/p/2", "https://other.example/x"}
	if strings.Join(urls, " ") != strings.Join(want, " ") || !strings.Contains(out["content"].(string), "Lamp") {
		t.Fatalf("links %v", urls)
	}
	out = fetchJSON(t, server, `{"url":"%s/","selector":"#p2","links":true}`)
	if l := out["links"].([]any); len(l) != 1 || l[0].(map[string]any)["text"] != "More" {
		t.Fatalf("links within selection: %v", l)
	}

	res := NewAllowingPrivate().Fetch(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/data.json","selector":"div"}`))
	if !res.Error || !strings.Contains(res.Content, "HTML pages only") {
		t.Fatal(res.Content)
	}
}

func TestSelectorParsing(t *testing.T) {
	for _, ok := range []string{"div", "*", "a[href]", "a[href='/x']", `input[type="text"]`, "ul > li a", "h1, h2", ".a.b#c", "main>.x"} {
		if _, err := parseSelector(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "> a", "a >", "a > > b", "a:hover", "a::before", "a + b", "a ~ b", "[x", "#", ".", "a#b#c", strings.Repeat("a ", 300)} {
		if _, err := parseSelector(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := Prepare(json.RawMessage(`{"url":"https://example.com","selector":"a:hover"}`)); err == nil {
		t.Fatal("bad selector passed approval preparation")
	}
	preview, err := Prepare(json.RawMessage(`{"url":"https://example.com","selector":".price","links":true}`))
	if err != nil || !strings.Contains(preview, "Selector: .price") || !strings.Contains(preview, "Links: listed") {
		t.Fatal(preview, err)
	}
}

func FuzzSelector(f *testing.F) {
	for _, s := range []string{"main > .product.card", "a[href='x']", "h1, h2 > b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		sel, err := parseSelector(s)
		if err != nil {
			return
		}
		doc, _ := parseDocument([]byte(shop))
		if nodes, err := selectNodes(context.Background(), doc, sel); err == nil {
			base, _ := url.Parse("https://example.com/")
			renderSelected(context.Background(), nodes, "markdown", base)
			collectLinks(nodes, base)
		}
	})
}
