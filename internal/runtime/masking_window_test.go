package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// C3: when masking is enough, the newest maskKeepRecent tool results stay
// verbatim and only older ones become receipts.
func TestMaskingKeepsRecentWindow(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.ContextWindow = 200000
	big := strings.Repeat("y", 9000)
	a.commit(core.Message{Role: "user", Content: "investigate"})
	for i := 0; i < 15; i++ {
		id := fmt.Sprint("r", i)
		a.commit(core.Message{Role: "assistant", ToolCalls: []core.Call{toolCall(id, "read_file", map[string]any{"path": id})}})
		a.commit(core.Message{Role: "tool", ToolCallID: id, Content: `{"content":"` + id + big + `"}`})
	}
	a.commit(core.Message{Role: "assistant", Content: "noted"})
	a.Config.CompactionThreshold = 50 // ~135k used of 200k: masking triggers
	if err := a.compact(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	masked, intact := 0, 0
	for _, m := range a.History {
		if m.Role != "tool" {
			continue
		}
		if strings.HasPrefix(m.Content, maskPrefix) {
			masked++
		} else {
			intact++
		}
	}
	if masked != 5 || intact != 10 {
		t.Fatalf("masked %d intact %d, want 5 and 10", masked, intact)
	}
}
