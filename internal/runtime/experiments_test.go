package runtime

import (
	"context"
	"encoding/json"
	"fmt"
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

// recoverable-compression: masked receipts carry a recall id bound to the
// exact recorded result, pages are lossless UTF-8 and bounded, and a reused
// provider call id cannot resolve to another result.
func TestRecallMaskedObservation(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Experiments = []string{core.ExperimentRecoverableCompression}
	big := strings.Repeat("ab€", 3000) // 3-byte runes cross page boundaries
	call := toolCall("k1", "read_file", map[string]any{"path": "x"})
	for _, m := range []core.Message{{Role: "user", Content: "go"}, {Role: "assistant", ToolCalls: []core.Call{call}}, {Role: "tool", ToolCallID: "k1", Content: big}} {
		a.commit(m)
	}
	view, n := maskObservations(append(a.History, core.Message{Role: "user", Content: "next"}), len(a.History))
	view = withRecallIDs(view, a.History)
	id := recallID("k1", big)
	if n == 0 || !strings.Contains(view[len(view)-2].Content, `"recall_id":"`+id+`"`) {
		t.Fatalf("receipt %s", view[len(view)-2].Content)
	}
	var got strings.Builder
	offset := 0
	for pages := 0; ; pages++ {
		r := a.executeRecall(json.RawMessage(fmt.Sprintf(`{"id":%q,"offset":%d}`, id, offset)))
		var page struct {
			Content string `json:"content"`
			Next    int    `json:"next_offset"`
			More    bool   `json:"more"`
		}
		if r.Error || json.Unmarshal([]byte(r.Content), &page) != nil || pages > 10 {
			t.Fatal(r.Content)
		}
		got.WriteString(page.Content)
		if !page.More {
			break
		}
		offset = page.Next
	}
	if got.String() != big {
		t.Fatal("pages lost or changed bytes")
	}
	if r := a.executeRecall(json.RawMessage(fmt.Sprintf(`{"id":%q,"offset":9223372036854775807}`, id))); r.Error || !strings.Contains(r.Content, `"content":""`) {
		t.Fatalf("huge offset: %+v", r)
	}
	// The provider reuses k1 for a later, different result: the old receipt
	// still resolves to the old content only.
	a.commit(core.Message{Role: "assistant", ToolCalls: []core.Call{call}})
	a.commit(core.Message{Role: "tool", ToolCallID: "k1", Content: "newer output"})
	r := a.executeRecall(json.RawMessage(fmt.Sprintf(`{"id":%q}`, id)))
	if r.Error || strings.Contains(r.Content, "newer output") {
		t.Fatalf("receipt resolved to another result: %s", r.Content)
	}
	if r := a.executeRecall(json.RawMessage(`{"id":"k1"}`)); !r.Error {
		t.Fatal("a bare call id resolved")
	}
}

// Follow-up review: two submit_evidence calls in one reply run serially;
// the first is recorded, the second is told it already is (race-free).
func TestEvidenceCallsInOneReply(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Config.Experiments = []string{core.ExperimentSubmissionEvidence}
	a.Grants["create_file"] = true
	ev := func(id string) core.Call {
		return toolCall(id, "submit_evidence", map[string]any{"intended_behavior": "x", "sources_inspected": []string{}, "test_scope": "none", "checks_run": []map[string]string{}, "unresolved_gaps": []string{}})
	}
	a.Provider = scripted(
		[]core.Call{toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "x\n"})},
		[]core.Call{ev("e1"), ev("e2")},
	)
	a.Run(context.Background(), "go")
	recorded := 0
	for _, r := range a.Store.Records() {
		if r.Type == "submission_evidence" {
			recorded++
		}
	}
	if recorded != 1 {
		t.Fatalf("records %d", recorded)
	}
}
