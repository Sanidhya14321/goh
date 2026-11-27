package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var mcpName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidMCPName accepts the server names tools can be namespaced with
// (mcp__NAME__TOOL): letters, digits, underscore and hyphen, no "__".
func ValidMCPName(name string) error {
	if !mcpName.MatchString(name) || strings.Contains(name, "__") {
		return fmt.Errorf("MCP server name %q must be 1-64 letters, digits, underscores or hyphens, without a double underscore", name)
	}
	return nil
}

// SaveMCPServer adds or replaces the MCP server name in the user settings
// text raw, keeping every other setting.
func SaveMCPServer(raw, name string, server MCPConfig) (string, error) {
	if err := ValidMCPName(name); err != nil {
		return "", err
	}
	if (server.Command == "") == (server.URL == "") {
		return "", fmt.Errorf("an MCP server has either a command or a url")
	}
	return editDocument(raw, func(doc map[string]json.RawMessage, _ map[string]map[string]json.RawMessage) error {
		servers, err := mcpSection(doc)
		if err != nil {
			return err
		}
		servers[name], _ = json.Marshal(server)
		doc["mcp"], _ = json.Marshal(servers)
		return nil
	})
}

// RemoveMCPServer deletes the MCP server name from the user settings text.
func RemoveMCPServer(raw, name string) (string, error) {
	return editDocument(raw, func(doc map[string]json.RawMessage, _ map[string]map[string]json.RawMessage) error {
		servers, err := mcpSection(doc)
		if err != nil {
			return err
		}
		if _, ok := servers[name]; !ok {
			return fmt.Errorf("no MCP server named %q in the user settings (see goh mcp list)", name)
		}
		delete(servers, name)
		if len(servers) == 0 {
			delete(doc, "mcp")
		} else {
			doc["mcp"], _ = json.Marshal(servers)
		}
		return nil
	})
}

func mcpSection(doc map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	servers := map[string]json.RawMessage{}
	if b, ok := doc["mcp"]; ok {
		if err := json.Unmarshal(b, &servers); err != nil || servers == nil {
			return nil, fmt.Errorf("the settings' mcp section is not an object")
		}
	}
	return servers, nil
}
