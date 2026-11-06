package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
	"github.com/charmbracelet/x/ansi"
)

func TestLayoutBoundsAndPalette(t *testing.T) {
	for _, size := range [][2]int{{24, 10}, {60, 18}, {80, 24}, {120, 36}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m := &model{a: &agent.Agent{Config: core.Config{Model: "fixture"}, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}}, width: size[0], height: size[1], log: strings.Repeat("wide 界 text\n", 100), input: strings.Repeat("界", 200)}
			for _, state := range []string{"chat", "permission", "palette"} {
				m.palette = state == "palette"
				m.pending = nil
				if state == "permission" {
					m.pending = &permissionMsg{request: core.Permission{Tool: "edit_file", Preview: m.log}}
				}
				view := m.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) > m.height {
					t.Fatalf("%s rows = %d, height = %d", state, len(lines), m.height)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > m.width {
						t.Fatalf("%s overflowed terminal width", state)
					}
				}
			}
		})
	}
	m := &model{blocked: "setup required", palette: true}
	m.Update(tea.KeyPressMsg{Code: 'p', Text: "provider"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.navigation != "provider" || cmd == nil || m.active {
		t.Fatal("palette failed to open provider setup")
	}
}

func BenchmarkTranscriptView(b *testing.B) {
	m := &model{a: &agent.Agent{Config: core.Config{Model: "fixture"}, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}}, width: 120, height: 36, log: strings.Repeat("Large output line with Unicode 界 and safe text\n", 2500)}
	m.View()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m.View()
	}
}
