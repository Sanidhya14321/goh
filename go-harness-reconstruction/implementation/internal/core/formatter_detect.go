package core

import (
	"os/exec"
	"strings"
)

// knownFormatters are widely used formatters GoHarness can add when
// formatter_auto is on. "$FILE" is replaced with the edited file.
var knownFormatters = []struct {
	name string
	cfg  FormatterConfig
}{
	{"gofmt", FormatterConfig{Command: "gofmt", Args: []string{"-w", "$FILE"}, Extensions: []string{".go"}}},
	{"rustfmt", FormatterConfig{Command: "rustfmt", Args: []string{"$FILE"}, Extensions: []string{".rs"}}},
	{"ruff", FormatterConfig{Command: "ruff", Args: []string{"format", "$FILE"}, Extensions: []string{".py", ".pyi"}}},
	{"prettier", FormatterConfig{Command: "prettier", Args: []string{"--write", "$FILE"}, Extensions: []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".css", ".scss", ".json"}}},
	{"clang-format", FormatterConfig{Command: "clang-format", Args: []string{"-i", "$FILE"}, Extensions: []string{".c", ".h", ".cc", ".cpp", ".hpp"}}},
	{"shfmt", FormatterConfig{Command: "shfmt", Args: []string{"-w", "$FILE"}, Extensions: []string{".sh", ".bash"}}},
}

// WithDetectedFormatters adds known formatters found on PATH when
// FormatterAuto is set. Configured formatters win by name and by extension
// (a disabled one still claims its extensions), and only executables already
// installed on PATH are used; nothing from the repository is run.
func (c Config) WithDetectedFormatters(lookPath func(string) (string, error)) Config {
	if !c.FormatterAuto {
		return c
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	taken := map[string]bool{}
	for _, cfg := range c.Formatters {
		for _, ext := range cfg.Extensions {
			taken[strings.ToLower(ext)] = true
		}
	}
	if _, ok := c.Formatters["gofmt"]; ok {
		taken[".go"] = true // gofmt defaults to .go when configured without extensions
	}
	for _, known := range knownFormatters {
		if _, exists := c.Formatters[known.name]; exists {
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
		next := make(map[string]FormatterConfig, len(c.Formatters)+1)
		for k, v := range c.Formatters {
			next[k] = v
		}
		cfg := known.cfg
		cfg.Args = append([]string(nil), cfg.Args...)
		cfg.Extensions = append([]string(nil), cfg.Extensions...)
		next[known.name] = cfg
		c.Formatters = next
		for _, ext := range cfg.Extensions {
			taken[ext] = true
		}
	}
	return c
}
