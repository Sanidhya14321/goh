package lspclient

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/independent-harness/internal/core"
)

// TestRealClangd exercises a real language server end to end when clangd is
// installed (skipped otherwise): approval-gated start, push diagnostics,
// the post-edit error report and go-to-definition.
func TestRealClangd(t *testing.T) {
	if testing.Short() {
		t.Skip("real server test")
	}
	clangd, err := exec.LookPath("clangd")
	if err != nil {
		t.Skip("clangd not installed")
	}
	project := t.TempDir()
	src := "int add(int a, int b) { return a + b; }\nint main(void) { int x = add(1, 2); char *p = x; return 0; }\n"
	if err := os.WriteFile(filepath.Join(project, "main.c"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	approvals := 0
	m := New(map[string]core.LSPConfig{"clangd": {Command: clangd, Extensions: []string{".c"}}}, project,
		func(context.Context, core.Permission) (core.Decision, error) {
			approvals++
			return core.Decision{Allow: true}, nil
		})
	defer m.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	run := func(op string, line, char int) map[string]any {
		in, _ := json.Marshal(map[string]any{"operation": op, "filePath": "main.c", "line": line, "character": char})
		r := m.Execute(ctx, "lsp", in)
		if r.Error {
			t.Fatalf("%s: %s", op, r.Content)
		}
		var out map[string]any
		json.Unmarshal([]byte(r.Content), &out)
		return out
	}
	// Diagnostics may arrive after the first request; poll briefly.
	var errs []string
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		// "Not published yet" is expected while the server warms up.
		in, _ := json.Marshal(map[string]any{"operation": "diagnostics", "filePath": "main.c", "line": 1, "character": 1})
		m.Execute(ctx, "lsp", in)
		if errs, _, _ = m.EditErrors(ctx, "main.c"); len(errs) > 0 {
			break
		}
	}
	if approvals != 1 {
		t.Fatalf("server start needs exactly one approval, got %d", approvals)
	}
	if len(errs) == 0 || !strings.HasPrefix(errs[0], "2:") {
		t.Fatalf("expected the int-to-pointer error on line 2, got %v", errs)
	}
	def := run("goToDefinition", 2, 27) // the call to add
	b, _ := json.Marshal(def)
	if !strings.Contains(string(b), "main.c") || !strings.Contains(string(b), `"line":0`) {
		t.Fatalf("definition did not resolve to line 1: %s", b)
	}

	// Kill the server: the next request fails without replay, and the one
	// after that restarts it behind a second approval.
	m.mu.Lock()
	m.servers["clangd"].process.cmd.Process.Kill()
	m.mu.Unlock()
	time.Sleep(200 * time.Millisecond)
	in, _ := json.Marshal(map[string]any{"operation": "goToDefinition", "filePath": "main.c", "line": 2, "character": 27})
	for i := 0; i < 3; i++ {
		if r := m.Execute(ctx, "lsp", in); !r.Error {
			break
		}
	}
	if approvals != 2 {
		t.Fatalf("restart needs exactly one more approval, got %d total", approvals)
	}
	if r := m.Execute(ctx, "lsp", in); r.Error {
		t.Fatalf("restarted server unusable: %s", r.Content)
	}
}
