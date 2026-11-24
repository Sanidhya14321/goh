package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func weighContent(m core.Message) int { return len(m.Content) + 64 }

func TestRecentMessagesProjection(t *testing.T) {
	s, _ := fixture(t)
	user := func(text string) core.Message { return core.Message{Role: "user", Content: text} }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.appendAt("message", user("same prompt"), 1000))
	must(s.appendAt("message", core.Message{Role: "assistant", Content: strings.Repeat("old answer ", 400)}, 1100))
	// A high-volume region with no messages is skipped without decoding.
	for i := range 300 {
		must(s.appendAt("tool_result", map[string]any{"id": fmt.Sprint(i), "result": strings.Repeat("r", 500)}, 1200))
	}
	must(s.appendAt("message", user("legacy prompt"), 0)) // a version 1 record: no time
	var call core.Call
	call.ID, call.Function.Name, call.Function.Arguments = "c1", "read_file", `{"path":"a.go"}`
	must(s.appendAt("message", core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, 3000))
	must(s.appendAt("message", core.Message{Role: "tool", ToolCallID: "c1", Content: "package a"}, 3100))
	must(s.appendAt("message", user("same prompt"), 5000))

	all := s.RecentMessages(1<<20, weighContent)
	if len(all) != 6 || all[0].Time != 1000 || all[5].Time != 5000 || all[2].Time != 0 {
		t.Fatalf("full projection: %+v", all)
	}
	// Order is journal order: the call before its result.
	if all[3].Message.ToolCalls[0].ID != "c1" || all[4].Message.ToolCallID != "c1" || all[3].Index >= all[4].Index {
		t.Fatal("tool call and result out of order")
	}
	// A budget that drops the old answer also drops the earlier duplicate,
	// and the remaining duplicate keeps its own time.
	recent := s.RecentMessages(1000, weighContent)
	if len(recent) != 4 || recent[0].Message.Content != "legacy prompt" || recent[3].Time != 5000 {
		t.Fatalf("bounded projection: %+v", recent)
	}
	// One record larger than the budget is still returned (never nothing).
	must(s.appendAt("message", core.Message{Role: "assistant", Content: strings.Repeat("x", 5000)}, 6000))
	if last := s.RecentMessages(100, weighContent); len(last) != 1 || last[0].Time != 6000 {
		t.Fatalf("oversized last record: %d", len(last))
	}
	if s.Len() != len(s.Records()) {
		t.Fatal("Len disagrees with Records")
	}
	if r, ok := s.Last("tool_result", "session_start"); !ok || r.Type != "tool_result" {
		t.Fatal("Last", r.Type)
	}
	if _, ok := s.Last("no_such_type"); ok {
		t.Fatal("Last found a missing type")
	}
}

// journalOf writes a session of n records (every fourth a message, about
// 1 KiB each) straight to disk, so a benchmark need not sync each append.
func journalOf(b *testing.B, n int) (dir, id string) {
	dir = filepath.Join(b.TempDir(), "sessions")
	s, err := New(dir, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	id = s.ID
	s.Close()
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		b.Fatal(err)
	}
	w := bufio.NewWriter(f)
	body := strings.Repeat("y", 900)
	for i := range n {
		var data any = map[string]any{"id": fmt.Sprint(i), "result": body}
		kind := "tool_result"
		if i%4 == 0 {
			kind, data = "message", core.Message{Role: "assistant", Content: fmt.Sprintf("answer %d %s", i, body)}
		}
		encoded, _ := json.Marshal(data)
		line, _ := json.Marshal(Record{Version: Version, Type: kind, Data: encoded, Time: int64(i + 1)})
		w.Write(append(line, '\n'))
	}
	if err := w.Flush(); err != nil {
		b.Fatal(err)
	}
	f.Close()
	return dir, id
}

// The tail the reopened chat shows (the TUI's 256 KiB budget).
const benchBudget = 256 << 10

func benchmarkProjection(b *testing.B, n int) {
	dir, id := journalOf(b, n)
	s, err := OpenReadOnly(dir, id, "")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if len(s.RecentMessages(benchBudget, weighContent)) == 0 {
			b.Fatal("empty")
		}
	}
}

// benchmarkCopyAll is what the projection replaces: copy every record, then
// decode every message.
func benchmarkCopyAll(b *testing.B, n int) {
	dir, id := journalOf(b, n)
	s, err := OpenReadOnly(dir, id, "")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		var messages []core.Message
		for _, r := range s.Records() {
			if r.Type == "message" {
				var m core.Message
				json.Unmarshal(r.Data, &m)
				messages = append(messages, m)
			}
		}
	}
}

func benchmarkOpenAndProject(b *testing.B, n int) {
	dir, id := journalOf(b, n)
	b.ReportAllocs()
	for b.Loop() {
		s, err := OpenReadOnly(dir, id, "")
		if err != nil {
			b.Fatal(err)
		}
		s.RecentMessages(benchBudget, weighContent)
	}
}

func BenchmarkRecentMessages1k(b *testing.B)         { benchmarkProjection(b, 1000) }
func BenchmarkRecentMessages20k(b *testing.B)        { benchmarkProjection(b, 20000) }
func BenchmarkCopyAllMessages1k(b *testing.B)        { benchmarkCopyAll(b, 1000) }
func BenchmarkCopyAllMessages20k(b *testing.B)       { benchmarkCopyAll(b, 20000) }
func BenchmarkOpenAndRecentMessages1k(b *testing.B)  { benchmarkOpenAndProject(b, 1000) }
func BenchmarkOpenAndRecentMessages20k(b *testing.B) { benchmarkOpenAndProject(b, 20000) }
