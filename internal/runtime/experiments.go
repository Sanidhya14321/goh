package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// Research prototypes (core.Experiment*). Each is off by default and leaves
// the default behaviour untouched; see docs/benchmarks for how to compare an
// arm with the baseline.

// --- submission-evidence -------------------------------------------------

func submitEvidenceDefinition() core.Tool {
	str := map[string]any{"type": "string"}
	list := map[string]any{"type": "array", "items": str, "maxItems": 50}
	check := map[string]any{"type": "object", "properties": map[string]any{"command": str, "result": str}, "required": []string{"command", "result"}, "additionalProperties": false}
	return core.NewTool("submit_evidence", "Before finishing changed work, record the evidence for the current changes: the intended behaviour, the source you inspected (files, callers, real definitions), the test scope, the checks you actually ran in this run with their results, and any gaps left unverified. Recorded against the current revision; it does not change files.", map[string]any{"type": "object", "properties": map[string]any{
		"intended_behavior": str, "sources_inspected": list, "test_scope": str,
		"checks_run":      map[string]any{"type": "array", "items": check, "maxItems": 20},
		"unresolved_gaps": list,
	}, "required": []string{"intended_behavior", "sources_inspected", "test_scope", "checks_run", "unresolved_gaps"}, "additionalProperties": false})
}

type evidenceRecord struct {
	IntendedBehavior string   `json:"intended_behavior"`
	SourcesInspected []string `json:"sources_inspected"`
	TestScope        string   `json:"test_scope"`
	ChecksRun        []struct {
		Command string `json:"command"`
		Result  string `json:"result"`
	} `json:"checks_run"`
	UnresolvedGaps []string `json:"unresolved_gaps"`
}

func parseEvidence(b json.RawMessage) (evidenceRecord, error) {
	var e evidenceRecord
	if len(b) > 64<<10 || json.Unmarshal(b, &e) != nil {
		return e, errors.New("submit_evidence needs a JSON object (at most 64 KiB)")
	}
	if strings.TrimSpace(e.IntendedBehavior) == "" || strings.TrimSpace(e.TestScope) == "" {
		return e, errors.New("submit_evidence needs intended_behavior and test_scope")
	}
	return e, nil
}

// executeEvidence journals the record against the current revision and
// says which claimed checks this run actually executed.
func (a *Agent) executeEvidence(b json.RawMessage) tools.Result {
	e, err := parseEvidence(b)
	if err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	if a.evidenceSubmitted && a.evidenceRevision == a.fingerprint && !a.noFingerprint {
		// One accepted record per revision; calls run serially (prepare
		// declares a state change), so the first one wins deterministically.
		out, _ := json.Marshal(map[string]any{"recorded": false, "revision": a.fingerprint, "note": "evidence for this revision is already recorded"})
		return tools.Result{Content: string(out)}
	}
	ran := map[string]bool{}
	for _, c := range a.runCommands {
		ran[strings.TrimSpace(c)] = true
	}
	var observed, unobserved []string
	for _, c := range e.ChecksRun {
		if ran[strings.TrimSpace(c.Command)] {
			observed = append(observed, c.Command)
		} else {
			unobserved = append(unobserved, c.Command)
		}
	}
	revision := a.fingerprint
	if a.noFingerprint {
		revision = ""
	}
	record := map[string]any{"evidence": e, "revision": revision, "checks_observed": observed, "checks_not_observed": unobserved, "current_check": a.evidence}
	if err := a.Store.Append("submission_evidence", record); err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	a.evidenceRevision = revision
	a.evidenceSubmitted = true
	a.event("submission_evidence", record)
	reply := map[string]any{"recorded": true, "revision": revision, "checks_observed_in_this_run": observed}
	if len(unobserved) > 0 {
		reply["checks_not_run_in_this_run"] = unobserved
		reply["note"] = "These claimed checks were not run in this run; run them or remove the claim before finishing."
	}
	out, _ := json.Marshal(reply)
	return tools.Result{Content: string(out)}
}

const evidencePrompt = "[Automatic check, not a new request] Before finishing, call submit_evidence for your current changes: the intended behaviour, the source you inspected, the test scope, the checks you ran in this run with their results, and any unresolved gaps. Run any check you would otherwise only claim. Then finish."

// needsEvidence reports whether the submission-evidence gate applies: the
// experiment is on, this is a top-level build run that changed files, and
// no record covers the current revision yet. It asks once per run.
func (a *Agent) needsEvidence() bool {
	if !a.Config.Experiment(core.ExperimentSubmissionEvidence) || a.depth > 0 || a.isPlan() || a.completionChecked {
		return false
	}
	if !a.mayHaveChanged() {
		return false
	}
	return !a.evidenceSubmitted || a.evidenceRevision != a.fingerprint
}

// --- recoverable-compression ---------------------------------------------

func recallDefinition() core.Tool {
	return core.NewTool("recall_observation", "Recall the exact output of an earlier tool result that was masked to save context, by its recall_id (the tool call id in the masked receipt). Returns up to 8192 bytes from offset; only results recorded in this session.", map[string]any{"type": "object", "properties": map[string]any{
		"id":     map[string]any{"type": "string"},
		"offset": map[string]any{"type": "integer", "minimum": 0},
	}, "required": []string{"id"}, "additionalProperties": false})
}

const recallPage = 8192

// recallID binds a receipt to one exact recorded result: the tool call id
// (which a provider may reuse across turns) plus a digest of the content.
func recallID(callID, content string) string {
	sum := sha256.Sum256([]byte(content))
	return callID + "#" + hex.EncodeToString(sum[:8])
}

// executeRecall returns a page of a recorded tool result from this
// session's journal: the same data the model already saw, under the same
// session, never another session's or a file it was not shown. A result
// read from a file that file_rules deny now is not returned.
func (a *Agent) executeRecall(b json.RawMessage) tools.Result {
	var args struct {
		ID     string `json:"id"`
		Offset int64  `json:"offset"`
	}
	if json.Unmarshal(b, &args) != nil || args.ID == "" || args.Offset < 0 {
		return tools.Result{Error: true, Content: "recall_observation needs an id"}
	}
	callID, _, _ := strings.Cut(args.ID, "#")
	content, found, path := "", false, ""
	paths := map[string]string{} // call id -> path its tool read
	for _, r := range a.Store.Records() {
		switch r.Type {
		case "tool_start":
			var d struct {
				ID        string `json:"id"`
				Arguments struct {
					Path string `json:"path"`
				} `json:"arguments"`
			}
			if json.Unmarshal(r.Data, &d) == nil {
				paths[d.ID] = d.Arguments.Path
			}
		case "message":
			var m core.Message
			if json.Unmarshal(r.Data, &m) == nil && m.Role == "tool" && m.ToolCallID == callID && !strings.HasPrefix(m.Content, maskPrefix) && recallID(m.ToolCallID, m.Content) == args.ID {
				content, found, path = m.Content, true, paths[m.ToolCallID]
			}
		}
	}
	if !found {
		return tools.Result{Error: true, Content: "no recorded tool result with that recall_id in this session"}
	}
	if path != "" && a.Tools != nil && a.Tools.Denied(filepath.ToSlash(path)) {
		return tools.Result{Error: true, Content: "that result came from a file your file_rules now deny"}
	}
	// Clamp before any arithmetic, then keep pages on UTF-8 boundaries so
	// nothing is lost or replaced between pages.
	start := int(min(args.Offset, int64(len(content))))
	for start < len(content) && !utf8.RuneStart(content[start]) {
		start++
	}
	end := min(len(content), start+recallPage)
	for end < len(content) && end > start && !utf8.RuneStart(content[end]) {
		end--
	}
	out, _ := json.Marshal(map[string]any{"id": args.ID, "offset": start, "next_offset": end, "total_bytes": len(content), "more": end < len(content), "content": content[start:end]})
	return tools.Result{Content: string(out)}
}

// withRecallIDs adds the recall id to masked receipts in view (only under
// the experiment; the baseline receipts are unchanged). original is the
// history view was masked from, index for index.
func withRecallIDs(view, original []core.Message) []core.Message {
	out := append([]core.Message(nil), view...)
	for i, m := range out {
		if m.Role != "tool" || !strings.HasPrefix(m.Content, maskPrefix) || i >= len(original) || original[i].ToolCallID != m.ToolCallID {
			continue
		}
		var r map[string]any
		if json.Unmarshal([]byte(m.Content), &r) != nil {
			continue
		}
		r["recall_id"] = recallID(m.ToolCallID, original[i].Content)
		r["note"] = fmt.Sprint(r["note"]) + " Call recall_observation with recall_id for the exact output."
		if b, err := json.Marshal(r); err == nil {
			out[i].Content = string(b)
		}
	}
	return out
}

// experimentDefinitions are the tools the enabled experiments add.
func (a *Agent) experimentDefinitions() []core.Tool {
	var defs []core.Tool
	if a.Config.Experiment(core.ExperimentSubmissionEvidence) && a.depth == 0 && !a.isPlan() {
		defs = append(defs, submitEvidenceDefinition())
	}
	if a.Config.Experiment(core.ExperimentRecoverableCompression) {
		defs = append(defs, recallDefinition())
	}
	return defs
}
