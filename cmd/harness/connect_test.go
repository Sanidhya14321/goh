package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestConnectOptionsListTheCatalog(t *testing.T) {
	opts := connectOptions(false)
	if opts[0].Value != "chatgpt" || opts[len(opts)-1].Value != "custom" || len(opts) != len(core.ProviderCatalog)+2 {
		t.Fatal(opts)
	}
	if len(connectOptions(true)) != len(opts)+2 {
		t.Fatal("reauthorize and switch missing when signed in")
	}
}

// checkRoute turns provider replies into actionable messages and keeps only
// chat models, renamed for the chat endpoint.
func TestCheckRoute(t *testing.T) {
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
	models, err := checkRoute(context.Background(), c, "good", p)
	if err != nil || len(models) != 1 || models[0].ID != "chat-1" {
		t.Fatal(models, err)
	}
	if _, err := checkRoute(context.Background(), c, "bad", p); err == nil || !strings.Contains(err.Error(), "did not accept this key") {
		t.Fatal(err)
	}
	if _, err := checkRoute(context.Background(), c, "embeddings-only", p); err == nil || !strings.Contains(err.Error(), "no chat models") {
		t.Fatal(err)
	}
	// A custom provider (no preset) is named by its host.
	if _, err := checkRoute(context.Background(), c, "bad", core.ProviderPreset{}); err == nil || !strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatal(err)
	}
	local := p
	local.KeyURL, local.BaseURL = "", "http://127.0.0.1:1/v1"
	if _, err := checkRoute(context.Background(), local.Apply(core.Config{}), "", local); err == nil || !strings.Contains(err.Error(), "not reachable") {
		t.Fatal(err)
	}
}

func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
}

// Two providers keep their own keys; switching and removing work through
// the same code the manager and `goh providers` use, and a removed
// provider's key goes with it.
func TestProvidersEndToEnd(t *testing.T) {
	isolateHome(t)
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0600); err != nil {
		t.Fatal(err)
	}
	anthropic := core.ProviderSettings{Protocol: "anthropic_messages", Address: "https://api.anthropic.com/v1", Model: "claude-x", Window: 200000}
	gateway := core.ProviderSettings{Protocol: "chat_completions", Address: "https://gateway.example/v1", Model: "gw", Window: 8192}
	if err := saveProvider("anthropic", anthropic, "sk-ant-1", true); err != nil {
		t.Fatal(err)
	}
	if err := saveProvider("gateway", gateway, "gw-key", true); err != nil {
		t.Fatal(err)
	}
	if err := saveProvider("bad", core.ProviderSettings{Protocol: "chat_completions", Address: "ftp://x", Model: "m", Window: 8192}, "", true); err == nil {
		t.Fatal("invalid URL saved")
	}
	var out bytes.Buffer
	if err := runProviders([]string{"list"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "* gateway") || !strings.Contains(out.String(), "  anthropic") || !strings.Contains(out.String(), "saved key") {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := runProviders([]string{"use", "anthropic"}, &out); err != nil || !strings.Contains(out.String(), "New chats use anthropic") {
		t.Fatal(out.String(), err)
	}
	_, _, cfg, _ := userSettings()
	if key, _ := cfg.ProviderKey(); key != "sk-ant-1" {
		t.Fatal("active provider lost its own key")
	}
	if e, _ := cfg.FindProvider("gateway"); keyState(e, cfg) != "saved key" {
		t.Fatal("other provider's key not kept:", keyState(e, cfg))
	}
	out.Reset()
	if err := runProviders([]string{"remove", "anthropic"}, &out); err != nil || !strings.Contains(out.String(), "New chats use gateway") ||
		!strings.Contains(out.String(), "saved key was deleted") {
		t.Fatal(out.String(), err)
	}
	_, _, cfg, _ = userSettings()
	if cfg.Model != "gw" {
		t.Fatal("did not hand over to the remaining provider")
	}
	if key, _ := (core.ProviderEntry{Protocol: anthropic.Protocol, Address: anthropic.Address}).Route(core.Defaults()).ProviderKey(); key != "" {
		t.Fatal("removed provider's key kept")
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"theme": "dark"`) {
		t.Fatal("unrelated settings lost")
	}
	if err := runProviders([]string{"use"}, &out); err == nil {
		t.Fatal("bad usage accepted")
	}
}

// ChatGPT is saved as a provider without a context window, which is looked
// up from the account when chat starts.
func TestChatGPTSavesAsAProvider(t *testing.T) {
	isolateHome(t)
	if err := saveProvider("chatgpt", core.ProviderSettings{Protocol: core.ChatGPTProtocol, Address: "https://api.openai.com/v1", Model: "gpt-x"}, "", true); err != nil {
		t.Fatal(err)
	}
	_, _, cfg, _ := userSettings()
	if cfg.EffectiveProtocol() != core.ChatGPTProtocol || cfg.Model != "gpt-x" || cfg.ContextWindow != 0 {
		t.Fatalf("%+v", cfg)
	}
	if e, _ := cfg.FindProvider("chatgpt"); !e.Active || keyState(e, cfg) != "not signed in" {
		t.Fatal("ChatGPT entry wrong")
	}
}
