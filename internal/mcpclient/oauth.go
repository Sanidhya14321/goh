package mcpclient

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"

	"github.com/Sanidhya14321/goh/internal/core"
)

// OAuth for remote MCP servers. Signing in is explicit (`goh mcp login`):
// it discovers the server's authorization server, registers GoHarness as a
// client (or uses a configured client ID), and completes an authorization
// code flow with PKCE through the browser and a loopback callback. The
// token is saved as a secret keyed by the server URL. Runs then use and
// refresh the saved token; they never open a browser or register a client,
// and a server that still refuses says how to sign in.

// signIn is what is saved for one server.
type signIn struct {
	ServerURL    string    `json:"server_url"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	AuthURL      string    `json:"auth_url"`
	TokenURL     string    `json:"token_url"`
	AuthStyle    int       `json:"auth_style,omitempty"`
	Scopes       []string  `json:"scopes,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitzero"`
}

func (s signIn) config() *oauth2.Config {
	return &oauth2.Config{ClientID: s.ClientID, ClientSecret: s.ClientSecret, Scopes: s.Scopes,
		Endpoint: oauth2.Endpoint{AuthURL: s.AuthURL, TokenURL: s.TokenURL, AuthStyle: oauth2.AuthStyle(s.AuthStyle)}}
}

func (s signIn) token() *oauth2.Token {
	return &oauth2.Token{AccessToken: s.AccessToken, RefreshToken: s.RefreshToken, TokenType: s.TokenType, Expiry: s.Expiry}
}

// SignInError says a remote MCP server needs (another) sign-in.
type SignInError struct{ Server string }

func (e *SignInError) Error() string {
	return fmt.Sprintf("MCP server %s requires sign-in: run `goh mcp login %s`", e.Server, e.Server)
}

// UsesOAuth reports whether goh signs in to this server (see usesOAuth).
func UsesOAuth(cfg core.MCPConfig) bool { return usesOAuth(cfg) }

// usesOAuth reports whether a server's requests are authorized by a saved
// sign-in: remote, and not already given an Authorization header.
func usesOAuth(cfg core.MCPConfig) bool {
	if cfg.URL == "" || cfg.Transport == "sse" {
		return false
	}
	for name := range cfg.Headers {
		if strings.EqualFold(name, "Authorization") {
			return false
		}
	}
	for name := range cfg.HeadersEnv {
		if strings.EqualFold(name, "Authorization") {
			return false
		}
	}
	return true
}

// authClient is the HTTP client for metadata, registration and token
// requests: bounded, and following redirects only to https (or loopback).
func authClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return checkRemoteURL(r.URL.String())
	}}
}

// tokenContext carries authClient for oauth2's token refreshes; it is not a
// request context, because a token source outlives any one request.
func tokenContext() context.Context {
	return context.WithValue(context.Background(), oauth2.HTTPClient, authClient())
}

// savingSource saves each new token it hands out, so a refreshed token (and
// a rotated refresh token) survives the process.
type savingSource struct {
	mu   sync.Mutex
	base oauth2.TokenSource
	save signIn
	last string
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.base.Token()
	if err != nil || t.AccessToken == s.last {
		return t, err
	}
	s.last = t.AccessToken
	s.save.AccessToken, s.save.TokenType, s.save.Expiry = t.AccessToken, t.TokenType, t.Expiry
	if t.RefreshToken != "" {
		s.save.RefreshToken = t.RefreshToken
	}
	if b, err := json.Marshal(s.save); err == nil {
		_ = core.SaveMCPCredential(s.save.ServerURL, b) // an unsaved refresh is only redone next time
	}
	return t, nil
}

func newSavingSource(saved signIn) *savingSource {
	return &savingSource{base: saved.config().TokenSource(tokenContext(), saved.token()), save: saved, last: saved.AccessToken}
}

// savedHandler authorizes a run's requests with the saved sign-in. It never
// starts a sign-in: a refusal becomes a SignInError.
type savedHandler struct {
	server string
	source oauth2.TokenSource
}

func (h *savedHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	if h.source == nil {
		return nil, nil
	}
	return h.source, nil
}

func (h *savedHandler) Authorize(context.Context, *http.Request, *http.Response) error {
	return &SignInError{Server: h.server}
}

// runHandler is the OAuth handler for a run's connection to server name.
func runHandler(name string, cfg core.MCPConfig) (auth.OAuthHandler, error) {
	h := &savedHandler{server: name}
	b, err := core.ReadMCPCredential(cfg.URL)
	if errors.Is(err, fs.ErrNotExist) {
		return h, nil
	}
	if err != nil {
		return nil, fmt.Errorf("MCP server %s sign-in: %w", name, err)
	}
	var saved signIn
	if json.Unmarshal(b, &saved) != nil || saved.ServerURL != cfg.URL || saved.AccessToken == "" {
		return h, nil // unusable or for another URL: the server will ask again
	}
	h.source = newSavingSource(saved)
	return h, nil
}

// Login signs in to the remote MCP server name and connects once with the
// new token; it returns how many tools the server offers. open shows the
// authorization page to the user.
func Login(ctx context.Context, name string, cfg core.MCPConfig, open func(string) error) (int, error) {
	if cfg.URL == "" {
		return 0, fmt.Errorf("MCP server %s is local (a command); only remote servers sign in", name)
	}
	if err := checkRemoteURL(cfg.URL); err != nil {
		return 0, fmt.Errorf("MCP server %s: %w", name, err)
	}
	if cfg.Transport == "sse" {
		return 0, fmt.Errorf("MCP server %s uses the older SSE transport, which goh does not sign in to; set its token with headers_env", name)
	}
	if !usesOAuth(cfg) {
		return 0, fmt.Errorf("MCP server %s sends its own Authorization header (headers_env); remove it to sign in instead", name)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	redirect := "http://" + listener.Addr().String() + "/callback"
	callbacks := make(chan *auth.AuthorizationResult, 1)
	var expectState string
	var stateMu sync.Mutex
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, MaxHeaderBytes: 16 << 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			stateMu.Lock()
			state := expectState
			stateMu.Unlock()
			if r.Method != http.MethodGet || r.Host != listener.Addr().String() || r.URL.Path != "/callback" ||
				state == "" || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
				http.Error(w, "Invalid sign-in callback", http.StatusBadRequest)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			if q.Get("error") != "" {
				http.Error(w, "Sign-in was refused: "+q.Get("error"), http.StatusBadRequest)
				select {
				case callbacks <- nil:
				default:
				}
				return
			}
			select {
			case callbacks <- &auth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}:
				_, _ = io.WriteString(w, "GoHarness received the sign-in response. Return to the terminal.")
			default:
				http.Error(w, "Sign-in callback already received", http.StatusConflict)
			}
		})}
	defer server.Close()
	go func() { _ = server.Serve(listener) }()

	fetch := func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
		stateMu.Lock()
		expectState = stateOf(args.URL)
		stateMu.Unlock()
		if err := open(args.URL); err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-callbacks:
			if result == nil {
				return nil, errors.New("the authorization server refused the sign-in")
			}
			return result, nil
		}
	}
	var saveErr error
	savedOnce := false
	config := &auth.AuthorizationCodeHandlerConfig{
		RedirectURL:              redirect,
		AuthorizationCodeFetcher: fetch,
		RequestRefreshToken:      true,
		Client:                   authClient(),
		NewTokenSource: func(_ context.Context, c *oauth2.Config, t *oauth2.Token) (oauth2.TokenSource, error) {
			saved := signIn{ServerURL: cfg.URL, ClientID: c.ClientID, ClientSecret: c.ClientSecret, AuthURL: c.Endpoint.AuthURL,
				TokenURL: c.Endpoint.TokenURL, AuthStyle: int(c.Endpoint.AuthStyle), Scopes: c.Scopes,
				AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, TokenType: t.TokenType, Expiry: t.Expiry}
			b, err := json.Marshal(saved)
			if err == nil {
				err = core.SaveMCPCredential(cfg.URL, b)
			}
			if err != nil {
				saveErr = err
				return nil, err
			}
			savedOnce = true
			return newSavingSource(saved), nil
		},
	}
	if cfg.OAuthClientID != "" {
		config.PreregisteredClient = &oauthex.ClientCredentials{ClientID: cfg.OAuthClientID}
	} else {
		config.DynamicClientRegistrationConfig = &auth.DynamicClientRegistrationConfig{Metadata: &oauthex.ClientRegistrationMetadata{
			ClientName: core.ProductName, RedirectURIs: []string{redirect}, GrantTypes: []string{"authorization_code", "refresh_token"},
			ResponseTypes: []string{"code"}, TokenEndpointAuthMethod: "none",
		}}
	}
	handler, err := auth.NewAuthorizationCodeHandler(config)
	if err != nil {
		return 0, err
	}
	httpClient, err := remoteClient(cfg)
	if err != nil {
		return 0, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: core.ProductName, Version: core.ProductVersion}, &mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}, MultiRoundTrip: &mcp.MultiRoundTripOptions{Disabled: true}})
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: cfg.URL, HTTPClient: httpClient, MaxRetries: -1, DisableStandaloneSSE: true, OAuthHandler: handler}, nil)
	if saveErr != nil {
		return 0, fmt.Errorf("saving the sign-in: %w", saveErr)
	}
	if err != nil {
		return 0, fmt.Errorf("MCP server %s sign-in: %w", name, err)
	}
	defer session.Close()
	tools := 0
	for _, err := range session.Tools(ctx, nil) {
		if err != nil {
			return 0, fmt.Errorf("MCP server %s: signed in, but listing tools failed: %w", name, err)
		}
		tools++
	}
	if !savedOnce {
		return 0, fmt.Errorf("MCP server %s accepted the connection without a sign-in; nothing was saved", name)
	}
	return tools, nil
}

// Logout forgets the saved sign-in for a server; it reports whether one
// was saved.
func Logout(cfg core.MCPConfig) (bool, error) {
	return core.RemoveMCPCredential(cfg.URL)
}

func stateOf(authorizationURL string) string {
	u, err := url.Parse(authorizationURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("state")
}
