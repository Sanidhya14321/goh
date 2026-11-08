// Package websearch queries a user-configured SearXNG-compatible search
// endpoint (GET /search?q=…&format=json). Nothing is searched unless the
// user configures websearch_url, and each query needs approval.
package websearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/tools"
)

// Definition is the model-facing tool.
func Definition() core.Tool {
	return core.NewTool("websearch", "Search the web through the user's configured search service. Returns titles, URLs and snippets; use webfetch to read a page. Each search needs the user's approval.",
		map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 10}}, "required": []string{"query"}, "additionalProperties": false})
}

// CheckURL accepts https, or http for localhost, without credentials.
func CheckURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("websearch_url must be an http(s) base URL without credentials, query or fragment")
	}
	if u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); u.Scheme == "http" && (host == "localhost" || ip != nil && ip.IsLoopback()) {
		return nil
	}
	return errors.New("websearch_url must use https (http only for localhost)")
}

type args struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

func parse(raw json.RawMessage) (args, error) {
	var a args
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil || strings.TrimSpace(a.Query) == "" || len(a.Query) > 500 || a.Limit < 0 || a.Limit > 10 {
		return a, errors.New("websearch takes a query (up to 500 characters) and an optional limit of 1-10")
	}
	if a.Limit == 0 {
		a.Limit = 5
	}
	return a, nil
}

// Prepare returns the approval preview.
func Prepare(base string, raw json.RawMessage) (string, error) {
	a, err := parse(raw)
	if err != nil {
		return "", err
	}
	u, _ := url.Parse(base)
	return fmt.Sprintf("Search %s for: %s", u.Host, a.Query), nil
}

// Search runs the query and returns bounded results.
func Search(ctx context.Context, base string, raw json.RawMessage, client *http.Client) tools.Result {
	fail := func(err error) tools.Result { return tools.Result{Error: true, Content: err.Error()} }
	a, err := parse(raw)
	if err != nil {
		return fail(err)
	}
	if err := CheckURL(base); err != nil {
		return fail(err)
	}
	target, _ := url.Parse(strings.TrimRight(base, "/") + "/search")
	target.RawQuery = url.Values{"q": {a.Query}, "format": {"json"}}.Encode()
	if client == nil {
		client = &http.Client{}
	}
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fail(fmt.Errorf("search failed: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("search service returned HTTP %d", resp.StatusCode))
	}
	var body struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&body); err != nil {
		return fail(errors.New("search service did not return JSON results"))
	}
	type result struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet,omitempty"`
	}
	var out []result
	for _, r := range body.Results {
		u, err := url.Parse(r.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		out = append(out, result{Title: clip(r.Title, 200), URL: clip(r.URL, 2000), Snippet: clip(r.Content, 400)})
		if len(out) == a.Limit {
			break
		}
	}
	b, _ := json.Marshal(map[string]any{"query": a.Query, "results": out})
	return tools.Result{Content: string(b)}
}

func clip(s string, n int) string {
	s = strings.ToValidUTF8(strings.Join(strings.Fields(s), " "), "")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
