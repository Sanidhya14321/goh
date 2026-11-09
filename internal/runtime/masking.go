package runtime

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
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
		b, _ := json.Marshal(maskReceipt(names[m.ToolCallID], m.Content))
		m.Content = string(b)
		out[i] = m
		masked++
	}
	return out, masked
}

// readOnlyTools can be run again safely to see their output.
var readOnlyTools = map[string]bool{"read_file": true, "grep": true, "glob": true, "list_files": true, "search": true,
	"repo_map": true, "lsp": true, "read_artifact": true, "memory_recall": true, "skill": true, "snapshot_list": true,
	"snapshot_diff": true, "formatter_status": true, "job_output": true}

// receipt replaces a masked tool result. It keeps the facts later turns
// rely on (status, exit code, path and hash, artifact id) so the model never
// needs to repeat an action just to recall what happened.
type receipt struct {
	Masked     bool   `json:"masked"` // first, for the maskPrefix check
	Tool       string `json:"tool"`
	Bytes      int    `json:"bytes"`
	Error      bool   `json:"error,omitempty"`
	Status     string `json:"status,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	Path       string `json:"path,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	ArtifactID string `json:"artifact_id,omitempty"`
	Note       string `json:"note"`
}

func maskReceipt(tool, content string) receipt {
	r := receipt{Masked: true, Tool: tool, Bytes: len(content)}
	var envelope struct {
		Content string `json:"content"`
		Error   bool   `json:"error"`
	}
	inner := content
	if json.Unmarshal([]byte(content), &envelope) == nil && envelope.Content != "" {
		r.Error, inner = envelope.Error, envelope.Content
	}
	var fields struct {
		Status     string `json:"status"`
		ExitCode   *int   `json:"exit_code"`
		Path       string `json:"path"`
		SHA256     string `json:"sha256"`
		ArtifactID string `json:"artifact_id"`
	}
	if json.Unmarshal([]byte(inner), &fields) == nil {
		r.Status, r.ExitCode, r.Path, r.SHA256, r.ArtifactID = fields.Status, fields.ExitCode, fields.Path, fields.SHA256, fields.ArtifactID
	}
	switch {
	case readOnlyTools[tool]:
		r.Note = "Older output omitted to save context; it is kept in the session journal. Run this read-only tool again if you need the details."
	case r.ArtifactID != "":
		r.Note = "Older output omitted to save context. This action already ran: do not run it again to see its output; use read_artifact with artifact_id."
	default:
		r.Note = "Older output omitted to save context. This action already ran and its effects stand: do not run it again just to see its output."
	}
	return r
}

// shrinkLargeResults replaces tool results larger than limit bytes with a
// receipt plus the first part of the output, so one oversized result cannot
// end a run. All results in history are complete (call/result pairs stay).
func shrinkLargeResults(history []core.Message, limit int) ([]core.Message, int) {
	if limit < 1024 {
		limit = 1024
	}
	names := map[string]string{}
	out := append([]core.Message(nil), history...)
	shrunk := 0
	for i, m := range out {
		for _, c := range m.ToolCalls {
			names[c.ID] = c.Function.Name
		}
		if m.Role != "tool" || len(m.Content) <= limit || strings.HasPrefix(m.Content, maskPrefix) {
			continue
		}
		r := maskReceipt(names[m.ToolCallID], m.Content)
		head := m.Content[:limit/2]
		for len(head) > 0 && !utf8.ValidString(head) {
			head = head[:len(head)-1]
		}
		r.Note = "Output too large for the context window; only the beginning is shown. " + r.Note
		if names[m.ToolCallID] == "read_file" {
			r.Note += " Use read_file with start_line and end_line to page through the file."
		}
		b, _ := json.Marshal(struct {
			receipt
			Head string `json:"head"`
		}{r, head})
		m.Content = string(b)
		out[i] = m
		shrunk++
	}
	return out, shrunk
}

// maskKeepRecent is how many of the newest tool results masking tries to
// leave intact.
const maskKeepRecent = 10

// recentWindowCut returns the history index before which masking may act so
// that the newest keep tool results stay visible; never later than cut.
func recentWindowCut(history []core.Message, cut, keep int) int {
	seen := 0
	for i := len(history) - 1; i > 0; i-- {
		if history[i].Role == "tool" {
			if seen++; seen == keep {
				if i < cut {
					return i
				}
				return cut
			}
		}
	}
	return 1 // fewer than keep results: nothing needs masking for the window
}
