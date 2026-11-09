package runtime

import (
	"encoding/json"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/memory"
	"github.com/Sanidhya14321/goh/internal/tools"
)

const withdrawnNote = "This recalled note is no longer valid (deleted, its files changed, or memory was turned off) and has been withdrawn. Do not rely on it."

// revalidateRecalls rechecks memory_recall results before they are sent to
// the model again. The journal keeps the original for audit; the model only
// sees notes that still exist with unchanged evidence.
func (a *Agent) revalidateRecalls(history []core.Message) []core.Message {
	recallIDs := map[string]bool{}
	for _, m := range history {
		for _, c := range m.ToolCalls {
			if c.Function.Name == "memory_recall" {
				recallIDs[c.ID] = true
			}
		}
	}
	if len(recallIDs) == 0 {
		return history
	}
	var ledger *memory.Ledger
	if a.Config.Memory {
		ledger, _ = memory.New(a.Config.SessionDir, a.Store.Project)
	}
	out := append([]core.Message(nil), history...)
	for i, m := range out {
		if m.Role != "tool" || !recallIDs[m.ToolCallID] {
			continue
		}
		var envelope tools.Result
		var body struct {
			Notes     []memory.Note `json:"notes"`
			Authority string        `json:"authority"`
		}
		if json.Unmarshal([]byte(m.Content), &envelope) != nil || envelope.Error || json.Unmarshal([]byte(envelope.Content), &body) != nil {
			continue
		}
		ids := make([]string, 0, len(body.Notes))
		for _, n := range body.Notes {
			ids = append(ids, n.ID)
		}
		var valid map[string]bool
		if ledger != nil {
			valid, _ = ledger.Valid(ids)
		}
		kept := body.Notes[:0:0]
		withdrawn := 0
		for _, n := range body.Notes {
			if valid[n.ID] {
				kept = append(kept, n)
			} else {
				withdrawn++
			}
		}
		if withdrawn == 0 {
			continue
		}
		next := map[string]any{"notes": kept, "authority": body.Authority, "withdrawn": withdrawn, "note": withdrawnNote}
		inner, _ := json.Marshal(next)
		b, _ := json.Marshal(tools.Result{Content: string(inner)})
		m.Content = string(b)
		out[i] = m
	}
	return out
}
