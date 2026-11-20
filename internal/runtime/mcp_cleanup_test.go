package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// uncertainMCP approves a server start, then reports it failed with a
// cleanup that could not be confirmed.
type uncertainMCP struct{ a *Agent }

func (u *uncertainMCP) Definitions() []core.Tool               { return nil }
func (u *uncertainMCP) Has(string) bool                        { return false }
func (u *uncertainMCP) Validate(string, json.RawMessage) error { return nil }
func (u *uncertainMCP) Execute(context.Context, string, json.RawMessage) tools.Result {
	return tools.Result{}
}
func (u *uncertainMCP) Connect(ctx context.Context) error {
	_, err := u.a.Permission(ctx, core.Permission{Tool: "mcp_start:flaky", Arguments: json.RawMessage(`{}`)})
	return err
}
func (u *uncertainMCP) Failures() []core.IntegrationFailure {
	return []core.IntegrationFailure{{Name: "flaky", Error: "initialization failed", CleanupIncomplete: true}}
}
func (u *uncertainMCP) Close() error {
	return errors.New("MCP cleanup_incomplete: process did not settle")
}

// Review P1: a failed MCP start whose cleanup is uncertain is never
// journaled as failed_and_stopped; it stays pending and the run says so.
func TestMCPStartWithUncertainCleanupStaysPending(t *testing.T) {
	a, _ := testAgent(t)
	a.Grants["mcp_start:flaky"] = true
	a.ExternalFactory = func() External { return &uncertainMCP{a: a} }
	a.Provider = scripted()
	status, err := a.Run(context.Background(), "hello")
	if status != "cleanup_incomplete" || err == nil || !strings.Contains(err.Error(), "could not be confirmed stopped") {
		t.Fatal(status, err)
	}
	for _, r := range a.Store.Records() {
		if r.Type == "tool_result" && strings.Contains(string(r.Data), "failed_and_stopped") {
			t.Fatal("uncertain cleanup journaled as stopped")
		}
	}
	if len(a.Store.Pending()) == 0 {
		t.Fatal("the start must stay an uncertain effect")
	}
}

// restartUncertain reports a restart whose cleanup could not be confirmed.
type restartUncertain struct{}

func (restartUncertain) Definitions() []core.Tool {
	return []core.Tool{core.NewTool("mcp__s__t", "tool", map[string]any{"type": "object"})}
}
func (restartUncertain) Has(n string) bool                      { return n == "mcp__s__t" }
func (restartUncertain) Validate(string, json.RawMessage) error { return nil }
func (restartUncertain) Execute(context.Context, string, json.RawMessage) tools.Result {
	return tools.Result{Error: true, Content: `{"status":"cleanup_incomplete","error":"MCP server s ` + core.MCPRestartFailed + `: cleanup_incomplete"}`}
}

// Follow-up review: a reconnect whose cleanup is uncertain ends the run as
// cleanup_incomplete with the call left pending, never failed_and_stopped.
func TestMCPRestartWithUncertainCleanupStops(t *testing.T) {
	a, _ := testAgent(t)
	a.Grants["mcp__s__t"] = true
	a.ExternalFactory = func() External { return restartUncertain{} }
	a.Provider = scripted([]core.Call{toolCall("m1", "mcp__s__t", map[string]any{})})
	status, _ := a.Run(context.Background(), "use it")
	if status != "cleanup_incomplete" || len(a.Store.Pending()) == 0 {
		t.Fatalf("status %s pending %v", status, a.Store.Pending())
	}
	for _, r := range a.Store.Records() {
		if strings.Contains(string(r.Data), "failed_and_stopped") {
			t.Fatal("uncertain restart journaled as stopped")
		}
	}
}
