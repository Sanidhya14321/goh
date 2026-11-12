package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestListModels(t *testing.T) {
	var auth, path string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		w.Write([]byte(`{"data":[{"id":"b-model"},{"id":"a-model"},{"id":"a-model"},{"id":""}]}`))
	}))
	defer s.Close()
	t.Setenv("FIXTURE_MODELS_KEY", "k-123")
	cfg := core.Config{Endpoint: s.URL + "/v1/chat/completions", APIKeyEnv: "FIXTURE_MODELS_KEY", Model: "x"}
	ids, err := ListModels(context.Background(), cfg, s.Client())
	if err != nil || strings.Join(ids, ",") != "a-model,b-model" || path != "/v1/models" || auth != "Bearer k-123" {
		t.Fatal(ids, err, path, auth)
	}
	redirect := httptest.NewServer(http.RedirectHandler("https://elsewhere.example/v1/models", http.StatusFound))
	defer redirect.Close()
	if _, err := ListModels(context.Background(), core.Config{Endpoint: redirect.URL + "/v1/chat/completions", Model: "x"}, redirect.Client()); err == nil {
		t.Fatal("redirect followed")
	}
	if _, err := ListModels(context.Background(), core.Config{Protocol: core.ChatGPTProtocol, Model: "x"}, nil); err == nil {
		t.Fatal("ChatGPT should use the account catalog")
	}
	for in, want := range map[string]string{
		"https://api.example.com/v1/chat/completions": "https://api.example.com/v1/models",
		"https://api.example.com/v1/responses":        "https://api.example.com/v1/models",
		"https://api.anthropic.com/v1/messages":       "https://api.anthropic.com/v1/models",
	} {
		if got, _ := ModelsURL(in); got != want {
			t.Fatal(in, got)
		}
	}
	if _, err := ModelsURL("https://x.example/other"); err == nil {
		t.Fatal("unknown endpoint shape accepted")
	}
}

// Provider setup checks a pasted key before saving it: the key goes in the
// protocol's own header, a refusal is ErrKeyRejected, and reported context
// windows are kept.
func TestListModelInfoChecksAnUnsavedKey(t *testing.T) {
	var gotAuth, gotAPIKey string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotAPIKey = r.Header.Get("Authorization"), r.Header.Get("x-api-key")
		if gotAuth != "Bearer good" && gotAPIKey != "good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"data":[{"id":"b","context_length":131072},{"id":"a","max_input_tokens":200000},{"id":"c"},{"id":"a"}]}`))
	}))
	defer s.Close()
	chat := core.Config{BaseURL: s.URL + "/v1", Model: "x"}
	if _, err := ListModelInfo(context.Background(), chat, "bad", s.Client()); !errors.Is(err, ErrKeyRejected) {
		t.Fatal(err)
	}
	infos, err := ListModelInfo(context.Background(), chat, "good", s.Client())
	if err != nil || gotAuth != "Bearer good" {
		t.Fatal(err, gotAuth)
	}
	want := []ModelInfo{{"a", 200000}, {"b", 131072}, {"c", 0}}
	if len(infos) != 3 || infos[0] != want[0] || infos[1] != want[1] || infos[2] != want[2] {
		t.Fatal(infos)
	}
	anthropic := core.Config{BaseURL: s.URL + "/v1", Model: "x", Protocol: "anthropic_messages"}
	if _, err := ListModelInfo(context.Background(), anthropic, "good", s.Client()); err != nil || gotAPIKey != "good" || gotAuth != "" {
		t.Fatal(err, gotAPIKey, gotAuth)
	}
}

// Google answers a bad key with 400; providers with a public model list are
// checked on a key endpoint of their own host.
func TestKeyRefusalsAndKeyCheck(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/key" && r.Header.Get("Authorization") == "Bearer good":
			w.Write([]byte(`{}`))
		case r.URL.Path == "/v1/key":
			w.WriteHeader(http.StatusUnauthorized)
		case r.URL.Path == "/v1/models" && r.Header.Get("Authorization") == "Bearer google-bad":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":400,"message":"Please pass a valid API key"}}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"model missing"}`))
		}
	}))
	defer s.Close()
	c := core.Config{BaseURL: s.URL + "/v1", Model: "x"}
	if _, err := ListModelInfo(context.Background(), c, "google-bad", s.Client()); !errors.Is(err, ErrKeyRejected) {
		t.Fatal(err)
	}
	if _, err := ListModelInfo(context.Background(), c, "other", s.Client()); err == nil || errors.Is(err, ErrKeyRejected) {
		t.Fatal("an unrelated 400 reported as a rejected key:", err)
	}
	if err := CheckKey(context.Background(), c, s.URL+"/v1/key", "good", s.Client()); err != nil {
		t.Fatal(err)
	}
	if err := CheckKey(context.Background(), c, s.URL+"/v1/key", "bad", s.Client()); !errors.Is(err, ErrKeyRejected) {
		t.Fatal(err)
	}
	if err := CheckKey(context.Background(), c, "https://elsewhere.example/key", "good", s.Client()); err == nil {
		t.Fatal("key sent to another host")
	}
}
