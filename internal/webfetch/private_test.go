package webfetch

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Review R8: by default webfetch must not reach loopback, private-network or
// link-local services, including via a hostname that resolves to loopback.
func TestDefaultClientRefusesNonPublicAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("internal-only")) }))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	c := New()
	for _, u := range []string{srv.URL, "http://localhost:" + port + "/"} {
		raw, _ := json.Marshal(map[string]any{"url": u, "format": "text"})
		r := c.Fetch(context.Background(), raw)
		if !r.Error || strings.Contains(r.Content, "internal-only") || !strings.Contains(r.Content, "loopback") {
			t.Fatalf("%s: %+v", u, r)
		}
	}
	if NewAllowingPrivate().Fetch(context.Background(), mustJSON(srv.URL)).Error {
		t.Fatal("explicit opt-in still refused loopback")
	}
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "::1", "fd00::1", "fe80::1"} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatalf("%s treated as public", ip)
		}
	}
	for _, ip := range []string{"93.184.216.34", "2606:4700::1111"} {
		if !publicIP(net.ParseIP(ip)) {
			t.Fatalf("%s treated as private", ip)
		}
	}
}

func mustJSON(u string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"url": u, "format": "text"})
	return b
}
