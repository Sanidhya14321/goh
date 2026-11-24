package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Sanidhya14321/goh/internal/mcpclient"
)

// OpenBrowser shows a page in the user's browser, best effort, and reports
// whether one opened. The entry point sets it; tests leave it nil.
var OpenBrowser func(ctx context.Context, address string) bool

// mcpSignInMsg is a line of /mcp login progress for the transcript.
type mcpSignInMsg struct{ text string }

// mcpLogin and mcpLogout are the sign-in steps; tests replace them.
var (
	mcpLogin  = mcpclient.Login
	mcpLogout = mcpclient.Logout
)

// mcpSignIn runs /mcp login NAME or /mcp logout NAME. Signing in waits for
// the browser, so it runs in the background like a prompt and Esc cancels
// it. The sign-in is used from the next run on.
func (m *model) mcpSignIn(action string, args []string) tea.Cmd {
	var remote []string
	for name, cfg := range m.a.Config.MCP {
		if cfg.URL != "" {
			remote = append(remote, name)
		}
	}
	sort.Strings(remote)
	if len(args) != 1 {
		m.log += "\nUsage: /mcp " + action + " NAME"
		if len(remote) > 0 {
			m.log += " (remote servers: " + Safe(strings.Join(remote, ", ")) + ")"
		}
		m.log += "\n"
		return nil
	}
	name := args[0]
	cfg, ok := m.a.Config.MCP[name]
	switch {
	case !ok:
		m.log += "\nNo MCP server named " + Safe(name) + " in the configuration.\n"
		return nil
	case cfg.URL == "":
		m.log += "\nMCP server " + Safe(name) + " is local (a command); only remote servers sign in.\n"
		return nil
	}
	if action == "logout" {
		removed, err := mcpLogout(cfg)
		switch {
		case err != nil:
			m.log += "\n[error] " + Safe(err.Error()) + "\n"
		case removed:
			m.log += "\nSigned out of " + Safe(name) + ".\n"
		default:
			m.log += "\nNot signed in to " + Safe(name) + ".\n"
		}
		return nil
	}
	m.active, m.status = true, "Signing in to "+Safe(name)+" in the browser (Esc cancels)"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	m.cancel = cancel
	done := make(chan struct{})
	m.runDone = done
	send := func(msg tea.Msg) bool {
		select {
		case m.events <- msg:
			return true
		case <-m.closed:
			return false
		}
	}
	go func() {
		defer close(done)
		defer cancel()
		open := func(address string) error {
			opened := OpenBrowser != nil && OpenBrowser(ctx, address)
			line := "Sign in to " + name + " in the browser that opened. If none did, visit:\n" + address
			if !opened {
				line = "Sign in to " + name + ": visit\n" + address
			}
			send(mcpSignInMsg{line})
			return nil
		}
		tools, err := mcpLogin(ctx, name, cfg, open)
		status := "completed"
		text := fmt.Sprintf("✓ Signed in to %s (%s). New runs use it.", name, plural(tools, "tool", "tools"))
		if err != nil {
			status, text = "failed", "[error] "+err.Error()
			if ctx.Err() != nil {
				status, text = "cancelled", "Sign-in to "+name+" cancelled."
			}
		}
		if send(mcpSignInMsg{text}) {
			send(doneMsg{status, nil})
		}
	}()
	return m.wait()
}
