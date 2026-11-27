package webfetch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func proxyGet(t *testing.T, p *proxy, target string) (int, string) {
	t.Helper()
	c, err := net.Dial("tcp", p.address())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprintf(c, "GET %s HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n", target)
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func proxyConnect(t *testing.T, p *proxy, hostport string) int {
	t.Helper()
	c, err := net.Dial("tcp", p.address())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", hostport, hostport)
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func TestRenderProxyAddressRule(t *testing.T) {
	var hits atomic.Int32
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); io.WriteString(w, "local-secret") }))
	defer local.Close()
	host := strings.TrimPrefix(local.URL, "http://")

	strict, err := newProxy(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer strict.close()
	if code, body := proxyGet(t, strict, local.URL+"/"); code != 502 || strings.Contains(body, "secret") {
		t.Fatal("plain request to loopback passed", code, body)
	}
	if code := proxyConnect(t, strict, host); code != 403 {
		t.Fatal("CONNECT to loopback passed", code)
	}
	for _, h := range []string{"localhost:" + strings.Split(host, ":")[1], "[::1]:443", "169.254.169.254:80", "10.0.0.1:443"} {
		if code := proxyConnect(t, strict, h); code != 403 {
			t.Fatal("CONNECT passed for", h, code)
		}
	}
	if hits.Load() != 0 {
		t.Fatal("the local server was reached")
	}

	open, err := newProxy(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer open.close()
	if code, body := proxyGet(t, open, local.URL+"/"); code != 200 || body != "local-secret" {
		t.Fatal("allowed proxy failed", code, body)
	}
	if code := proxyConnect(t, open, host); code != 200 {
		t.Fatal("allowed CONNECT failed", code)
	}
}

func TestFindBrowser(t *testing.T) {
	if _, err := FindBrowser("chrome"); err == nil {
		t.Fatal("relative path accepted")
	}
	if _, err := FindBrowser(t.TempDir()); err == nil {
		t.Fatal("directory accepted")
	}
	var c *Client
	if c.CanRender() || New().CanRender() {
		t.Fatal("render on by default")
	}
	if _, ok := New().Definition().Function.Parameters["properties"].(map[string]any)["render"]; ok {
		t.Fatal("render offered without a browser")
	}
	res := New().Fetch(context.Background(), json.RawMessage(`{"url":"https://example.com","render":true}`))
	if !res.Error || !strings.Contains(res.Content, "webfetch_browser") {
		t.Fatal(res.Content)
	}
}

const scriptedPage = `<!doctype html><html><head><title>App</title></head><body><div id="app">Loading…</div>
<script>
fetch("/data").then(r => r.json()).then(d => {
  document.getElementById("app").innerHTML = "<main><h1>" + d.title + "</h1><a href='/next'>Next</a></main>";
});
</script></body></html>`

// The real browser, where one is installed.
func TestRenderWithBrowser(t *testing.T) {
	browser, err := FindBrowser("auto")
	if err != nil || testing.Short() {
		t.Skip("no Chromium-family browser installed")
	}
	var dataHits atomic.Int32
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data" {
			dataHits.Add(1)
			io.WriteString(w, `{"title":"Built by script 93"}`)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, scriptedPage)
	}))
	defer site.Close()

	plain := NewAllowingPrivate().Fetch(context.Background(), json.RawMessage(`{"url":"`+site.URL+`/"}`))
	if strings.Contains(plain.Content, "Built by script") {
		t.Fatal("fixture renders without scripts")
	}
	c := NewAllowingPrivate()
	c.SetBrowser(browser)
	start := time.Now()
	res := c.Fetch(context.Background(), json.RawMessage(`{"url":"`+site.URL+`/","render":true,"links":true,"timeout":60}`))
	if res.Error && runtime.GOOS != "darwin" && (strings.Contains(res.Content, "returned no page") || strings.Contains(res.Content, "could not start")) {
		// CI Linux runners often cannot start the browser's own sandbox.
		t.Skip("the browser could not run here:", res.Content)
	}
	if res.Error || !strings.Contains(res.Content, "Built by script 93") || !strings.Contains(res.Content, site.URL+"/next") {
		t.Fatalf("rendered page: %s", res.Content)
	}
	t.Logf("render took %s", time.Since(start))

	// The same browser under the public-only rule cannot reach this
	// machine: loopback goes through the proxy, which refuses it.
	strict := New()
	strict.SetBrowser(browser)
	before := dataHits.Load()
	res = strict.Fetch(context.Background(), json.RawMessage(`{"url":"`+site.URL+`/","render":true,"timeout":60}`))
	if !res.Error && strings.Contains(res.Content, "Built by script") || dataHits.Load() != before {
		t.Fatalf("public-only render reached loopback: %s", res.Content)
	}
}
