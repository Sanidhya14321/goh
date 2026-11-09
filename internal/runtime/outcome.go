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
