package tui

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
	"github.com/charmbracelet/x/ansi"
)

// fuzzEvents is every kind of input a terminal can deliver: printable and
// control keys, navigation, mouse buttons and wheel, paste, focus and resizes
// down to degenerate sizes.
func fuzzEvents() []tea.Msg {
	var events []tea.Msg
	for _, r := range "aZ9 /!@y sn?q" {
		events = append(events, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	for _, code := range []rune{tea.KeyEnter, tea.KeyEscape, tea.KeyTab, tea.KeyBackspace, tea.KeyDelete,
		tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd, tea.KeyInsert, tea.KeyF1} {
		events = append(events, tea.KeyPressMsg{Code: code})
	}
	events = append(events, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	for _, r := range "abeklnpuwx" { // Ctrl chords; Ctrl+C/D/Z are covered separately
		events = append(events, tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl})
	}
	for _, b := range []tea.MouseButton{tea.MouseLeft, tea.MouseRight, tea.MouseMiddle} {
		events = append(events, tea.MouseClickMsg{Button: b, X: 5, Y: 5}, tea.MouseReleaseMsg{Button: b})
	}
	events = append(events,
		tea.MouseWheelMsg{Button: tea.MouseWheelUp}, tea.MouseWheelMsg{Button: tea.MouseWheelDown},
		tea.MouseWheelMsg{Button: tea.MouseWheelLeft}, tea.MouseMotionMsg{X: 3, Y: 3},
		tea.PasteMsg{Content: "pasted\ttext\nsecond line 界"}, tea.PasteMsg{Content: ""},
		tea.PasteMsg{Content: strings.Repeat("x", 5000)},
		tea.FocusMsg{}, tea.BlurMsg{},
		tea.WindowSizeMsg{Width: 1, Height: 1}, tea.WindowSizeMsg{Width: 20, Height: 5},
		tea.WindowSizeMsg{Width: 59, Height: 17}, tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.WindowSizeMsg{Width: 250, Height: 70}, tea.WindowSizeMsg{Width: 0, Height: 0},
	)
	return events
}

type fuzzProvider struct{}

func (fuzzProvider) Complete(_ context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	return core.Reply{Message: core.Message{Role: "assistant", Content: "ok"}, Finish: "stop"}, nil
}

func fuzzModel(t *testing.T) *model {
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = t.TempDir(), "fixture", 1<<20
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	a.Provider = fuzzProvider{}
	return &model{a: a, events: make(chan tea.Msg, 128), width: 100, height: 30, status: "ready",
		closed: make(chan struct{}), keys: keymapFor(&a.Config),
		log: "\nYou: hello\nreply with **bold** and a table\n\n| a | b |\n|---|---|\n| 1 | 2 |\n```go\nfunc x() {}\n```\n"}
}

// update runs one event with a deadline so a handler that blocks (for
// example on an unbuffered reply channel) fails the test instead of hanging.
func update(t *testing.T, m *model, msg tea.Msg, trail []string) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		m.Update(msg)
		_ = m.View()
	}()
	select {
	case p := <-done:
		if p != nil {
			t.Fatalf("panic %v after %s", p, strings.Join(trail, " → "))
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("handler blocked after %s", strings.Join(trail, " → "))
	}
	view := m.View()
	if m.width < 1 || m.height < 1 {
		return
	}
	rows := strings.Split(view.Content, "\n")
	if len(rows) > m.height {
		t.Fatalf("%d rows exceed height %d after %s", len(rows), m.height, strings.Join(trail, " → "))
	}
	for i, row := range rows {
		if w := ansi.StringWidth(row); w > m.width {
			t.Fatalf("row %d is %d cells wide in a %d-wide terminal after %s: %q", i, w, m.width, strings.Join(trail, " → "), ansi.Strip(row))
		}
	}
}

func fuzzType(m *model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func fuzzStates(t *testing.T) map[string]func() *model {
	states := map[string]func() *model{
		"home": func() *model { return fuzzModel(t) },
		"palette": func() *model {
			m := fuzzModel(t)
			m.palette = true
			return m
		},
		"approval": func() *model {
			m := fuzzModel(t)
			m.active = true
			m.pending = &permissionMsg{request: core.Permission{Tool: "shell", Preview: "Command: ls\n" + strings.Repeat("界 long line ", 40)},
				response: make(chan core.Decision, 4)}
			return m
		},
		"question": func() *model {
			m := fuzzModel(t)
			m.active = true
			q := questionMsg{request: core.QuestionRequest{Questions: []core.Question{
				{Question: "Pick one?", Header: "One", Options: []core.QuestionOption{{Label: "A"}, {Label: "B"}}},
				{Question: "Pick many?", Header: "Many", Multiple: true, Options: []core.QuestionOption{{Label: "C"}}},
			}}, response: make(chan questionResponse, 4)}
			m.Update(q)
			return m
		},
		"running": func() *model {
			m := fuzzModel(t)
			m.active = true
			m.status = "running"
			return m
		},
	}
	for _, entry := range paletteEntries {
		command := entry.command
		switch command {
		case "/editor", "/connect", "/provider", "/settings", "/workspace", "/sessions", "/models":
			continue // these hand the terminal to another program
		}
		states["cmd "+command] = func() *model {
			m := fuzzModel(t)
			fuzzType(m, command)
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			return m
		}
	}
	return states
}

func TestEveryEventInEveryState(t *testing.T) {
	events := fuzzEvents()
	for name, build := range fuzzStates(t) {
		t.Run(name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(len(name))))
			// Each event once from a fresh state, then a long random walk.
			for i, event := range events {
				m := build()
				update(t, m, event, []string{name, fmt.Sprintf("#%d %T%+v", i, event, event)})
			}
			m := build()
			trail := []string{name}
			for range 300 {
				event := events[rng.Intn(len(events))]
				trail = append(trail, fmt.Sprintf("%T%+v", event, event))
				if len(trail) > 12 {
					trail = append(trail[:1], trail[len(trail)-11:]...)
				}
				update(t, m, event, trail)
			}
		})
	}
}

// The setup screens run as their own programs; the same events must be safe.
func TestEveryEventInSetupScreens(t *testing.T) {
	screens := map[string]func() tea.Model{
		"options": func() tea.Model {
			return &optionPicker{title: "Pick", current: "b", width: 80, height: 24,
				options: []Option{{Value: "a", Label: "Alpha"}, {Value: "b", Label: strings.Repeat("界", 60)}}}
		},
		"options-empty": func() tea.Model { return &optionPicker{title: "Empty", width: 80, height: 24} },
		"sessions": func() tea.Model {
			return &chooser{project: "/one", current: "x", width: 120, height: 30, sessions: []session.Info{
				{ID: "x", Project: "/one"}, {ID: "y", Project: "/two"}}}
		},
		"sessions-empty": func() tea.Model { return &chooser{project: "/one", width: 80, height: 24} },
		"launcher":       func() tea.Model { return &launcher{status: "Chat setup required", width: 80, height: 24} },
		"settings": func() tea.Model {
			return &settingsForm{title: "Provider", hint: "Edit", width: 80, height: 24,
				save:   func([]SettingsField) error { return nil },
				fields: []SettingsField{{Label: "URL", Value: "http://x"}, {Label: "Key", Value: "secret", Secret: true}}}
		},
	}
	events := fuzzEvents()
	for name, build := range screens {
		t.Run(name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(len(name))))
			check := func(m tea.Model, msg tea.Msg, trail string) tea.Model {
				done := make(chan any, 1)
				var next tea.Model
				go func() {
					defer func() { done <- recover() }()
					next, _ = m.Update(msg)
					_ = next.View()
				}()
				select {
				case p := <-done:
					if p != nil {
						t.Fatalf("panic %v after %s", p, trail)
					}
				case <-time.After(2 * time.Second):
					t.Fatalf("blocked after %s", trail)
				}
				w, h := 80, 24
				if size, ok := sizeOf(next); ok {
					w, h = size[0], size[1]
				}
				if w > 0 && h > 0 {
					rows := strings.Split(next.View().Content, "\n")
					if len(rows) > h {
						t.Fatalf("%d rows exceed height %d after %s", len(rows), h, trail)
					}
					for i, row := range rows {
						if ansi.StringWidth(row) > w {
							t.Fatalf("row %d is %d wide in %d after %s: %q", i, ansi.StringWidth(row), w, trail, ansi.Strip(row))
						}
					}
				}
				return next
			}
			for _, event := range events {
				check(build(), event, fmt.Sprintf("%T%+v", event, event))
			}
			m := build()
			for i := range 400 {
				event := events[rng.Intn(len(events))]
				m = check(m, event, fmt.Sprintf("step %d %T%+v", i, event, event))
			}
		})
	}
}

func sizeOf(m tea.Model) ([2]int, bool) {
	switch v := m.(type) {
	case *optionPicker:
		return [2]int{v.width, v.height}, true
	case *chooser:
		return [2]int{v.width, v.height}, true
	case *launcher:
		return [2]int{v.width, v.height}, true
	case *settingsForm:
		return [2]int{v.width, v.height}, true
	}
	return [2]int{}, false
}
