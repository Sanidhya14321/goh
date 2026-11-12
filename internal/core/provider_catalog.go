package core

import "strings"

// ProviderPreset is a known provider that /connect can set up from a key
// alone: the route is fixed, the key page is linked, and models come from
// the provider's own list, so no model names are hard-coded here.
type ProviderPreset struct {
	ID          string
	Name        string
	Description string
	Protocol    string
	BaseURL     string
	// KeyURL is where the account owner creates an API key; empty when no
	// key is needed (a local server).
	KeyURL string
	// KeyCheckURL, when set, verifies the key on the provider's host;
	// needed where the model list is public (OpenRouter).
	KeyCheckURL string
	// Window is used when the model list does not report one.
	Window int
	// ModelPrefix is removed from listed IDs (Gemini lists "models/NAME"
	// but its chat endpoint takes NAME).
	ModelPrefix string
	// Skip hides listed models that cannot chat (embeddings, audio, images).
	Skip []string
}

// ProviderCatalog lists the presets in the order /connect shows them.
var ProviderCatalog = []ProviderPreset{
	{ID: "anthropic", Name: "Anthropic (Claude API)", Description: "API key from the Claude Console; billed per token",
		Protocol: "anthropic_messages", BaseURL: "https://api.anthropic.com/v1", KeyURL: "https://platform.claude.com/settings/keys", Window: 200000},
	{ID: "openai", Name: "OpenAI API", Description: "API key from the OpenAI Platform; billed per token",
		Protocol: "openai_responses", BaseURL: "https://api.openai.com/v1", KeyURL: "https://platform.openai.com/api-keys", Window: 128000,
		Skip: []string{"embedding", "tts", "whisper", "dall-e", "moderation", "davinci", "babbage", "image", "audio", "realtime", "transcribe", "search", "sora"}},
	{ID: "openrouter", Name: "OpenRouter", Description: "One key for many model vendors",
		Protocol: "chat_completions", BaseURL: "https://openrouter.ai/api/v1", KeyURL: "https://openrouter.ai/settings/keys", Window: 32768,
		KeyCheckURL: "https://openrouter.ai/api/v1/key"},
	{ID: "gemini", Name: "Google Gemini", Description: "API key from Google AI Studio (OpenAI-compatible endpoint)",
		Protocol: "chat_completions", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", KeyURL: "https://aistudio.google.com/apikey", Window: 128000,
		ModelPrefix: "models/", Skip: []string{"embedding", "aqa", "imagen", "veo", "tts", "image", "audio"}},
	{ID: "groq", Name: "Groq", Description: "API key from GroqCloud",
		Protocol: "chat_completions", BaseURL: "https://api.groq.com/openai/v1", KeyURL: "https://console.groq.com/keys", Window: 32768,
		Skip: []string{"whisper", "tts", "guard", "playai"}},
	{ID: "ollama", Name: "Ollama (local)", Description: "Models running on this computer; no key",
		Protocol: "chat_completions", BaseURL: "http://localhost:11434/v1", Window: 8192},
}

// FindPreset returns the catalog entry with this ID.
func FindPreset(id string) (ProviderPreset, bool) {
	for _, p := range ProviderCatalog {
		if p.ID == id {
			return p, true
		}
	}
	return ProviderPreset{}, false
}

// Apply routes c to the preset. Settings that belong to another provider
// (an api_key_env, a different model) are cleared by SetProviderRoute or
// replaced when the model is chosen.
func (p ProviderPreset) Apply(c Config) Config {
	c.SetProviderRoute(p.Protocol, "", p.BaseURL)
	return c
}

// ChatModel maps a listed ID to the name the chat endpoint takes, and
// reports false for models that cannot chat.
func (p ProviderPreset) ChatModel(id string) (string, bool) {
	lower := strings.ToLower(id)
	for _, s := range p.Skip {
		if strings.Contains(lower, s) {
			return "", false
		}
	}
	return strings.TrimPrefix(id, p.ModelPrefix), true
}

// NeedsKey reports whether the preset needs an API key.
func (p ProviderPreset) NeedsKey() bool { return p.KeyURL != "" }
