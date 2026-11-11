package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/charmbracelet/x/ansi"
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

func TestScrolledReaderStaysPutWhileOutputStreams(t *testing.T) {
	m := fuzzModel(t)
	for i := range 200 {
		m.log += fmt.Sprintf("line %d\n", i)
	}
	m.View()
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	before := ansi.Strip(m.View().Content)
	m.log += "new 1\nnew 2\nnew 3\n"
	after := ansi.Strip(m.View().Content)
	pick := func(s string) string {
		for _, l := range strings.Split(s, "\n") {
			if strings.Contains(l, "line ") {
				return strings.TrimSpace(l)
			}
		}
		return ""
	}
	if pick(before) != pick(after) {
		t.Fatalf("view drifted: %q → %q", pick(before), pick(after))
	}
}

func TestScrolledUpShowsHintAndEscReturns(t *testing.T) {
	m := fuzzModel(t)
	m.log += strings.Repeat("row\n", 200)
	m.View()
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if !strings.Contains(ansi.Strip(m.View().Content), "Scrolled up") {
		t.Fatal("no scrolled-up hint")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.scrollOffset != 0 || strings.Contains(ansi.Strip(m.View().Content), "Scrolled up") {
		t.Fatal("Esc should return to the latest output")
	}
}

func TestApprovalScrollDoesNotOvershoot(t *testing.T) {
	m := fuzzModel(t)
	m.active = true
	m.pending = &permissionMsg{request: core.Permission{Tool: "shell", Preview: strings.Repeat("x\n", 60)}, response: make(chan core.Decision, 1)}
	for range 50 {
		m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
		m.View()
	}
	bottom := m.reviewOffset
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	m.View()
	if m.reviewOffset >= bottom {
		t.Fatalf("PgUp after overshoot did nothing (%d → %d)", bottom, m.reviewOffset)
	}
}

func TestUpTakesBackQueuedPrompt(t *testing.T) {
	m := fuzzModel(t)
	m.active = true
	fuzzType(m, "oops")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.queue) != 1 {
		t.Fatal("not queued")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if len(m.queue) != 0 || m.input != "oops" {
		t.Fatalf("queue %v input %q", m.queue, m.input)
	}
}

func TestQueuedPromptNotLostIntoOpenPanel(t *testing.T) {
	m := fuzzModel(t)
	m.queue = []string{"next task"}
	m.find = &findState{query: "x", matches: []int{0}}
	m.Update(queuedSubmitMsg{})
	if m.input != "next task" || len(m.queue) != 0 {
		t.Fatalf("queued prompt lost: input %q queue %v", m.input, m.queue)
	}
}

func TestSidebarShowsOpenTasksAndStripsControls(t *testing.T) {
	m := fuzzModel(t)
	for i := range 20 {
		m.todos = append(m.todos, core.Todo{Content: fmt.Sprintf("task %d", i), Status: "completed"})
	}
	m.todos = append(m.todos, core.Todo{Content: "current work", Status: "in_progress"})
	rows := strings.Join(m.sidebar("id", "model"), "\n")
	if !strings.Contains(rows, "current work") || !strings.Contains(rows, "earlier done") {
		t.Fatalf("open task hidden:\n%s", rows)
	}
}

func TestTinyTerminalCannotApproveUnseenOperation(t *testing.T) {
	m := fuzzModel(t)
	m.width, m.height = 45, 3
	response := make(chan core.Decision, 1)
	m.pending = &permissionMsg{request: core.Permission{Tool: "shell", Preview: "Command: rm -rf build"}, response: response}
	m.View()
	m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	select {
	case d := <-response:
		t.Fatalf("approved unseen: %+v", d)
	default:
	}
	if m.pending == nil || !strings.Contains(m.status, "Enlarge") {
		t.Fatalf("status %q", m.status)
	}
	m.width, m.height = 100, 30
	m.View()
	m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if d := <-response; !d.Allow {
		t.Fatal("visible approval refused")
	}
}

func TestCompactViewHidesThinkingAndScrolls(t *testing.T) {
	m := fuzzModel(t)
	m.width, m.height = 52, 15
	m.hideThinking = true
	m.addReasoning("SECRET-THOUGHT")
	m.flushReasoning()
	for i := range 60 {
		m.log += fmt.Sprintf("line %02d\n", i)
	}
	content := m.View().Content
	if strings.Contains(content, "SECRET-THOUGHT") {
		t.Fatal("hidden reasoning rendered in the compact view")
	}
	if !strings.Contains(content, "line 59") {
		t.Fatalf("tail not shown:\n%s", content)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if content := m.View().Content; strings.Contains(content, "line 59") {
		t.Fatalf("PgUp did not scroll the compact view:\n%s", content)
	}
}

func TestQuestionSelectionStaysOnScreen(t *testing.T) {
	m := fuzzModel(t)
	m.width, m.height = 64, 18
	options := []core.QuestionOption{}
	for i := range 20 {
		options = append(options, core.QuestionOption{Label: fmt.Sprintf("option-%02d", i)})
	}
	m.question = &questionState{selected: map[int]bool{}, message: questionMsg{request: core.QuestionRequest{Questions: []core.Question{{Header: "Pick", Question: "Which?", Options: options}}}, response: make(chan questionResponse, 1)}}
	m.View()
	for range 15 {
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m.View()
	}
	if content := m.View().Content; !strings.Contains(content, "option-15") {
		t.Fatalf("selected option scrolled out of view:\n%s", content)
	}
}

func TestSettingsFormKeepsFocusAndErrorVisible(t *testing.T) {
	fields := []SettingsField{}
	for _, l := range []string{"Protocol", "Base URL", "Endpoint", "Model", "Context window", "Output budget", "API key"} {
		fields = append(fields, SettingsField{Label: l, Value: "value-" + l})
	}
	m := &settingsForm{title: "Provider", hint: "Configure the provider.", fields: fields, width: 62, height: 16, index: 6, problem: "API key is required for this endpoint"}
	content := m.View().Content
	if !strings.Contains(content, "> API key") || !strings.Contains(content, "API key is required") || !strings.Contains(content, "Ctrl+S save") {
		t.Fatalf("focus, error or controls hidden:\n%s", content)
	}
}
