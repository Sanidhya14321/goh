package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestConnectOptionsListTheCatalog(t *testing.T) {
	opts := connectOptions(false)
	if opts[0].Value != "chatgpt" || opts[len(opts)-1].Value != "api" || len(opts) != len(core.ProviderCatalog)+2 {
		t.Fatal(opts)
	}
	if len(connectOptions(true)) != len(opts)+1 {
		t.Fatal("reauthorize missing when signed in")
	}
}

// verifyKey turns provider replies into actionable messages and keeps only
// chat models, renamed for the chat endpoint.
func TestVerifyKey(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer good":
			w.Write([]byte(`{"data":[{"id":"models/chat-1"},{"id":"models/text-embedding-2"}]}`))
		case "Bearer embeddings-only":
			w.Write([]byte(`{"data":[{"id":"models/text-embedding-2"}]}`))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer s.Close()
	p := core.ProviderPreset{ID: "x", Name: "Fixture", Protocol: "chat_completions", BaseURL: s.URL + "/v1",
		KeyURL: "https://example.invalid/keys", Window: 1000, ModelPrefix: "models/", Skip: []string{"embedding"}}
	c := p.Apply(core.Config{})
	models, err := verifyKey(context.Background(), p, c, "good")
	if err != nil || len(models) != 1 || models[0].ID != "chat-1" {
		t.Fatal(models, err)
	}
	if _, err := verifyKey(context.Background(), p, c, "bad"); err == nil || !strings.Contains(err.Error(), "did not accept this key") {
		t.Fatal(err)
	}
	if _, err := verifyKey(context.Background(), p, c, "embeddings-only"); err == nil || !strings.Contains(err.Error(), "no chat models") {
		t.Fatal(err)
	}
	local := p
	local.KeyURL, local.BaseURL = "", "http://127.0.0.1:1/v1"
	if _, err := verifyKey(context.Background(), local, local.Apply(core.Config{}), ""); err == nil || !strings.Contains(err.Error(), "not reachable") {
		t.Fatal(err)
	}
}

// Saving a preset writes the verified key under the preset's own scope and
// changes only the provider fields of the settings.
func TestSavePresetKeepsOtherSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	t.Setenv("ANTHROPIC_API_KEY", "")
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	raw := `{"base_url":"https://gateway.example/v1","api_key_env":"GATEWAY_KEY","model":"old","context_window":4096,"theme":"dark"}`
	cfg, err := core.DecodeConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := core.FindPreset("anthropic")
	next := p.Apply(cfg)
	next.Model, next.ContextWindow = "claude-fixture", p.Window
	if err := savePreset(path, raw, cfg, next, p, "sk-ant-fixture"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	written, err := core.DecodeConfig(string(b))
	if err != nil {
		t.Fatal(err)
	}
	if written.Model != "claude-fixture" || written.EffectiveProtocol() != "anthropic_messages" || written.APIKeyEnv != "" ||
		written.ContextWindow != 200000 || !strings.Contains(string(b), `"theme": "dark"`) {
		t.Fatal(string(b))
	}
	if key, err := written.ProviderKey(); err != nil || key != "sk-ant-fixture" {
		t.Fatal(key, err)
	}
	if key, _ := cfg.ProviderKey(); key != "" {
		t.Fatal("key saved under the old gateway's scope")
	}
}
