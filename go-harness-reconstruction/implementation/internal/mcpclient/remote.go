package mcpclient

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"example.com/independent-harness/internal/core"
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
