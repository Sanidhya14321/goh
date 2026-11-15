package tui

import (
	"reflect"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Every screen (chat, pickers, forms) is its own Bubble Tea program, and
// moving between them quits one and starts the next. A program that quits
// drops input it has already read, so keys typed quickly after Tab, Enter on
// a picker or a slash command used to vanish. runProgram closes a screen only
// once typing pauses, and replays the keys typed meanwhile into the next
// screen, as if one program had been running throughout.

const (
	settleQuiet   = 60 * time.Millisecond  // input pause that ends a closing screen
	settleLimit   = 400 * time.Millisecond // longest a closing screen waits
	typeAheadKeys = 512                    // keys kept across one change of screen
)

var typeAhead struct {
	sync.Mutex
	keys []tea.Msg
}

func addTypeAhead(msg tea.Msg) {
	typeAhead.Lock()
	defer typeAhead.Unlock()
	if len(typeAhead.keys) < typeAheadKeys {
		typeAhead.keys = append(typeAhead.keys, msg)
	}
}

// takeTypeAhead returns and clears the keys typed while the last screen
// closed.
func takeTypeAhead() []tea.Msg {
	typeAhead.Lock()
	defer typeAhead.Unlock()
	keys := typeAhead.keys
	typeAhead.keys = nil
	return keys
}

// runProgram runs m as a full-screen program, carrying keystrokes across the
// change of screen in both directions.
func runProgram(m tea.Model) error {
	_, err := tea.NewProgram(&settler{inner: m, replay: takeTypeAhead()}).Run()
	return err
}

type (
	quitRequested struct{}
	settleTick    struct{}
	replayKeys    struct{}
)

type settler struct {
	inner          tea.Model
	replay         []tea.Msg
	closing        bool
	started, typed time.Time
}

func (s *settler) Init() tea.Cmd {
	cmd := interceptQuit(s.inner.Init())
	if len(s.replay) > 0 {
		cmd = tea.Batch(cmd, func() tea.Msg { return replayKeys{} })
	}
	return cmd
}

func (s *settler) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(replayKeys); ok {
		// Keys from the previous screen go first, one at a time; any that
		// arrive after this screen closes pass on to the next one.
		keys := s.replay
		s.replay = nil
		var cmds []tea.Cmd
		for _, key := range keys {
			_, cmd := s.Update(key)
			cmds = append(cmds, cmd)
		}
		return s, tea.Batch(cmds...)
	}
	if !s.closing {
		if _, ok := msg.(quitRequested); ok {
			s.closing, s.started, s.typed = true, time.Now(), time.Now()
			return s, tea.Tick(settleQuiet, func(time.Time) tea.Msg { return settleTick{} })
		}
		next, cmd := s.inner.Update(msg)
		s.inner = next
		if isQuit(cmd) {
			// Close now, so keys already queued behind this one carry over
			// instead of reaching the closing screen.
			return s.Update(quitRequested{})
		}
		return s, interceptQuit(cmd)
	}
	switch msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		addTypeAhead(msg)
		s.typed = time.Now()
	case settleTick:
		if time.Since(s.typed) >= settleQuiet || time.Since(s.started) >= settleLimit {
			return s, tea.Quit
		}
		return s, tea.Tick(settleQuiet, func(time.Time) tea.Msg { return settleTick{} })
	}
	return s, nil
}

func (s *settler) View() tea.View { return s.inner.View() }

// isQuit reports whether cmd is tea.Quit itself, which every screen returns
// to close.
func isQuit(cmd tea.Cmd) bool {
	return cmd != nil && reflect.ValueOf(cmd).Pointer() == reflect.ValueOf(tea.Quit).Pointer()
}

// interceptQuit turns a quit inside a batch into quitRequested, so the
// settler can collect late input before the program ends.
func interceptQuit(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		switch v := cmd().(type) {
		case tea.QuitMsg:
			return quitRequested{}
		case tea.BatchMsg:
			out := make(tea.BatchMsg, len(v))
			for i, c := range v {
				out[i] = interceptQuit(c)
			}
			return out
		default:
			return v
		}
	}
}
