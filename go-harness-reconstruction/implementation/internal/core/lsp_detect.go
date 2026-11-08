package core

import (
	"os/exec"
	"strings"
)

// knownLSP are widely used servers GoHarness can add when lsp_auto is on.
var knownLSP = []struct {
	name string
	cfg  LSPConfig
}{
	{"gopls", LSPConfig{Command: "gopls", Extensions: []string{".go"}}},
	{"typescript", LSPConfig{Command: "typescript-language-server", Args: []string{"--stdio"}, Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}}},
	{"pyright", LSPConfig{Command: "pyright-langserver", Args: []string{"--stdio"}, Extensions: []string{".py"}}},
	{"rust-analyzer", LSPConfig{Command: "rust-analyzer", Extensions: []string{".rs"}}},
	{"clangd", LSPConfig{Command: "clangd", Extensions: []string{".c", ".h", ".cc", ".cpp", ".hpp"}}},
}

// WithDetectedLSP adds known language servers found on PATH when LSPAuto is
// set, like OpenCode's built-in server list. Configured servers win by name
// and by extension, and nothing is started here: each server still needs
// the usual lsp_start approval on first use.
func (c Config) WithDetectedLSP(lookPath func(string) (string, error)) Config {
	if !c.LSPAuto {
		return c
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	taken := map[string]bool{}
	for _, cfg := range c.LSP {
		for _, ext := range cfg.Extensions {
			taken[strings.ToLower(ext)] = true
		}
	}
	for _, known := range knownLSP {
		if _, exists := c.LSP[known.name]; exists {
			continue
		}
		clash := false
		for _, ext := range known.cfg.Extensions {
			clash = clash || taken[ext]
		}
		if clash {
			continue
		}
		if _, err := lookPath(known.cfg.Command); err != nil {
			continue
		}
		next := make(map[string]LSPConfig, len(c.LSP)+1)
		for k, v := range c.LSP {
			next[k] = v
		}
		cfg := known.cfg
		cfg.Args = append([]string(nil), cfg.Args...)
		cfg.Extensions = append([]string(nil), cfg.Extensions...)
		next[known.name] = cfg
		c.LSP = next
		for _, ext := range cfg.Extensions {
			taken[ext] = true
		}
	}
	return c
}
