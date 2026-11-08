package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rewriteAsV1 strips times and marks every record version 1, as written by
// releases before record timestamps.
func rewriteAsV1(t *testing.T, dir, id string) {
	t.Helper()
	path := filepath.Join(dir, id+".jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var r Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		r.Version, r.Time = 1, 0
		enc, _ := json.Marshal(r)
		out = append(out, string(enc))
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNewRecordsCarryTimeAndOldJournalsStillOpen(t *testing.T) {
	s, dir := fixture(t)
	if err := s.Append("message", map[string]string{"role": "user", "content": "hi"}); err != nil {
		t.Fatal(err)
	}
	for _, r := range s.Records() {
		if r.Version != 2 || r.Time <= 0 {
			t.Fatal("new record lacks a time", r)
		}
	}
	id := s.ID
	s.Close()
	rewriteAsV1(t, dir, id)

	old, err := Open(dir, id)
	if err != nil {
		t.Fatal("version 1 journal must still open:", err)
	}
	if err := old.Append("message", map[string]string{"role": "assistant", "content": "hello"}); err != nil {
		t.Fatal(err)
	}
	records := old.Records()
	if records[0].Version != 1 || records[len(records)-1].Version != 2 || records[len(records)-1].Time <= 0 {
		t.Fatal("mixed journal versions wrong", records[0], records[len(records)-1])
	}
	old.Close()

	forked, err := Fork(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer forked.Close()
	var sawOld, sawNew bool
	for _, r := range forked.Records() {
		if r.Type != "message" {
			continue
		}
		if r.Version == 1 && r.Time == 0 {
			sawOld = true
		}
		if r.Version == 2 && r.Time == records[len(records)-1].Time {
			sawNew = true
		}
	}
	if !sawOld || !sawNew {
		t.Fatal("fork did not keep original record times (or their absence)")
	}
}

func TestInconsistentVersionAndTimeRejected(t *testing.T) {
	for _, r := range []Record{{Version: 1, Time: 5}, {Version: 2}, {Version: 3, Time: 5}} {
		if validVersion(r) {
			t.Fatal("accepted", r)
		}
	}
}
