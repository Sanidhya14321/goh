package tui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDetailsToggle(t *testing.T) {
	output := "ok  pkg/a\nmiddle 1\nmiddle 2\nmiddle 3\nok  pkg/b\nok  pkg/c\nFAIL pkg/d"
	log := "\nYou: run tests\n" + toolResultLines("✓ ", Safe(output)) + toolResultLines("", Safe("\x00Rforged"))
	shown := strings.ReplaceAll(withDetails(log, detailsShown), verbatimMark, "")
	if !strings.Contains(shown, "✓ ok  pkg/a\n  middle 1\n  middle 2\n  middle 3\n  ok  pkg/b\n  ok  pkg/c\n  FAIL pkg/d") || strings.Contains(shown, "earlier lines") || strings.Contains(shown, "\x00") {
		t.Fatalf("%q", shown)
	}
	// Compact: the result line, then the last lines, where commands report.
	compact := strings.ReplaceAll(withDetails(log, detailsHidden), verbatimMark, "")
	if !strings.Contains(compact, "✓ ok pkg/a\n  … 3 earlier lines · /details shows all\n  ok  pkg/b\n  ok  pkg/c\n  FAIL pkg/d") || strings.Contains(compact, "middle") || strings.Contains(compact, "\x00") {
		t.Fatalf("%q", compact)
	}
	if strings.Count(shown, "forged") != 1 || strings.Count(compact, "forged") != 1 {
		t.Fatal("content must not forge markers")
	}
	m := &model{log: log, width: 80, height: 24}
	m.startFind("middle 2")
	if m.find != nil {
		t.Fatal("compact details should not be searchable on screen")
	}
	m.toggleDetails()
	m.startFind("middle 2")
	if m.find == nil || len(m.find.matches) != 1 {
		t.Fatal("find should see each shown line once")
	}
}

// Tool output is file and command text: Markdown syntax in it must render
// literally (a Python "# comment" is not a heading; "**kwargs" keeps "**").
func TestToolOutputRendersVerbatim(t *testing.T) {
	code := "def f(*args, **kwargs):\n    # TODO: handle fiscal years\n    return a ** b\n```\n| a | b |\n|---|---|"
	log := "\nYou: read it\n" + toolResultLines("✓ ", Safe(code)) + "## Reply heading\n"
	var text []string
	for _, row := range transcriptRows(withDetails(log, detailsShown), 100, "", false) {
		text = append(text, row.text)
	}
	got := strings.Join(text, "\n")
	for _, want := range []string{"def f(*args, **kwargs):", "    # TODO: handle fiscal years", "return a ** b", "```", "| a | b |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("tool output line %q not shown verbatim:\n%s", want, got)
		}
	}
	if strings.Contains(got, "## Reply") || !strings.Contains(got, "Reply heading") {
		t.Fatalf("assistant Markdown should still render:\n%s", got)
	}
}

func TestListingResultsAreReadable(t *testing.T) {
	got := toolResultText(`{"matches":[{"path":"a.py","line":20,"text":"    # TODO x"}],"skipped_files":0,"truncated":false}`)
	if got != "1 match\na.py:20: # TODO x" {
		t.Fatalf("%q", got)
	}
	if got := toolResultText(`{"count":2,"files":["a.go","b.go"],"truncated":true}`); got != "2 files\na.go\nb.go\n[truncated]" {
		t.Fatalf("%q", got)
	}
	if got := toolArgumentText([]byte(`{"url":"https://go.dev/","format":"text"}`)); got != "https://go.dev/" {
		t.Fatalf("%q", got)
	}
	if got := toolArgumentText([]byte(`{"patchText":"*** Begin Patch\n*** Update File: a.go\n@@\n-x\n+y\n*** Add File: b.go\n+z\n*** End Patch"}`)); got != "update a.go, add b.go" {
		t.Fatalf("%q", got)
	}
	if got := toolArgumentText([]byte(`{"todos":[{"content":"a","status":"completed"},{"content":"b","status":"pending"}]}`)); got != "2 tasks, 1 done" {
		t.Fatalf("%q", got)
	}
}

func TestArtifactReadLabel(t *testing.T) {
	for raw, want := range map[string]string{
		`{"id":"fetch-1.log","offset":4000,"limit":4000}`: "fetch-1.log · bytes 4000–8000",
		`{"id":"shell-2.log","tail":true}`:                "shell-2.log · end",
		`{"id":"job-3"}`:                                  "job-3",
		`{"id":"fetch-1.log","pattern":"go1\\."}`:         "fetch-1.log · go1\\.",
	} {
		if got := toolArgumentText(json.RawMessage(raw)); got != want {
			t.Errorf("%s: %q", raw, got)
		}
	}
}
