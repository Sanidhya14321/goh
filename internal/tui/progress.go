package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// ErrCancelled is returned by WithProgress when the user pressed Esc.
var ErrCancelled = errors.New("cancelled")

type progressDone struct{ err error }

// progressScreen shows that a request is running and owns the keyboard
// while it does: Esc (or Ctrl+C) cancels the request on this screen, and no
// key typed meanwhile reaches the screen that opens next.
type progressScreen struct {
	title, backdrop string
	cancel          context.CancelFunc
	run             func() error
	err             error
	cancelled       bool
	width, height   int
}

func (p *progressScreen) Init() tea.Cmd {
	return func() tea.Msg { return progressDone{p.run()} }
}

func (p *progressScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = v.Width, v.Height
	case progressDone:
		p.err = v.err
		return p, tea.Quit
	case tea.KeyPressMsg:
		switch v.String() {
		case "esc", "ctrl+c":
			p.cancelled = true
			p.cancel()
		}
		// Every other key is swallowed: it was typed at this screen.
	}
	return p, nil
}

func (p *progressScreen) View() tea.View {
	width, height := p.width, p.height
	if width < 1 || height < 1 {
		return blankView()
	}
	boxWidth := min(76, max(1, width-4))
	left := max(0, (width-boxWidth)/2)
	padding := strings.Repeat(" ", left)
	status := "  " + p.title + " …"
	hint := "  Esc cancel"
	if p.cancelled {
		status, hint = "  Cancelling …", "  "
	}
	top := max(0, height/2-2)
	rows := make([]string, top)
	rows = append(rows,
		padding+cell(ansi.Truncate(status, boxWidth, "…"), boxWidth, accent, panel),
		padding+cell(hint, boxWidth, muted, panel))
	return overlayView(p.backdrop, rows, left, top, width, height)
}

// WithProgress runs work with a cancellable context while a screen says
// what is happening. Esc cancels it there; the result is ErrCancelled then.
func WithProgress(title, backdrop string, work func(context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &progressScreen{title: title, backdrop: backdrop, cancel: cancel, run: func() error { return work(ctx) }}
	if err := runProgram(p); err != nil {
		return err
	}
	if p.cancelled {
		return ErrCancelled
	}
	return p.err
}
