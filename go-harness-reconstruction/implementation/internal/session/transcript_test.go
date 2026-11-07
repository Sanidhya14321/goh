package session

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownTranscriptAndPrivatePublication(t *testing.T) {
	store, _ := completedFixture(t)
	defer store.Close()
	for _, withTools := range []bool{false, true} {
		var transcript bytes.Buffer
		if err := store.WriteTranscript(&transcript, withTools); err != nil {
			t.Fatal(err)
		}
		text := transcript.String()
		for _, expected := range []string{store.ID, "## User\n\nInvestigate and remember the task", "## Assistant\n\nRead completed"} {
			if !strings.Contains(text, expected) {
				t.Fatalf("missing saved conversation: %q", expected)
			}
		}
		for _, excluded := range []string{"Continue verification", "Untrusted summary", "Verify change", "permission"} {
			if strings.Contains(text, excluded) {
				t.Fatalf("included internal state: %q", excluded)
			}
		}
		for _, details := range []string{"    read_file", "    {\"path\":\"README.md\"}", "    read content"} {
			if strings.Contains(text, details) != withTools {
				t.Fatalf("tool details visibility did not match option: %q", details)
			}
		}
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "transcript.md")
	write := func(w io.Writer) error { return store.WriteTranscript(w, false) }
	if err := WriteExportFile(file, write); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("export is not private: %v %v", info, err)
	}
	if err := WriteExportFile(file, write); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate destination: %v", err)
	}
	want := errors.New("simulated write failure")
	err = WriteExportFile(filepath.Join(dir, "failed.md"), func(w io.Writer) error {
		_, _ = io.WriteString(w, "partial output")
		return want
	})
	if !errors.Is(err, want) {
		t.Fatal(err)
	}
	if files, err := os.ReadDir(dir); err != nil || len(files) != 1 || files[0].Name() != "transcript.md" {
		t.Fatalf("partial output left in destination: %v %v", files, err)
	}
}
