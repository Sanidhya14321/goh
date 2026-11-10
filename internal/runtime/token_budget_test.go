package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The run stops before a model turn once max_run_tokens is used, after the
// requested tools have their results.
func TestMaxRunTokens(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MaxRunTokens = 150
	a.Grants["read_file"] = true
	turns := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turns++
		r := answer(toolCall("r", "list_files", map[string]any{}))
		r.Usage = &core.Usage{Prompt: 90, Completion: 10, Total: 100}
		return r, nil
	})
	status, err := a.Run(context.Background(), "go")
	if status != "budget_exhausted" || err == nil || !strings.Contains(err.Error(), "max_run_tokens") || turns != 2 {
		t.Fatalf("status %s err %v turns %d", status, err, turns)
	}
}
