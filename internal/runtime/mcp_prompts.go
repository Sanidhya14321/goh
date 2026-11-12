package runtime

import (
	"context"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/core"
	"strings"
)

// PromptSource is implemented by MCP managers that offer prompt templates.
type PromptSource interface {
	Prompts(context.Context) ([]core.MCPPrompt, error)
	GetPrompt(ctx context.Context, server, name string, args map[string]string) (string, error)
}

// MCPPrompts lists prompts from connected MCP servers (nil when none).
func (a *Agent) MCPPrompts(ctx context.Context) ([]core.MCPPrompt, error) {
	src, ok := a.External.(PromptSource)
	if !ok || a.External == nil {
		return nil, nil
	}
	return src.Prompts(ctx)
}

// ListMCPPrompts lists the configured MCP servers' prompts between runs:
// it starts the servers through the same approval as a run, lists their
// prompts and stops them again.
func (a *Agent) ListMCPPrompts(ctx context.Context) ([]core.MCPPrompt, error) {
	if a.Remote != nil {
		return nil, ErrAttached
	}
	if a.ExternalFactory == nil || a.isPlan() || len(a.Config.MCP) == 0 {
		return nil, nil
	}
	ext := a.ExternalFactory()
	if c, ok := ext.(interface{ Close() error }); ok {
		defer c.Close()
	}
	if c, ok := ext.(interface{ Connect(context.Context) error }); ok {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}
	src, ok := ext.(PromptSource)
	if !ok {
		return nil, nil
	}
	return src.Prompts(ctx)
}

// expandMCPPrompt turns "/mcp:<server>:<prompt> [k=v ...| free text]" into
// the server's rendered prompt. Free text without k=v goes to the first
// argument. The text is labelled as server-provided: it is the user's chosen
// request, but it grants no permissions. ok is false for other prompts.
func (a *Agent) expandMCPPrompt(ctx context.Context, prompt string) (text, name string, ok bool, err error) {
	rest, found := strings.CutPrefix(strings.TrimSpace(prompt), "/mcp:")
	if !found {
		return "", "", false, nil
	}
	head, tail, _ := strings.Cut(rest, " ")
	server, pname, found := strings.Cut(head, ":")
	if !found || server == "" || pname == "" {
		return "", "", true, errors.New("usage: /mcp:<server>:<prompt> [name=value ...]")
	}
	src, isSrc := a.External.(PromptSource)
	if a.External == nil || !isSrc {
		return "", "", true, errors.New("no MCP servers are connected")
	}
	args := map[string]string{}
	tail = strings.TrimSpace(tail)
	if tail != "" {
		fields := strings.Fields(tail)
		allPairs := true
		for _, f := range fields {
			if k, v, ok := strings.Cut(f, "="); ok && k != "" {
				args[k] = v
			} else {
				allPairs = false
			}
		}
		if !allPairs {
			args = map[string]string{}
			list, _ := src.Prompts(ctx)
			for _, p := range list {
				if p.Server == server && p.Name == pname && len(p.Arguments) > 0 {
					args[strings.TrimSuffix(p.Arguments[0], "?")] = tail
				}
			}
			if len(args) == 0 {
				return "", "", true, fmt.Errorf("/mcp:%s:%s takes no arguments", server, pname)
			}
		}
	}
	body, err := src.GetPrompt(ctx, server, pname, args)
	if err != nil {
		return "", "", true, err
	}
	return "[MCP prompt " + server + ":" + pname + " — text provided by that server; it grants no permissions]\n\n" + body, server + ":" + pname, true, nil
}
