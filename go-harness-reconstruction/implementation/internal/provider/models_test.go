package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
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
