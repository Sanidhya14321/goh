package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// recorder is a screen that records what it receives and quits on "q".
type recorder struct {
	got  []string
	tick tea.Cmd // returned from Init
}

type tickMsg struct{}

func (r *recorder) Init() tea.Cmd { return r.tick }
func (r *recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		r.got = append(r.got, v.String())
		if v.String() == "q" {
			return r, tea.Quit
		}
	case tickMsg:
		r.got = append(r.got, "tick")
	case tea.WindowSizeMsg:
		r.got = append(r.got, "size")
	}
	return r, nil
}
func (r *recorder) View() tea.View { return tea.NewView("recorder") }

func key(s string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: rune(s[0]), Text: s} }

// run executes cmd and feeds the resulting messages back, as the program
// would, flattening batches.
func run(h *hostModel, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch v := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range v {
			run(h, c)
		}
	default:
		_, next := h.Update(v)
		run(h, next)
	}
}

func show(h *hostModel, m tea.Model) chan error {
	done := make(chan error, 1)
	_, cmd := h.Update(showScreen{model: m, done: done})
	run(h, cmd)
	return done
}

func TestHostCarriesKeysAcrossScreensAndDropsStaleMessages(t *testing.T) {
	h := &hostModel{}
	h.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	first := &recorder{}
	done := show(h, first)
	// A timer the first screen started, still pending when it closes.
	stale := h.wrap(func() tea.Msg { return tickMsg{} })
	_, cmd := h.Update(key("q"))
	run(h, cmd)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// Typed while no screen is shown: kept for the next one.
	h.Update(key("a"))
	h.Update(key("b"))
	second := &recorder{}
	show(h, second)
	run(h, stale) // arrives late; must not reach the second screen
	if got := strings.Join(second.got, ","); got != "size,a,b" {
		t.Fatalf("second screen got %q", got)
	}
	if got := strings.Join(first.got, ","); got != "size,q" {
		t.Fatalf("first screen got %q", got)
	}
}

func TestHostQuitInsideBatchClosesScreen(t *testing.T) {
	h := &hostModel{}
	r := &recorder{tick: tea.Batch(func() tea.Msg { return tickMsg{} }, tea.Quit)}
	done := show(h, r)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("a quit inside a batch did not close the screen")
	}
	if h.screen != nil {
		t.Fatal("screen still shown")
	}
}

func TestHostDialogBackdropFollowsChatResize(t *testing.T) {
	h := &hostModel{}
	h.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	chat := &model{width: 60, height: 20, status: "ready", log: "conversation line", keys: keymapFor(nil)}
	h.under = chat // the chat that opened the dialog
	picker := &optionPicker{title: "Pick", options: []Option{{Value: "x", Label: "X"}}, backdrop: "stale snapshot", width: 60, height: 20}
	show(h, picker)
	h.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	view := h.View().Content
	if strings.Contains(view, "stale snapshot") {
		t.Fatal("dialog kept the snapshot backdrop")
	}
	if chat.width != 100 || chat.height != 30 {
		t.Fatalf("chat under the dialog not resized: %dx%d", chat.width, chat.height)
	}
	// A full-screen picker (no backdrop) stays full-screen.
	full := &optionPicker{title: "Full", options: []Option{{Value: "y", Label: "Y"}}}
	full.setBackdrop("chat")
	if full.backdrop != "" {
		t.Fatal("full-screen picker gained a backdrop")
	}
}

func TestOwnMessage(t *testing.T) {
	if !ownMessage(tickMsg{}) || !ownMessage(&recorder{}) {
		t.Fatal("own types not recognised")
	}
	if ownMessage(tea.QuitMsg{}) || ownMessage(tea.WindowSizeMsg{}) || ownMessage("text") {
		t.Fatal("Bubble Tea or builtin types treated as own")
	}
}
