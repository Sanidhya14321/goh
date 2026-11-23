package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeGoh writes a stand-in goh: it emits run events and runs body in the
// project folder (the last --project argument value).
func fakeGoh(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "goh")
	script := "#!/bin/sh\nwhile [ $# -gt 0 ]; do [ \"$1\" = --project ] && proj=$2; shift; done\ncd \"$proj\"\n" + body +
		"\necho '{\"type\":\"tool_requested\",\"data\":{}}'\necho '{\"type\":\"assistant_completed\",\"data\":{\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":7,\"cache_read_tokens\":40}}}'\necho '{\"type\":\"run_finished\",\"data\":{\"status\":\"completed\",\"outcome\":\"unverified\"}}'\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunnerGradesCleanlyAndFlagsFalseCompletion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures")
	}
	if _, err := os.Stat("../../bench/tasks/py-median/task.json"); err != nil {
		t.Skip("task fixtures not present")
	}
	tasks, err := loadTasks("../../bench/tasks", "py-median")
	if err != nil {
		t.Fatal(err)
	}
	sol, _ := filepath.Abs("../../bench/tasks/py-median/solution/stats.py")
	work := t.TempDir()
	if r := runTask(tasks[0], fakeGoh(t, "true"), "base", nil, 1, work); r.Error == "" || r.Status != "" {
		t.Fatalf("a task without an image ran without -host-grading: %+v", r)
	}
	hostGrading = true
	defer func() { hostGrading = false }()
	good := runTask(tasks[0], fakeGoh(t, "cp "+sol+" stats.py"), "base", nil, 1, work)
	if !good.Resolved || good.FalseComplete || good.ToolCalls != 1 || good.PromptTokens != 100 || good.CacheRead != 40 || good.Status != "completed" {
		t.Fatalf("solved run: %+v", good)
	}
	lazy := runTask(tasks[0], fakeGoh(t, "true"), "base", nil, 1, work)
	if lazy.Resolved || !lazy.FalseComplete {
		t.Fatalf("unsolved run claiming completion: %+v", lazy)
	}
	// An agent that rewrites the hidden test to pass trivially is graded
	// against the real hidden test and flagged.
	cheat := runTask(tasks[0], fakeGoh(t, "printf 'import unittest\\n' > test_median.py"), "base", nil, 1, work)
	if cheat.Resolved || !cheat.HiddenTampered {
		t.Fatalf("tampering run: %+v", cheat)
	}
	if s := summarize([]Result{good, lazy, cheat}); s == "" {
		t.Fatal("empty summary")
	}
}

// Answer-graded tasks: every pattern must match (case-insensitive, across
// lines) and no forbidden one; the per-category table groups by arm.
func TestAnswerSpecAndPromptRangeTable(t *testing.T) {
	a := &answerSpec{Patterns: []string{`\b175\b`, `4,?750`}, Forbid: []string{`not sure`}}
	for reply, want := range map[string]bool{
		"lateFeeCents: 175 cents/day,\ncapped at 4,750.": true,
		"175 per day":                      false,
		"I'm not sure, maybe 175 and 4750": false,
		"":                                 false,
	} {
		if got, err := a.check(reply); err != nil || got != want {
			t.Errorf("%q: %v %v", reply, got, err)
		}
	}
	if _, err := (&answerSpec{Patterns: []string{"("}}).check("x"); err == nil {
		t.Error("bad pattern accepted")
	}
	yes, no := true, false
	table := promptRangeTable([]Result{
		{Arm: "a", Category: "chat", Size: "tiny", Scored: true, Resolved: true, Requests: 1, PromptTokens: 2500, WithinBudget: &yes, Seconds: 2},
		{Arm: "a", Category: "question", Size: "large", Scored: true, Requests: 5, ToolCalls: 4, PromptTokens: 30000, CacheRead: 15000, WithinBudget: &no, ProjectWrites: 1, Seconds: 30},
		{Arm: "b", Category: "chat", Size: "tiny", Scored: false},
	})
	for _, want := range []string{"| chat | tiny | a | 1 | 1/1 | 1.0 | 0.0 | 2500 | 0% |", "| question | large | a | 1 | 0/1 | 5.0 | 4.0 | 30000 | 50% |", "| 0/1 | 1 |"} {
		if !strings.Contains(table, want) {
			t.Errorf("missing %q in\n%s", want, table)
		}
	}
	if strings.Contains(table, "| b |") {
		t.Error("unscored attempt counted")
	}
}
