package acp

import (
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"regexp"
	"strings"

	"example.com/independent-harness/internal/core"
)

var serverName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

func mergeMCP(config map[string]core.MCPConfig, requested []json.RawMessage) (map[string]core.MCPConfig, error) {
	if len(requested) == 0 {
		return config, nil
	}
	if len(requested) > 16 {
		return nil, errors.New("At most 16 client MCP servers are supported")
	}
	merged := make(map[string]core.MCPConfig, len(config)+len(requested))
	maps.Copy(merged, config)
	for _, raw := range requested {
		var server struct {
			Type    string   `json:"type"`
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
			URL     string   `json:"url"`
			Env     []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"env"`
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		}
		if json.Unmarshal(raw, &server) != nil || !serverName.MatchString(server.Name) || strings.Contains(server.Name, "__") {
			return nil, errors.New("Client MCP servers require a valid name")
		}
		if _, exists := merged[server.Name]; exists {
			return nil, errors.New("Duplicate MCP server name")
		}
		if server.Type == "http" {
			// Remote servers from the client: the URL is checked again (https,
			// or http for localhost) when the server starts, after approval.
			if server.URL == "" || server.Command != "" || len(server.Headers) > 32 {
				return nil, errors.New("Client HTTP MCP servers require a url")
			}
			headers := make(map[string]string, len(server.Headers))
			for _, h := range server.Headers {
				if h.Name == "" || strings.ContainsAny(h.Name, " :\r\n") || strings.ContainsAny(h.Value, "\r\n") {
					return nil, errors.New("Invalid MCP header")
				}
				headers[h.Name] = h.Value
			}
			merged[server.Name] = core.MCPConfig{URL: server.URL, Headers: headers}
			continue
		}
		if (server.Type != "" && server.Type != "stdio") ||
			!filepath.IsAbs(server.Command) || strings.ContainsRune(server.Command, 0) ||
			len(server.Args) > 128 || len(server.Env) > 128 {
			return nil, errors.New("Client MCP servers require an absolute stdio command or an http url")
		}
		for _, arg := range server.Args {
			if strings.ContainsRune(arg, 0) {
				return nil, errors.New("Invalid MCP argument")
			}
		}
		env := make(map[string]string, len(server.Env))
		for _, item := range server.Env {
			if item.Name == "" || strings.ContainsAny(item.Name, "=\x00") || strings.ContainsRune(item.Value, 0) {
				return nil, errors.New("Invalid MCP environment variable")
			}
			if _, exists := env[item.Name]; exists {
				return nil, errors.New("Duplicate MCP environment variable")
			}
			env[item.Name] = item.Value
		}
		merged[server.Name] = core.MCPConfig{Command: server.Command, Args: server.Args, Env: env}
	}
	return merged, nil
}
