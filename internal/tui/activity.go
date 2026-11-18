package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

// activity is what a run is doing right now, for the status line: waiting
// for the model, thinking, writing the reply, writing a tool call (with its
// size so far) or running a tool.
type activity struct {
	phase  string // "", "waiting", "thinking", "reply", "writing", "running"
	tool   string
	target string
	chars  int
	since  time.Time // when this phase started
	steps  int       // model requests in this run
	tools  int       // tools started in this run
}

// track updates the activity from a run event.
func (a *activity) track(e core.Event) {
	now := time.Now()
	set := func(phase string) {
		if a.phase != phase {
			a.phase, a.since = phase, now
		}
	}
	switch e.Type {
	case "run_started":
		*a = activity{}
	case "assistant_started":
		a.steps++
		a.tool, a.target, a.chars = "", "", 0
		a.phase, a.since = "waiting", now
	case "reasoning_updated":
		set("thinking")
	case "assistant_updated":
		set("reply")
	case "tool_progress":
		p, ok := e.Data.(core.ToolProgress)
		if !ok {
			b, _ := json.Marshal(e.Data)
			json.Unmarshal(b, &p)
		}
		if a.phase != "writing" || a.tool != p.Tool {
			a.phase, a.since = "writing", now
		}
		a.tool, a.target, a.chars = Safe(p.Tool), Safe(p.Target), p.Chars
	case "tool_started":
		b, _ := json.Marshal(e.Data)
		var d struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		json.Unmarshal(b, &d)
		a.tools++
		a.phase, a.since = "running", now
		a.tool, a.target, a.chars = Safe(d.Name), Safe(oneLine(toolArgumentText(d.Arguments), 60)), 0
	case "tool_finished":
		a.phase = ""
	case "run_finished":
		*a = activity{}
	}
}

// label is the activity in words, with how long it has lasted.
func (a *activity) label(model string) string {
	elapsed := ""
	if !a.since.IsZero() {
		if s := time.Since(a.since); s >= time.Second {
			elapsed = " · " + humanDuration(s.Milliseconds()/1000*1000)
		}
	}
	target := a.target
	if target == "" {
		target = a.tool
	}
	switch a.phase {
	case "waiting":
		if model == "" {
			model = "the model"
		}
		return "Waiting for " + model + elapsed
	case "thinking":
		return "Thinking" + elapsed
	case "reply":
		return "Writing the reply"
	case "writing":
		label := "Writing " + target
		if a.tool == "shell" || a.tool == "job_start" {
			label = "Preparing command " + target
		}
		if tokens := a.chars / 4; tokens > 0 {
			label += " · " + strings.TrimSuffix(humanTokens(tokens), " tokens") + " tokens"
			if s := time.Since(a.since).Seconds(); s >= 2 {
				label += fmt.Sprintf(" (%d/s)", int(float64(tokens)/s))
			}
		}
		return label
	case "running":
		if a.tool == "shell" || a.tool == "job_start" {
			return "Running " + a.target + elapsed
		}
		return "Running " + a.tool + " " + a.target + elapsed
	}
	return "Working"
}

// counts is "step 3 · 5 tools" once a run has done more than one thing.
func (a *activity) counts() string {
	if a.steps <= 1 && a.tools == 0 {
		return ""
	}
	s := fmt.Sprintf("step %d", a.steps)
	switch a.tools {
	case 0:
	case 1:
		s += " · 1 tool"
	default:
		s += fmt.Sprintf(" · %d tools", a.tools)
	}
	return s
}
