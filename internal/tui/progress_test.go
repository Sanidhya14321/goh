package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Review: a model list loading behind a progress screen is cancelled by Esc
// on that screen, and other keys are swallowed there.
func TestProgressScreenOwnsKeys(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &progressScreen{title: "Loading models", cancel: cancel, width: 60, height: 10}
	if _, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil || ctx.Err() != nil {
		t.Fatal("Enter acted on the progress screen")
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if ctx.Err() == nil || !p.cancelled {
		t.Fatal("Esc did not cancel the request")
	}
	if _, cmd := p.Update(progressDone{err: ctx.Err()}); cmd == nil {
		t.Fatal("screen did not close when the request ended")
	}
}
