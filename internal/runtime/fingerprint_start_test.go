package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The start fingerprint is taken beside the first request; a shell command
// in the first reply that changes a file is still noticed.
func TestStartFingerprintCoversTheFirstCommand(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644)
	a.Grants["shell"] = true
	a.Provider = scripted([]core.Call{toolCall("s", "shell", map[string]any{"command": "echo b >> a.txt"})})
	a.Run(context.Background(), "go")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeUnverified {
		t.Fatalf("outcome %v, want %v", got, OutcomeUnverified)
	}
}

// A reply without tool calls skips the closing walk: nothing the session
// started could have changed a file, so an edit made meanwhile by someone
// else is not the run's change.
func TestReplyWithoutToolsIsNoChange(t *testing.T) {
	a, root := testAgent(t)
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		os.WriteFile(filepath.Join(root, "edited-meanwhile.txt"), []byte("x\n"), 0o644)
		return answerText("hello"), nil
	})
	a.Run(context.Background(), "hi")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeNoChanges {
		t.Fatalf("outcome %v, want %v", got, OutcomeNoChanges)
	}
}
