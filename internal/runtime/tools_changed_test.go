package runtime

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// growingMCP offers mcp__s__one; calling it adds mcp__s__two.
type growingMCP struct{ grown atomic.Bool }

func (g *growingMCP) Definitions() []core.Tool {
	empty := map[string]any{"type": "object"}
	defs := []core.Tool{core.NewTool("mcp__s__one", "one", empty)}
	if g.grown.Load() {
		defs = append(defs, core.NewTool("mcp__s__two", "two", empty))
	}
	return defs
}
func (g *growingMCP) Has(n string) bool {
	return n == "mcp__s__one" || n == "mcp__s__two" && g.grown.Load()
}
func (g *growingMCP) Validate(string, json.RawMessage) error { return nil }
func (g *growingMCP) Execute(context.Context, string, json.RawMessage) tools.Result {
	g.grown.Store(true)
	return tools.Result{Content: "done"}
}
func (g *growingMCP) Generation() uint64 {
	if g.grown.Load() {
		return 1
	}
	return 0
}

func TestToolListRebuiltWhenServerChangesIt(t *testing.T) {
	a, _ := testAgent(t)
	a.Grants["mcp__s__one"] = true
	g := &growingMCP{}
	a.ExternalFactory = func() External { return g }
	var offered [][]string
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, _ []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		var names []string
		for _, d := range defs {
			if len(d.Function.Name) > 5 && d.Function.Name[:5] == "mcp__" {
				names = append(names, d.Function.Name)
			}
		}
		offered = append(offered, names)
		turn++
		if turn == 1 {
			call := core.Call{ID: "c1", Type: "function"}
			call.Function.Name, call.Function.Arguments = "mcp__s__one", `{}`
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, Finish: "tool_calls"}, nil
		}
		return core.Reply{Message: core.Message{Role: "assistant", Content: "finished"}, Finish: "stop"}, nil
	})
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatal(err)
	}
	if len(offered) < 2 || len(offered[0]) != 1 || len(offered[1]) != 2 {
		t.Fatalf("tool lists per request: %v", offered)
	}
}
