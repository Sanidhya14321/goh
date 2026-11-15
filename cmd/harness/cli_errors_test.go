package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

// A fresh user, or a script with a typo, gets a message that says what to
// do instead of an internal error.
func TestCLIErrorsSayWhatToDo(t *testing.T) {
	isolateHome(t)
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{"protocol":"chat_completions","base_url":"http://127.0.0.1:1/v1","model":"m","context_window":8192}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"run", "hello"}, "no model provider is connected yet"},
		{[]string{"run", "--config", cfg}, "usage: goh run MESSAGE"},
		{[]string{"run", "--config", cfg, "--project", filepath.Join(t.TempDir(), "missing"), "hi"}, "does not exist"},
		{[]string{"resume"}, "usage: goh resume SESSION_ID"},
		{[]string{"resume", "--config", cfg}, "usage: goh resume SESSION_ID"},
		{[]string{"resume", "--config", cfg, "nope", "--prompt", "hi"}, "no saved session"},
		{[]string{"compact"}, "usage: goh compact SESSION_ID"},
	} {
		err := runCommand(c.args, true, "", "")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("goh %s: %v, want %q", strings.Join(c.args, " "), err, c.want)
		}
	}
	if _, err := (core.Config{}).ProviderEndpoint(); !errors.Is(err, core.ErrNoProvider) {
		t.Fatal(err)
	}
	if _, err := session.Open(t.TempDir(), "20260101T000000-0123456789abcdef01234567"); !errors.Is(err, session.ErrNotFound) {
		t.Fatal(err)
	}
}
