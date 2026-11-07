package core

import "testing"

func TestSavedProviderKeyScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GOH_TEST_KEY", "")
	cfg := Config{BaseURL: "https://gateway.example/v1", APIKeyEnv: "GOH_TEST_KEY"}
	check := func(config Config, want string) {
		t.Helper()
		key, err := config.ProviderKey()
		if err != nil || key != want {
			t.Fatal("unexpected credential lookup result", err)
		}
	}
	if err := cfg.SaveProviderKey("test-only"); err != nil {
		t.Fatal(err)
	}
	check(cfg, "test-only")
	check(Config{BaseURL: "https://other.example/v1", APIKeyEnv: cfg.APIKeyEnv}, "")
	check(Config{BaseURL: cfg.BaseURL, APIKeyEnv: cfg.APIKeyEnv, Protocol: "anthropic_messages"}, "")
	t.Setenv("GOH_TEST_KEY", "override")
	check(cfg, "override")
	t.Setenv("GOH_TEST_KEY", "")
	if err := cfg.SaveProviderKey(""); err != nil {
		t.Fatal(err)
	}
	check(cfg, "")
	if err := (Config{BaseURL: "http://gateway.example/v1"}).SaveProviderKey("test-only"); err == nil {
		t.Fatal("accepted a key for remote plaintext HTTP")
	}
}

func TestOfficialEnvironmentKeysStayOnOfficialOrigins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "official-only")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-only")
	t.Setenv("GATEWAY_KEY", "explicit-gateway")
	for _, item := range []struct {
		cfg  Config
		key  string
		fail bool
	}{
		{Config{BaseURL: "https://api.openai.com/v1"}, "official-only", false},
		{Config{BaseURL: "https://api.anthropic.com/v1", Protocol: "anthropic_messages"}, "anthropic-only", false},
		{Config{BaseURL: "https://gateway.example/v1", APIKeyEnv: "OPENAI_API_KEY"}, "", false},
		{Config{BaseURL: "https://gateway.example/v1"}, "", false},
		{Config{BaseURL: "https://gateway.example/v1", APIKeyEnv: "GATEWAY_KEY"}, "explicit-gateway", false},
		{Config{BaseURL: "http://gateway.example/v1", APIKeyEnv: "GATEWAY_KEY"}, "", true},
		{Config{BaseURL: "http://localhost:8080/v1", APIKeyEnv: "GATEWAY_KEY"}, "explicit-gateway", false},
	} {
		key, err := item.cfg.ProviderKey()
		if key != item.key || (err != nil) != item.fail {
			t.Errorf("unexpected credential routing for %s (%v)", item.cfg.BaseURL, err)
		}
	}
}
