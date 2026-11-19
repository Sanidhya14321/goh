package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Check kinds: what one fail_to_pass or pass_to_pass command established.
// Only "passed" is a pass; the rest are distinct non-pass results.
const (
	checkPassed      = "passed"       // exit 0 and (when identities are known) every expected test passed
	checkTestFailure = "test_failure" // a test assertion failed
	checkBuild       = "build_failure"
	checkNoTests     = "no_tests" // nothing ran, or an expected test was skipped or missing
	checkTimeout     = "timeout"
	checkToolError   = "tool_error" // the runner itself failed (container gone, command missing)
)

// checkResult is one run of a grading command with its observed tests.
type checkResult struct {
	Kind       string            `json:"kind"`
	Exit       int               `json:"exit"`
	Structured bool              `json:"structured"` // per-test results were parsed
	Tests      map[string]string `json:"tests,omitempty"`
	Missing    []string          `json:"missing,omitempty"` // expected tests that did not pass
	Output     string            `json:"-"`
}

// Failed lists tests that failed.
func (c checkResult) Failed() []string { return c.with("fail") }

// Passed lists tests that passed.
func (c checkResult) Passed() []string { return c.with("pass") }

func (c checkResult) with(state string) []string {
	var out []string
	for name, s := range c.Tests {
		if s == state {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

var goTestCmd = regexp.MustCompile(`(^|[\s;&|(])go\s+test\b`)

// structuredCommand adds machine-readable output to a known test runner:
// go test gets -json. unittest -v and pytest -rA output is parsed as is.
func structuredCommand(command string) string {
	if loc := goTestCmd.FindStringIndex(command); loc != nil && !strings.Contains(command, "-json") {
		return command[:loc[1]] + " -json" + command[loc[1]:]
	}
	return command
}

// runCheck runs a grading command and classifies what it established.
// expected, when set, are the test identities that must pass (matched by
// suffix, so "TestSum" matches "pkg.TestSum").
func runCheck(box, dir, command string, seconds int, expected []string) checkResult {
	code, out := csh(box, dir, structuredCommand(command), seconds)
	r := checkResult{Exit: code, Output: out}
	r.Tests, r.Structured = parseTests(out)
	switch {
	case code == 124 || code == -1 && strings.Contains(out, "deadline exceeded"):
		r.Kind = checkTimeout
		return r
	case code == 126 || code == 127 || code == -1:
		r.Kind = checkToolError
		return r
	}
	for _, want := range expected {
		if state := lookupTest(r.Tests, want); state != "pass" {
			r.Missing = append(r.Missing, want+"="+orNone(state))
		}
	}
	failed := len(r.Failed()) > 0
	switch {
	case code == 0 && len(r.Missing) == 0 && (!r.Structured || len(r.Passed()) > 0):
		r.Kind = checkPassed
	case failed:
		r.Kind = checkTestFailure
	case code != 0 && buildFailure(out):
		r.Kind = checkBuild
	case code != 0 && !r.Structured:
		r.Kind = checkTestFailure // an unparsed runner: its exit code is the evidence
	case code != 0:
		r.Kind = checkBuild
	default:
		r.Kind = checkNoTests // exit 0, but nothing (or not what was expected) passed
	}
	return r
}

func orNone(s string) string {
	if s == "" {
		return "not_run"
	}
	return s
}

func lookupTest(tests map[string]string, want string) string {
	if s, ok := tests[want]; ok {
		return s
	}
	for name, s := range tests {
		if strings.HasSuffix(name, "."+want) || strings.HasSuffix(name, "::"+want) || strings.HasSuffix(name, "/"+want) {
			return s
		}
	}
	return ""
}

var (
	buildFailRe    = regexp.MustCompile(`(?m)\[build failed\]|^# \S+\n|setup failed|cannot find package|undefined: |SyntaxError|ImportError|ModuleNotFoundError`)
	unittestLineRe = regexp.MustCompile(`(?m)^(\w+) \(([\w.]+)\)(?: \.\.\.|\n.*\.\.\.) (ok|FAIL|ERROR|skipped.*|expected failure|unexpected success)$`)
	pytestLineRe   = regexp.MustCompile(`(?m)^(PASSED|FAILED|ERROR|SKIPPED|XFAIL|XPASS) (\S+::\S+)`)
)

func buildFailure(out string) bool { return buildFailRe.MatchString(out) }

// parseTests reads per-test results from go test -json, unittest -v or
// pytest -rA output.
func parseTests(out string) (map[string]string, bool) {
	tests := map[string]string{}
	structured := false
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var e struct{ Action, Package, Test string }
		if json.Unmarshal(line, &e) != nil || e.Action == "" {
			continue
		}
		structured = true
		if e.Test == "" {
			continue
		}
		name := e.Package + "." + e.Test
		switch e.Action {
		case "pass":
			tests[name] = "pass"
		case "fail":
			tests[name] = "fail"
		case "skip":
			tests[name] = "skip"
		}
	}
	for _, m := range unittestLineRe.FindAllStringSubmatch(out, -1) {
		structured = true
		name := m[2] + "." + m[1]
		if !strings.HasSuffix(m[2], "."+m[1]) && !strings.HasSuffix(m[2], m[1]) {
			name = m[2] + "." + m[1]
		} else {
			name = m[2]
		}
		switch {
		case m[3] == "ok" || m[3] == "expected failure":
			tests[name] = "pass"
		case strings.HasPrefix(m[3], "skipped"):
			tests[name] = "skip"
		default:
			tests[name] = "fail"
		}
	}
	for _, m := range pytestLineRe.FindAllStringSubmatch(out, -1) {
		structured = true
		switch m[1] {
		case "PASSED", "XFAIL":
			tests[m[2]] = "pass"
		case "SKIPPED":
			tests[m[2]] = "skip"
		default:
			tests[m[2]] = "fail"
		}
	}
	return tests, structured
}

// snapshotHashes records the content hash of every regular file under dir.
func snapshotHashes(dir string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		sum := sha256.Sum256(b)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	return out, err
}

// testLikePath reports a test or test-discovery file, which the grader
// keeps under its own control.
func testLikePath(rel string) bool {
	base := filepath.Base(rel)
	return strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		base == "conftest.py" || base == "pytest.ini" || base == "tox.ini" || strings.HasPrefix(rel, "testdata/") || strings.Contains(rel, "/testdata/")
}

// protectedChanges compares the candidate tree with the pre-run baseline:
// test and discovery files (and the hidden tests' paths) that the candidate
// modified or deleted.
func protectedChanges(baseline, candidate map[string]string, hidden map[string]bool) []string {
	var out []string
	for rel, sum := range baseline {
		if !testLikePath(rel) && !hidden[rel] {
			continue
		}
		if now, ok := candidate[rel]; !ok {
			out = append(out, rel+" (deleted)")
		} else if now != sum {
			out = append(out, rel+" (modified)")
		}
	}
	for rel := range candidate {
		if _, ok := baseline[rel]; !ok && hidden[rel] {
			out = append(out, rel+" (added over a hidden test)")
		}
	}
	sort.Strings(out)
	return out
}

// restoreProtected puts the baseline's test and discovery files back into
// the grade tree, so the grader's own checks decide the result.
func restoreProtected(baselineDir, gradeDir string, baseline map[string]string) error {
	for rel := range baseline {
		if !testLikePath(rel) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(baselineDir, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		target := filepath.Join(gradeDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// splitArgs splits an arm's flags like a shell would split words, honouring
// single and double quotes and backslashes, without evaluating anything
// (no variables, globs or command substitution).
func splitArgs(s string) ([]string, error) {
	// Bytes, not runes: every delimiter is ASCII, and argv must keep any
	// byte sequence exactly (ranging over runes rewrote invalid UTF-8).
	var out []string
	var cur strings.Builder
	inWord, quote, escaped := false, byte(0), false
	for i := 0; i < len(s); i++ {
		r := s[i]
		switch {
		case escaped:
			cur.WriteByte(r)
			escaped, inWord = false, true
		case r == '\\' && quote != '\'':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteByte(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(r)
			inWord = true
		}
	}
	if quote != 0 || escaped {
		return nil, errors.New("unbalanced quote or trailing backslash")
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out, nil
}

var armName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,39}$`)

type arm struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

// parseArms validates arm specs (name=flags): unique, safe names and flags
// split without shell evaluation.
func parseArms(specs []string) ([]arm, error) {
	seen := map[string]bool{}
	var out []arm
	for _, spec := range specs {
		name, flags, _ := strings.Cut(spec, "=")
		if !armName.MatchString(name) {
			return nil, fmt.Errorf("arm %q: name must be letters, digits, '.', '_' or '-' (at most 40)", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("arm %q given twice", name)
		}
		seen[name] = true
		args, err := splitArgs(flags)
		if err != nil {
			return nil, fmt.Errorf("arm %q: %w", name, err)
		}
		out = append(out, arm{Name: name, Args: args})
	}
	return out, nil
}
