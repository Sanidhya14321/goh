package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONCConfig(t *testing.T) {
	raw := `{
  // model choice
  "model": "say \"/* not a comment */\" and \\", /* inline */
  "context_window": 65536,
  "endpoint": "https://example.test/v1/chat//completions", // URL keeps its //
  "fallback_profiles": [],
}
`
	c, err := DecodeConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.Model != `say "/* not a comment */" and \` || c.ContextWindow != 65536 || c.Endpoint != "https://example.test/v1/chat//completions" {
		t.Fatalf("%+v", c)
	}
	if !HasJSONCSyntax(raw) || HasJSONCSyntax(`{"model":"x"}`) {
		t.Fatal("HasJSONCSyntax")
	}
}

func TestJSONCErrorsKeepLinePositions(t *testing.T) {
	if _, err := StripJSONC(`{"a": 1 /* open`); err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Fatal(err)
	}
	plain, err := StripJSONC("{\n/* a\nb */\n\"x\": 1\n}")
	if err != nil || strings.Count(plain, "\n") != 4 {
		t.Fatalf("%q %v", plain, err)
	}
	if _, err := DecodeConfig(`{"model": "x", "nope": 1, // c
}`); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatal("unknown fields must still be rejected", err)
	}
	if got, _ := StripJSONC(`{"a":"x,}","b":[1,]}`); got != `{"a":"x,}","b":[1 ]}` {
		t.Fatalf("%q", got)
	}
}

// Plain JSON must pass through unchanged, whatever its strings contain.
func FuzzStripJSONCIdentityOnJSON(f *testing.F) {
	for _, seed := range []string{`{"a":"//x"}`, `["/*",",]"]`, `{"k":"\"\\/"}`, `1`, `{"a":[1,2]}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !json.Valid([]byte(s)) {
			return
		}
		got, err := StripJSONC(s)
		if err != nil || got != s {
			t.Fatalf("changed valid JSON %q -> %q (%v)", s, got, err)
		}
	})
}
