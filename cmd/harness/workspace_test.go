package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The home folder holds goh's saved keys, so it cannot be a project. The
// refusal is a named error the launcher turns into "choose a project
// folder" rather than "add a provider", which could never fix it.
func TestHomeFolderIsRefusedWithAFolderFix(t *testing.T) {
	isolateHome(t)
	home, _ := os.UserHomeDir()
	if err := core.CheckProjectCredentialBoundary(home); !errors.Is(err, core.ErrProjectHoldsKeys) {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "app")
	os.Mkdir(project, 0700)
	if err := core.CheckProjectCredentialBoundary(project); err != nil {
		t.Fatal(err)
	}
	if displayPath(home) != "your home folder (~)" || displayPath(filepath.Join(home, "code")) != filepath.Join("~", "code") {
		t.Fatal(displayPath(home), displayPath(filepath.Join(home, "code")))
	}
}

// /models on a chat started from flags lists that chat's route, not the
// saved provider, and does not save the choice.
func TestSameRoute(t *testing.T) {
	saved := core.Config{Protocol: "chat_completions", BaseURL: "https://api.groq.com/openai/v1/"}
	if !sameRoute(core.Config{Protocol: "chat_completions", BaseURL: "https://api.groq.com/openai/v1", Model: "x"}, saved) {
		t.Fatal("same provider with another model")
	}
	if sameRoute(core.Config{Protocol: "chat_completions", Endpoint: "http://127.0.0.1:1/v1/chat/completions"}, saved) {
		t.Fatal("flag route treated as saved")
	}
}

// Groq lists speech and safety models next to chat models; only chat
// models are offered.
func TestGroqHidesNonChatModels(t *testing.T) {
	p, ok := core.PresetForRoute("chat_completions", "https://api.groq.com/openai/v1")
	if !ok {
		t.Fatal("groq preset not found")
	}
	for _, id := range []string{"whisper-large-v3", "canopylabs/orpheus-v1-english", "meta-llama/llama-prompt-guard-2-22m"} {
		if _, ok := p.ChatModel(id); ok {
			t.Fatal(id)
		}
	}
	if _, ok := p.ChatModel("openai/gpt-oss-120b"); !ok {
		t.Fatal("chat model hidden")
	}
}

func TestWorkspacesCommand(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	if err := core.OpenWorkspace(dir); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runWorkspaces(nil, &out); err != nil || !strings.Contains(out.String(), filepath.Base(dir)) {
		t.Fatal(out.String(), err)
	}
	out.Reset()
	if err := runWorkspaces([]string{"forget", dir}, &out); err != nil || core.IsTrustedWorkspace(dir) {
		t.Fatal(out.String(), err)
	}
	if err := runWorkspaces([]string{"forget"}, &out); err == nil {
		t.Fatal("usage accepted")
	}
}
