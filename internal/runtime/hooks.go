package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/platform"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// hookOutputLimit bounds what a hook may add to a result or a refusal.
const hookOutputLimit = 8 << 10

// hookExitBlock is the exit code a pre_tool hook uses to refuse a tool; on a
// post_tool hook it marks its output as feedback the model must see.
const hookExitBlock = 2

type hookOutcome struct {
	exit   int
	stdout string
	stderr string
	err    error
}

// matchingHooks returns configured hooks for event and tool, in config order.
func (a *Agent) matchingHooks(event, tool string) []core.HookConfig {
	var out []core.HookConfig
	for _, h := range a.Config.Hooks {
		if h.Event != event {
			continue
		}
		if len(h.Tools) == 0 {
			out = append(out, h)
			continue
		}
		for _, p := range h.Tools {
			if ok, _ := path.Match(p, tool); ok {
				out = append(out, h)
				break
			}
		}
	}
	return out
}

// runHook runs one hook directly (no shell) in the project directory, with
// the filtered child environment plus GOH_* context, the payload as JSON on
// stdin, a deadline and process-group cleanup.
func (a *Agent) runHook(ctx context.Context, h core.HookConfig, event string, payload map[string]any) hookOutcome {
	timeout := time.Duration(h.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload["event"] = event
	payload["session_id"] = a.Store.ID
	payload["project"] = a.Store.Project
	in, err := json.Marshal(payload)
	if err != nil {
		return hookOutcome{exit: -1, err: err}
	}
	cmd := exec.Command(h.Command, h.Args...)
	cmd.Dir = a.Store.Project
	cmd.Env = append(platform.ChildEnv(), "GOH_EVENT="+event, "GOH_SESSION_ID="+a.Store.ID, "GOH_PROJECT="+a.Store.Project)
	if tool, _ := payload["tool"].(string); tool != "" {
		cmd.Env = append(cmd.Env, "GOH_TOOL="+tool)
	}
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	platform.NewGroup(cmd)
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return hookOutcome{exit: -1, err: err}
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = platform.KillGroup(cmd.Process.Pid)
		<-done
		err = fmt.Errorf("hook timed out after %s", timeout)
		return hookOutcome{exit: -1, stdout: stdout.String(), stderr: stderr.String(), err: err}
	}
	out := hookOutcome{stdout: stdout.String(), stderr: stderr.String()}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			out.exit = exit.ExitCode()
		} else {
			out.exit, out.err = -1, err
		}
	}
	return out
}

// preToolHooks runs pre_tool hooks after approval and before the tool. Any
// refusal (exit 2) or hook failure stops the tool: hooks fail closed.
func (a *Agent) preToolHooks(ctx context.Context, c core.Call) (tools.Result, bool) {
	for _, h := range a.matchingHooks("pre_tool", c.Function.Name) {
		o := a.runHook(ctx, h, "pre_tool", map[string]any{"tool": c.Function.Name, "call_id": c.ID, "arguments": json.RawMessage(c.Function.Arguments)})
		if o.exit == 0 && o.err == nil {
			continue
		}
		reason := strings.TrimSpace(o.stderr)
		if reason == "" {
			reason = strings.TrimSpace(o.stdout)
		}
		msg := "Blocked by a user hook (" + h.Command + ")"
		if o.exit != hookExitBlock {
			msg = "A user hook failed, so the tool was not run (" + h.Command + ")"
			if o.err != nil {
				reason = strings.TrimSpace(o.err.Error() + " " + reason)
			} else {
				reason = strings.TrimSpace(fmt.Sprintf("exit %d %s", o.exit, reason))
			}
		}
		if reason != "" {
			msg += ": " + reason
		}
		a.event("hook", map[string]any{"event": "pre_tool", "tool": c.Function.Name, "command": h.Command, "blocked": true, "exit": o.exit})
		return tools.Result{Error: true, Content: msg}, false
	}
	return tools.Result{}, true
}

// postToolHooks runs post_tool hooks after a tool. Output from a hook that
// exits 2 is attached to the result for the model; other outcomes are only
// reported as events, since the tool's effects have already happened.
func (a *Agent) postToolHooks(ctx context.Context, c core.Call, res tools.Result) tools.Result {
	for _, h := range a.matchingHooks("post_tool", c.Function.Name) {
		o := a.runHook(ctx, h, "post_tool", map[string]any{"tool": c.Function.Name, "call_id": c.ID, "arguments": json.RawMessage(c.Function.Arguments), "error": res.Error, "result": res.Content})
		a.event("hook", map[string]any{"event": "post_tool", "tool": c.Function.Name, "command": h.Command, "exit": o.exit})
		if o.exit != hookExitBlock {
			continue
		}
		feedback := strings.TrimSpace(o.stderr + "\n" + o.stdout)
		if feedback == "" {
			continue
		}
		var body map[string]any
		if json.Unmarshal([]byte(res.Content), &body) != nil || body == nil {
			body = map[string]any{"content": res.Content}
		}
		notes, _ := body["hook_feedback"].([]any)
		body["hook_feedback"] = append(notes, map[string]any{"command": h.Command, "output": feedback})
		if b, err := json.Marshal(body); err == nil && len(b) <= tools.MaxInline {
			res.Content = string(b)
		}
	}
	return res
}

// runFinishedHooks notifies hooks that a run ended; they cannot change it.
func (a *Agent) runFinishedHooks(status string, err error) {
	hooks := a.matchingHooks("run_finished", "")
	if len(hooks) == 0 {
		return
	}
	payload := map[string]any{"status": status}
	if err != nil {
		payload["error"] = err.Error()
	}
	for _, h := range hooks {
		// Not tied to the run's context: a cancelled run still notifies.
		o := a.runHook(context.Background(), h, "run_finished", payload)
		a.event("hook", map[string]any{"event": "run_finished", "command": h.Command, "exit": o.exit})
	}
}

// limitedBuffer keeps the first hookOutputLimit bytes and discards the rest.
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := hookOutputLimit - b.Len(); room > 0 {
		if len(p) > room {
			b.Buffer.Write(p[:room])
		} else {
			b.Buffer.Write(p)
		}
	}
	return len(p), nil
}
