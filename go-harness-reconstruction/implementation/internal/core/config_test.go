package core

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestConfigPrecedenceAndCredentialIndirection(t *testing.T) {
	t.Setenv("HARNESS_ENDPOINT", "http://localhost/environment")
	t.Setenv("HARNESS_MODEL", "environment-model")
	t.Setenv("HARNESS_CONTEXT_WINDOW", "64000")
	t.Setenv("HARNESS_API_KEY_ENV", "FIXTURE_SECRET_NAME")
	p := filepath.Join(t.TempDir(), "config.json")
	if e := os.WriteFile(p, []byte(`{"endpoint":"http://localhost/config","model":"config-model","context_window":32000,"output_budget":1000}`), 0600); e != nil {
		t.Fatal(e)
	}
	c, e := LoadConfig(p)
	if e != nil {
		t.Fatal(e)
	}
	if c.Endpoint != "http://localhost/environment" || c.Model != "environment-model" || c.ContextWindow != 64000 || c.OutputBudget != 1000 || c.APIKeyEnv != "FIXTURE_SECRET_NAME" {
		t.Fatalf("precedence: %+v", c)
	}
	if e = c.Validate(); e != nil {
		t.Fatal(e)
	}
	t.Setenv("HARNESS_CONTEXT_WINDOW", "10000 invalid")
	if _, e = LoadConfig(p); e == nil {
		t.Fatal("accepted malformed integer")
	}
}
func TestConfigurationRejectsUnsupportedProfiles(t *testing.T) {
	c := Defaults()
	c.Endpoint = "https://localhost/v1/chat/completions"
	c.Model = "fixture"
	c.ContextWindow = 32000
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*Config){func(c *Config) { c.Endpoint = "file:///tmp/model" }, func(c *Config) { c.Endpoint = "https://secret:secret@localhost" }, func(c *Config) { c.ContextWindow = 4000 }, func(c *Config) { c.MaxTurns = 0 }, func(c *Config) { c.OutputParameter = "unknown" }, func(c *Config) { c.Model = "" }} {
		bad := c
		mutate(&bad)
		if bad.Validate() == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestProviderBaseURLAndOverrides(t *testing.T) {
	c := Config{
		Endpoint: "https://example.test/exact?version=1",
		Profiles: map[string]ModelProfile{
			"gateway": {BaseURL: str("https://example.test/gateway/v1/")},
		},
	}
	assertURL := func(config Config, want string) {
		t.Helper()
		got, err := config.ProviderEndpoint()
		if err != nil || got != want {
			t.Fatalf("endpoint = %q, %v; want %q", got, err, want)
		}
	}
	t.Setenv("HARNESS_BASE_URL", "")
	t.Setenv("HARNESS_ENDPOINT", "")
	t.Setenv("HARNESS_PROTOCOL", "")
	assertURL(c, "https://example.test/exact?version=1")
	c.overlayProfile(c.Profiles["gateway"])
	assertURL(c, "https://example.test/gateway/v1/chat/completions")
	c.Protocol = "anthropic_messages"
	assertURL(c, "https://example.test/gateway/v1/messages")

	t.Setenv("HARNESS_BASE_URL", "https://example.test/override")
	resolved, err := c.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	assertURL(resolved, "https://example.test/override/chat/completions")
	t.Setenv("HARNESS_BASE_URL", "")
	t.Setenv("HARNESS_ENDPOINT", "https://example.test/environment?version=2")
	resolved, err = c.Resolve("gateway", "")
	if err != nil {
		t.Fatal(err)
	}
	assertURL(resolved, "https://example.test/environment?version=2")
	for _, base := range []string{"https://user:secret@example.test/v1", "https://example.test/v1?secret=1", "https://example.test/v1#secret", "https://:443/v1", "/v1"} {
		if _, err := (Config{BaseURL: base}).ProviderEndpoint(); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid base URL produced error %v", err)
		}
	}
	if _, err := (Config{BaseURL: c.BaseURL, Endpoint: "https://example.test/exact"}).ProviderEndpoint(); err == nil {
		t.Fatal("accepted two addresses")
	}
}

func TestConfigRejectsSpecialAndOversizedFiles(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "config.fifo")
	if e := syscall.Mkfifo(fifo, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadConfig(fifo); e == nil {
		t.Fatal("FIFO config accepted")
	}
	big := filepath.Join(dir, "big.json")
	if e := os.WriteFile(big, []byte(strings.Repeat(" ", (1<<20)+1)), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadConfig(big); e == nil {
		t.Fatal("oversized config accepted")
	}
}
