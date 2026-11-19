package runtime

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// Run outcomes recorded with a completed run. "completed" only says the model
// stopped; the outcome says whether its file changes were checked.
const (
	OutcomeNoChanges  = "no_changes" // no file was changed
	OutcomeVerified   = "verified"   // a recognized check succeeded after the last change
	OutcomeUnverified = "unverified" // files changed and nothing checked them afterwards
)

// Change tracking states reported with a run's verification detail.
const (
	TrackingKnown   = "known"   // the project fingerprint covered every change
	TrackingUnknown = "unknown" // the project could not be fingerprinted (too large or unreadable)
)

var fileChangeTools = map[string]bool{"create_file": true, "write_file": true, "edit_file": true, "apply_patch": true}

// effectTools can change files outside the edit tools: shell commands, child
// agents and background jobs. After each, the project is compared with its
// last known state.
var effectTools = map[string]bool{"shell": true, "task": true, "job_start": true, "job_stop": true, "job_output": true}

// Evidence is the check that last established a run's verification outcome.
// It is reported in run_finished (as "verification") so a consumer can tell
// what was checked: a recognized test or build command the model ran, or the
// user's configured verify command.
type Evidence struct {
	Source   string `json:"source"`          // "shell" (model ran it) or "configured" (verify.command)
	Command  string `json:"command"`         // the command that ran
	Kind     string `json:"kind"`            // test, build, assert, run or custom
	Scope    string `json:"scope,omitempty"` // for tests: selected or full
	Passed   bool   `json:"passed"`
	ExitCode *int   `json:"exit_code,omitempty"`
	// Revision is the project fingerprint the check saw; empty when the
	// change tracking is unknown.
	Revision string `json:"revision,omitempty"`
	// RepairAttempt is the number of repairs used before this check
	// (configured checks only).
	RepairAttempt int    `json:"repair_attempt,omitempty"`
	ArtifactID    string `json:"artifact_id,omitempty"`
}

// resetOutcome clears the per-operation verification state. Every Run,
// RunShell and Compact starts from it, so an operation never reports a
// result left by an earlier one.
func (a *Agent) resetOutcome(fingerprint bool) {
	a.changedFiles, a.verifiedAfterChange, a.changeUnknown = false, false, false
	a.repairsUsed, a.verifyFailed, a.completionChecked, a.wroteTestDouble = 0, false, false, false
	a.narrowTestPassed, a.fullTestPassed = false, false
	a.evidence = nil
	a.runCommands, a.evidenceSubmitted, a.evidenceRevision = nil, false, ""
	var fingerprinted bool
	a.fingerprint, fingerprinted = "", false
	if a.Tools != nil && fingerprint {
		a.fingerprint, fingerprinted = a.Tools.Fingerprint()
	}
	a.noFingerprint = !fingerprinted
}

// markChanged records a file change: earlier checks no longer cover the
// project.
func (a *Agent) markChanged() {
	a.changedFiles, a.verifiedAfterChange = true, false
	a.narrowTestPassed, a.fullTestPassed = false, false
	a.evidence = nil
}

// trackOutcome updates the run outcome from a finished tool call. Only a
// recognized check (a test runner, a build, lint or type check, an explicit
// assertion such as test or diff, or running the program) counts as
// verification; inspecting the project (pwd, ls, git status) does not.
func (a *Agent) trackOutcome(c core.Call, r tools.Result) {
	if fileChangeTools[c.Function.Name] {
		if r.Error {
			return
		}
		a.markChanged()
		if writesTestDouble(c) {
			a.wroteTestDouble = true
		}
		if !a.noFingerprint {
			a.fingerprint, _ = a.Tools.Fingerprint()
		}
		return
	}
	if !effectTools[c.Function.Name] {
		return
	}
	var args struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal([]byte(c.Function.Arguments), &args)
	if c.Function.Name == "shell" && !r.Error && len(a.runCommands) < 500 {
		a.runCommands = append(a.runCommands, args.Command)
	}
	if a.noFingerprint {
		// Changes cannot be detected, so any command that may write leaves
		// the coverage unknown, and later checks still run.
		if c.Function.Name != "shell" || (!readOnlyCommand(args.Command) && checkKind(args.Command) == "") {
			a.changeUnknown = true
			a.verifiedAfterChange = false
			a.evidence = nil
		}
	} else if a.noticeUntrackedChanges() {
		return // the command changed files, so it did not check them
	}
	if c.Function.Name != "shell" || r.Error {
		return
	}
	var out struct {
		ExitCode *int `json:"exit_code"`
	}
	if !(a.changedFiles || a.changeUnknown) || json.Unmarshal([]byte(r.Content), &out) != nil || out.ExitCode == nil || *out.ExitCode != 0 {
		return
	}
	kind := checkKind(args.Command)
	if kind == "" {
		return // ran fine, but checked nothing
	}
	a.verifiedAfterChange = true
	scope := ""
	if kind == "test" {
		scope = testScope(args.Command)
		switch scope {
		case "selected":
			a.narrowTestPassed = true
		case "full":
			a.fullTestPassed = true
		}
	}
	code := 0
	a.evidence = &Evidence{Source: "shell", Command: truncateRunes(args.Command, 500), Kind: kind, Scope: scope, Passed: true, ExitCode: &code, Revision: a.fingerprint}
}

// noticeUntrackedChanges compares the project with the state after the last
// known change. Files changed by a shell command or background job count as
// a change like an edit: the outcome and the configured verify then cover
// them. Ignored files and build or test caches do not count.
func (a *Agent) noticeUntrackedChanges() bool {
	if a.Tools == nil || a.noFingerprint {
		return false
	}
	now, ok := a.Tools.Fingerprint()
	if !ok {
		// The project became unreadable or too large during the run.
		a.noFingerprint, a.changeUnknown = true, true
		a.verifiedAfterChange, a.evidence = false, nil
		return true
	}
	if a.fingerprint == "" || now == a.fingerprint {
		if a.fingerprint == "" {
			a.fingerprint = now
		}
		return false
	}
	a.fingerprint = now
	a.markChanged()
	return true
}

// mayHaveChanged reports whether the configured verifier must run: files
// changed, or a command that may have changed them ran while changes could
// not be tracked.
func (a *Agent) mayHaveChanged() bool { return a.changedFiles || a.changeUnknown }

func (a *Agent) outcome() string {
	switch {
	case a.verifyFailed:
		return OutcomeFailedVerification
	case !a.mayHaveChanged():
		return OutcomeNoChanges
	case a.verifiedAfterChange:
		return OutcomeVerified
	}
	return OutcomeUnverified
}

// outcomeDetail is the additive verification detail recorded beside the
// outcome in run_finished.
func (a *Agent) outcomeDetail() map[string]any {
	d := map[string]any{"change_tracking": TrackingKnown}
	if a.noFingerprint {
		d["change_tracking"] = TrackingUnknown
	}
	if a.evidence != nil {
		d["evidence"] = a.evidence
	}
	return d
}

// testDoubleRe finds a stand-in the model defines for code it did not write.
// It matches named doubles (FakeX, mockY), stand-ins named after an external
// role (a class or struct ending in Gateway, Client, Service and so on), and
// common mocking libraries.
var testDoubleRe = regexp.MustCompile(`(?i)\b(class|type|struct|def|func|function|const|let|var)\s+\w*(fake|mock|stub|dummy|spy)\w*` +
	`|(?m)^\s*(class\s+\w*|type\s+\w*)(gateway|client|service|repository|repo|api|server|store|provider|backend|processor)\b` +
	`|\bunittest\.mock\b|\bMagicMock\b|\bMock\(|\bmonkeypatch\b|\bjest\.(fn|mock)\b|\bsinon\.|\bgomock\b|\bmockito\b`)

// isTestPath reports whether path looks like a test file in common layouts.
func isTestPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(p)
	return strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || base == "conftest.py" ||
		strings.Contains(p, "/tests/") || strings.HasPrefix(p, "tests/") || strings.Contains(p, "/test/") || strings.HasPrefix(p, "test/")
}

// writesTestDouble reports whether a file change adds a fake, mock or stub to a
// test file. Checks that pass against such a stand-in can hide a change that
// misuses the real dependency (an invented parameter, say); benchmark runs
// showed exactly this false completion.
func writesTestDouble(c core.Call) bool {
	var args struct {
		Path      string `json:"path"`
		Content   string `json:"content"`
		NewText   string `json:"new_text"`
		PatchText string `json:"patchText"`
	}
	if json.Unmarshal([]byte(c.Function.Arguments), &args) != nil {
		return false
	}
	if args.PatchText != "" {
		file := ""
		for _, line := range strings.Split(args.PatchText, "\n") {
			for _, h := range []string{"*** Add File: ", "*** Update File: "} {
				if strings.HasPrefix(line, h) {
					file = strings.TrimSpace(strings.TrimPrefix(line, h))
				}
			}
			if strings.HasPrefix(line, "+") && isTestPath(file) && testDoubleRe.MatchString(line) {
				return true
			}
		}
		return false
	}
	return isTestPath(args.Path) && testDoubleRe.MatchString(args.Content+"\n"+args.NewText)
}

const testDoubleCheckPrompt = "[Automatic check, not a new request] Your checks in this run use a fake, mock or stub you wrote. A passing check against your own stand-in does not show the change works with the real code. Open the real definitions your change calls (functions, methods, constructors and their parameter names) and confirm every call matches them exactly; do not add parameters the real code does not accept. Run the project's existing tests too. Fix any mismatch, then finish."

const selectedTestsCheckPrompt = "[Automatic check, not a new request] The tests that passed since your last change were picked by name (-run, -k, ::, -t or similar). Run the full tests of every package or module you changed, without the name filter, and fix any failure, including existing tests your change broke. Then finish."

const completionCheckPrompt = "[Automatic check, not a new request] You changed files in this run but no test, build or other check has succeeded since your last change. Run the most relevant check now and fix any failure. If no check applies, say why in one sentence. Then finish."

// needsCompletionCheck reports whether to remind the model, once per run, to
// check its changes before finishing. Completion gated on evidence improves
// solve rates (ECLoop, arXiv:2607.28815), and weaker models often submit
// without running any test (DeepSWE, arXiv:2607.07946). A configured verify
// command already enforces checking, so the reminder is skipped then.
func (a *Agent) needsCompletionCheck() bool {
	if a.completionChecked || a.depth > 0 || a.isPlan() || a.Config.Verify != nil {
		return false
	}
	if a.Config.CompletionCheck != nil && !*a.Config.CompletionCheck {
		return false
	}
	return a.changedFiles && (!a.verifiedAfterChange || a.wroteTestDouble || a.onlySelectedTests())
}

// onlySelectedTests reports whether the passing tests since the last change
// ran only tests picked by name. In the real-project benchmark every miss
// was a fix the model's own passing checks called done, including one that
// broke an existing test in the same package; a name-filtered run cannot
// see such breakage.
func (a *Agent) onlySelectedTests() bool {
	return a.verifiedAfterChange && a.narrowTestPassed && !a.fullTestPassed
}

var (
	// testCommandRe finds a test runner invocation in a shell command.
	testCommandRe = regexp.MustCompile(`(^|[\s;&|(/])(go\s+test|pytest|py\.test|python3?\s+-m\s+(pytest|unittest)|jest|vitest|mocha|cargo\s+(nextest\s+run|test)|npm\s+(run\s+)?test|yarn\s+test|pnpm\s+(run\s+)?test|rspec|phpunit|dotnet\s+test|mvn\s+test|gradlew?\s+test|make\s+test|ctest)\b`)
	// selectedTestRe finds options that run only tests picked by name.
	selectedTestRe = regexp.MustCompile(`(\s-run[=\s]|\s-k\s|\s-k=|::|\s-t\s|--testNamePattern|--grep\s|\s--filter[=\s]|\s-Dtest=|--tests\s)`)
	// cargoNamedRe is cargo test given a test name before any option.
	cargoNamedRe = regexp.MustCompile(`cargo\s+test\s+[^-\s]`)
)

// testScope classifies a shell command: "selected" when it runs tests picked
// by name, "full" when it runs a test suite, "" when it runs no tests.
func testScope(command string) string {
	if !testCommandRe.MatchString(command) {
		return ""
	}
	if selectedTestRe.MatchString(" "+command) || cargoNamedRe.MatchString(command) {
		return "selected"
	}
	return "full"
}

// completionCheck returns the reminder to send and the reason recorded with it.
func (a *Agent) completionCheck() (prompt, reason string) {
	if a.verifiedAfterChange && a.wroteTestDouble {
		return testDoubleCheckPrompt, "checks used a test double written in this run"
	}
	if a.onlySelectedTests() {
		return selectedTestsCheckPrompt, "only tests selected by name passed after the last change"
	}
	return completionCheckPrompt, "files changed without a passing check"
}

var (
	// buildCheckRe finds a build, compile, lint or type check.
	buildCheckRe = regexp.MustCompile(`(^|[\s;&|(/])(go\s+(build|vet)|gofmt\s+-l|staticcheck|golangci-lint|cargo\s+(build|check|clippy)|tsc|eslint|mypy|pyright|ruff(\s+check)?|flake8|pylint|python3?\s+-m\s+(py_compile|compileall|mypy|ruff|flake8)|javac|gcc|g\+\+|clang|cc|make|cmake\s+--build|ninja|npm\s+run\s+(build|lint|typecheck|check)|yarn\s+(build|lint|tsc)|pnpm\s+(run\s+)?(build|lint|typecheck)|dotnet\s+build|mvn\s+(compile|verify|package)|gradlew?\s+(build|check|assemble)|swift\s+build|zig\s+build)\b`)
	// assertCheckRe finds an explicit assertion about files or output.
	assertCheckRe = regexp.MustCompile(`^(test|\[|\[\[|diff|cmp)$`)
	// runCheckRe finds a command that runs the program or a script.
	runCheckRe = regexp.MustCompile(`(^|[\s;&|(/])(go\s+run|cargo\s+run|deno\s+run|bun\s+run|npm\s+(start|run\s+\S+)|swift\s+run|dotnet\s+run)\b`)
)

// checkKind classifies a successful shell command as verification evidence:
// "test" (a test runner), "build" (build, lint or type check), "assert"
// (test, [, diff, cmp, grep -q) or "run" (running the program or a script).
// It returns "" for commands that only inspect the project (pwd, ls, cat,
// git status, echo), which prove nothing about a change.
func checkKind(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if testScope(command) != "" {
		return "test"
	}
	if buildCheckRe.MatchString(command) {
		return "build"
	}
	if runCheckRe.MatchString(command) {
		return "run"
	}
	segments := splitShell(command)
	for _, words := range segments {
		for len(words) > 0 && strings.Contains(words[0], "=") && !strings.HasPrefix(words[0], "=") {
			words = words[1:] // FOO=1 cmd
		}
		if len(words) == 0 {
			continue
		}
		name := words[0]
		args := words[1:]
		switch {
		case assertCheckRe.MatchString(name):
			return "assert"
		case name == "grep" || name == "rg":
			for _, a := range args {
				if a == "-q" || a == "--quiet" || (strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a, "q")) {
					return "assert"
				}
			}
		case name == "python" || name == "python3" || name == "node" || name == "ruby" || name == "perl" || name == "php" || name == "bash" || name == "sh" || name == "zsh":
			if len(args) > 0 && args[0] != "--version" && args[0] != "-v" && args[0] != "-V" {
				return "run"
			}
		case strings.HasPrefix(name, "./") || (strings.Contains(name, "/") && !strings.HasPrefix(name, "/")):
			return "run" // a project script or binary
		}
	}
	return ""
}
