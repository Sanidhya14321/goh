package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Budget awareness. A run capped by max_run_seconds, max_tool_calls or
// max_run_tokens tells the model how much is left when a budget crosses
// half, three quarters and nine tenths of its limit, so it can plan the
// rest of the run instead of being stopped mid-task. "Budget-Aware Tool-Use
// Enables Effective Agent Scaling" (arXiv:2511.17006) shows an agent that
// sees its remaining budget uses it better; here the note is sent at most
// three times per budget, on the newest tool result, so earlier requests
// stay byte-identical for the prompt cache. Runs without a cap get nothing.

var budgetMarks = []int{50, 75, 90}

// budgetState is per-run: the highest mark already reported per budget,
// and when the run began.
type budgetState struct {
	started  time.Time
	reported map[string]int
}

func (a *Agent) resetBudget(started time.Time) {
	a.budget = budgetState{started: started, reported: map[string]int{}}
}

// budgetNote returns one line when a budget has crossed a mark not yet
// reported, or "". Only the top-level run reports: child tasks share the
// parent's time and stop with their own error.
func (a *Agent) budgetNote(ctx context.Context) string {
	if a.depth != 0 || a.budget.reported == nil {
		return ""
	}
	var parts []string
	crossed, top := false, 0
	check := func(name string, used, limit float64, left string) {
		if limit <= 0 {
			return
		}
		pct := int(used * 100 / limit)
		mark := 0
		for _, m := range budgetMarks {
			if pct >= m {
				mark = m
			}
		}
		if mark > a.budget.reported[name] {
			a.budget.reported[name] = mark
			crossed = true
		}
		if mark > 0 {
			top = max(top, pct)
			parts = append(parts, left)
		}
	}
	if s := a.Config.MaxRunSeconds; s > 0 {
		used := time.Since(a.budget.started).Seconds()
		if d, ok := ctx.Deadline(); ok {
			used = float64(s) - time.Until(d).Seconds()
		}
		left := max(0, float64(s)-used)
		check("time", used, float64(s), fmt.Sprintf("%s of %s left", shortDuration(left), shortDuration(float64(s))))
	}
	if n := a.Config.MaxToolCalls; n > 0 {
		check("tool_calls", float64(a.toolCalls), float64(n), fmt.Sprintf("%d of %d tool calls left", max(0, n-a.toolCalls), n))
	}
	if n := a.Config.MaxRunTokens; n > 0 {
		check("tokens", float64(a.runTokens), float64(n), fmt.Sprintf("%d of %d tokens left", max(0, n-a.runTokens), n))
	}
	if !crossed {
		return ""
	}
	advice := "Plan the remaining work to fit."
	if top >= 90 {
		advice = "Finish now: make sure the required results are in place."
	}
	return "Run budget: " + strings.Join(parts, "; ") + ". " + advice
}

// shortDuration renders seconds as "45s", "12m" or "1h05m".
func shortDuration(sec float64) string {
	s := int(sec + 0.5)
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	default:
		return fmt.Sprintf("%dh%02dm", s/3600, s/60%60)
	}
}
