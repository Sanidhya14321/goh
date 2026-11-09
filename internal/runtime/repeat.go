package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sanidhya14321/goh/internal/core"
)

// repeatLimit is how many identical consecutive calls (same tool and exact
// arguments) are allowed before the user is asked, as in OpenCode's doom-loop
// guard. Repetition is a common failure of tool-using agents (SWE-agent); a
// stuck model otherwise spends its whole turn budget.
const repeatLimit = 3

const repeatStopped = "Stopped: this exact call was repeated"

// checkRepeat returns a non-empty tool result text when the call must not run.
// From the third identical call on, the user must approve it even if the tool
// itself is already granted; without an interactive approver the run stops.
// A "session" approval of repeated_call allows repeats for the rest of the
// session. Returned errors are journal failures.
func (a *Agent) checkRepeat(ctx context.Context, c core.Call, preview string) (string, error) {
	key := c.Function.Name + "\x00" + c.Function.Arguments
	if key == a.lastCall {
		a.repeats++
	} else {
		a.lastCall, a.repeats = key, 1
	}
	if a.repeats < repeatLimit || c.Function.Name == "question" || c.Function.Name == "job_output" {
		return "", nil
	}
	if len(preview) > 2048 {
		preview = preview[:2048] + "…"
	}
	d, err := a.Permission(ctx, core.Permission{Tool: "repeated_call",
		Preview:   fmt.Sprintf("The model has requested this exact %s call %d times in a row. It may be stuck.\n\n%s\n\nAllow it again?", c.Function.Name, a.repeats, preview),
		Arguments: []byte(c.Function.Arguments)})
	switch {
	case errors.Is(err, errPermission):
		return fmt.Sprintf("%s %d times; nobody was available to approve it.", repeatStopped, a.repeats), nil
	case err != nil:
		if ctx.Err() != nil {
			return "Not executed: run interrupted", nil
		}
		return "", err
	case !d.Allow:
		return fmt.Sprintf("Permission denied: the user declined repeating this exact %s call. Try a different approach.", c.Function.Name), nil
	}
	return "", nil
}
