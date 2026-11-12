package core

import (
	"strings"
	"testing"
)

func TestProviderCatalogRoutesValidate(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range ProviderCatalog {
		if seen[p.ID] || p.Name == "" || p.Window <= 0 {
			t.Fatalf("bad preset %+v", p)
		}
		seen[p.ID] = true
		base := Defaults()
		base.Model, base.ContextWindow, base.APIKeyEnv = "m", p.Window, "OLD_GATEWAY_KEY"
		c := p.Apply(base)
		if err := c.Validate(); err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		if c.APIKeyEnv != "" {
			t.Fatalf("%s kept another provider's key variable", p.ID)
		}
		if p.NeedsKey() && (len(p.KeyURL) < 8 || p.KeyURL[:8] != "https://") {
			t.Fatalf("%s key page is not HTTPS", p.ID)
		}
	}
	if _, ok := FindPreset("anthropic"); !ok {
		t.Fatal("anthropic missing")
	}
}

func TestPresetChatModels(t *testing.T) {
	gemini, _ := FindPreset("gemini")
	if id, ok := gemini.ChatModel("models/gemini-x-pro"); !ok || id != "gemini-x-pro" {
		t.Fatal(id, ok)
	}
	if _, ok := gemini.ChatModel("models/text-embedding-004"); ok {
		t.Fatal("embedding model offered for chat")
	}
	openai, _ := FindPreset("openai")
	for _, id := range []string{"whisper-1", "tts-1-hd", "dall-e-3", "omni-moderation-latest", "gpt-x-realtime"} {
		if _, ok := openai.ChatModel(id); ok {
			t.Fatal(id, "offered for chat")
		}
	}
	if _, ok := openai.ChatModel("gpt-x"); !ok {
		t.Fatal("chat model hidden")
	}
}

func TestKeyCheckURLsStayOnTheProviderHost(t *testing.T) {
	for _, p := range ProviderCatalog {
		if p.KeyCheckURL == "" {
			continue
		}
		base, check := p.BaseURL[:len("https://")], p.KeyCheckURL[:len("https://")]
		if base != check || !strings.HasPrefix(p.KeyCheckURL, p.BaseURL[:strings.Index(p.BaseURL[8:], "/")+8]) {
			t.Fatalf("%s checks keys off its host: %s", p.ID, p.KeyCheckURL)
		}
	}
}
