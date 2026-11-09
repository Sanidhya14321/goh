package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// Review R10: after a crash between journaling a tool_result and adding it
// to history, resume must reuse the recorded result, not report the
// effect as uncertain; a call that never started is "not executed".
func TestRepairUsesJournaledResults(t *testing.T) {
	a, _ := testAgent(t)
	done := toolCall("done", "shell", map[string]any{"command": "make build"})
	never := toolCall("never", "shell", map[string]any{"command": "make deploy"})
	if err := a.commit(core.Message{Role: "user", Content: "build"}); err != nil {
		t.Fatal(err)
	}
	if err := a.commit(core.Message{Role: "assistant", ToolCalls: []core.Call{done, never}}); err != nil {
		t.Fatal(err)
	}
	a.start(done)
	a.finish(done, tools.Result{Content: `{"exit_code":0,"output":"BUILD-OK-7731"}`})
	// crash here: no tool messages were committed, "never" never started
	a.Store.Close()
	s, err := session.Open(a.Config.SessionDir, a.Store.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resumed, err := New(a.Config, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.RepairInterruptedHistory(); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range resumed.History {
		if m.Role == "tool" {
			got[m.ToolCallID] = m.Content
		}
	}
	if !strings.Contains(got["done"], "BUILD-OK-7731") || strings.Contains(got["done"], "uncertain") {
		t.Fatalf("recorded result not reused: %s", got["done"])
	}
	if !strings.Contains(got["never"], "Not executed") {
		t.Fatalf("never-started call misreported: %s", got["never"])
	}
}

// Review R13: masked results keep their facts and never tell the model to
// re-run an action with effects.
func TestMaskReceiptsKeepFactsAndDiscourageRepeats(t *testing.T) {
	big := strings.Repeat("x", 600)
	shellOut := `{"content":"{\"exit_code\":2,\"status\":\"completed\",\"artifact_id\":\"shell-1.log\",\"output\":\"` + big + `\"}","error":false}`
	readOut := `{"content":"{\"path\":\"a.go\",\"sha256\":\"abc\",\"content\":\"` + big + `\"}"}`
	h := []core.Message{{Role: "system", Content: "s"},
		{Role: "assistant", ToolCalls: []core.Call{toolCall("s1", "shell", map[string]any{"command": "make deploy"}), toolCall("r1", "read_file", map[string]any{"path": "a.go"})}},
		{Role: "tool", ToolCallID: "s1", Content: shellOut}, {Role: "tool", ToolCallID: "r1", Content: readOut}, {Role: "user", Content: "next"}}
	out, n := maskObservations(h, 4)
	if n != 2 {
		t.Fatalf("masked %d", n)
	}
	shell, read := out[2].Content, out[3].Content
	for _, want := range []string{`"exit_code":2`, `"artifact_id":"shell-1.log"`, "do not run it again"} {
		if !strings.Contains(shell, want) {
			t.Fatalf("shell receipt missing %s: %s", want, shell)
		}
	}
	if strings.Contains(shell, "run the tool again") || !strings.Contains(read, `"sha256":"abc"`) || !strings.Contains(read, "read-only") {
		t.Fatalf("receipts: %s | %s", shell, read)
	}
}

// Review R12: one large early read in a small window must not end the run
// with budget_exhausted before the model sees anything.
func TestLargeEarlyResultDoesNotExhaustBudget(t *testing.T) {
	a, root := testAgent(t)
	a.Config.ContextWindow, a.Config.OutputBudget = 32768, 4096
	big := strings.Repeat("line of source code here\n", 1800) // ~45 KB
	if err := os.WriteFile(filepath.Join(root, "big.txt"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	turn := 0
	var second string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			return answer(toolCall("r", "read_file", map[string]any{"path": "big.txt"})), nil
		}
		second = h[len(h)-1].Content
		return answer(), nil
	})
	status, err := a.Run(context.Background(), "read big.txt")
	if status != "completed" {
		t.Fatalf("status %s: %v", status, err)
	}
	if !strings.Contains(second, "start_line") || !strings.Contains(second, "line of source code") {
		t.Fatalf("model did not get an excerpt with paging advice: %.300s", second)
	}
}
