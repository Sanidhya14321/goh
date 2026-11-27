package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// checkRemoteURL accepts https URLs, and http only for loopback hosts, with
// no embedded credentials or fragment.
func checkRemoteURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("url must be an absolute http(s) URL without credentials or fragment")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
			return nil
		}
		return errors.New("plain http is only allowed for localhost; use https")
	}
	return errors.New("url must use https")
}

// remoteClient adds configured headers from environment variables and
// refuses redirects, so requests and their headers never leave the
// configured server.
func remoteClient(cfg core.MCPConfig) (*http.Client, error) {
	headers := http.Header{}
	for name, value := range cfg.Headers {
		if name == "" || strings.ContainsAny(name, " :\r\n") || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("invalid header %q", name)
		}
		headers.Set(name, value)
	}
	for name, env := range cfg.HeadersEnv {
		if name == "" || strings.ContainsAny(name, " :\r\n") {
			return nil, fmt.Errorf("invalid header name %q", name)
		}
		value, ok := os.LookupEnv(env)
		if !ok || value == "" {
			return nil, fmt.Errorf("header %s: environment variable %s is not set", name, env)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("header %s: value contains a line break", name)
		}
		headers.Set(name, value)
	}
	return &http.Client{
		Timeout:       0, // streaming responses; the SDK applies request contexts
		Transport:     headerTransport{base: http.DefaultTransport, headers: headers},
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("MCP server redirects are refused") },
	}, nil
}

type headerTransport struct {
	base    http.RoundTripper
	headers http.Header
}

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range t.headers {
		r.Header[k] = v
	}
	return t.base.RoundTrip(r)
}

// sameOrigin wraps c so every request must go to raw's scheme, host and
// port: an SSE server's message endpoint cannot send goh's requests and
// headers elsewhere.
func sameOrigin(c *http.Client, raw string) *http.Client {
	u, _ := url.Parse(raw)
	next := *c
	next.Transport = originTransport{base: c.Transport, scheme: u.Scheme, host: u.Host}
	return &next
}

type originTransport struct {
	base         http.RoundTripper
	scheme, host string
}

func (t originTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != t.scheme || !strings.EqualFold(r.URL.Host, t.host) {
		return nil, fmt.Errorf("MCP server named a message URL on another origin (%s://%s); refused", r.URL.Scheme, r.URL.Host)
	}
	return t.base.RoundTrip(r)
}

// sseTransport opens the SSE stream with a context that outlives start-up:
// the SDK binds the stream to the context Connect gets, and goh's start-up
// context ends after discovery. Closing the session closes the stream; ctx
// still bounds the wait for the server's first event.
type sseTransport struct{ inner *mcp.SSEClientTransport }

func (t sseTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	type result struct {
		conn mcp.Connection
		err  error
	}
	done := make(chan result, 1)
	life, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go func() {
		conn, err := t.inner.Connect(life)
		done <- result{conn, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			cancel()
			return nil, r.err
		}
		return closingConn{r.conn, cancel}, nil
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}

type closingConn struct {
	mcp.Connection
	cancel context.CancelFunc
}

func (c closingConn) Close() error {
	err := c.Connection.Close()
	c.cancel()
	return err
}
