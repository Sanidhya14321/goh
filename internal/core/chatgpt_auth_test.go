package core

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/Sanidhya14321/goh/internal/testutil"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type chatGPTTestTransport func(*http.Request) (*http.Response, error)

func (f chatGPTTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChatGPTAuthorizationStateIdentityAndRenewal(t *testing.T) {
	isolateConfig(t)
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	var nonce, redirect string
	var challenge string
	clientID := "oaiapp_fixture"
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = chatGPTTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "127.0.0.1" {
			return oldTransport.RoundTrip(r)
		}
		var result any
		switch r.URL.String() {
		case chatGPTIssuer + "/.well-known/jwks.json":
			result = map[string]any{"keys": []any{map[string]string{"kid": "fixture", "kty": "RSA",
				"n": encode(private.N.Bytes()), "e": "AQAB"}}}
		case chatGPTIssuer + "/api/accounts/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("client_id") != clientID || r.Form.Get("resource") != chatGPTResource {
				t.Error("exchange lost OAuth binding")
			}
			if r.Form.Get("grant_type") == "refresh_token" {
				switch r.Form.Get("refresh_token") {
				case "fixture-renewal":
					result = chatGPTTokens{Access: "renewed-access", Refresh: "renewed-refresh", Type: "Bearer", Lifetime: 3600}
				case "renewed-refresh":
					result = chatGPTTokens{Access: "forced-access", Refresh: "forced-refresh", Type: "Bearer", Lifetime: 3600}
				default:
					t.Error("wrong refresh token")
				}
				break
			}
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("code") != "fixture-code" || r.Form.Get("redirect_uri") != redirect || encode(digest[:]) != challenge {
				t.Error("exchange lost callback or PKCE binding")
			}
			claims, _ := json.Marshal(map[string]any{"iss": chatGPTIssuer, "sub": "fixture-user",
				"aud": []string{clientID}, "nonce": nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
			token := encode([]byte(`{"alg":"RS256","kid":"fixture"}`)) + "." + encode(claims)
			hash := sha256.Sum256([]byte(token))
			signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, hash[:])
			if err != nil {
				t.Error(err)
			}
			result = chatGPTTokens{Access: "fixture-access", Refresh: "fixture-renewal", Identity: token + "." + encode(signature),
				Type: "Bearer", Scope: chatGPTScope, Lifetime: 3600}
		case chatGPTResource + "/models":
			if r.Header.Get("Authorization") != "Bearer fixture-access" {
				t.Error("catalog lacked access token")
			}
			result = map[string]any{"models": []ChatGPTModelInfo{{Slug: "model-hidden", Visibility: "hidden"}, {Slug: "model-visible", Visibility: "list"}}}
		default:
			t.Error("unexpected OAuth destination")
		}
		data, _ := json.Marshal(result)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data))), Header: http.Header{}}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	model, err := LoginChatGPT(ctx, func(address string) error {
		u, err := url.Parse(address)
		if err != nil {
			return err
		}
		q := u.Query()
		nonce, redirect, challenge = q.Get("nonce"), q.Get("redirect_uri"), q.Get("code_challenge")
		if q.Get("client_id") != "dynamic_agent_client" || q.Get("agent_name_hint") != "GoHarness" ||
			q.Get("code_challenge_method") != "S256" || nonce == "" {
			t.Error("invalid authorization request")
		}
		bad, err := http.Get(redirect + "?state=wrong&code=fixture-code")
		if err != nil {
			return err
		}
		bad.Body.Close()
		if bad.StatusCode != 400 {
			t.Error("accepted incorrect callback state")
		}
		callback := url.Values{"state": {q.Get("state")}, "code": {"fixture-code"}, "client_id": {clientID}}
		ok, err := http.Get(redirect + "?" + callback.Encode())
		if err != nil {
			return err
		}
		ok.Body.Close()
		return nil
	})
	if err != nil || model != "model-visible" {
		t.Fatalf("unexpected sign-in outcome: %q %v", model, err)
	}
	path, _ := chatGPTPath("chatgpt-account.json")
	if info, err := os.Stat(path); err != nil || testutil.UnixModes() && info.Mode().Perm() != 0600 {
		t.Fatal("credentials were not private", err)
	}
	account, err := readChatGPTAccount()
	if err != nil || account.Subject != "fixture-user" {
		t.Fatal("verified account not saved", err)
	}
	if _, err := verifyChatGPTIdentity(ctx, account.Tokens.Identity, clientID, "wrong"); err == nil {
		t.Fatal("accepted incorrect identity nonce")
	}
	account.Expires = time.Now().Add(-time.Minute)
	if err := saveChatGPTAccount(account); err != nil {
		t.Fatal(err)
	}
	token, err := ChatGPTAccessToken(ctx)
	if err != nil || token != "renewed-access" {
		t.Fatal("refresh failed", err)
	}
	account, err = readChatGPTAccount()
	if err != nil || account.Tokens.Refresh != "renewed-refresh" {
		t.Fatal("rotated credential not saved", err)
	}
	// A 401 on an unexpired token forces one renewal.
	if token, err := RenewChatGPTAccessToken(ctx, "renewed-access"); err != nil || token != "forced-access" {
		t.Fatal("forced renewal failed", token, err)
	}
	// A stale rejection (another process already renewed) reuses the new token.
	if token, err := RenewChatGPTAccessToken(ctx, "renewed-access"); err != nil || token != "forced-access" {
		t.Fatal("renewed twice for one rejection", token, err)
	}
}

func TestChatGPTEndpointRejectsCustomDestinations(t *testing.T) {
	for _, endpoint := range []string{"http://127.0.0.1:8080/responses", "https://proxy.example/v1/responses",
		"https://api.openai.com/v1/responses?extra=1"} {
		if _, err := (Config{Protocol: ChatGPTProtocol, Endpoint: endpoint}).ProviderEndpoint(); err == nil {
			t.Fatal("accepted a different ChatGPT token destination")
		}
	}
}

func TestCorruptCredentialsAreReplacedBySignIn(t *testing.T) {
	isolateConfig(t)
	path, err := chatGPTPath("chatgpt-account.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, setup := range map[string]func(){
		"corrupt":    func() { os.WriteFile(path, []byte("{not json"), 0o600) },
		"world-read": func() { os.WriteFile(path, []byte("{}"), 0o644); os.Chmod(path, 0o644) },
	} {
		setup()
		_, err := ChatGPTModel()
		if err == nil || !strings.Contains(err.Error(), "goh auth login") {
			t.Fatalf("%s: error should name the recovery command: %v", name, err)
		}
		if name == "world-read" && testutil.UnixModes() && !strings.Contains(err.Error(), path) {
			t.Fatalf("permission error should name the file: %v", err)
		}
		// A new sign-in saves over the damaged file.
		var tokens chatGPTTokens
		tokens.Scope = "openid chatgpt.tokens.use.direct"
		tokens.Access = "token"
		if err := saveChatGPTAccount(chatGPTAccount{Tokens: tokens, ClientID: "c", Subject: "s", Model: "m", Expires: time.Now().Add(time.Hour)}); err != nil {
			t.Fatalf("%s: sign-in could not replace the file: %v", name, err)
		}
		if model, err := ChatGPTModel(); err != nil || model != "m" {
			t.Fatalf("%s: credentials unreadable after sign-in: %v", name, err)
		}
	}
}

// The CLI paths, not only the save helper: sign-in gets past a damaged file
// (it is replaced on success) and sign-out removes it.
func TestDamagedCredentialsDoNotBlockLoginOrLogout(t *testing.T) {
	isolateConfig(t)
	path, err := chatGPTPath("chatgpt-account.json")
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte(`{"client_id":"c","sub`), 0o600)
	opened := errors.New("browser step reached")
	if _, err := LoginChatGPT(context.Background(), func(string) error { return opened }); !errors.Is(err, opened) {
		t.Fatalf("login stopped at the damaged file: %v", err)
	}
	if err := LogoutChatGPT(context.Background()); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("damaged file not removed")
	}
}

// Switching accounts registers afresh (new client and host identity, so a
// workspace-scoped registration does not block another workspace), accepts
// a different account, and replaces and revokes the old sign-in only after
// the new one succeeds.
func TestSwitchChatGPTAccount(t *testing.T) {
	isolateConfig(t)
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	old := chatGPTAccount{ClientID: "oaiapp_old", Subject: "old-user", Model: "m",
		Tokens:  chatGPTTokens{Access: "old-access", Refresh: "old-refresh", Type: "Bearer", Scope: chatGPTScope, Lifetime: 3600},
		Expires: time.Now().Add(time.Hour)}
	if err := saveChatGPTAccount(old); err != nil {
		t.Fatal(err)
	}
	hostPath, _ := chatGPTPath("chatgpt-host")
	if err := writePrivate(hostPath, "urn:uuid:old-host"); err != nil {
		t.Fatal(err)
	}
	var nonce string
	revoked := url.Values{}
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = chatGPTTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "127.0.0.1" {
			return oldTransport.RoundTrip(r)
		}
		var result any = map[string]any{}
		switch r.URL.String() {
		case chatGPTIssuer + "/.well-known/jwks.json":
			result = map[string]any{"keys": []any{map[string]string{"kid": "k", "kty": "RSA", "n": encode(private.N.Bytes()), "e": "AQAB"}}}
		case chatGPTIssuer + "/api/accounts/oauth/token":
			_ = r.ParseForm()
			if r.Form.Get("client_id") != "oaiapp_new" {
				t.Error("exchange used the old registration")
			}
			claims, _ := json.Marshal(map[string]any{"iss": chatGPTIssuer, "sub": "new-user", "aud": []string{"oaiapp_new"},
				"nonce": nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
			token := encode([]byte(`{"alg":"RS256","kid":"k"}`)) + "." + encode(claims)
			hash := sha256.Sum256([]byte(token))
			signature, _ := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, hash[:])
			result = chatGPTTokens{Access: "new-access", Refresh: "new-refresh", Identity: token + "." + encode(signature),
				Type: "Bearer", Scope: chatGPTScope, Lifetime: 3600}
		case chatGPTIssuer + "/api/accounts/oauth/revoke":
			_ = r.ParseForm()
			revoked = r.Form
		case chatGPTResource + "/models":
			result = map[string]any{"models": []ChatGPTModelInfo{{Slug: "m"}}}
		default:
			t.Error("unexpected destination", r.URL.String())
		}
		data, _ := json.Marshal(result)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data))), Header: http.Header{}}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// An abandoned switch (OpenAI showed an error page) keeps the old sign-in.
	abandoned := errors.New("abandoned")
	if _, err := SwitchChatGPTAccount(ctx, func(string) error { return abandoned }); !errors.Is(err, abandoned) {
		t.Fatal(err)
	}
	if a, err := readChatGPTAccount(); err != nil || a.Subject != "old-user" {
		t.Fatal("old sign-in lost", err)
	}
	if b, _ := os.ReadFile(hostPath); string(b) != "urn:uuid:old-host" {
		t.Fatal("host identity changed by a failed switch")
	}

	var sentHost string
	_, err = SwitchChatGPTAccount(ctx, func(address string) error {
		u, _ := url.Parse(address)
		q := u.Query()
		nonce, sentHost = q.Get("nonce"), q.Get("ext_agent_host_id")
		if q.Get("client_id") != "dynamic_agent_client" || q.Get("agent_name_hint") != "GoHarness" || sentHost == "urn:uuid:old-host" {
			t.Error("switch reused the old registration:", q.Get("client_id"), sentHost)
		}
		cb := url.Values{"state": {q.Get("state")}, "code": {"c"}, "client_id": {"oaiapp_new"}}
		resp, err := http.Get(q.Get("redirect_uri") + "?" + cb.Encode())
		if err == nil {
			resp.Body.Close()
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err := readChatGPTAccount()
	if err != nil || a.Subject != "new-user" || a.ClientID != "oaiapp_new" {
		t.Fatal("new account not saved", a.Subject, a.ClientID, err)
	}
	if b, _ := os.ReadFile(hostPath); string(b) != sentHost {
		t.Fatal("new host identity not kept")
	}
	if revoked.Get("token") != "old-refresh" || revoked.Get("client_id") != "oaiapp_old" {
		t.Fatal("old sign-in not revoked", revoked)
	}
}
