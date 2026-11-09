package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// RunShell runs a command the user typed as "!command", as in OpenCode. The
// user typing it is the approval, so no prompt is shown, but everything else
// matches a model-requested shell call: plan mode refuses it, uncertain effects
// block it, and tool_start/tool_result records make an interrupted command an
// uncertain effect on resume. The conversation records the user's "!command",
// the shell call and its output (as tool data), so the model sees the result
// on the next prompt. No model request is made.
func (a *Agent) RunShell(ctx context.Context, command string) (status string, err error) {
	a.seq = 0
	a.runID = ID()
	status = "failed"
	started := time.Now()
	a.event("run_started", map[string]any{"operation": "user_shell", "project": a.Store.Project})
	defer func() { status, err = a.endRun(ctx, status, err, started, false) }()
	command = strings.TrimSpace(command)
	if command == "" {
		return "failed", errors.New("usage: !command")
	}
	if e := a.checkMode("shell"); e != nil {
		return "failed", e
	}
	if len(a.Store.Pending()) > 0 {
		a.event("recovery_required", map[string]any{"pending": a.Store.Pending()})
		return "failed", errors.New("uncertain tool effects require acknowledgment before running commands")
	}
	args, _ := json.Marshal(map[string]string{"command": command})
	if _, _, e := a.Tools.Prepare("shell", args); e != nil {
		return "failed", e
	}
	call := core.Call{ID: "user_shell_" + ID(), Type: "function"}
	call.Function.Name, call.Function.Arguments = "shell", string(args)
	approval := map[string]any{"tool": "shell", "allowed": true, "session": false, "source": "user_typed"}
	if e := a.Store.Append("permission", approval); e != nil {
		return "failed", e
	}
	if e := a.commit(core.Message{Role: "user", Content: "!" + command}); e != nil {
		return "failed", e
	}
	if e := a.commit(core.Message{Role: "assistant", ToolCalls: []core.Call{call}}); e != nil {
		return "failed", e
	}
	if e := a.start(call); e != nil {
		return "failed", e
	}
	result := a.execute(ctx, call)
	if e := a.finish(call, result); e != nil {
		return "failed", e
	}
	if e := a.saveResults([]core.Call{call}, []tools.Result{result}); e != nil {
		return "failed", e
	}
	if ctx.Err() != nil {
		return "cancelled", ctx.Err()
	}
	return "completed", nil
}
