package tui

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
)

// The host keeps one Bubble Tea program running for a whole interactive
// session. Chat, pickers, forms and setup screens are shown in it one after
// another instead of each starting its own program, so the terminal never
// leaves the alternate screen between them, keys typed during a change of
// screen go to the next screen, and a dialog drawn over the chat follows the
// chat when the terminal is resized.
//
// Callers keep their blocking style: runProgram hands the screen to the host
// and waits until that screen quits.

var host struct {
	sync.Mutex
	active  bool          // between StartHost and its stop
	program *tea.Program  // nil once the program has ended
	ended   chan struct{} // closed when the program has ended
	waiting chan error    // the caller of the screen currently shown
}

// ErrHostEnded is returned to screens opened after the shared program has
// ended (for example after SIGTERM), so no new program takes the terminal.
var ErrHostEnded = errors.New("terminal interface closed")

// StartHost starts the shared program. Screens shown until stop is called
// run inside it. stop ends the program and waits for the terminal to be
// restored.
func StartHost() (stop func()) {
	host.Lock()
	if host.active {
		host.Unlock()
		return func() {}
	}
	p := tea.NewProgram(&hostModel{})
	ended := make(chan struct{})
	host.active, host.program, host.ended = true, p, ended
	host.Unlock()
	go func() {
		// A program that ends by itself (SIGTERM) closes the shown screen
		// as if it had quit; SIGINT reports an interrupt, as a standalone
		// program would.
		_, err := p.Run()
		host.Lock()
		if host.waiting != nil {
			host.waiting <- err
			host.waiting = nil
		}
		host.program = nil
		host.Unlock()
		close(ended)
	}()
	return func() {
		host.Lock()
		program := host.program
		host.Unlock()
		if program != nil {
			program.Send(hostStop{})
		}
		<-ended
		host.Lock()
		host.active = false
		host.Unlock()
	}
}

// hosted reports whether screens go to the shared program.
func hosted() bool {
	host.Lock()
	defer host.Unlock()
	return host.active
}

// showHosted shows m in the shared program and waits until it quits.
func showHosted(m tea.Model) error {
	host.Lock()
	if host.program == nil {
		host.Unlock()
		return ErrHostEnded
	}
	done := make(chan error, 1)
	host.waiting = done
	p := host.program
	host.Unlock()
	p.Send(showScreen{model: m, done: done})
	return <-done
}

// Suspended runs fn with the terminal handed back to normal line mode, for
// a prompt read straight from standard input while the host is running.
func Suspended(fn func() error) error {
	host.Lock()
	p := host.program
	host.Unlock()
	if p == nil {
		return fn()
	}
	if err := p.ReleaseTerminal(); err != nil {
		return err
	}
	defer p.RestoreTerminal()
	return fn()
}

type (
	showScreen struct {
		model tea.Model
		done  chan error
	}
	hostStop  struct{}
	hostQuit  struct{ gen int }
	screenMsg struct {
		gen int
		msg tea.Msg
	}
)

// backdropSetter is a screen drawn over the previous one; the host keeps its
// backdrop current, so it reflows with the screen underneath on resize.
type backdropSetter interface {
	setBackdrop(string)
}

type hostModel struct {
	screen        tea.Model
	under         tea.Model // the last chat screen, drawn under dialogs
	last          tea.Model // the last screen of any kind, kept up between screens
	done          chan error
	gen           int
	width, height int
	sized         bool
	pending       []tea.Msg // keys typed while no screen was shown
}

// Init asks the terminal for its background color. The answer arrives
// asynchronously and only tints panels; nothing waits for it.
func (h *hostModel) Init() tea.Cmd { return tea.RequestBackgroundColor }

func (h *hostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case hostStop:
		h.finish(nil)
		return h, tea.Quit
	case showScreen:
		return h, h.show(v)
	case hostQuit:
		if v.gen == h.gen {
			h.finish(nil)
		}
		return h, nil
	case screenMsg:
		if v.gen != h.gen || h.screen == nil {
			return h, nil // from a screen that has closed
		}
		return h, h.deliver(v.msg)
	case tea.BackgroundColorMsg:
		if !setTerminalBackground(v.Color) || h.screen == nil {
			return h, nil
		}
		return h, h.deliver(themeChanged{}) // cached rows carry the old colors
	case tea.WindowSizeMsg:
		h.width, h.height, h.sized = v.Width, v.Height, true
		if h.under != nil {
			h.under, _ = h.under.Update(v) // only its view is used
		}
		if h.last != nil && h.last != h.under {
			h.last, _ = h.last.Update(v)
		}
	case tea.KeyPressMsg, tea.PasteMsg:
		if h.screen == nil {
			if len(h.pending) < typeAheadKeys {
				h.pending = append(h.pending, msg)
			}
			return h, nil
		}
	}
	if h.screen == nil {
		return h, nil
	}
	return h, h.deliver(msg)
}

// show replaces the current screen with s.model, giving it the terminal
// size and any keys typed since the last screen closed.
func (h *hostModel) show(s showScreen) tea.Cmd {
	if h.screen != nil {
		h.finish(nil)
	}
	h.gen++
	h.screen, h.done = s.model, s.done
	cmds := []tea.Cmd{h.wrap(h.screen.Init())}
	if h.sized {
		cmds = append(cmds, h.deliver(tea.WindowSizeMsg{Width: h.width, Height: h.height}))
	}
	keys := h.pending
	h.pending = nil
	for _, key := range keys {
		if h.screen == nil {
			// A replayed key closed the screen; the rest wait for the next.
			h.pending = append(h.pending, key)
			continue
		}
		cmds = append(cmds, h.deliver(key))
	}
	return tea.Batch(cmds...)
}

// deliver sends msg to the current screen and wraps the command it returns.
func (h *hostModel) deliver(msg tea.Msg) tea.Cmd {
	h.syncBackdrop()
	next, cmd := h.screen.Update(msg)
	h.screen = next
	if isQuit(cmd) {
		// Close now, so keys already queued behind this one go to the
		// next screen instead of this one.
		h.finish(nil)
		return nil
	}
	return h.wrap(cmd)
}

// finish closes the current screen and releases its caller.
func (h *hostModel) finish(err error) {
	if h.screen == nil {
		return
	}
	if _, chat := h.screen.(*model); chat {
		h.under = h.screen
	}
	h.last = h.screen
	h.screen = nil
	h.gen++ // messages still in flight for the closed screen are dropped
	host.Lock()
	if host.waiting == h.done {
		host.waiting = nil
	}
	host.Unlock()
	if h.done != nil {
		h.done <- err
		h.done = nil
	}
}

func (h *hostModel) syncBackdrop() {
	if b, ok := h.screen.(backdropSetter); ok && h.under != nil {
		b.setBackdrop(h.under.View().Content)
	}
}

func (h *hostModel) View() tea.View {
	if h.screen != nil {
		h.syncBackdrop()
		return h.screen.View()
	}
	if h.last != nil {
		// Keep the last screen up while the caller prepares the next one.
		return h.last.View()
	}
	return blankView()
}

// wrap tags the messages a screen's command produces with that screen, so
// a timer or result that arrives after the screen closed is not delivered
// to the next one. Program-level messages (exec, clipboard, terminal
// queries) belong to Bubble Tea and pass through unchanged.
func (h *hostModel) wrap(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	gen := h.gen
	var tag func(tea.Cmd) tea.Cmd
	tag = func(c tea.Cmd) tea.Cmd {
		if c == nil {
			return nil
		}
		if isQuit(c) {
			return func() tea.Msg { return hostQuit{gen: gen} }
		}
		return func() tea.Msg {
			switch v := c().(type) {
			case nil:
				return nil
			case tea.QuitMsg:
				return hostQuit{gen: gen}
			case tea.BatchMsg:
				out := make(tea.BatchMsg, len(v))
				for i, inner := range v {
					out[i] = tag(inner)
				}
				return out
			default:
				if ownMessage(v) {
					return screenMsg{gen: gen, msg: v}
				}
				return v
			}
		}
	}
	return tag(cmd)
}

// ownMessage reports whether msg is one of this program's own types rather
// than a Bubble Tea type the program itself must handle.
func ownMessage(msg tea.Msg) bool {
	t := reflect.TypeOf(msg)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t != nil && strings.HasPrefix(t.PkgPath(), "github.com/Sanidhya14321/goh/")
}

// Dialogs opened over the chat keep their backdrop live; full-screen ones
// (no backdrop) stay full-screen.
func (m *optionPicker) setBackdrop(s string) {
	if m.backdrop != "" {
		m.backdrop = s
	}
}

func (m *providerManager) setBackdrop(s string) {
	if m.backdrop != "" {
		m.backdrop = s
	}
}
