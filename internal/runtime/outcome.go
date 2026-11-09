package runtime

import (
	"encoding/json"

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
	return a.changedFiles && !a.verifiedAfterChange
}
