package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

type promptExternal struct{ got map[string]string }

func (promptExternal) Definitions() []core.Tool               { return nil }
func (promptExternal) Has(string) bool                        { return false }
func (promptExternal) Validate(string, json.RawMessage) error { return nil }
func (promptExternal) Execute(context.Context, string, json.RawMessage) tools.Result {
	return tools.Result{}
}
func (promptExternal) Prompts(context.Context) ([]core.MCPPrompt, error) {
	return []core.MCPPrompt{{Server: "s", Name: "review", Arguments: []string{"file"}}}, nil
}
func (p *promptExternal) GetPrompt(_ context.Context, server, name string, args map[string]string) (string, error) {
	p.got = args
	return "Review " + args["file"], nil
}

func TestMCPPromptExpansion(t *testing.T) {
	a, _ := testAgent(t)
	ext := &promptExternal{}
	a.External = ext
	var sent string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		sent = h[len(h)-1].Content
		return core.Reply{Message: core.Message{Role: "assistant", Content: "ok"}, Finish: "stop"}, nil
	})
	if status, err := a.Run(context.Background(), "/mcp:s:review main.go"); status != "completed" {
		t.Fatal(status, err)
	}
	if ext.got["file"] != "main.go" || !strings.Contains(sent, "Review main.go") || !strings.Contains(sent, "grants no permissions") {
		t.Fatalf("args %v sent %q", ext.got, sent)
	}
	if _, err := a.Run(context.Background(), "/mcp:s:review file=x.go"); err != nil || ext.got["file"] != "x.go" {
		t.Fatal(err, ext.got)
	}
	if status, _ := a.Run(context.Background(), "/mcp:bad"); status != "failed" {
		t.Fatal("malformed prompt accepted")
	}
}

// lazyPromptExternal only serves prompts after Connect, like a real manager.
type lazyPromptExternal struct {
	promptExternal
	connected int
}

func (l *lazyPromptExternal) Connect(context.Context) error { l.connected++; return nil }
func (l *lazyPromptExternal) GetPrompt(ctx context.Context, server, name string, args map[string]string) (string, error) {
	if l.connected == 0 {
		return "", errNotConnected
	}
	return l.promptExternal.GetPrompt(ctx, server, name, args)
}

var errNotConnected = errorString("no connected MCP server")

type errorString string

func (e errorString) Error() string { return string(e) }

func TestMCPPromptConnectsBeforeExpanding(t *testing.T) {
	a, _ := testAgent(t)
	ext := &lazyPromptExternal{}
	a.ExternalFactory = func() External { return ext }
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "ok"}, Finish: "stop"}, nil
	})
	if status, err := a.Run(context.Background(), "/mcp:s:review main.go"); status != "completed" {
		t.Fatal(status, err)
	}
	if ext.connected != 1 {
		t.Fatalf("connected %d times, want once per run", ext.connected)
	}
}

// closingPromptExternal fails once closed, like a real manager after a run.
type closingPromptExternal struct {
	lazyPromptExternal
	closed bool
}

func (c *closingPromptExternal) Close() error { c.closed = true; return nil }
func (c *closingPromptExternal) Prompts(ctx context.Context) ([]core.MCPPrompt, error) {
	if c.closed || c.connected == 0 {
		return nil, errorString("connection closed")
	}
	return c.promptExternal.Prompts(ctx)
}

// /mcp lists prompts between runs, before the first run and after one,
// with a fresh manager that is stopped afterwards.
func TestListMCPPromptsBetweenRuns(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MCP = map[string]core.MCPConfig{"s": {Command: "unused"}}
	var made []*closingPromptExternal
	a.ExternalFactory = func() External {
		e := &closingPromptExternal{}
		made = append(made, e)
		return e
	}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "ok"}, Finish: "stop"}, nil
	})
	for _, when := range []string{"before a run", "after a run"} {
		list, err := a.ListMCPPrompts(context.Background())
		if err != nil || len(list) != 1 || list[0].Name != "review" {
			t.Fatalf("%s: %v %v", when, list, err)
		}
		if last := made[len(made)-1]; !last.closed {
			t.Fatalf("%s: the listing manager was left running", when)
		}
		if when == "before a run" {
			if status, err := a.Run(context.Background(), "hello"); status != "completed" {
				t.Fatal(status, err)
			}
			if a.External != nil {
				t.Fatal("the run's closed manager was kept")
			}
		}
	}
}
