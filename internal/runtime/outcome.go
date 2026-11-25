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
	a.fingerprint, a.noFingerprint, a.pendingFingerprint = "", true, nil
	if a.Tools != nil && fingerprint {
		// Walking a large project takes a noticeable time (about 0.15 s for
		// 6,000 files), so it runs beside the first model request; nothing
		// reads or changes the project before that reply (settleFingerprint).
		p := &pendingFingerprint{done: make(chan struct{})}
		go func(t *tools.Engine) { p.sum, p.ok = t.Fingerprint(); close(p.done) }(a.Tools)
		a.pendingFingerprint = p
	}
}

// pendingFingerprint is the project fingerprint being taken at the start of
// an operation.
type pendingFingerprint struct {
	done chan struct{}
	sum  string
	ok   bool
}

// settleFingerprint waits for the start-of-operation fingerprint. It is
// called once a model reply arrives, before any tool runs or the outcome is
// read, and before a user shell command.
func (a *Agent) settleFingerprint() {
	p := a.pendingFingerprint
	if p == nil {
		return
	}
	a.pendingFingerprint = nil
	<-p.done
	a.fingerprint, a.noFingerprint = p.sum, !p.ok
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
	} else if !(c.Function.Name == "shell" && readOnlyCommand(args.Command)) && a.noticeUntrackedChanges() {
		// A command recognized as read-only (the recognizer that also lets
		// it run without asking) skips the project walk; changes from
		// background jobs are still noticed at the next writing command and
		// when the run ends.
		if c.Function.Name != "shell" || !checksAfterChanges(args.Command) {
			return // the command changed files, so it did not check them
		}
		// gofmt -w x.go && go test ./...: the check ran after the changes.
	}
	if c.Function.Name != "shell" || r.Error {
		return
	}
	var out struct {
		ExitCode *int   `json:"exit_code"`
		Output   string `json:"output"`
	}
	if !(a.changedFiles || a.changeUnknown) || json.Unmarshal([]byte(r.Content), &out) != nil || out.ExitCode == nil || *out.ExitCode != 0 {
		return
	}
	kind := checkKind(args.Command)
	if kind == "" {
		return // ran fine, but checked nothing
	}
	if !checkStatusAttributable(args.Command) {
		// The shell may have handled a failing check (|| true, a later
		// command after ;, a pipe): exit 0 is not evidence it passed.
		return
	}
	if kind == "test" && ranNoTests(out.Output) {
		return // the runner found no tests: nothing was checked
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

const completionCheckPrompt = "[Automatic check, not a new request] You changed files in this run but no test, build or other check has succeeded since your last change. Run the most relevant check now and fix any failure. If no check applies, say why in one sentence. Then finish: your next reply is the final one, so state what you changed in a sentence or two and the check's result."

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
// by name, "full" when it runs a test suite, "" when it runs no tests. Only
// the programs the command executes count (see programs), so text that
// merely mentions a test command is not a test run.
func testScope(command string) string {
	scope := ""
	for _, seg := range programs(command) {
		if !startsWith(testCommandRe, seg) {
			continue
		}
		if selectedTestRe.MatchString(" "+seg) || cargoNamedRe.MatchString(seg) {
			if scope == "" {
				scope = "selected"
			}
			continue
		}
		scope = "full"
	}
	return scope
}

// startsWith reports whether re matches seg at its start; the program is
// matched by its base name, so ./gradlew and node_modules/.bin/jest count.
func startsWith(re *regexp.Regexp, seg string) bool {
	program, rest, _ := strings.Cut(seg, " ")
	if i := strings.LastIndex(program, "/"); i >= 0 {
		program = program[i+1:]
	}
	seg = strings.TrimSpace(program + " " + rest)
	loc := re.FindStringIndex(seg)
	return loc != nil && loc[0] == 0
}

// nonPrograms only print, change directory or set state: their arguments
// are text, never a check.
var nonPrograms = map[string]bool{"echo": true, "printf": true, "cd": true, "pushd": true, "popd": true, "true": true, ":": true,
	"export": true, "set": true, "unset": true, "sleep": true, "pwd": true, "ls": true, "cat": true, "tee": true, "head": true, "tail": true}

// wrappers run the rest of their words as the program.
var wrappers = map[string]int{"timeout": 1, "time": 0, "nice": 0, "env": 0, "npx": 0, "xvfb-run": 0, "command": 0, "exec": 0}

// programs returns each simple command of a shell command line as the
// program it runs with its arguments, after environment assignments and
// wrappers (timeout 60, npx, uv run, poetry run…), skipping commands that
// only print or move around. An unparseable command yields nothing, so it
// is never counted as a check.
func programs(command string) []string {
	var out []string
	if stripped, ok := stripHeredocs(command); ok {
		command = stripped // a here-document body is input, not commands
	}
	for _, words := range splitShell(strings.TrimSpace(command)) {
		for {
			for len(words) > 0 && strings.Contains(words[0], "=") && !strings.HasPrefix(words[0], "=") && !strings.HasPrefix(words[0], "-") {
				words = words[1:] // FOO=1 cmd
			}
			if len(words) == 0 {
				break
			}
			if skip, ok := wrappers[words[0]]; ok {
				words = words[1:]
				for len(words) > 0 && strings.HasPrefix(words[0], "-") {
					words = words[1:] // wrapper options (timeout -s KILL)
				}
				if skip > 0 && len(words) > 0 {
					words = words[skip:]
				}
				continue
			}
			if len(words) >= 2 && (words[0] == "uv" || words[0] == "poetry" || words[0] == "pipenv" || words[0] == "pdm" || words[0] == "hatch") && words[1] == "run" {
				words = words[2:]
				continue
			}
			if len(words) >= 2 && words[0] == "bundle" && words[1] == "exec" {
				words = words[2:]
				continue
			}
			break
		}
		if len(words) == 0 || nonPrograms[words[0]] {
			continue
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

// zeroTestsRe finds a test runner's report that it ran no tests.
var zeroTestsRe = regexp.MustCompile(`(?m)(^Ran 0 tests\b|\bno tests ran\b|\bcollected 0 items\b|^No tests found|\brunning 0 tests\b|\[no tests to run\]|\[no test files\])`)

// goPassedRe is a Go package that ran tests and passed, timed or "(cached)":
// a cached pass is the recorded result of these exact sources and tests.
var goPassedRe = regexp.MustCompile(`(?m)^ok\s+\S+\s+([\d.]+s|\(cached\))(\s|$)`)

// ranNoTests reports whether a test command's output says it executed no
// tests: such a run passes without checking anything. Go output counts as
// a run when any package passed with tests (not "[no tests to run]").
func ranNoTests(output string) bool {
	if !zeroTestsRe.MatchString(output) {
		return false
	}
	if goPassedRe.MatchString(output) {
		for _, line := range strings.Split(output, "\n") {
			if goPassedRe.MatchString(line) && !strings.Contains(line, "[no tests to run]") {
				return false // some package really ran tests
			}
		}
	}
	return true
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
	segs := programs(command)
	if testScope(command) != "" {
		return "test"
	}
	for _, seg := range segs {
		if startsWith(buildCheckRe, seg) {
			return "build"
		}
	}
	for _, seg := range segs {
		if startsWith(runCheckRe, seg) {
			return "run"
		}
	}
	for _, seg := range segs {
		words := strings.Fields(seg)
		name, args := words[0], words[1:]
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
