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
