package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
)

func TestPasteWhileRunningQueuesIntoComposer(t *testing.T) {
	m := fuzzModel(t)
	m.active = true
	m.Update(tea.PasteMsg{Content: "follow up"})
	if m.input != "follow up" {
		t.Fatalf("paste during a run was dropped: %q", m.input)
	}
}

func TestEmptyQuestionRequestDoesNotPanic(t *testing.T) {
	m := fuzzModel(t)
	response := make(chan questionResponse, 1)
	m.Update(questionMsg{response: response})
	if m.question != nil || len(response) != 1 {
		t.Fatal("an empty question request must be answered, not opened")
	}
}

func TestQuestionCustomEscReturnsToOptions(t *testing.T) {
	m := fuzzModel(t)
	response := make(chan questionResponse, 1)
	m.Update(questionMsg{request: core.QuestionRequest{Questions: []core.Question{{Question: "Q?", Options: []core.QuestionOption{{Label: "A"}}}}}, response: response})
	m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.question == nil || m.question.custom || len(response) != 0 {
		t.Fatal("Esc in a custom answer should go back to the options")
	}
	if strings.Contains(m.question.view(), " — \n") {
		t.Fatal("empty description rendered a dangling dash")
	}
}

func TestApprovalAcceptsUppercase(t *testing.T) {
	m := fuzzModel(t)
	response := make(chan core.Decision, 1)
	m.pending = &permissionMsg{request: core.Permission{Tool: "shell"}, response: response}
	m.Update(tea.KeyPressMsg{Code: 'Y', Text: "Y"})
	if d := <-response; !d.Allow {
		t.Fatal("caps-lock Y should allow")
	}
}

func TestSlashTabCyclesEveryMatch(t *testing.T) {
	m := fuzzModel(t)
	fuzzType(m, "/t")
	seen := map[string]bool{}
	for range len(m.slashMatches()) {
		m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		seen[m.input] = true
	}
	for _, want := range []string{"/todos", "/theme", "/timeline", "/thinking", "/timestamps"} {
		if !seen[want] {
			t.Fatalf("Tab never reached %s: %v", want, seen)
		}
	}
}

func TestPickerPasteIsOneLine(t *testing.T) {
	p := &optionPicker{width: 80, height: 24, options: []Option{{Value: "a", Label: "a"}}}
	p.Update(tea.PasteMsg{Content: "gpt\n5\t mini\n"})
	if p.query != "gpt 5 mini" {
		t.Fatalf("query %q", p.query)
	}
}

func TestSettingsWarnsBeforeDiscardingAndTrimsPastedKeys(t *testing.T) {
	f := &settingsForm{width: 80, height: 24, save: func([]SettingsField) error { return nil },
		fields: []SettingsField{{Label: "Key", Secret: true}, {Label: "URL"}}}
	f.Update(tea.PasteMsg{Content: "  sk-abc\n"})
	if f.fields[0].Value != "sk-abc" {
		t.Fatalf("pasted key %q", f.fields[0].Value)
	}
	if _, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}); cmd != nil || !strings.Contains(f.problem, "Unsaved") {
		t.Fatal("first Esc with edits must warn")
	}
	if _, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}); cmd == nil {
		t.Fatal("second Esc should discard")
	}
	f.Update(tea.PasteMsg{Content: "two\nlines"})
	if !strings.Contains(f.problem, "single line") {
		t.Fatal("multi-line paste into a single-line field must be explained")
	}
}

func TestFindSurvivesTranscriptTrim(t *testing.T) {
	m := fuzzModel(t)
	m.log = strings.Repeat("needle\n", 50)
	m.startFind("needle")
	m.log = "short"
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
}

func TestSettingsKeepsTypedSpaces(t *testing.T) {
	f := &settingsForm{width: 80, height: 24, fields: []SettingsField{{Label: "Name"}}}
	fuzzType2(f, "a b")
	if f.fields[0].Value != "a b" {
		t.Fatalf("typed value %q", f.fields[0].Value)
	}
}

func fuzzType2(m tea.Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}
