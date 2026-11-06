package mcpclient

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
)

type scriptedModel struct {
	tool  string
	turns int
}

func (m *scriptedModel) Complete(_ context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	m.turns++
	if m.turns > 1 {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "Verified fixture response."}, Finish: "stop"}, nil
	}
	var call core.Call
	call.ID = "fixture-call"
	call.Type = "function"
	call.Function.Name = m.tool
	call.Function.Arguments = `{}`
	if m.tool == "mcp__fixture__echo" {
		call.Function.Arguments = `{"text":"paired result"}`
	}
	return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, Finish: "tool_calls"}, nil
}
func TestAgentMCPJournalsAndRecovery(t *testing.T) {
	for _, test := range []struct {
		tool, status string
		pending      bool
	}{{"echo", "completed", false}, {"disconnect", "failed", true}} {
		t.Run(test.tool, func(t *testing.T) {
			project := t.TempDir()
			dir := t.TempDir()
			store, err := session.New(dir, project)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			cfg := core.Defaults()
			cfg.Model = "scripted"
			cfg.ContextWindow = 1 << 20
			cfg.SessionDir = dir
			a, err := agent.New(cfg, store)
			if err != nil {
				t.Fatal(err)
			}
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			configs := map[string]core.MCPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestMCPHelper$"}, Env: map[string]string{"HARNESS_MCP_HELPER": "1"}}}
			a.ExternalFactory = func() agent.External { return New(configs, project, a.Permission) }
			name := "mcp__fixture__" + test.tool
			a.Grants["mcp_start:fixture"] = true
			a.Grants[name] = true
			model := &scriptedModel{tool: name}
			a.Provider = model
			var events []core.Event
			a.Emit = func(e core.Event) { events = append(events, e) }
			status, runErr := a.Run(context.Background(), "Exercise the explicit MCP fixture")
			if status != test.status {
				t.Fatalf("status %s err %v", status, runErr)
			}
			if got := len(store.Pending()) > 0; got != test.pending {
				t.Fatalf("pending=%v wanted %v", got, test.pending)
			}
			terminal := 0
			for _, event := range events {
				if event.Type == "run_finished" {
					terminal++
				}
			}
			if terminal != 1 || events[len(events)-1].Type != "run_finished" {
				t.Fatalf("terminal events %d", terminal)
			}

			toolStarts, toolResults, serverStarts, serverResults := 0, 0, 0, 0
			for _, record := range store.Records() {
				var data struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				json.Unmarshal(record.Data, &data)
				if record.Type == "tool_start" {
					if data.ID == "fixture-call" {
						toolStarts++
					} else if data.Name == "mcp_start:fixture" {
						serverStarts++
					}
				}
				if record.Type == "tool_result" {
					if data.ID == "fixture-call" {
						toolResults++
					} else {
						serverResults++
					}
				}
			}
			if toolStarts != 1 || serverStarts != 1 || serverResults != 1 {
				t.Fatalf("tool starts %d server starts/results %d/%d", toolStarts, serverStarts, serverResults)
			}
			if test.pending && toolResults != 0 {
				t.Fatal("uncertain tool result was committed as complete")
			}
			if !test.pending && toolResults != 1 {
				t.Fatalf("tool result records %d", toolResults)
			}

			if test.pending {
				id := store.ID
				store.Close()
				reopened, err := session.Open(dir, id)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				resumed, err := agent.New(cfg, reopened)
				if err != nil {
					t.Fatal(err)
				}
				if len(resumed.Grants) != 0 {
					t.Fatal("MCP authority survived resumption")
				}
				if status, err := resumed.Run(context.Background(), "Continue"); status != "failed" || err == nil {
					t.Fatal("uncertain effects did not block continuation")
				}
			}
		})
	}
}

func TestAgentMCPStartupRecovery(t *testing.T) {
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "permission_unavailable", true: "approved_startup_failed"}[allow], func(t *testing.T) {
			project, dir := t.TempDir(), t.TempDir()
			store, err := session.New(dir, project)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			cfg := core.Defaults()
			cfg.Model = "fixture"
			cfg.ContextWindow = 1 << 20
			cfg.SessionDir = dir
			a, err := agent.New(cfg, store)
			if err != nil {
				t.Fatal(err)
			}
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			configs := map[string]core.MCPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestMCPHelper$"}, Env: map[string]string{"HARNESS_MCP_HELPER": "1", "HARNESS_MCP_BAD_SCHEMA": "1"}}}
			a.ExternalFactory = func() agent.External { return New(configs, project, a.Permission) }
			a.Grants["mcp_start:fixture"] = allow
			provider := &scriptedModel{tool: "mcp__fixture__echo"}
			a.Provider = provider
			status, err := a.Run(context.Background(), "Initialize explicit fixture server")
			if err == nil {
				t.Fatal("expected startup failure")
			}
			want := "needs_permission"
			if allow {
				want = "failed"
			}
			if status != want {
				t.Fatalf("status %s expected %s: %v", status, want, err)
			}
			if provider.turns != 0 {
				t.Fatal("provider called after failed startup")
			}
			if got := len(store.Pending()) > 0; got != allow {
				t.Fatalf("uncertain startup=%v, approved=%v", got, allow)
			}
		})
	}
}
