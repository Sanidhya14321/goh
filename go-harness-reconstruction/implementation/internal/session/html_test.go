package session

import (
	"bytes"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestHTMLTranscriptEscapesAndRedacts(t *testing.T) {
	s, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Append("message", core.Message{Role: "user", Content: `<script>alert(1)</script> key sk-proj-abcdefghijklmnopqrstuvwx1234`})
	s.Append("message", core.Message{Role: "assistant", Content: "Done & dusted"})
	var b bytes.Buffer
	if err := s.WriteHTMLTranscript(&b, false); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, "<script>") || !strings.Contains(out, "&lt;script&gt;") {
		t.Fatal("content not escaped")
	}
	if strings.Contains(out, "abcdefghijklmnopqrstuvwx1234") || !strings.Contains(out, "sk-proj-[redacted]") {
		t.Fatal("secret not redacted")
	}
	if !strings.Contains(out, "Done &amp; dusted") || !strings.Contains(out, "default-src 'none'") {
		t.Fatal(out)
	}
}
