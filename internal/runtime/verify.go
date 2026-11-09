package runtime

import (
	"context"
	"encoding/json"
	"fmt"

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
	if v == nil || !a.changedFiles || a.depth > 0 || a.isPlan() {
		return false, nil
	}
	timeout := v.Timeout
	if timeout == 0 {
		timeout = 300
	}
	args, _ := json.Marshal(map[string]any{"command": v.Command, "timeout_seconds": timeout})
	res := a.Tools.Execute(ctx, "shell", args)
	var out struct {
		ExitCode   *int   `json:"exit_code"`
		Output     string `json:"output"`
		ArtifactID string `json:"artifact_id"`
	}
	_ = json.Unmarshal([]byte(res.Content), &out)
	passed := !res.Error && out.ExitCode != nil && *out.ExitCode == 0
	record := map[string]any{"command": v.Command, "passed": passed, "repair_attempt": a.repairsUsed, "artifact_id": out.ArtifactID}
	if out.ExitCode != nil {
		record["exit_code"] = *out.ExitCode
	}
	if e := a.Store.Append("verification", record); e != nil {
		return false, e
	}
	a.event("verification", record)
	if passed {
		a.verifiedAfterChange, a.verifyFailed = true, false
		return false, nil
	}
	a.verifiedAfterChange = false
	allowed := 1
	if v.RepairAttempts != nil {
		allowed = *v.RepairAttempts
	}
	if a.repairsUsed >= allowed {
		a.verifyFailed = true
		return false, nil
	}
	a.repairsUsed++
	tail := out.Output
	if tail == "" {
		tail = res.Content
	}
	if len(tail) > 6000 {
		tail = "…" + tail[len(tail)-6000:]
	}
	msg := fmt.Sprintf("[Automatic check, not a new request] The configured verification command failed after your changes (repair attempt %d of %d).\nCommand: %s\nOutput (end):\n%s\nFix the cause, then finish.", a.repairsUsed, allowed, v.Command, tail)
	return true, a.commit(core.Message{Role: "user", Content: msg, Automatic: true})
}
