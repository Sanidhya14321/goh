package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Version is shown in the footer; set by the command at startup.
var Version string

type spinMsg struct{}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

// humanTokens formats a token count compactly ("" for zero).
func humanTokens(n int) string {
	switch {
	case n <= 0:
		return ""
	case n < 1000:
		return fmt.Sprintf("%d tokens", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.1fk tokens", float64(n)/1000)
	default:
		return fmt.Sprintf("%.2fM tokens", float64(n)/1e6)
	}
}

// humanDuration formats milliseconds compactly ("" for zero).
func humanDuration(ms int64) string {
	switch {
	case ms <= 0:
		return ""
	case ms < 1000:
		return fmt.Sprintf("%d ms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	default:
		return fmt.Sprintf("%dm %02ds", ms/60_000, (ms%60_000)/1000)
	}
}

// statusLine is the header line under the title: a spinner and elapsed time
// while running, the outcome with usage afterwards. Empty fields are omitted.
func (m *model) statusLine() string {
	parts := []string{}
	if m.active {
		label := "Working"
		switch {
		case m.pending != nil:
			label = "Waiting for your approval"
		case m.question != nil:
			label = "Waiting for your answer"
		}
		elapsed := int64(0)
		if !m.runStarted.IsZero() {
			elapsed = time.Since(m.runStarted).Milliseconds()
		}
		head := spinFrames[m.spinFrame%len(spinFrames)] + " " + label
		if m.pending != nil || m.question != nil {
			head = "● " + label
		}
		parts = append(parts, head)
		if d := humanDuration(elapsed / 1000 * 1000); d != "" {
			parts = append(parts, d)
		}
		// A notice raised during the run (a provider retry, a full queue)
		// shows beside the spinner instead of being hidden by it.
		if m.status != "" && m.status != "running" {
			parts = append(parts, Safe(oneLine(m.status, 120)))
		}
	} else if m.status != "" {
		parts = append(parts, Safe(oneLine(m.status, 200)))
		if d := humanDuration(m.durationMS); d != "" {
			parts = append(parts, d)
		}
	}
	if t := humanTokens(m.usage); t != "" {
		parts = append(parts, t)
	}
	return strings.Join(parts, "  ·  ")
}

// shortPath shows a path relative to the home folder and keeps its tail
// when it is wider than width.
func shortPath(path string, width int) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
			path = filepath.Join("~", rel)
		}
	}
	r := []rune(path)
	if width > 1 && len(r) > width {
		path = "…" + string(r[len(r)-width+1:])
	}
	return path
}

// relativizePreview shortens the project prefix to a relative path in the
// header lines of approval previews so they fit the screen. Only lines at
// the positions the tools write headers are touched: the first two lines of
// a file diff, the "File N:" list at the top of a patch, and the last
// "Directory:" line (every command preview ends with it and a fixed footer).
// Commands, file contents and diff bodies are shown exactly as they will
// run: a heredoc or a removed line can look like a header, and rewriting it
// would show text that differs from what executes.
func relativizePreview(preview, project string) string {
	if project == "" {
		return preview
	}
	project = strings.TrimRight(project, "/")
	lines := strings.Split(preview, "\n")
	strip := func(path string) (string, bool) {
		if rel, ok := strings.CutPrefix(path, project+"/"); ok {
			return rel, true
		}
		return path, false
	}
	switch {
	case len(lines) >= 2 && strings.HasPrefix(lines[0], "--- ") && strings.HasPrefix(lines[1], "+++ "):
		for i, label := range []string{"--- ", "+++ "} {
			if rel, ok := strip(lines[i][len(label):]); ok {
				lines[i] = label + rel
			}
		}
	case strings.HasPrefix(preview, "Apply patch") || strings.HasPrefix(preview, "File 1: "):
		for i, line := range lines {
			if i == 0 && !strings.HasPrefix(line, "File ") {
				continue
			}
			label, rest, ok := patchFileLine(line)
			if !ok {
				break // the list ends at the first other line
			}
			if rel, ok := strip(rest); ok {
				lines[i] = label + strings.Replace(rel, " -> "+project+"/", " -> ", 1)
			}
		}
	default:
		for i := len(lines) - 1; i >= 0; i-- {
			rest, ok := strings.CutPrefix(lines[i], "Directory: ")
			if !ok {
				continue
			}
			if rest == project {
				lines[i] = "Directory: . (project root)"
			} else if rel, ok := strip(rest); ok {
				lines[i] = "Directory: " + rel
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}

// patchFileLine splits an apply_patch list line, "File N: KIND /abs/path[ ->
// /abs/dest]", into its label and the path text.
func patchFileLine(line string) (label, rest string, ok bool) {
	head, after, found := strings.Cut(line, ": ")
	n, isFile := strings.CutPrefix(head, "File ")
	if !found || !isFile || n == "" || strings.Trim(n, "0123456789") != "" {
		return "", "", false
	}
	kind, path, found := strings.Cut(after, " ")
	if !found || kind == "" || strings.ToUpper(kind) != kind {
		return "", "", false
	}
	return head + ": " + kind + " ", path, true
}
