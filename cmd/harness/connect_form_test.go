package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// stubCheck accepts exactly the keys in good, as checkRoute would.
func stubCheck(t *testing.T, good ...string) {
	t.Helper()
	old := checkRoute
	t.Cleanup(func() { checkRoute = old })
	checkRoute = func(_ context.Context, _ core.Config, key string, _ core.ProviderPreset) ([]provider.ModelInfo, error) {
		for _, g := range good {
			if key == g {
				return []provider.ModelInfo{{ID: "m", Window: 9000}}, nil
			}
		}
		return nil, keyRejection("rejected")
	}
}

func writeSettings(t *testing.T, body string) {
	t.Helper()
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(path), 0o700)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Editing a custom provider's URL with the key left blank keeps its saved
// key: it moves to the new address, and the old copy is deleted.
func TestEditCustomProviderMovesSavedKey(t *testing.T) {
	isolateHome(t)
	writeSettings(t, `{}`)
	gw := core.ProviderSettings{Protocol: "chat_completions", Address: "https://gw.example/v1", Model: "m", Window: 9000}
	if err := saveProvider("gw", gw, "gw-key", true); err != nil {
		t.Fatal(err)
	}
	stubCheck(t, "gw-key")
	e, cfg, err := findEntry("gw")
	if err != nil {
		t.Fatal(err)
	}
	old := e.Route(cfg)
	form, err := checkCustomForm(context.Background(), []string{"gw", "https://gw2.example/v1", "chat_completions", "", "m", "9000", ""}, &e, &old, cfg)
	if err != nil {
		t.Fatalf("blank key on a new URL was refused: %v", err)
	}
	if form.key != "gw-key" || form.moved == nil {
		t.Fatalf("saved key not carried over: %+v", form)
	}
	if err := saveProviderAs("gw", form.name, form.settings, form.key, true); err != nil {
		t.Fatal(err)
	}
	if !keyInUse(*form.moved) {
		form.moved.SaveProviderKey("")
	}
	_, _, cfg, _ = userSettings()
	if key, _ := cfg.ProviderKey(); key != "gw-key" {
		t.Fatalf("key at the new address = %q", key)
	}
	if key, _ := old.ProviderKey(); key != "" {
		t.Fatal("old address kept a copy of the key")
	}
}

// A provider whose key comes from a variable this process lacks can still
// be saved with a typed model, with a warning instead of a refusal.
func TestCustomProviderWithUnsetKeyVariable(t *testing.T) {
	isolateHome(t)
	writeSettings(t, `{}`)
	stubCheck(t, "the-real-key")
	t.Setenv("MY_GW_KEY", "")
	_, _, cfg, _ := userSettings()
	form, err := checkCustomForm(context.Background(), []string{"gw", "https://gw.example/v1", "chat_completions", "", "m", "", "MY_GW_KEY"}, nil, nil, cfg)
	if err != nil || !strings.Contains(form.warning, "MY_GW_KEY is not set") {
		t.Fatalf("form %+v err %v", form, err)
	}
	if _, err := checkCustomForm(context.Background(), []string{"gw", "https://gw.example/v1", "chat_completions", "", "", "", "MY_GW_KEY"}, nil, nil, cfg); err == nil || !strings.Contains(err.Error(), "MY_GW_KEY is not set") {
		t.Fatalf("blank model with unset variable: %v", err)
	}
	// A wrong key typed into the form is still refused.
	if _, err := checkCustomForm(context.Background(), []string{"gw", "https://gw.example/v1", "chat_completions", "wrong", "m", "", ""}, nil, nil, cfg); !errors.Is(err, provider.ErrKeyRejected) {
		t.Fatalf("wrong key: %v", err)
	}
}
