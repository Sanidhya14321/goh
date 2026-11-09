package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func journalWith(t *testing.T, dir, project string, messages ...core.Message) string {
	t.Helper()
	s, err := New(dir, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range messages {
		if err := s.Append("message", m); err != nil {
			t.Fatal(err)
		}
	}
	id := s.ID
	s.Close()
	return id
}

// corruptLine replaces journal line n (0-based) with text, keeping the rest.
func corruptLine(t *testing.T, dir, id string, n int, text string) {
	t.Helper()
	p := filepath.Join(dir, id+".jsonl")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(b), "\n")
	lines[n] = text + "\n"
	if err := os.WriteFile(p, []byte(strings.Join(lines, "")), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListSurvivesOneDamagedJournal(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	good := journalWith(t, dir, project, core.Message{Role: "user", Content: "fine"})
	bad := journalWith(t, dir, project, core.Message{Role: "user", Content: "x"})
	corruptLine(t, dir, bad, 0, "garbage header")
	infos, err := List(dir)
	if err != nil || len(infos) != 1 || infos[0].ID != good {
		t.Fatal("healthy session hidden by a damaged one", infos, err)
	}
	_, damaged, err := ListAll(dir)
	if err != nil || len(damaged) != 1 || damaged[0].ID != bad || !strings.Contains(damaged[0].Problem, "header") {
		t.Fatal(damaged, err)
	}
}

func TestSalvageKeepsConversationBeforeDamage(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	id := journalWith(t, dir, project,
		core.Message{Role: "user", Content: "first"},
		core.Message{Role: "assistant", Content: "answer one"},
		core.Message{Role: "user", Content: "second"},
		core.Message{Role: "assistant", Content: "answer two"})
	// Line 0 is session_start; damage the "second" prompt (line 3).
	corruptLine(t, dir, id, 3, `{"version":1,"type":"message","data":{"role":"user"`)
	if _, err := Open(dir, id); err == nil {
		t.Fatal("damaged journal opened normally")
	}
	original, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	store, report, err := Salvage(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if report.Source != id || report.Session != store.ID || report.DamagedAt == 0 || !strings.Contains(report.Problem, "unreadable record") {
		t.Fatalf("%+v", report)
	}
	turns := store.RecentUserTurns(10)
	if len(turns) != 1 || turns[0].Prompt != "first" {
		t.Fatal("salvage should keep exactly the turns before the damage", turns)
	}
	after, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	if string(after) != string(original) {
		t.Fatal("salvage modified the damaged journal")
	}
}

func TestSalvageDropsIncompleteToolGroup(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	call := core.Call{ID: "c1", Type: "function"}
	call.Function.Name, call.Function.Arguments = "read_file", `{"path":"x"}`
	id := journalWith(t, dir, project,
		core.Message{Role: "user", Content: "read it"},
		core.Message{Role: "assistant", ToolCalls: []core.Call{call}},
		core.Message{Role: "tool", ToolCallID: "c1", Content: "contents"})
	corruptLine(t, dir, id, 3, "NOT JSON") // the tool result is lost
	store, report, err := Salvage(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if report.DroppedGroup == 0 {
		t.Fatal("an unanswered tool call must be dropped", report)
	}
	for _, r := range store.Records() {
		if r.Type != "message" {
			continue
		}
		var m core.Message
		json.Unmarshal(r.Data, &m)
		if len(m.ToolCalls) > 0 {
			t.Fatal("salvaged session kept a tool call without its result")
		}
	}
}

func TestSalvageRefusesIntactOrHeaderless(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	intact := journalWith(t, dir, project, core.Message{Role: "user", Content: "ok"})
	if _, _, err := Salvage(dir, intact); err == nil || !strings.Contains(err.Error(), "not damaged") {
		t.Fatal(err)
	}
	headless := journalWith(t, dir, project, core.Message{Role: "user", Content: "ok"})
	corruptLine(t, dir, headless, 0, "garbage")
	if _, _, err := Salvage(dir, headless); err == nil || !strings.Contains(err.Error(), "header is damaged") {
		t.Fatal(err)
	}
}
