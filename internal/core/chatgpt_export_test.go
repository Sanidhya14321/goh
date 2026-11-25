package core

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// An exported sign-in has no refresh or identity token, is renewed first
// when it would expire too soon, and cannot be renewed where it is used.
func TestExportChatGPTAccessCannotRenew(t *testing.T) {
	isolateConfig(t)
	saved := chatGPTAccount{
		Tokens:   chatGPTTokens{Access: "old-access", Refresh: "the-refresh", Identity: "the-id", Type: "Bearer", Scope: "openid chatgpt.tokens.use.direct", Lifetime: 3600},
		ClientID: "oaiapp_fixture", Subject: "user-1", Expires: time.Now().Add(time.Hour), Model: "gpt-x",
	}
	if err := saveChatGPTAccount(saved); err != nil {
		t.Fatal(err)
	}
	renewals := 0
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	http.DefaultTransport = chatGPTTestTransport(func(r *http.Request) (*http.Response, error) {
		r.ParseForm()
		if r.URL.String() != chatGPTIssuer+"/api/accounts/oauth/token" || r.Form.Get("refresh_token") != "the-refresh" {
			t.Errorf("unexpected request %s", r.URL)
		}
		renewals++
		return jsonResponse(chatGPTTokens{Access: "new-access", Refresh: "next-refresh", Type: "Bearer", Lifetime: 10 * 3600}), nil
	})

	b, expires, err := ExportChatGPTAccess(context.Background(), 30*time.Minute)
	if err != nil || renewals != 0 || time.Until(expires) > time.Hour {
		t.Fatalf("long enough: err=%v renewals=%d", err, renewals)
	}
	b, expires, err = ExportChatGPTAccess(context.Background(), 6*time.Hour)
	if err != nil || renewals != 1 || time.Until(expires) < 9*time.Hour {
		t.Fatalf("renew first: err=%v renewals=%d expires=%v", err, renewals, expires)
	}
	var exported chatGPTAccount
	if err := json.Unmarshal(b, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Tokens.Access != "new-access" || exported.Tokens.Refresh != "" || exported.Tokens.Identity != "" || exported.Model != "gpt-x" {
		t.Fatalf("exported %+v", exported.Tokens)
	}
	if a, _ := readChatGPTAccount(); a.Tokens.Refresh != "next-refresh" {
		t.Fatal("this machine lost its refresh token")
	}

	// Where the copy is used: valid until it expires, then sign-in required
	// without any request.
	isolateConfig(t)
	if err := saveChatGPTAccount(exported); err != nil {
		t.Fatal(err)
	}
	if tok, err := ChatGPTAccessToken(context.Background()); err != nil || tok != "new-access" {
		t.Fatalf("copy: %q %v", tok, err)
	}
	exported.Expires = time.Now().Add(-time.Minute)
	saveChatGPTAccount(exported)
	if _, err := ChatGPTAccessToken(context.Background()); !errors.Is(err, ErrChatGPTLogin) || renewals != 1 {
		t.Fatalf("expired copy: %v, renewals %d", err, renewals)
	}
}

func jsonResponse(v any) *http.Response {
	data, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data))), Header: http.Header{}}
}
