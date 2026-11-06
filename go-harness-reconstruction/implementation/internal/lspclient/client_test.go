package lspclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/testutil"
)

func TestLSPFixtureHelper(t *testing.T) {
	if os.Getenv("LSP_FIXTURE_HELPER") != "1" {
		return
	}
	testutil.RunLSP()
	os.Exit(0)
}
func setup(t *testing.T, mode string) (*Manager, string, string) {
	t.Helper()
	project := t.TempDir()
	path := filepath.Join(project, "fixture.lsptest")
	if e := os.WriteFile(path, []byte("a😀bc\nsecond\n"), 0600); e != nil {
		t.Fatal(e)
	}
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	log := filepath.Join(t.TempDir(), "messages.jsonl")
	m := New(map[string]core.LSPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestLSPFixtureHelper$"}, Extensions: []string{".lsptest"}, Env: map[string]string{"LSP_FIXTURE_HELPER": "1", "LSP_FIXTURE_MODE": mode, "LSP_FIXTURE_LOG": log}}}, project, func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: true}, nil })
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	return m, path, log
}
func argumentsFor(op string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"operation": op, "filePath": "fixture.lsptest", "line": 1, "character": 3, "query": "Fixture"})
	return b
}
func TestLSPInitializeQueriesUTF16SyncAndDiagnostics(t *testing.T) {
	m, path, log := setup(t, "")
	starts := 0
	m.OnStarted = func(name string) error { starts++; return nil }
	for _, op := range []string{"hover", "goToDefinition", "findReferences", "documentSymbol", "workspaceSymbol", "goToImplementation", "diagnostics"} {
		r := m.Execute(context.Background(), "lsp", argumentsFor(op))
		if r.Error {
			t.Fatal(op, r.Content)
		}
		if op == "hover" {
			if !strings.Contains(r.Content, `"character":3`) {
				t.Fatal("rune->UTF16 position incorrect", r.Content)
			}
		}
	}
	if starts != 1 {
		t.Fatal("server restarted", starts)
	}
	os.WriteFile(path, []byte("changed\n"), 0600)
	r := m.Execute(context.Background(), "lsp", argumentsFor("hover"))
	if r.Error {
		t.Fatal(r.Content)
	}
	m.Close()
	b, e := os.ReadFile(log)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(b, []byte("textDocument/didChange")) || !bytes.Contains(b, []byte("shutdown")) {
		t.Fatal("sync/shutdown missing")
	}
}
func TestLSPAdmissionFailureAndBounds(t *testing.T) {
	m, path, _ := setup(t, "")
	m.approve = func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{}, nil }
	if r := m.Execute(context.Background(), "lsp", argumentsFor("hover")); !r.Error || !strings.Contains(r.Content, "needs_permission") {
		t.Fatal(r)
	}
	if len(m.servers) != 0 {
		t.Fatal("denied server started")
	}
	fifo := filepath.Join(filepath.Dir(path), "fifo.lsptest")
	if e := syscall.Mkfifo(fifo, 0600); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"../outside.lsptest", "fifo.lsptest"} {
		raw := json.RawMessage(fmt.Sprintf(`{"operation":"hover","filePath":%q,"line":1,"character":1}`, name))
		if e := m.Validate("lsp", raw); e == nil {
			t.Fatal("unsafe path accepted", name)
		}
	}
	os.Symlink(path, filepath.Join(filepath.Dir(path), "link.lsptest"))
	if e := m.Validate("lsp", json.RawMessage(`{"operation":"hover","filePath":"link.lsptest","line":1,"character":1}`)); e == nil {
		t.Fatal("symlink accepted")
	}
	if e := m.Validate("lsp", json.RawMessage(`{"operation":"hover","filePath":"fixture.lsptest","line":1,"character":1,"query":null}`)); e == nil {
		t.Fatal("null query accepted")
	}
}
func TestLSPDisconnectCancelMalformedAndLarge(t *testing.T) {
	for _, mode := range []string{"disconnect", "hang", "malformed", "bad-initialize", "bad-encoding", "large", "request-edit"} {
		t.Run(mode, func(t *testing.T) {
			m, _, log := setup(t, mode)
			ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
			defer cancel()
			r := m.Execute(ctx, "lsp", argumentsFor("hover"))
			if mode == "request-edit" {
				if r.Error {
					t.Fatal(r)
				}
				m.Close()
				b, _ := os.ReadFile(log)
				if !strings.Contains(string(b), "-32601") {
					t.Fatal("server-requested effect was not refused")
				}
			} else if !r.Error {
				t.Fatal("failure accepted", mode, r)
			}
			if len(r.Content) > maxResult {
				t.Fatal("unbounded result")
			}
			if mode == "disconnect" {
				r = m.Execute(context.Background(), "lsp", argumentsFor("hover"))
				if !r.Error {
					t.Fatal("disconnected server replayed")
				}
			}
		})
	}
}
func TestLSPFramingAndPositions(t *testing.T) {
	for _, s := range []string{"Content-Length: 2\r\n\r\n{}", "Content-Length: 2\r\nContent-Type: application/vscode-jsonrpc\r\n\r\n{}"} {
		b, e := readFrame(bufio.NewReader(strings.NewReader(s)))
		if e != nil || string(b) != "{}" {
			t.Fatal(string(b), e)
		}
	}
	for _, s := range []string{"Content-Length: 2\n\n{}", "Content-Length: 2\r\nContent-Length: 2\r\n\r\n{}", "Content-Length: 99999999\r\n\r\n", strings.Repeat("X", 9000)} {
		if _, e := readFrame(bufio.NewReader(strings.NewReader(s))); e == nil {
			t.Fatal("malformed frame accepted")
		}
	}
	position, e := utf16Position("a😀bc\r\nnext", 1, 3)
	if e != nil || position["character"] != 3 {
		t.Fatal(position, e)
	}
	if _, e = utf16Position("a", 1, 3); e == nil {
		t.Fatal("invalid column accepted")
	}
}
func FuzzLSPFrame(f *testing.F) {
	f.Add("Content-Length: 2\r\n\r\n{}")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 1<<16 {
			t.Skip()
		}
		readFrame(bufio.NewReader(strings.NewReader(s)))
	})
}

func TestLSPLanguageAndLongestSuffixSelection(t *testing.T) {
	for _, tc := range [][3]string{{"file.ts", "", "typescript"}, {"file.tsx", "", "typescriptreact"}, {"file.py", "", "python"}, {"file.unknown", "custom-language", "custom-language"}} {
		if got := languageID(tc[0], tc[1]); got != tc[2] {
			t.Fatal(got, tc)
		}
	}
	m := New(map[string]core.LSPConfig{"generic": {Command: "fixture", Extensions: []string{".go"}}, "tests": {Command: "fixture", Extensions: []string{".test.go"}}}, t.TempDir(), nil)
	defer m.Close()
	name, _, e := m.selectServer("unit.test.go")
	if e != nil || name != "tests" {
		t.Fatal(name, e)
	}
}
