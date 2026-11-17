package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/mcpclient"
)

// A configured MCP server that a headless run was not allowed to start is
// left out with a notice; the run itself goes on with the built-in tools.
func TestHeadlessRunSkipsUnapprovedMCPServer(t *testing.T) {
	a, _ := testAgent(t)
	a.Approve = nil
	a.Config.MCP = map[string]core.MCPConfig{"ops": {Command: "/nonexistent/never-started"}}
	a.External = mcpclient.New(a.Config.MCP, a.Store.Project, a.Permission)
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
	})
	status, err := a.Run(context.Background(), "summarize the repo")
	if status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	found := false
	for _, r := range a.Store.Records() {
		if r.Type == "integration_degraded" && strings.Contains(string(r.Data), "--allow mcp_start:ops") {
			found = true
		}
		if r.Type == "permission" {
			t.Fatal("a skipped server must not be asked about", string(r.Data))
		}
	}
	if !found {
		t.Fatal("skip notice not recorded")
	}
}
