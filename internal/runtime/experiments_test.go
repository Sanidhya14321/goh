package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func hasTool(defs []core.Tool, name string) bool {
	for _, d := range defs {
		if d.Function.Name == name {
			return true
		}
	}
	return false
}

// Experiment off: no new tools, so the baseline arm sees the same request.
func TestExperimentsOffLeaveBaselineUnchanged(t *testing.T) {
	a, _ := testAgent(t)
	defs := a.localDefinitions()
	if hasTool(defs, "submit_evidence") || hasTool(defs, "recall_observation") {
		t.Fatal("experiment tools offered without the experiment")
	}
	a.Config.Experiments = []string{core.ExperimentSubmissionEvidence, core.ExperimentRecoverableCompression}
	defs = a.localDefinitions()
	if !hasTool(defs, "submit_evidence") || !hasTool(defs, "recall_observation") {
		t.Fatal("experiment tools missing")
	}
	if err := (core.Config{Experiments: []string{"nope"}}).Validate(); err == nil || !strings.Contains(err.Error(), "unknown experiment") {
		t.Fatal(err)
	}
}

// submission-evidence: changed work is asked for a structured record once;
// the record is journaled against the current revision, and claimed checks
// that did not run in this run are named.
func TestSubmissionEvidenceGate(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Config.CompletionCheck = nil
	a.Config.Experiments = []string{core.ExperimentSubmissionEvidence}
	a.Grants["create_file"], a.Grants["shell"] = true, true
	asked, turn := 0, 0
	var evidenceReply string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		last := h[len(h)-1]
		switch {
		case last.Automatic && strings.Contains(last.Content, "submit_evidence"):
			asked++
			return answer(toolCall("ev", "submit_evidence", map[string]any{
				"intended_behavior": "new.txt exists", "sources_inspected": []string{"new.txt"}, "test_scope": "file presence",
				"checks_run":      []map[string]string{{"command": "test -f new.txt", "result": "exit 0"}, {"command": "go test ./...", "result": "ok"}},
				"unresolved_gaps": []string{},
			})), nil
		case last.Role == "tool" && last.ToolCallID == "ev":
			evidenceReply = last.Content
			return answer(), nil
		case turn == 1:
			return answer(toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})), nil
		case turn == 2:
			return answer(toolCall("t", "shell", map[string]any{"command": "test -f new.txt"})), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "go"); status != "completed" {
		t.Fatal(status, err)
	}
	if asked != 1 || !strings.Contains(evidenceReply, `checks_not_run_in_this_run`) || !strings.Contains(evidenceReply, `go test ./...`) {
		t.Fatalf("asked %d reply %s", asked, evidenceReply)
	}
	found := false
	for _, r := range a.Store.Records() {
		if r.Type == "submission_evidence" {
			var d struct {
				Revision string   `json:"revision"`
				Observed []string `json:"checks_observed"`
			}
			json.Unmarshal(r.Data, &d)
			found = d.Revision != "" && d.Revision == a.fingerprint && len(d.Observed) == 1
		}
	}
	if !found {
		t.Fatal("evidence not journaled against the current revision")
	}
}

// recoverable-compression: masked receipts carry a recall id and the exact
// output comes back from this session's journal, in bounded pages.
func TestRecallMaskedObservation(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Experiments = []string{core.ExperimentRecoverableCompression}
	big := strings.Repeat("line of output\n", 1000)
	call := toolCall("k1", "read_file", map[string]any{"path": "x"})
	for _, m := range []core.Message{{Role: "user", Content: "go"}, {Role: "assistant", ToolCalls: []core.Call{call}}, {Role: "tool", ToolCallID: "k1", Content: big}} {
		a.commit(m)
	}
	view, n := maskObservations(append(a.History, core.Message{Role: "user", Content: "next"}), len(a.History))
	view = withRecallIDs(view)
	if n == 0 || !strings.Contains(view[len(view)-2].Content, `"recall_id":"k1"`) {
		t.Fatalf("receipt %s", view[len(view)-2].Content)
	}
	r := a.executeRecall(json.RawMessage(`{"id":"k1","offset":8192}`))
	var page struct {
		Content string `json:"content"`
		Total   int    `json:"total_bytes"`
		More    bool   `json:"more"`
	}
	json.Unmarshal([]byte(r.Content), &page)
	if r.Error || page.Total != len(big) || page.Content != big[8192:] || page.More {
		t.Fatalf("second page: %+v", page.More)
	}
	json.Unmarshal([]byte(a.executeRecall(json.RawMessage(`{"id":"k1"}`)).Content), &page)
	if page.Content != big[:8192] || !page.More {
		t.Fatal("first page")
	}
	if r := a.executeRecall(json.RawMessage(`{"id":"other"}`)); !r.Error {
		t.Fatal("recalled an unknown id")
	}
}
