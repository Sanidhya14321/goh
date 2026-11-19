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
