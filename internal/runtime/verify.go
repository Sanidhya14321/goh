package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// OutcomeFailedVerification: the configured check still failed after the
// allowed repair attempts.
const OutcomeFailedVerification = "failed_verification"

// verifyAndMaybeRepair runs the user's configured check when the run changed
// files. On success the outcome becomes verified. On failure, while repair
// attempts remain, the output is handed back to the model and the loop goes
// on (retry true); otherwise the outcome records the failure. The check is
// the user's own command from their config, so it runs without a prompt but
// inside the sandbox when that is enabled.
func (a *Agent) verifyAndMaybeRepair(ctx context.Context) (retry bool, err error) {
	v := a.Config.Verify
	if v == nil || !a.mayHaveChanged() || a.depth > 0 || a.isPlan() {
		return false, nil
	}
	timeout := v.Timeout
	if timeout == 0 {
		timeout = 300
	}
	allowed := 1
	if v.RepairAttempts != nil {
		allowed = *v.RepairAttempts
	}
	// The verifier is an effect like any command: its start is journaled
	// first, so an interruption leaves a record of what was running.
	startRec := map[string]any{"command": v.Command, "repair_attempt": a.repairsUsed, "repair_allowance": allowed, "change_tracking": a.outcomeDetail()["change_tracking"]}
	if e := a.Store.Append("verification_started", startRec); e != nil {
		return false, e
	}
	a.event("verification_started", startRec)
	args, _ := json.Marshal(map[string]any{"command": v.Command, "timeout_seconds": timeout})
	res := a.Tools.Execute(ctx, "shell", args)
	var out struct {
		ExitCode   *int   `json:"exit_code"`
		Output     string `json:"output"`
		ArtifactID string `json:"artifact_id"`
		Status     string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Content), &out)
	passed := !res.Error && out.ExitCode != nil && *out.ExitCode == 0
	// The evidence names the revision the check left behind; a check that
	// rewrote files (a formatter, say) is reported as having done so.
	worktreeChanged := false
	revision := ""
	if !a.noFingerprint && a.Tools != nil {
		if now, ok := a.Tools.Fingerprint(); ok {
			worktreeChanged = a.fingerprint != "" && now != a.fingerprint
			a.fingerprint, revision = now, now
		} else {
			a.noFingerprint, a.changeUnknown = true, true
		}
	}
	record := map[string]any{"command": v.Command, "passed": passed, "repair_attempt": a.repairsUsed, "repair_allowance": allowed, "artifact_id": out.ArtifactID}
	if out.ExitCode != nil {
		record["exit_code"] = *out.ExitCode
	}
	if out.Status != "" {
		record["status"] = out.Status
	}
	if worktreeChanged {
		record["worktree_changed"] = true
	}
	if revision != "" {
		record["revision"] = revision
	}
	if e := a.Store.Append("verification", record); e != nil {
		return false, e
	}
	a.event("verification", record)
	a.evidence = &Evidence{Source: "configured", Command: v.Command, Kind: "custom", Passed: passed, ExitCode: out.ExitCode, Revision: revision, RepairAttempt: a.repairsUsed, ArtifactID: out.ArtifactID}
	if worktreeChanged {
		a.changedFiles = true
	}
	if passed {
		a.verifiedAfterChange, a.verifyFailed = true, false
		return false, nil
	}
	a.verifiedAfterChange = false
	if ctx.Err() != nil {
		return false, nil // interrupted: the run ends as cancelled, not as a failed check
	}
	if a.repairsUsed >= allowed {
		a.verifyFailed = true
		return false, nil
	}
	a.repairsUsed++
	tail := a.outputTail(ctx, out.ArtifactID, out.Output, res.Content)
	msg := fmt.Sprintf("[Automatic check, not a new request] The configured verification command failed after your changes (repair attempt %d of %d).\nCommand: %s\nOutput (end):\n%s\nFix the cause, then finish.", a.repairsUsed, allowed, v.Command, tail)
	return true, a.commit(core.Message{Role: "user", Content: msg, Automatic: true})
}

// outputTail returns the last 6000 bytes of a check's output. The shell
// result inlines only the start of long output; the retained artifact holds
// the rest, and failures and summaries are usually at the end.
func (a *Agent) outputTail(ctx context.Context, artifact, inline, raw string) string {
	const max = 6000
	if artifact != "" {
		args, _ := json.Marshal(map[string]any{"id": artifact, "tail": true, "limit": max})
		if r := a.Tools.Execute(ctx, "read_artifact", args); !r.Error {
			var page struct {
				Content string `json:"content"`
				Offset  int64  `json:"offset"`
			}
			if json.Unmarshal([]byte(r.Content), &page) == nil && page.Content != "" {
				if page.Offset > 0 {
					return "…" + page.Content
				}
				return page.Content
			}
		}
	}
	tail := inline
	if tail == "" {
		tail = raw
	}
	if len(tail) > max {
		tail = "…" + strings.ToValidUTF8(tail[len(tail)-max:], "")
	}
	return tail
}
