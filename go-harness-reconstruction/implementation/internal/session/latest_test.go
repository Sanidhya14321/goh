package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/independent-harness/internal/testutil"
)

func TestLatestPicksNewestForProject(t *testing.T) {
	dir := t.TempDir()
	projA, projB := testutil.TempDir(t), testutil.TempDir(t) // symlink-resolved, as stored
	if _, err := Latest(dir, projA); !errors.Is(err, ErrNoSession) {
		t.Fatal(err)
	}
	mk := func(project string, age time.Duration) string {
		s, err := New(dir, project)
		if err != nil {
			t.Fatal(err)
		}
		id := s.ID
		s.Close()
		when := time.Now().Add(-age)
		os.Chtimes(filepath.Join(dir, id+".jsonl"), when, when)
		return id
	}
	mk(projA, 3*time.Hour)
	newest := mk(projA, time.Minute)
	mk(projA, 2*time.Hour)
	mk(projB, 0)
	got, err := Latest(dir, projA)
	if err != nil || got != newest {
		t.Fatal(got, newest, err)
	}
}
