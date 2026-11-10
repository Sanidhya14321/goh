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
	OutcomeVerified   = "verified"   // a shell command succeeded after the last change
	OutcomeUnverified = "unverified" // files changed and nothing checked them afterwards
)

var fileChangeTools = map[string]bool{"create_file": true, "write_file": true, "edit_file": true, "apply_patch": true}

// trackOutcome updates the run outcome from a finished tool call. This is a
// heuristic: any successful shell command after the last edit counts as a
// check, whatever it ran.
func (a *Agent) trackOutcome(c core.Call, r tools.Result) {
	if r.Error {
		return
	}
	switch {
	case fileChangeTools[c.Function.Name]:
		a.changedFiles, a.verifiedAfterChange = true, false
		if writesTestDouble(c) {
			a.wroteTestDouble = true
		}
	case c.Function.Name == "shell" && a.changedFiles:
		var out struct {
			ExitCode *int `json:"exit_code"`
		}
		if json.Unmarshal([]byte(r.Content), &out) == nil && out.ExitCode != nil && *out.ExitCode == 0 {
			a.verifiedAfterChange = true
		}
	}
}

func (a *Agent) outcome() string {
	switch {
	case a.verifyFailed:
		return OutcomeFailedVerification
	case !a.changedFiles:
		return OutcomeNoChanges
	case a.verifiedAfterChange:
		return OutcomeVerified
	}
	return OutcomeUnverified
}

// testDoubleRe finds a stand-in the model defines for code it did not write.
var testDoubleRe = regexp.MustCompile(`(?i)\b(class|type|struct|def|func|function|const|let|var)\s+\w*(fake|mock|stub|dummy)\w*`)

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
	return a.changedFiles && (!a.verifiedAfterChange || a.wroteTestDouble)
}

// completionCheck returns the reminder to send and the reason recorded with it.
func (a *Agent) completionCheck() (prompt, reason string) {
	if a.verifiedAfterChange && a.wroteTestDouble {
		return testDoubleCheckPrompt, "checks used a test double written in this run"
	}
	return completionCheckPrompt, "files changed without a passing check"
}
