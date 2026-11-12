package core

import (
	"strings"
	"testing"
)

func mustDecode(t *testing.T, raw string) Config {
	t.Helper()
	c, err := DecodeConfig(raw)
	if err != nil {
		t.Fatal(err, raw)
	}
	return c
}

// Several providers can be saved, switched between and removed; the active
// one is the top-level route, and unrelated settings survive every edit.
func TestProviderRegistry(t *testing.T) {
	raw := `{"base_url":"https://gateway.example/v1","model":"gw-model","context_window":8192,"theme":"dark",
		"profiles":{"fast":{"model":"gw-small"}}}`
	c := mustDecode(t, raw)
	list := c.Providers()
	if len(list) != 1 || !list[0].Unsaved || !list[0].Active || list[0].Name != UnsavedProviderName || list[0].Title() != "Custom" {
		t.Fatalf("hand-configured provider not listed: %+v", list)
	}

	raw, err := SaveProvider(raw, "gateway", ProviderSettings{Protocol: "chat_completions", Address: "https://gateway.example/v1", Model: "gw-model", Window: 8192}, true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = SaveProvider(raw, "anthropic", ProviderSettings{Protocol: "anthropic_messages", Address: "https://api.anthropic.com/v1", Model: "claude-x", Window: 200000}, true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = SaveProvider(raw, "chatgpt", ProviderSettings{Protocol: ChatGPTProtocol, Address: chatGPTResource, Model: "gpt-x"}, false)
	if err != nil {
		t.Fatal(err)
	}
	c = mustDecode(t, raw)
	list = c.Providers()
	names := []string{}
	for _, e := range list {
		names = append(names, e.Name)
		if e.Active != (e.Name == "anthropic") {
			t.Fatalf("wrong active provider: %+v", list)
		}
	}
	if strings.Join(names, ",") != "anthropic,chatgpt,gateway" {
		t.Fatal("model-only profile listed as a provider, or provider missing:", names)
	}
	if c.EffectiveProtocol() != "anthropic_messages" || c.Model != "claude-x" || c.ContextWindow != 200000 {
		t.Fatal("top level does not follow the active provider")
	}
	if e, _ := c.FindProvider("anthropic"); e.Title() != "Anthropic (Claude API)" {
		t.Fatal(e.Title())
	}

	// Switching to ChatGPT drops the previous provider's window, so the
	// account's own window is looked up.
	raw, err = ActivateProvider(raw, "chatgpt")
	if err != nil {
		t.Fatal(err)
	}
	c = mustDecode(t, raw)
	if c.EffectiveProtocol() != ChatGPTProtocol || c.ContextWindow != 0 || c.Model != "gpt-x" {
		t.Fatalf("%+v", c)
	}
	if _, err := c.ProviderEndpoint(); err != nil {
		t.Fatal(err)
	}
	if e, _ := c.FindProvider("chatgpt"); !e.Active || e.Title() != "ChatGPT" {
		t.Fatal("ChatGPT not active")
	}
	if _, err := ActivateProvider(raw, "fast"); err == nil {
		t.Fatal("model-only profile activated as a provider")
	}

	// Removing the active provider moves to the one named.
	raw, err = RemoveProvider(raw, "chatgpt", true, "gateway")
	if err != nil {
		t.Fatal(err)
	}
	c = mustDecode(t, raw)
	if c.BaseURL != "https://gateway.example/v1" || c.Model != "gw-model" || len(c.Providers()) != 2 {
		t.Fatalf("%+v", c.Providers())
	}
	if !strings.Contains(raw, `"theme": "dark"`) || c.Profiles["fast"].Model == nil {
		t.Fatal("unrelated settings lost")
	}
	// Removing a provider that is not active leaves the top level alone.
	raw, err = RemoveProvider(raw, "anthropic", false, "")
	if err != nil || mustDecode(t, raw).Model != "gw-model" {
		t.Fatal(err)
	}
	// Removing the last one clears the route.
	raw, err = RemoveProvider(raw, "gateway", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if c = mustDecode(t, raw); c.BaseURL != "" || c.Model != "" || len(c.Providers()) != 0 {
		t.Fatalf("%+v", c)
	}
}

func TestProviderRegistryRefusals(t *testing.T) {
	raw := `{"profiles":{"gw":{"base_url":"https://gateway.example/v1","model":"m"}},"agents":{"rev":{"description":"d","instructions":"i","mode":"plan","profile":"gw"}}}`
	if _, err := RemoveProvider(raw, "gw", false, ""); err == nil || !strings.Contains(err.Error(), "agent") {
		t.Fatal("removed a provider an agent uses:", err)
	}
	if _, err := SaveProvider(`{"model":"x", // note
	}`, "a", ProviderSettings{Protocol: "chat_completions", Address: "https://a.example/v1"}, true); err == nil {
		t.Fatal("rewrote a settings file with comments")
	}
	for _, name := range []string{"", "has space", UnsavedProviderName} {
		if _, err := SaveProvider(`{}`, name, ProviderSettings{Protocol: "chat_completions", Address: "https://a.example/v1"}, true); err == nil {
			t.Fatalf("name %q accepted", name)
		}
	}
	c := mustDecode(t, raw)
	if c.SuggestProviderName("gw") != "gw-2" || c.SuggestProviderName("My Gateway!") != "my-gateway" {
		t.Fatal(c.SuggestProviderName("gw"), c.SuggestProviderName("My Gateway!"))
	}
}
