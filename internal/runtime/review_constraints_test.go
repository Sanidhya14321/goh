package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// Review R17: an early user constraint must survive two compactions
// verbatim, not only as summary prose.
func TestUserConstraintsSurviveRepeatedCompaction(t *testing.T) {
	a, _ := testAgent(t)
	const constraint = "CONSTRAINT-42: never modify files under tests/ and keep the public API unchanged"
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "Summary: worked on the parser."}, Finish: "stop"}, nil
	})
	turns := func(msgs ...string) {
		for i, m := range msgs {
			role := "user"
			if i%2 == 1 {
				role = "assistant"
			}
			if err := a.commit(core.Message{Role: role, Content: m}); err != nil {
				t.Fatal(err)
			}
		}
	}
	has := func() bool {
		for _, m := range a.History {
			if m.Role == "user" && strings.Contains(m.Content, constraint) {
				return true
			}
		}
		return false
	}
	turns(constraint, "understood", "refactor the lexer", strings.Repeat("lexer work ", 500), "now the parser", "parser done", "and docs", "docs done")
	if _, err := a.Compact(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !has() {
		t.Fatal("constraint lost after first compaction")
	}
	turns("next: add tests elsewhere", strings.Repeat("more work ", 500), "polish", "polished", "final check", "checked")
	if _, err := a.Compact(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !has() {
		t.Fatal("constraint lost after second compaction")
	}
}
