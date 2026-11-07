package core

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type chatGPTTestTransport func(*http.Request) (*http.Response, error)

func (f chatGPTTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChatGPTAuthorizationStateIdentityAndRenewal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
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
				if r.Form.Get("refresh_token") != "fixture-renewal" {
					t.Error("wrong refresh token")
				}
				result = chatGPTTokens{Access: "renewed-access", Refresh: "renewed-refresh", Type: "Bearer", Lifetime: 3600}
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
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
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
}

func TestChatGPTEndpointRejectsCustomDestinations(t *testing.T) {
	for _, endpoint := range []string{"http://127.0.0.1:8080/responses", "https://proxy.example/v1/responses",
		"https://api.openai.com/v1/responses?extra=1"} {
		if _, err := (Config{Protocol: ChatGPTProtocol, Endpoint: endpoint}).ProviderEndpoint(); err == nil {
			t.Fatal("accepted a different ChatGPT token destination")
		}
	}
}
