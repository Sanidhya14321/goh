package tui

import (
	"os"

	tea "charm.land/bubbletea/v2"
)

// mouseMode captures the mouse so the wheel scrolls GoHarness instead of the
// terminal's own scrollback (which would reveal the shell behind the
// alternate screen). GOH_NO_MOUSE=1 leaves the mouse to the terminal for
// native text selection; most terminals also allow Shift/Option-drag to
// select while the mouse is captured.
func mouseMode() tea.MouseMode {
	if os.Getenv("GOH_NO_MOUSE") != "" {
		return tea.MouseModeNone
	}
	return tea.MouseModeCellMotion
}

// blankView is drawn until the terminal reports its size, so the first frame
// never overflows a small window and pushes the screen down.
func blankView() tea.View {
	view := tea.NewView("")
	view.AltScreen = true
	view.MouseMode = mouseMode()
	return view
}

// wheelAsKey turns a wheel event into an arrow key for list-style screens.
// Other mouse events (clicks, right clicks, drags, motion) return ok=false
// with handled=true: they are deliberately ignored.
func wheelAsKey(msg tea.Msg) (key tea.KeyPressMsg, ok, handled bool) {
	switch v := msg.(type) {
	case tea.MouseWheelMsg:
		switch v.Button {
		case tea.MouseWheelUp:
			return tea.KeyPressMsg{Code: tea.KeyUp}, true, true
		case tea.MouseWheelDown:
			return tea.KeyPressMsg{Code: tea.KeyDown}, true, true
		}
		return key, false, true
	case tea.MouseClickMsg, tea.MouseReleaseMsg, tea.MouseMotionMsg, tea.FocusMsg, tea.BlurMsg:
		return key, false, true
	}
	return key, false, false
}

// cancelRun stops the active run, if one was started.
func (m *model) cancelRun() {
	if m.cancel != nil {
		m.cancel()
	}
}
