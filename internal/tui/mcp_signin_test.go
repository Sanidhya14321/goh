package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Sanidhya14321/goh/internal/core"
)

// drain feeds the model's background messages back to it until the
// background work reports done.
func drain(t *testing.T, m *model) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg := <-m.events:
			m.Update(msg)
			if _, done := msg.(doneMsg); done {
				return
			}
		case <-deadline:
			t.Fatal("background work did not finish")
		}
	}
}

func TestMCPLoginFromTheChat(t *testing.T) {
	m := fuzzModel(t)
	m.a.Config.MCP = map[string]core.MCPConfig{"docs": {URL: "https://mcp.example/mcp"}, "local": {Command: "server"}}
	var opened []string
	savedOpen, savedLogin, savedLogout := OpenBrowser, mcpLogin, mcpLogout
	t.Cleanup(func() { OpenBrowser, mcpLogin, mcpLogout = savedOpen, savedLogin, savedLogout })
	OpenBrowser = func(_ context.Context, address string) bool { opened = append(opened, address); return false }
	mcpLogin = func(_ context.Context, name string, cfg core.MCPConfig, open func(string) error) (int, error) {
		if name != "docs" || cfg.URL != "https://mcp.example/mcp" {
			return 0, errors.New("wrong server")
		}
		open("https://auth.example/authorize?state=s")
		return 3, nil
	}
	send := func(text string) {
		m.input, m.inputCursor = text, len(text)
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	send("/mcp login docs")
	if !m.active || !strings.Contains(m.status, "Signing in to docs") {
		t.Fatal("sign-in not running:", m.status)
	}
	drain(t, m)
	if len(opened) != 1 || !strings.Contains(m.log, "https://auth.example/authorize?state=s") || !strings.Contains(m.log, "✓ Signed in to docs (3 tools)") || m.active {
		t.Fatalf("opened %v, active %v:\n%s", opened, m.active, m.log)
	}

	// Refusals stay in the chat without starting anything.
	for input, want := range map[string]string{
		"/mcp login":       "remote servers: docs",
		"/mcp login local": "only remote servers sign in",
		"/mcp login nope":  "No MCP server named nope",
	} {
		send(input)
		if m.active || !strings.Contains(m.log, want) {
			t.Fatalf("%s: want %q", input, want)
		}
	}

	mcpLogout = func(core.MCPConfig) (bool, error) { return true, nil }
	send("/mcp logout docs")
	if !strings.Contains(m.log, "Signed out of docs.") {
		t.Fatal("logout not reported")
	}

	// Esc cancels a sign-in waiting for the browser.
	mcpLogin = func(ctx context.Context, _ string, _ core.MCPConfig, _ func(string) error) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	}
	send("/mcp login docs")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m)
	if !strings.Contains(m.log, "Sign-in to docs cancelled.") || m.active {
		t.Fatalf("cancel not reported:\n%s", m.log[max(0, len(m.log)-300):])
	}
}

// A server that needs sign-in points to /mcp login in the chat.
func TestDegradedSignInSuggestsChatLogin(t *testing.T) {
	line := degradedLine(map[string]string{"kind": "mcp", "name": "docs", "error": "MCP server docs requires sign-in: run `goh mcp login docs`"})
	if !strings.Contains(line, "Sign in with /mcp login docs") {
		t.Fatal(line)
	}
}
