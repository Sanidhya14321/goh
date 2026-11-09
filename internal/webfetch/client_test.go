package webfetch

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/tools"
)

func raw(t *testing.T, url, format string) json.RawMessage {
	t.Helper()
	a := map[string]any{"url": url}
	if format != "" {
		a["format"] = format
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func decoded(t *testing.T, r tools.Result) map[string]any {
	t.Helper()
	if r.Error {
		t.Fatal(r.Content)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(r.Content), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestFetchFormatsAndSafeHTMLConversion(t *testing.T) {
	htmlBody := `<!doctype html><title>Title</title><h1>Hello &amp; world</h1><p>Read <strong>carefully</strong> and <a href="/plain">link</a>.</p><ul><li>One</li><li>Two</li></ul><pre><code>a &lt; b` + "\n" + `</code></pre><script>SECRET_SCRIPT</script><style>SECRET_STYLE</style><iframe>SECRET_FRAME</iframe><a href="javascript:bad()">visible</a>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/plain" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, "plain 😀\n")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlBody)
	}))
	defer server.Close()
	c := New()
	defer c.Close()
	for _, format := range []string{"text", "markdown", "html"} {
		t.Run(format, func(t *testing.T) {
			v := decoded(t, c.Fetch(context.Background(), raw(t, server.URL, format)))
			content := v["content"].(string)
			if format == "html" {
				if content != htmlBody {
					t.Fatal(content)
				}
			} else {
				if !strings.Contains(content, "Hello & world") || strings.Contains(content, "SECRET_") {
					t.Fatal(content)
				}
				if format == "markdown" {
					for _, expected := range []string{"# Hello", "**carefully**", "[link](" + server.URL + "/plain)", "- One", "```"} {
						if !strings.Contains(content, expected) {
							t.Fatal(expected, content)
						}
					}
				}
			}
		})
	}
	v := decoded(t, c.Fetch(context.Background(), raw(t, server.URL+"/plain", "")))
	if v["content"] != "plain 😀\n" || v["format"] != "markdown" {
		t.Fatal(v)
	}
}
func TestRedirectBoundaryAndNoCredentialCookieProxyForwarding(t *testing.T) {
	var otherCalls atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { otherCalls.Add(1); fmt.Fprint(w, "outside") }))
	defer other.Close()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "" || r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("credentials forwarded", r.Header)
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "independent-harness") {
			t.Error("user agent missing")
		}
		switch r.URL.Path {
		case "/same":
			http.SetCookie(w, &http.Cookie{Name: "secret", Value: "must-not-forward"})
			http.Redirect(w, r, "/ok", 302)
		case "/cross":
			http.Redirect(w, r, other.URL, 302)
		case "/loop":
			http.Redirect(w, r, "/loop", 302)
		default:
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "ok")
		}
	}))
	defer server.Close()
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")
	t.Setenv("OPENAI_API_KEY", "must-not-forward")
	c := New()
	defer c.Close()
	if c.http.Transport.(*http.Transport).Proxy != nil || c.http.Jar != nil {
		t.Fatal("proxy/cookie jar enabled")
	}
	v := decoded(t, c.Fetch(context.Background(), raw(t, server.URL+"/same", "text")))
	if v["content"] != "ok" || v["url"] != server.URL+"/ok" {
		t.Fatal(v)
	}
	if r := c.Fetch(context.Background(), raw(t, server.URL+"/cross", "text")); !r.Error || !strings.Contains(r.Content, "cross-origin") {
		t.Fatal(r)
	}
	if otherCalls.Load() != 0 {
		t.Fatal("outside origin contacted")
	}
	before := requests.Load()
	if r := c.Fetch(context.Background(), raw(t, server.URL+"/loop", "text")); !r.Error || !strings.Contains(r.Content, "redirect limit") {
		t.Fatal(r)
	}
	if requests.Load()-before != 6 {
		t.Fatal("unexpected redirect bound", requests.Load()-before)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	decoded(t, c.Fetch(context.Background(), raw(t, server.URL+"/ok", "text")))
}
func TestResponseValidationAndBoundedResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			w.WriteHeader(403)
			fmt.Fprint(w, "secret error content")
		case "/binary":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte{0, 1, 2, 255})
		case "/invalid-utf8":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte{255})
		case "/latin":
			w.Header().Set("Content-Type", "text/plain; charset=iso-8859-1")
			fmt.Fprint(w, "body")
		case "/header":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", fmt.Sprint(MaxResponse+1))
			w.WriteHeader(200)
		case "/stream":
			w.Header().Set("Content-Type", "text/plain")
			w.(http.Flusher).Flush()
			fmt.Fprint(w, strings.Repeat("x", MaxResponse+1))
		case "/gzip":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Encoding", "gzip")
			z := gzip.NewWriter(w)
			fmt.Fprint(z, strings.Repeat("x", MaxResponse+1))
			z.Close()
		case "/long":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, strings.Repeat("\x01😀", 20000))
		case "/json":
			w.Header().Set("Content-Type", "application/problem+json")
			fmt.Fprint(w, `{"answer":42}`)
		default:
			fmt.Fprint(w, "sniffed plain")
		}
	}))
	defer server.Close()
	c := New()
	defer c.Close()
	for _, path := range []string{"status", "binary", "invalid-utf8", "latin", "header", "stream", "gzip"} {
		r := c.Fetch(context.Background(), raw(t, server.URL+"/"+path, "text"))
		if !r.Error || len(r.Content) > tools.MaxInline || strings.Contains(r.Content, "secret error content") {
			t.Fatal(path, r)
		}
	}
	r := c.Fetch(context.Background(), raw(t, server.URL+"/long", "text"))
	v := decoded(t, r)
	if len(r.Content) > tools.MaxInline || v["truncated"] != true || !utf8.ValidString(v["content"].(string)) {
		t.Fatal("result bound violated", len(r.Content))
	}
	v = decoded(t, c.Fetch(context.Background(), raw(t, server.URL+"/json", "text")))
	if v["content"] != `{"answer":42}` {
		t.Fatal(v)
	}
	decoded(t, c.Fetch(context.Background(), raw(t, server.URL, "text")))
}
func TestCancellationTimeoutAndInterruptedBody(t *testing.T) {
	started := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broken" {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "100")
			fmt.Fprint(w, "short")
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.(http.Flusher).Flush()
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer server.Close()
	c := New()
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan tools.Result, 1)
	go func() { done <- c.Fetch(ctx, raw(t, server.URL, "text")) }()
	<-started
	cancel()
	select {
	case r := <-done:
		if !r.Error {
			t.Fatal(r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fetch ignored cancellation")
	}
	a := map[string]any{"url": server.URL, "timeout": 1}
	b, _ := json.Marshal(a)
	start := time.Now()
	r := c.Fetch(context.Background(), b)
	if !r.Error || time.Since(start) > 2*time.Second {
		t.Fatal(r, time.Since(start))
	}
	if r := c.Fetch(context.Background(), raw(t, server.URL+"/broken", "text")); !r.Error || !strings.Contains(r.Content, "interrupted") {
		t.Fatal(r)
	}
	var absent *Client
	if r := absent.Fetch(context.Background(), raw(t, server.URL, "text")); !r.Error {
		t.Fatal("nil client succeeded")
	}
	if err := absent.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestArgumentAndURLAdmissionBeforeNetwork(t *testing.T) {
	for _, s := range []string{"file:///etc/passwd", "ftp://fixture/a", "http://user:password@fixture/a", "http://fixture/a#fragment", "http://fixture:99999/a", "http://fixture:0/a", "http://[::1%25lo0]/a", "http:///missing", "javascript:alert(1)", strings.Repeat("x", 5000)} {
		if _, err := Prepare(raw(t, s, "text")); err == nil {
			t.Fatal("admitted", s)
		}
	}
	for _, s := range []string{`null`, `[]`, `{}`, `{"url":null}`, `{"url":"http://fixture","format":null}`, `{"url":"http://fixture","format":""}`, `{"url":"http://fixture","timeout":0}`, `{"url":"http://fixture","timeout":121}`, `{"url":"http://fixture","timeout":0.5}`, `{"url":"http://fixture","headers":{}}`, `{"url":"http://fixture"} {}`} {
		if _, err := Prepare(json.RawMessage(s)); err == nil {
			t.Fatal("accepted", s)
		}
	}
	preview, err := Prepare(raw(t, "https://fixture/path?query=value", ""))
	if err != nil || !strings.Contains(preview, "query=value") || !strings.Contains(preview, "same origin") {
		t.Fatal(preview, err)
	}
}
func TestHTMLMalformedAndSuppressedElements(t *testing.T) {
	base, _ := url.Parse("https://fixture/base")
	for _, body := range []string{`<script>SECRET<script>nested</script><p>visible`, `<script/>SECRET</script><p>visible`, `<style>SECRET</style><p>visible`, `<template><p>SECRET</p></template><p>visible`, `<iframe>SECRET</iframe><p>visible`} {
		got, _, err := renderHTML(context.Background(), []byte(body), "text", base)
		if err != nil || strings.Contains(got, "SECRET") || !strings.Contains(got, "visible") {
			t.Fatal(got, err)
		}
	}
	if _, _, err := renderHTML(context.Background(), []byte(strings.Repeat("<div>", 300)), "text", base); err == nil {
		t.Fatal("unbounded nesting")
	}
}
func FuzzHTMLConversion(f *testing.F) {
	for _, s := range []string{"<h1>hello</h1>", "<script/>secret</script><p>ok", `<a href="javascript:bad">x`} {
		f.Add(s)
	}
	base, _ := url.Parse("http://fixture/")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > MaxResponse {
			return
		}
		text, _, _ := renderHTML(context.Background(), []byte(s), "markdown", base)
		if len(text) > maxConverted || !utf8.ValidString(text) {
			t.Fatal("unbounded/invalid conversion")
		}
	})
}

func FuzzWebfetchArguments(f *testing.F) {
	for _, s := range []string{`{"url":"http://fixture"}`, `null`, `{"url":"file:///x"}`, `{"url":"http://fixture","timeout":1}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		preview, err := Prepare(json.RawMessage(s))
		if err != nil {
			return
		}
		a, u, err := decode(json.RawMessage(s))
		if err != nil || u == nil || a.Timeout < 1 || a.Timeout > 120 || len(a.URL) > maxURL || len(preview) > 8192 {
			t.Fatal("inconsistent argument admission")
		}
	})
}
