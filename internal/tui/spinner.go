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
	} else if m.status != "" {
		parts = append(parts, m.status)
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

// relativizePreview replaces the project prefix in approval previews so diff
// headers show project-relative paths that fit the screen.
func relativizePreview(preview, project string) string {
	if project == "" {
		return preview
	}
	project = strings.TrimRight(project, "/")
	preview = strings.ReplaceAll(preview, project+"/", "")
	return strings.ReplaceAll(preview, "Directory: "+project+"\n", "Directory: . (project root)\n")
}
