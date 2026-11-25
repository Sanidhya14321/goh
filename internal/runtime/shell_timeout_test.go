package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// A command stopped at its own timeout is a result the model can act on,
// not the end of the run.
func TestTimedOutCommandReturnsToTheModel(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Grants["shell"] = true
	var seen string
	calls := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		calls++
		if calls == 1 {
			return answer(toolCall("s", "shell", map[string]any{"command": "sleep 5", "timeout_seconds": 1})), nil
		}
		seen = h[len(h)-1].Content
		return answerText("it timed out; I will run it in the background"), nil
	})
	status, err := a.Run(context.Background(), "go")
	if err != nil || status != "completed" {
		t.Fatalf("status %s, err %v", status, err)
	}
	if !strings.Contains(seen, "timed_out") || !strings.Contains(seen, "job_start") {
		t.Fatalf("the model saw %q", seen)
	}
}
