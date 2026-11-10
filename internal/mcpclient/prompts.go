package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptInfo is the listing entry for one server prompt; the model never
// fetches prompts by itself.
type PromptInfo = core.MCPPrompt

const maxPromptText = 64 << 10

// Prompts lists the prompts of every connected server that offers them.
func (m *Manager) Prompts(ctx context.Context) ([]PromptInfo, error) {
	m.mu.Lock()
	servers := make(map[string]*mcp.ClientSession, len(m.promptServers))
	for k, v := range m.promptServers {
		servers[k] = v
	}
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	var out []PromptInfo
	for server, s := range servers {
		for p, err := range s.Prompts(ctx, nil) {
			if err != nil {
				return out, fmt.Errorf("listing prompts from %s: %w", server, err)
			}
			info := PromptInfo{Server: server, Name: p.Name, Description: p.Description}
			for _, a := range p.Arguments {
				if a.Required {
					info.Arguments = append([]string{a.Name}, info.Arguments...)
				} else {
					info.Arguments = append(info.Arguments, a.Name+"?")
				}
			}
			if len(out) == 500 {
				return out, nil
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Server+":"+out[i].Name < out[j].Server+":"+out[j].Name
	})
	return out, nil
}

// GetPrompt renders a prompt to text. Only text content is used; the result
// is server-provided and carries no permission grants.
func (m *Manager) GetPrompt(ctx context.Context, server, name string, args map[string]string) (string, error) {
	m.mu.Lock()
	s := m.promptServers[server]
	m.mu.Unlock()
	if s == nil {
		return "", fmt.Errorf("no connected MCP server named %q offers prompts", server)
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := s.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: args})
	if err != nil {
		return "", fmt.Errorf("MCP prompt %s:%s: %w", server, name, err)
	}
	var b strings.Builder
	for _, msg := range res.Messages {
		if t, ok := msg.Content.(*mcp.TextContent); ok && t.Text != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(t.Text)
		}
	}
	if b.Len() == 0 {
		return "", errors.New("MCP prompt returned no text")
	}
	if b.Len() > maxPromptText {
		return "", errors.New("MCP prompt text exceeds 64 KiB")
	}
	return b.String(), nil
}
