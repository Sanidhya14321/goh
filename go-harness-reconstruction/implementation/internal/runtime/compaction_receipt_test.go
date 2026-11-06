package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tools"
)

func TestCompactionReceiptRetainsEvidenceAcrossPortableRoundTrip(t *testing.T) {
	a := contextAgent(t)
	a.Config.Model = "fixture"
	digest := strings.Repeat("a", 64)
	commit := func(message core.Message) {
		t.Helper()
		if err := a.commit(message); err != nil {
			t.Fatal(err)
		}
	}
	recordTool := func(name string, result tools.Result) {
		t.Helper()
		call := core.Call{ID: ID(), Type: "function"}
		call.Function.Name, call.Function.Arguments = name, "{}"
		commit(core.Message{Role: "assistant", ToolCalls: []core.Call{call}})
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		commit(core.Message{Role: "tool", ToolCallID: call.ID, Content: string(raw)})
	}
	commit(core.Message{Role: "user", Content: "repair the parser"})
	recordTool("apply_patch", tools.Result{Content: `{"files":[{"path":"old.go","destination":"parser.go","sha256":"` + digest + `"}]}`})
	recordTool("shell", tools.Result{Error: true, Content: `{"exit_code":2,"output":"raw diagnostic should stay in the journal"}`})
	commit(core.Message{Role: "assistant", Content: "recent group 1"})
	commit(core.Message{Role: "assistant", Content: "recent group 2"})

	a.Provider = &summaryModel{summary: "Work in progress."}
	if err := a.compactWithMode(context.Background(), nil, true); err != nil {
		t.Fatal(err)
	}
	records := a.Store.Records()
	var saved session.Compaction
	if err := json.Unmarshal(records[len(records)-1].Data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Receipt == nil || len(saved.Receipt.Evidence) != 2 || saved.Model != "fixture" || saved.After != estimate(a.modelHistory(), nil) {
		t.Fatalf("unexpected compaction state: %+v", saved)
	}
	summary := a.History[1].Content
	if !strings.Contains(summary, `"path":"parser.go"`) || !strings.Contains(summary, `"exit_code":2`) ||
		strings.Contains(summary, "raw diagnostic") || strings.Contains(summary, "old.go") {
		t.Fatalf("unexpected evidence projection: %s", summary)
	}
	id := a.Store.ID
	if err := a.Store.Close(); err != nil {
		t.Fatal(err)
	}
	var portable bytes.Buffer
	if err := session.Export(a.Config.SessionDir, id, &portable); err != nil {
		t.Fatal(err)
	}
	imported, err := session.Import(t.TempDir(), &portable, "")
	if err != nil {
		t.Fatal(err)
	}
	defer imported.Close()
	reopened, err := New(a.Config, imported)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.History[1].Content != summary {
		t.Fatal("receipt-backed working view changed across export and import")
	}
}

func TestResumeRejectsCompactionReceiptMismatch(t *testing.T) {
	a := contextAgent(t)
	state := session.Compaction{
		Messages: []core.Message{{Role: "system", Content: "framing"}, {Role: "user", Content: "task"}},
	}
	if err := state.SealReceipt(nil); err != nil {
		t.Fatal(err)
	}
	state.Messages[1].Content = "changed task"
	if err := a.Store.Append("context_compacted", state); err != nil {
		t.Fatal(err)
	}
	if _, err := New(a.Config, a.Store); err == nil || !strings.Contains(err.Error(), "compaction receipt") {
		t.Fatalf("changed compacted view accepted: %v", err)
	}
}
