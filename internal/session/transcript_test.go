package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Sanidhya14321/goh/internal/testutil"
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
	if err != nil || testutil.UnixModes() && info.Mode().Perm() != 0600 {
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

// A secret spanning several lines must be redacted as a whole before a tool
// block is split into indented lines: raw content (imported or legacy
// sessions) has real newlines, the current JSON envelope escaped ones.
func TestMarkdownTranscriptRedactsMultilineToolSecrets(t *testing.T) {
	// Synthetic, not a real key.
	body := "MIIEowIBAAKCAQEAsyntheticKEYbodyLINEone0000000000\nsyntheticKEYbodyLINEtwo1111111111111111111111\nsyntheticKEYbodyLINEthree22222222222222222"
	key := "-----BEGIN RSA PRIVATE KEY-----\n" + body + "\n-----END RSA PRIVATE KEY-----"
	token := "sk-proj-" + strings.Repeat("Zx9", 10)
	envelope := `{"content":` + strconvQuote("found:\n"+key+"\nand "+token) + `,"error":false}`
	for name, content := range map[string]string{
		"raw":       "found:\n" + key + "\ntrailing " + token + "\n",
		"truncated": "partial:\n-----BEGIN OPENSSH PRIVATE KEY-----\n" + body,
		"envelope":  envelope,
	} {
		t.Run(name, func(t *testing.T) {
			store, err := New(t.TempDir(), testutil.TempDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			call := map[string]any{"id": "c1", "type": "function", "function": map[string]string{"name": "read_file", "arguments": `{"path":"id_rsa"}`}}
			for _, m := range []map[string]any{
				{"role": "user", "content": "show the key"},
				{"role": "assistant", "tool_calls": []any{call}},
				{"role": "tool", "tool_call_id": "c1", "content": content},
			} {
				if err := store.Append("message", m); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := store.WriteTranscript(&out, true); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			for _, leaked := range []string{"syntheticKEYbody", "Zx9Zx9Zx9"} {
				if strings.Contains(text, leaked) {
					t.Fatalf("secret %q exported:\n%s", leaked, text)
				}
			}
			if !strings.Contains(text, "[redacted private key]") || !strings.Contains(text, "### Tool result") {
				t.Fatalf("redaction marker missing:\n%s", text)
			}
		})
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
