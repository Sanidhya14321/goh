package session

import (
	"encoding/json"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions")
	s, err := New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dir
}
func TestDurableRoundTripAndUncertainEffects(t *testing.T) {
	s, dir := fixture(t)
	if err := s.Append("message", map[string]string{"role": "user", "content": "edit file"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"done", "uncertain"} {
		if err := s.Append("tool_start", map[string]string{"id": id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Append("tool_result", map[string]string{"id": "done"}); err != nil {
		t.Fatal(err)
	}
	id := s.ID
	s.Close()
	reopened, err := Open(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got := strings.Join(reopened.Pending(), ","); got != "uncertain" {
		t.Fatalf("pending %q", got)
	}
	if err := reopened.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	if len(reopened.Pending()) != 0 {
		t.Fatal("acknowledgment not durable")
	}
	records := reopened.Records()
	records[0].Data[0] = '!'
	if !json.Valid(reopened.Records()[0].Data) {
		t.Fatal("record copy aliases store")
	}
	info, err := os.Stat(filepath.Join(dir, id+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if testutil.UnixModes() && info.Mode().Perm() != 0600 {
		t.Fatalf("permissions %o", info.Mode().Perm())
	}
	listed, err := List(dir)
	if err != nil || len(listed) != 1 || listed[0].ID != id {
		t.Fatalf("list %v %v", listed, err)
	}
}
func TestConflictingWriters(t *testing.T) {
	s, dir := fixture(t)
	other, err := Open(dir, s.ID)
	if err == nil {
		other.Close()
		t.Fatal("second writer admitted")
	}
	s.Close()
	other, err = Open(dir, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
}
func TestTornTailRecovery(t *testing.T) {
	for _, tail := range []string{`{"version":1,"type":`, `{"version":1,"type":"message","data":{}}`} {
		t.Run(tail, func(t *testing.T) {
			s, dir := fixture(t)
			id := s.ID
			s.Close()
			f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(tail); err != nil {
				t.Fatal(err)
			}
			f.Close()
			reopened, err := Open(dir, id)
			if err != nil {
				t.Fatal(err)
			}
			if got := reopened.Records(); len(got) != 2 || got[1].Type != "recovery" {
				t.Fatalf("records %#v", got)
			}
			reopened.Close()
			again, err := Open(dir, id)
			if err != nil {
				t.Fatal(err)
			}
			again.Close()
		})
	}
}
func TestRejectCorruptionAndVersions(t *testing.T) {
	for _, line := range []string{"invalid\n", "{}\n", "{\"version\":2,\"type\":\"message\",\"data\":{}}\n", "{\"version\":1,\"type\":\"message\",\"data\":{}} garbage\n"} {
		t.Run(line, func(t *testing.T) {
			s, dir := fixture(t)
			id := s.ID
			s.Close()
			path := filepath.Join(dir, id+".jsonl")
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			f.WriteString(line + `{"unfinished"`)
			f.Close()
			before, _ := os.ReadFile(path)
			if opened, err := Open(dir, id); err == nil {
				opened.Close()
				t.Fatal("corrupt journal opened")
			}
			after, _ := os.ReadFile(path)
			if string(after) != string(before) {
				t.Fatal("corruption modified journal")
			}
		})
	}
}
func TestRejectTraversalAndSymlink(t *testing.T) {
	_, dir := fixture(t)
	for _, id := range []string{"../x", "a/b", "", "."} {
		if _, err := Open(dir, id); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	path := filepath.Join(dir, "linked.jsonl")
	if err := os.Symlink(filepath.Join(dir, "target"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir, "linked"); err == nil {
		t.Fatal("accepted symlink")
	}
}
func TestAppendAfterClose(t *testing.T) {
	s, _ := fixture(t)
	s.Close()
	if err := s.Append("message", "text"); err == nil {
		t.Fatal("write to closed store succeeded")
	}
}
func FuzzJournal(f *testing.F) {
	f.Add([]byte("{\"version\":1,\"type\":\"message\",\"data\":{}}\n"))
	f.Add([]byte("{torn"))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1<<20 {
			t.Skip()
		}
		_, _, _, _ = readRecords(strings.NewReader(string(input)))
	})
}
