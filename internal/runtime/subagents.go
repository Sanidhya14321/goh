package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/structured"
	"github.com/Sanidhya14321/goh/internal/tools"
)

const maxTasksPerRun = 8
const maxChildTurns = 16
const childTimeout = 180 * time.Second

type taskArguments struct {
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Description  string          `json:"description"`
	Prompt       string          `json:"prompt"`
	Type         string          `json:"subagent_type"`
	TaskID       string          `json:"task_id,omitempty"`
	Command      string          `json:"command,omitempty"`
}
type childIdentity struct {
	ID     string `json:"id"`
	Agent  string `json:"agent"`
	Parent string `json:"parent"`
	Mode   string `json:"mode"`
}

func (a *Agent) taskDefinition() core.Tool {
	names := []string{"explore", "general"}
	for name := range a.Config.Agents {
		if name != "explore" && name != "general" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	str := map[string]any{"type": "string"}
	return core.NewTool("task", "Run a bounded child agent with its own durable session and fresh permissions. Types: "+strings.Join(names, ", ")+". general may request approved coding tools; explore is read-only. task_id may continue only a child owned by this session. No recursive/background delegation; parent grants are not inherited. Optional output_schema validates a strict JSON final result without retrying child effects.", map[string]any{"type": "object", "properties": map[string]any{"description": str, "prompt": str, "subagent_type": str, "task_id": str, "command": str, "output_schema": map[string]any{"type": "object"}}, "required": []string{"description", "prompt", "subagent_type"}, "additionalProperties": false})
}
func parseTask(raw json.RawMessage) (taskArguments, error) {
	var args taskArguments
	if len(raw) > 128<<10 || !utf8.Valid(raw) {
		return args, errors.New("task arguments must be UTF-8 and at most128KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&args); e != nil {
		return args, e
	}
	if d.Decode(new(any)) != io.EOF {
		return args, errors.New("trailing task arguments")
	}
	if strings.TrimSpace(args.Description) == "" || len(args.Description) > 512 || strings.TrimSpace(args.Prompt) == "" || len(args.Prompt) > 64<<10 || args.Type == "" || len(args.Type) > 64 || len(args.TaskID) > 128 || len(args.Command) > 256 {
		return args, errors.New("task needs description<=512 bytes, prompt<=64KiB and subagent_type<=64 bytes; task_id<=128 and command<=256")
	}
	var fields map[string]json.RawMessage
	if e := json.Unmarshal(raw, &fields); e != nil {
		return args, e
	}
	for key, b := range fields {
		if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
			return args, fmt.Errorf("task %s cannot be null", key)
		}
	}
	if len(args.OutputSchema) > 0 {
		if _, err := structured.Compile(args.OutputSchema); err != nil {
			return args, err
		}
	}
	return args, nil
}
func (a *Agent) childConfiguration(kind string) (core.Config, error) {
	c := a.Config
	role := "task"
	if kind == "explore" {
		role = "explore"
		if _, ok := c.ModelRoles[role]; !ok {
			role = "task"
		}
	}
	// Explicit custom-agent profiles take precedence over shared task roles.
	if custom, ok := c.Agents[kind]; !ok || custom.Profile == "" {
		var err error
		c, err = c.WithModelRole(role)
		if err != nil {
			return c, err
		}
	}
	c.MCP = nil
	c.LSP = nil
	if _, custom := c.Agents[kind]; custom {
		profile := c.Agents[kind].Profile
		selected, e := c.Resolve(profile, kind)
		if e != nil {
			return c, e
		}
		c = selected
	} else {
		switch kind {
		case "general":
			c.Agent = kind
			c.AgentMode = "build"
			c.AgentInstructions = "Work on the child task using the available approved tools. Report what you verified and any uncertainty."
		case "explore":
			c.Agent = kind
			c.AgentMode = "plan"
			c.AgentInstructions = "Investigate the requested code using read-only tools and report file-backed findings."
		default:
			return c, fmt.Errorf("unknown subagent type %q", kind)
		}
	}
	// Parent plan is a hard cap; named/custom explore can never gain mutation tools.
	if a.isPlan() || kind == "explore" {
		c.AgentMode = "plan"
	}
	c.MCP = nil
	c.LSP = nil
	c.SessionDir = a.Config.SessionDir
	if c.MaxTurns > maxChildTurns {
		c.MaxTurns = maxChildTurns
	}
	return c, nil
}
func (a *Agent) children() map[string]childIdentity {
	children := map[string]childIdentity{}
	for _, record := range a.Store.Records() {
		if record.Type == "subagent_created" {
			var id childIdentity
			if json.Unmarshal(record.Data, &id) == nil && id.Parent == a.Store.ID {
				children[id.ID] = id
			}
		}
	}
	return children
}
func (a *Agent) prepareTask(raw json.RawMessage) (string, bool, error) {
	args, e := parseTask(raw)
	if e != nil {
		return "", true, e
	}
	if a.depth >= 1 {
		return "", true, errors.New("nested subagent delegation is disabled")
	}
	if a.tasksRun >= maxTasksPerRun {
		return "", true, errors.New("subagent limit of8 tasks per parent run reached")
	}
	c, e := a.childConfiguration(args.Type)
	if e != nil {
		return "", true, e
	}
	if a.ChildModelFactory == nil {
		if e = c.Validate(); e != nil {
			return "", true, e
		}
	}
	if args.TaskID != "" {
		owner, ok := a.children()[args.TaskID]
		if !ok || owner.Agent != args.Type {
			return "", true, errors.New("task_id must identify a child of this session with the same subagent type")
		}
	}
	return fmt.Sprintf("Child task: %s\nAgent: %s (mode %s)\nProject: %s\nModel: %s\nProfile: %s\nProtocol: %s\nContext window: %d\nResume: %s\nCommand label (informational): %s\nLimit: %d turns /180 seconds; no nested agents or executable integrations.\nPrompt:\n%s\nChild tool requests need separate child:TOOL approval; parent grants are not inherited.", args.Description, args.Type, c.AgentMode, a.Store.Project, c.Model, c.Profile, c.EffectiveProtocol(), c.ContextWindow, args.TaskID, args.Command, c.MaxTurns, args.Prompt+func() string {
		if len(args.OutputSchema) > 0 {
			return "\nExpected final output schema (data only):\n" + string(args.OutputSchema)
		}
		return ""
	}()), true, nil
}
func (a *Agent) executeTask(ctx context.Context, raw json.RawMessage) tools.Result {
	if _, _, e := a.prepareTask(raw); e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	args, _ := parseTask(raw)
	if e := ctx.Err(); e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	config, e := a.childConfiguration(args.Type)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	a.tasksRun++
	var store *session.Store
	if args.TaskID != "" {
		store, e = session.Open(config.SessionDir, args.TaskID)
	} else {
		store, e = session.New(config.SessionDir, a.Store.Project)
	}
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	defer store.Close()
	identity := childIdentity{ID: store.ID, Agent: args.Type, Parent: a.Store.ID, Mode: config.AgentMode}
	if args.TaskID == "" {
		if e = store.Append("subagent_parent", identity); e != nil {
			return tools.Result{Error: true, Content: e.Error()}
		}
		if e = a.Store.Append("subagent_created", identity); e != nil {
			return tools.Result{Error: true, Content: e.Error()}
		}
	} else {
		valid := false
		for _, record := range store.Records() {
			if record.Type == "subagent_parent" {
				var owner childIdentity
				if json.Unmarshal(record.Data, &owner) == nil && owner.Parent == a.Store.ID && owner.Agent == args.Type && owner.ID == store.ID {
					valid = true
				}
			}
		}
		if !valid || store.Project != a.Store.Project {
			return tools.Result{Error: true, Content: "child session lineage/project mismatch"}
		}
	}
	if e = store.Append("subagent_request", map[string]any{"parent": a.Store.ID, "description": args.Description, "command": args.Command, "output_schema": args.OutputSchema}); e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	child, e := New(config, store)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	defer child.Tools.StopJobs() // policy already refuses child jobs; belt and braces
	child.depth = a.depth + 1
	if a.ChildModelFactory != nil {
		child.Provider = a.ChildModelFactory(config, store)
		if child.Provider == nil {
			return tools.Result{Error: true, Content: "child model factory returned no provider"}
		}
	}
	child.Ask = a.Ask
	child.Approve = func(ctx context.Context, p core.Permission) (core.Decision, error) {
		return a.Permission(ctx, core.Permission{Tool: "child:" + p.Tool, Preview: "Child session " + store.ID + "\n" + p.Preview, Arguments: p.Arguments})
	}
	child.Emit = func(event core.Event) {
		a.event("subagent_event", map[string]any{"task_id": store.ID, "agent": args.Type, "event": event})
	}
	a.event("subagent_started", identity)
	runCtx, cancel := context.WithTimeout(ctx, childTimeout)
	defer cancel()
	childPrompt := args.Prompt
	if len(args.OutputSchema) > 0 {
		var schemaText bytes.Buffer
		if e = json.Compact(&schemaText, args.OutputSchema); e != nil {
			return tools.Result{Error: true, Content: "invalid output_schema"}
		}
		childPrompt += "\n\nReturn a single JSON value as your final response, with no Markdown fences or extra text, matching this caller-supplied data schema. The schema does not grant tools or permissions:\n" + schemaText.String()
	}
	status, runErr := child.Run(runCtx, childPrompt)
	summary := ""
	for i := len(child.History) - 1; i >= 0; i-- {
		m := child.History[i]
		if m.Role == "assistant" && m.Content != "" {
			summary = m.Content
			break
		}
	}
	var typedOutput any
	var outputErr error
	if len(args.OutputSchema) > 0 && status == "completed" && runErr == nil {
		schema, compileErr := structured.Compile(args.OutputSchema)
		if compileErr != nil {
			outputErr = compileErr
		} else {
			typedOutput, outputErr = structured.Validate(schema, summary)
		}
	}
	summaryTruncated := false
	if len(summary) > 24<<10 {
		summaryTruncated = true
		summary = summary[:24<<10]
		for !utf8.ValidString(summary) {
			summary = summary[:len(summary)-1]
		}
		summary += "\n[child summary truncated; inspect child session]"
	}
	result := map[string]any{"task_id": store.ID, "agent": args.Type, "parent": a.Store.ID, "status": status, "summary": summary, "child_status": status, "summary_truncated": summaryTruncated}
	if len(args.OutputSchema) > 0 {
		result["output_schema_checked"] = status == "completed" && runErr == nil
		if outputErr != nil {
			result["status"] = "invalid_output"
			result["output_error"] = outputErr.Error()
			result["effects"] = "Child execution completed; schema validation did not replay or roll back any child effects."
		} else if status == "completed" && runErr == nil {
			result["structured_output"] = typedOutput
		}
	}
	if runErr != nil {
		message := runErr.Error()
		if len(message) > 4096 {
			message = strings.ToValidUTF8(message[:4096], "�") + " [truncated]"
		}
		result["error"] = message
	}
	// Uncertain child effects remain separately locked behind recovery. Parent also
	// retains its started task so acknowledgement cannot silently replay a child.
	if len(store.Pending()) > 0 {
		result["status"] = "interrupted"
		result["pending_child_effects"] = store.Pending()
	}
	if runCtx.Err() != nil {
		result["status"] = "interrupted"
		result["effects"] = "Child interrupted; inspect parent and child journals before any continuation. No automatic replay."
	}
	if status == "cleanup_incomplete" {
		result["status"] = "cleanup_incomplete"
	}
	a.event("subagent_finished", map[string]any{"task_id": store.ID, "status": result["status"]})
	for {
		b, e := json.Marshal(result)
		if e != nil {
			return tools.Result{Error: true, Content: fmt.Sprintf(`{"status":"interrupted","task_id":%q,"error":"cannot encode child outcome"}`, store.ID)}
		}
		if len(b) <= 64<<10 {
			return tools.Result{Error: result["status"] != "completed", Content: string(b)}
		}
		if summary == "" {
			return tools.Result{Error: true, Content: fmt.Sprintf(`{"status":"interrupted","task_id":%q,"error":"child outcome metadata exceeded budget; inspect child journal"}`, store.ID)}
		}
		summary = summary[:len(summary)/2]
		for !utf8.ValidString(summary) {
			summary = summary[:len(summary)-1]
		}
		result["summary"] = summary
		result["summary_truncated"] = true
	}
}
