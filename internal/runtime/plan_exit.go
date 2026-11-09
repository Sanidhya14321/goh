package runtime

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// plan_exit lets the read-only plan agent propose switching to build mode
// (OpenCode's plan_exit). The switch needs the user's approval through the
// normal permission prompt, which shows the plan; nothing changes mode
// inside the current run. The UI performs the switch after the run ends.

func planExitDefinition() core.Tool {
	return core.NewTool("plan_exit", "When the plan is complete, ask the user to switch to build mode to carry it out. Give the full plan; the user approves or declines. Do not call this before investigating.",
		map[string]any{"type": "object", "properties": map[string]any{"plan": map[string]any{"type": "string"}}, "required": []string{"plan"}, "additionalProperties": false})
}

func parsePlanExit(raw json.RawMessage) (string, error) {
	var args struct {
		Plan string `json:"plan"`
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&args); err != nil || strings.TrimSpace(args.Plan) == "" {
		return "", errors.New("plan_exit needs a non-empty plan")
	}
	if len(args.Plan) > 64<<10 {
		return "", errors.New("plan exceeds 64 KiB")
	}
	return args.Plan, nil
}

func (a *Agent) preparePlanExit(raw json.RawMessage) (string, bool, error) {
	if !a.isPlan() || a.depth > 0 {
		return "", true, errors.New("plan_exit is only available to the plan agent")
	}
	plan, err := parsePlanExit(raw)
	if err != nil {
		return "", true, err
	}
	return "Switch to Build mode and carry out this plan?\n\n" + plan, true, nil
}

func (a *Agent) executePlanExit(raw json.RawMessage) tools.Result {
	plan, err := parsePlanExit(raw)
	if err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	a.HandoffPlan = plan
	return tools.Result{Content: `{"status":"approved","next":"The user approved. Chat continues in build mode after this reply; end your turn now."}`}
}
