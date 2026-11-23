package runtime

import (
	"encoding/json"
	"strings"

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

// withoutRecallBodies replaces memory_recall results with a neutral receipt
// (no note text). Summaries and excerpts are persisted as opaque text that
// revalidateRecalls cannot recognise later, so a note body must never reach
// them: a note fresh when summarised can be stale on a later turn. Call and
// result pairing is kept; the journal keeps the original for audit, and the
// model can recall the note again through validated memory_recall.
func withoutRecallBodies(history []core.Message) []core.Message {
	recallIDs := map[string]bool{}
	var out []core.Message
	for i, m := range history {
		for _, c := range m.ToolCalls {
			if c.Function.Name == "memory_recall" {
				recallIDs[c.ID] = true
			}
		}
		if m.Role != "tool" || !recallIDs[m.ToolCallID] || strings.HasPrefix(m.Content, maskPrefix) {
			continue
		}
		if out == nil {
			out = append([]core.Message(nil), history...)
		}
		b, _ := json.Marshal(recallReceipt(m.Content))
		out[i].Content = string(b)
	}
	if out == nil {
		return history
	}
	return out
}

func recallReceipt(content string) receipt {
	r := maskReceipt("memory_recall", content)
	r.Note = "Project-note text omitted; notes are revalidated only through memory_recall. Run memory_recall again if the note is still needed."
	return r
}
