package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/testutil"
)

// Review: quoted --arm values are kept whole and never evaluated; names are
// validated and unique.
func TestParseArms(t *testing.T) {
	arms, err := parseArms([]string{`base=`, `verify=--verify "go test ./..." --reasoning-effort=high`, `odd=--prompt-suffix '$(touch /tmp/x)'`})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(arms[1].Args, []string{"--verify", "go test ./...", "--reasoning-effort=high"}) {
		t.Fatal(arms[1].Args)
	}
	if arms[2].Args[1] != "$(touch /tmp/x)" {
		t.Fatal("arm value was altered:", arms[2].Args)
	}
	for _, bad := range [][]string{{"a=x", "a=y"}, {"bad name=x"}, {"../x=y"}, {`q=--verify "unbalanced`}} {
		if _, err := parseArms(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestParseTestsAdapters(t *testing.T) {
	goOut := `{"Action":"run","Package":"ex/p","Test":"TestA"}
{"Action":"pass","Package":"ex/p","Test":"TestA"}
{"Action":"fail","Package":"ex/p","Test":"TestB"}
{"Action":"skip","Package":"ex/p","Test":"TestC"}
{"Action":"fail","Package":"ex/p"}`
	tests, ok := parseTests(goOut)
	if !ok || tests["ex/p.TestA"] != "pass" || tests["ex/p.TestB"] != "fail" || tests["ex/p.TestC"] != "skip" {
		t.Fatal(tests)
	}
	unittest := "test_total (test_cart.CartTest.test_total) ... ok\ntest_zero (test_cart.CartTest.test_zero) ... FAIL\ntest_old (test_cart.CartTest) ... skipped 'later'\n"
	tests, ok = parseTests(unittest)
	if !ok || tests["test_cart.CartTest.test_total"] != "pass" || tests["test_cart.CartTest.test_zero"] != "fail" || tests["test_cart.CartTest.test_old"] != "skip" {
		t.Fatal(tests)
	}
	pytest := "PASSED tests/test_a.py::test_one\nFAILED tests/test_a.py::test_two - assert 1 == 2\n"
	tests, ok = parseTests(pytest)
	if !ok || tests["tests/test_a.py::test_one"] != "pass" || tests["tests/test_a.py::test_two"] != "fail" {
		t.Fatal(tests)
	}
	if structuredCommand("cd x && go test -race ./...") != "cd x && go test -json -race ./..." {
		t.Fatal(structuredCommand("cd x && go test -race ./..."))
	}
}

// Empty selections, skipped expected tests, build errors and runner errors
// are distinct non-pass results.
func TestRunCheckKinds(t *testing.T) {
	testutil.RequireUnixShell(t)
	dir := t.TempDir()
	cases := map[string]struct {
		cmd      string
		expected []string
		want     string
	}{
		"pass":          {`echo 'PASSED t.py::test_a'`, nil, checkPassed},
		"empty":         {`echo '{"Action":"output","Package":"p"}'`, nil, checkNoTests},
		"skipped":       {`echo 'SKIPPED t.py::test_a'`, []string{"test_a"}, checkNoTests},
		"missing":       {`echo 'PASSED t.py::test_b'`, []string{"test_a"}, checkNoTests},
		"failure":       {`echo 'FAILED t.py::test_a'; exit 1`, nil, checkTestFailure},
		"build":         {`echo '# ex/p'; echo 'x.go:1: undefined: Foo'; exit 1`, nil, checkBuild},
		"tool":          {`exit 127`, nil, checkToolError},
		"timeout":       {`exit 124`, nil, checkTimeout},
		"unparsed-fail": {`exit 1`, nil, checkTestFailure},
		"unparsed-pass": {`true`, nil, checkPassed},
	}
	for name, c := range cases {
		if got := runCheck("", dir, c.cmd, 30, c.expected); got.Kind != c.want {
			t.Errorf("%s: %s, want %s", name, got.Kind, c.want)
		}
	}
}

// Review: protected files are compared with the actual pre-run baseline,
// including deletions, not with the hidden overlay.
func TestProtectedChanges(t *testing.T) {
	base := map[string]string{"a.go": "1", "a_test.go": "2", "test_x.py": "3", "conftest.py": "4"}
	cand := map[string]string{"a.go": "9", "a_test.go": "2", "conftest.py": "5", "hidden_test.go": "6"}
	got := protectedChanges(base, cand, map[string]bool{"hidden_test.go": true})
	want := []string{"conftest.py (modified)", "hidden_test.go (added over a hidden test)", "test_x.py (deleted)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

// Review: a truncated or doubled terminal event is surfaced.
func TestReadEventsValidatesStream(t *testing.T) {
	for in, want := range map[string]string{
		`{"type":"run_started","data":{"model":"m"}}` + "\n":                                                                                 "truncated: no run_finished",
		`{"type":"run_finished","data":{"status":"completed"}}` + "\n" + `{"type":"run_finished","data":{"status":"completed"}}` + "\n":      "2 run_finished events",
		"not json\n" + `{"type":"run_finished","data":{"status":"completed"}}` + "\n":                                                        "1 malformed event lines",
		`{"type":"run_started","data":{"model":"m","protocol":"p"}}` + "\n" + `{"type":"run_finished","data":{"status":"completed"}}` + "\n": "",
	} {
		var r Result
		readEvents(strings.NewReader(in), &r)
		if r.StreamProblem != want {
			t.Errorf("%q: problem %q want %q", in, r.StreamProblem, want)
		}
	}
}

// Infrastructure failures stay out of the resolution rate but lower
// reliability; false verification is counted on its own.
func TestSummarySeparatesInfraAndFalseVerification(t *testing.T) {
	s := summarize([]Result{
		{Arm: "a", Scored: true, Resolved: true},
		{Arm: "a", Scored: true, Outcome: "verified", FalseVerified: true, FalseComplete: true, Status: "completed"},
		{Arm: "a", Scored: false, FailureKind: "setup"},
	})
	if !strings.Contains(s, "| a | 3 | 1 | 1/2 (50%) | 1/3 (33%) | 1 | 1 |") {
		t.Fatal(s)
	}
}

// Review: concurrent invocations and repeated attempts never share an
// output folder.
func TestRunTaskFoldersAreFresh(t *testing.T) {
	testutil.RequireUnixShell(t)
	dir := t.TempDir()
	tk := task{ID: "tiny", Prompt: "p", FailToPass: "test -f done", dir: dir}
	os.MkdirAll(filepath.Join(dir, "repo"), 0o755)
	os.WriteFile(filepath.Join(dir, "repo", "x.txt"), []byte("x"), 0o644)
	hostGrading = true
	defer func() { hostGrading = false }()
	work := t.TempDir()
	goh := fakeGoh(t, "touch done")
	a := runTask(tk, goh, "base", nil, 1, work)
	b := runTask(tk, goh, "base", nil, 1, work)
	entries, _ := os.ReadDir(work)
	if len(entries) != 2 || !a.Resolved || !b.Resolved {
		t.Fatalf("folders %d, results %+v %+v", len(entries), a, b)
	}
}

// Research: retrieval telemetry from the journals, against gold files the
// agent never sees.
func TestRetrievalTelemetry(t *testing.T) {
	dir := t.TempDir()
	start := time.UnixMilli(1_000_000)
	journal := strings.Join([]string{
		`{"version":1,"type":"tool_start","time":1002000,"data":{"name":"grep","arguments":{"pattern":"x"}}}`,
		`{"version":1,"type":"tool_result","time":1003000,"data":{"result":{"content":"{\"matches\":[{\"path\":\"pkg/a.go\",\"line\":3}]}"}}}`,
		`{"version":1,"type":"tool_start","time":1005000,"data":{"name":"read_file","arguments":{"path":"pkg/a.go"}}}`,
		`{"version":1,"type":"tool_start","time":1006000,"data":{"name":"read_file","arguments":{"path":"pkg/a.go"}}}`,
	}, "\n") + "\n"
	os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(journal), 0o600)
	s := retrievalTelemetry(dir, []string{"pkg/a.go", "pkg/b.go"}, start)
	if s.Read != 1 || s.Recall != 0.5 || s.RepeatReads != 1 || s.FirstGoldSecs == nil || *s.FirstGoldSecs != 3 || len(s.Missed) != 1 {
		t.Fatalf("%+v", s)
	}
}

// Research: a test the agent adds counts as reproducer evidence only when
// it fails on the baseline and passes on the candidate, in isolated copies.
func TestReproducerAudit(t *testing.T) {
	testutil.RequireUnixShell(t)
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "repo"), 0o755)
	os.WriteFile(filepath.Join(dir, "repo", "calc.py"), []byte("def add(a, b):\n    return a - b\n"), 0o644)
	tk := task{ID: "calc", Prompt: "fix add", FailToPass: `python3 -c "import calc; assert calc.add(2, 3) == 5"`, dir: dir}
	hostGrading, reproducerAudit = true, true
	defer func() { hostGrading, reproducerAudit = false, false }()
	fix := `printf 'def add(a, b):\n    return a + b\n' > calc.py; printf 'import unittest, calc\nclass T(unittest.TestCase):\n    def test_add(self):\n        self.assertEqual(calc.add(2, 3), 5)\n' > test_repro.py`
	r := runTask(tk, fakeGoh(t, fix), "base", nil, 1, t.TempDir())
	if !r.Resolved || r.Reproducer != "fail_to_pass" {
		t.Fatalf("%+v", r)
	}
	weak := `printf 'def add(a, b):\n    return a + b\n' > calc.py; printf 'import unittest\nclass T(unittest.TestCase):\n    def test_true(self):\n        self.assertTrue(True)\n' > test_repro.py`
	if r := runTask(tk, fakeGoh(t, weak), "base", nil, 1, t.TempDir()); r.Reproducer != "no_signal" {
		t.Fatalf("%+v", r.Reproducer)
	}
}
