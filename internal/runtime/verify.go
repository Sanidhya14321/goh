package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
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
func (a *Agent) verifyAndMaybeRepair(ctx context.Context) (retry bool, status string, err error) {
	v := a.Config.Verify
	if v == nil || !a.mayHaveChanged() || a.depth > 0 || a.isPlan() {
		return false, "", nil
	}
	timeout := v.Timeout
	if timeout == 0 {
		timeout = 300
	}
	allowed := 1
	if v.RepairAttempts != nil {
		allowed = *v.RepairAttempts
	}
	var (
		passed, worktreeChanged bool
		revision                string
		out                     struct {
			ExitCode   *int   `json:"exit_code"`
			Output     string `json:"output"`
			ArtifactID string `json:"artifact_id"`
			Status     string `json:"status"`
		}
		res tools.Result
	)
	// A pass counts only for the revision it checked: a verifier that
	// rewrites files (a formatter, a code generator) is run once more on
	// its own result, and if it changes files again the outcome stays
	// unverified.
	for run := 0; run < 2; run++ {
		var st string
		res, st, err = a.runVerifier(ctx, v.Command, timeout, allowed)
		if err != nil {
			return false, st, err
		}
		out.ExitCode, out.Output, out.ArtifactID, out.Status = nil, "", "", ""
		_ = json.Unmarshal([]byte(res.Content), &out)
		passed = !res.Error && out.ExitCode != nil && *out.ExitCode == 0
		before := a.fingerprint
		worktreeChanged, revision = false, ""
		if !a.noFingerprint && a.Tools != nil {
			if now, ok := a.Tools.Fingerprint(); ok {
				worktreeChanged = before != "" && now != before
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
			record["revision_before"] = before
		}
		if revision != "" {
			record["revision"] = revision
		}
		if e := a.Store.Append("verification", record); e != nil {
			return false, "failed", e
		}
		a.event("verification", record)
		if worktreeChanged {
			a.changedFiles = true
		}
		if !passed || !worktreeChanged {
			break
		}
	}
	a.evidence = &Evidence{Source: "configured", Command: v.Command, Kind: "custom", Passed: passed, ExitCode: out.ExitCode, Revision: revision, RepairAttempt: a.repairsUsed, ArtifactID: out.ArtifactID}
	if passed && worktreeChanged {
		// The check kept rewriting files: its pass describes a revision that
		// no longer exists. Not verified, and not a failure either.
		a.evidence.Revision = ""
		a.verifiedAfterChange, a.verifyFailed = false, false
		return false, "", nil
	}
	if passed {
		a.verifiedAfterChange, a.verifyFailed = true, false
		return false, "", nil
	}
	a.verifiedAfterChange = false
	if ctx.Err() != nil {
		return false, "", nil // interrupted: the run ends as cancelled, not as a failed check
	}
	if a.repairsUsed >= allowed {
		a.verifyFailed = true
		return false, "", nil
	}
	a.repairsUsed++
	tail := a.outputTail(ctx, out.ArtifactID, out.Output, res.Content)
	msg := fmt.Sprintf("[Automatic check, not a new request] The configured verification command failed after your changes (repair attempt %d of %d).\nCommand: %s\nOutput (end):\n%s\nFix the cause, then finish.", a.repairsUsed, allowed, v.Command, tail)
	if e := a.commit(core.Message{Role: "user", Content: msg, Automatic: true}); e != nil {
		return false, "failed", e
	}
	return true, "", nil
}

// runVerifier runs the configured check as a durable effect: its tool_start
// is journaled before it runs (so an interruption leaves a pending effect
// that the next run must acknowledge) and its result only once the command
// has settled. An interrupted check, or one whose cleanup is uncertain,
// stays pending and stops the run before any further model work.
func (a *Agent) runVerifier(ctx context.Context, command string, timeout, allowed int) (tools.Result, string, error) {
	startRec := map[string]any{"command": command, "repair_attempt": a.repairsUsed, "repair_allowance": allowed, "change_tracking": a.outcomeDetail()["change_tracking"]}
	if e := a.Store.Append("verification_started", startRec); e != nil {
		return tools.Result{}, "failed", e
	}
	a.event("verification_started", startRec)
	args, _ := json.Marshal(map[string]any{"command": command, "timeout_seconds": timeout})
	call := core.Call{ID: "verify_" + ID(), Type: "function"}
	call.Function.Name, call.Function.Arguments = "verify", string(args)
	if e := a.start(call); e != nil {
		return tools.Result{}, "failed", e
	}
	res := a.Tools.Execute(ctx, "shell", args)
	var settled struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Content), &settled)
	switch {
	case settled.Status == "cleanup_incomplete":
		if e := a.Store.Append("tool_observation", map[string]any{"id": call.ID, "result": res}); e != nil {
			return res, "failed", e
		}
		return res, "cleanup_incomplete", errors.New("verification command cleanup incomplete; inspect the project and acknowledge before continuing")
	case settled.Status == "interrupted" || settled.Status == "timed_out" && ctx.Err() != nil || settled.Status == "cancelled" || ctx.Err() != nil:
		if e := a.Store.Append("tool_observation", map[string]any{"id": call.ID, "result": res}); e != nil {
			return res, "failed", e
		}
		if ctx.Err() != nil {
			return res, "cancelled", ctx.Err()
		}
		return res, "failed", errors.New("verification command was interrupted; its effects are uncertain")
	}
	if e := a.finish(call, res); e != nil {
		return res, "failed", e
	}
	return res, "", nil
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
