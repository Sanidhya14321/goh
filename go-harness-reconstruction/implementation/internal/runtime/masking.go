package runtime

import (
	"encoding/json"
	"strings"

	"example.com/independent-harness/internal/core"
)

// maskMinBytes leaves short tool results intact; they cost little and often
// carry the facts (hashes, exit codes) a later turn needs.
const maskMinBytes = 512

const maskPrefix = `{"masked":true`

// maskObservations returns a copy of history in which tool results before cut
// are replaced by a short placeholder. Call/result pairing is preserved, the
// full results stay in the journal, and recent groups after cut are untouched.
// This follows the observation-masking baseline in "The Complexity Trap"
// (arXiv:2508.21433); see research_notes for the evidence and its limits.
func maskObservations(history []core.Message, cut int) ([]core.Message, int) {
	names := map[string]string{}
	out := append([]core.Message(nil), history...)
	masked := 0
	for i := 1; i < cut && i < len(out); i++ {
		m := out[i]
		for _, c := range m.ToolCalls {
			names[c.ID] = c.Function.Name
		}
		if m.Role != "tool" || len(m.Content) < maskMinBytes || strings.HasPrefix(m.Content, maskPrefix) {
			continue
		}
		// A struct keeps "masked" first, which the prefix check relies on.
		b, _ := json.Marshal(struct {
			Masked bool   `json:"masked"`
			Tool   string `json:"tool"`
			Bytes  int    `json:"bytes"`
			Note   string `json:"note"`
		}{true, names[m.ToolCallID], len(m.Content), "Older tool output omitted to save context. It is kept in the session journal; run the tool again if you need it."})
		m.Content = string(b)
		out[i] = m
		masked++
	}
	return out, masked
}
