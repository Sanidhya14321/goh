package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func isolateUserConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(path)
}

func TestGuidanceFilesLoaded(t *testing.T) {
	dir := isolateUserConfig(t)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Always answer in British English."), 0o600)
	a, _ := testAgent(t)
	os.WriteFile(filepath.Join(a.Store.Project, "CLAUDE.md"), []byte("Use make test."), 0o600)
	b, err := New(a.Config, a.Store)
	if err != nil {
		t.Fatal(err)
	}
	system := b.History[0].Content
	// The user's own guidance is trusted and stays in the system message;
	// repository guidance (review R20) is a separate untrusted user message.
	if !strings.Contains(system, "British English") || strings.Contains(system, "Use make test.") || len(b.GuidanceFiles) != 2 {
		t.Fatal("trust boundary wrong:", system, b.GuidanceFiles)
	}
	sent := b.modelHistory()
	if len(sent) < 2 || sent[1].Role != "user" || !strings.Contains(sent[1].Content, "untrusted repository content") || !strings.Contains(sent[1].Content, "Use make test.") {
		t.Fatalf("project guidance not sent as untrusted data: %+v", sent)
	}
	// AGENTS.md takes precedence over CLAUDE.md.
	os.WriteFile(filepath.Join(a.Store.Project, "AGENTS.md"), []byte("Use go test."), 0o600)
	c, _ := New(a.Config, a.Store)
	if g := c.modelHistory()[1].Content; !strings.Contains(g, "Use go test.") || strings.Contains(g, "Use make test.") {
		t.Fatal("AGENTS.md should replace CLAUDE.md")
	}
}

func TestOversizedUserGuidanceSkipped(t *testing.T) {
	dir := isolateUserConfig(t)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(strings.Repeat("x", 65<<10)), 0o600)
	a, _ := testAgent(t)
	if len(a.GuidanceFiles) != 0 || strings.Contains(a.History[0].Content, "User guidance") {
		t.Fatal("oversized guidance loaded")
	}
}
