package mcpclient

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// oauthFixture is a loopback authorization server and an MCP server that
// accepts only its tokens, following the MCP authorization flow: a 401 with
// protected-resource metadata, authorization-server metadata, dynamic
// client registration, authorization code with PKCE, and refresh.
type oauthFixture struct {
	t         *testing.T
	as, mcp   *httptest.Server
	mu        sync.Mutex
	clients   map[string]string // client_id -> redirect_uri
	codes     map[string]codeGrant
	valid     map[string]bool // access tokens the MCP server accepts
	issued    int
	refresh   map[string]bool
	sawToken  []string // Authorization headers the MCP server received
	expiresIn int
}

type codeGrant struct{ client, redirect, challenge, resource string }

func newOAuthFixture(t *testing.T) *oauthFixture {
	f := &oauthFixture{t: t, clients: map[string]string{}, codes: map[string]codeGrant{}, valid: map[string]bool{}, refresh: map[string]bool{}, expiresIn: 3600}
	f.as = httptest.NewServer(http.HandlerFunc(f.serveAS))
	t.Cleanup(f.as.Close)
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, in echoIn) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echo: " + in.Text}}}, nil, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	f.mcp = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/oauth-protected-resource" || r.URL.Path == "/.well-known/oauth-protected-resource/mcp" {
			writeJSON(w, map[string]any{"resource": f.mcp.URL + "/mcp", "authorization_servers": []string{f.as.URL}})
			return
		}
		header := r.Header.Get("Authorization")
		f.mu.Lock()
		f.sawToken = append(f.sawToken, header)
		ok := f.valid[strings.TrimPrefix(header, "Bearer ")]
		f.mu.Unlock()
		if !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource/mcp"`, f.mcp.URL))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(f.mcp.Close)
	return f
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (f *oauthFixture) issue(w http.ResponseWriter) {
	f.issued++
	access, refresh := fmt.Sprintf("access-%d", f.issued), fmt.Sprintf("refresh-%d", f.issued)
	f.valid[access], f.refresh[refresh] = true, true
	writeJSON(w, map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": f.expiresIn})
}

func (f *oauthFixture) serveAS(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/.well-known/oauth-authorization-server":
		writeJSON(w, map[string]any{"issuer": f.as.URL, "authorization_endpoint": f.as.URL + "/authorize", "token_endpoint": f.as.URL + "/token",
			"registration_endpoint": f.as.URL + "/register", "response_types_supported": []string{"code"},
			"code_challenge_methods_supported": []string{"S256"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
			"token_endpoint_auth_methods_supported": []string{"none"}})
	case "/register":
		var meta struct {
			RedirectURIs []string `json:"redirect_uris"`
		}
		json.NewDecoder(r.Body).Decode(&meta)
		id := fmt.Sprintf("client-%d", len(f.clients)+1)
		f.clients[id] = meta.RedirectURIs[0]
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{"client_id": id, "redirect_uris": meta.RedirectURIs, "token_endpoint_auth_method": "none"})
	case "/authorize":
		q := r.URL.Query()
		redirect, known := f.clients[q.Get("client_id")]
		if !known || redirect != q.Get("redirect_uri") || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
			http.Error(w, "bad authorization request", http.StatusBadRequest)
			return
		}
		code := fmt.Sprintf("code-%d", len(f.codes)+1)
		f.codes[code] = codeGrant{client: q.Get("client_id"), redirect: redirect, challenge: q.Get("code_challenge"), resource: q.Get("resource")}
		http.Redirect(w, r, redirect+"?code="+code+"&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
	case "/token":
		r.ParseForm()
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			g, ok := f.codes[r.Form.Get("code")]
			delete(f.codes, r.Form.Get("code"))
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if !ok || g.client != r.Form.Get("client_id") || g.redirect != r.Form.Get("redirect_uri") || base64.RawURLEncoding.EncodeToString(sum[:]) != g.challenge {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, map[string]string{"error": "invalid_grant"})
				return
			}
			f.issue(w)
		case "refresh_token":
			if !f.refresh[r.Form.Get("refresh_token")] {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, map[string]string{"error": "invalid_grant"})
				return
			}
			delete(f.refresh, r.Form.Get("refresh_token")) // rotated
			f.issue(w)
		default:
			http.Error(w, "unsupported grant", http.StatusBadRequest)
		}
	default:
		http.NotFound(w, r)
	}
}

// browse plays the user's browser: it follows the authorization page's
// redirect to the loopback callback.
func browse(address string) error {
	resp, err := http.Get(address)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authorization page: %s", resp.Status)
	}
	return nil
}

func isolateCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
}

func TestMCPOAuthSignInRunAndRefresh(t *testing.T) {
	isolateCredentials(t)
	f := newOAuthFixture(t)
	cfg := core.MCPConfig{URL: f.mcp.URL + "/mcp"}
	allow := func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: true}, nil }

	// Before signing in, a run says how to sign in and never opens a browser.
	m := New(map[string]core.MCPConfig{"docs": cfg}, t.TempDir(), allow)
	err := connectErr(m)
	m.Close()
	if err == nil || !strings.Contains(err.Error(), "requires sign-in: run `goh mcp login docs`") || len(f.clients) != 0 {
		t.Fatalf("unsigned run: %v (registered clients: %d)", err, len(f.clients))
	}

	opened := 0
	tools, err := Login(context.Background(), "docs", cfg, func(address string) error { opened++; return browse(address) })
	if err != nil || tools != 1 || opened != 1 {
		t.Fatalf("login: %d tools, opened %d: %v", tools, opened, err)
	}
	saved, err := core.ReadMCPCredential(cfg.URL)
	if err != nil || !strings.Contains(string(saved), `"access_token":"access-1"`) {
		t.Fatalf("saved sign-in: %s %v", saved, err)
	}

	// A run uses the saved token.
	m = New(map[string]core.MCPConfig{"docs": cfg}, t.TempDir(), allow)
	if err := connectErr(m); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"text": "hi"})
	if r := m.Execute(context.Background(), "mcp__docs__echo", args); r.Error || !strings.Contains(r.Content, "echo: hi") {
		t.Fatal(r.Content)
	}
	m.Close()

	// An expired token is refreshed, and the rotated refresh token saved.
	var s struct {
		Expiry string `json:"expiry"`
	}
	json.Unmarshal(saved, &s)
	expired := strings.Replace(string(saved), `"expiry":"`+s.Expiry+`"`, `"expiry":"2000-01-01T00:00:00Z"`, 1)
	if err := core.SaveMCPCredential(cfg.URL, []byte(expired)); err != nil {
		t.Fatal(err)
	}
	m = New(map[string]core.MCPConfig{"docs": cfg}, t.TempDir(), allow)
	if err := connectErr(m); err != nil {
		t.Fatal("refresh:", err)
	}
	m.Close()
	saved, _ = core.ReadMCPCredential(cfg.URL)
	if !strings.Contains(string(saved), `"access_token":"access-2"`) || !strings.Contains(string(saved), `"refresh_token":"refresh-2"`) {
		t.Fatalf("refreshed token not saved: %s", saved)
	}

	// The token is only ever sent to its own server: another URL has none.
	other := core.MCPConfig{URL: f.mcp.URL + "/mcp?other=1"}
	m = New(map[string]core.MCPConfig{"other": other}, t.TempDir(), allow)
	if err := connectErr(m); err == nil || !strings.Contains(err.Error(), "goh mcp login other") {
		t.Fatal("another URL used this sign-in:", err)
	}
	m.Close()

	// Logout forgets it.
	if removed, err := Logout(cfg); !removed || err != nil {
		t.Fatal("logout", removed, err)
	}
	if _, err := core.ReadMCPCredential(cfg.URL); err == nil {
		t.Fatal("sign-in still saved after logout")
	}
}

// A server given its own Authorization header is not signed in to, and a
// local server cannot be.
func TestMCPOAuthOnlyForRemoteServersWithoutAuthHeader(t *testing.T) {
	if usesOAuth(core.MCPConfig{Command: "server"}) || usesOAuth(core.MCPConfig{URL: "https://x/mcp", HeadersEnv: map[string]string{"authorization": "TOKEN"}}) {
		t.Fatal("OAuth used where it does not apply")
	}
	if !usesOAuth(core.MCPConfig{URL: "https://x/mcp", HeadersEnv: map[string]string{"X-Team": "TEAM"}}) {
		t.Fatal("OAuth not used for a remote server")
	}
	if _, err := Login(context.Background(), "local", core.MCPConfig{Command: "server"}, browse); err == nil {
		t.Fatal("local server signed in")
	}
}

// connectErr is Connect's error or the first server failure it recorded.
func connectErr(m *Manager) error {
	if err := m.Connect(context.Background()); err != nil {
		return err
	}
	if failures := m.Failures(); len(failures) > 0 {
		return errors.New(failures[0].Error)
	}
	return nil
}
