package main

import (
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestChatGPTDefaultNeverOverridesExplicitProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no saved sign-in
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir()) // Windows home
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	explicit := core.Config{Model: "local-model", Endpoint: "http://127.0.0.1:9/v1/chat/completions"}
	if got := withChatGPTDefault(explicit); got.Model != "local-model" || got.Endpoint != explicit.Endpoint || got.EffectiveProtocol() == core.ChatGPTProtocol {
		t.Fatalf("explicit provider replaced: %+v", got)
	}
	if got := withChatGPTDefault(core.Config{}); got.EffectiveProtocol() == core.ChatGPTProtocol {
		t.Fatal("ChatGPT chosen without a saved sign-in")
	}
}
