package main

import (
	"os"
	"path/filepath"
	"runtime"
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
