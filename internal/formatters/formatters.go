// Package formatters resolves explicitly configured local formatters. It never
// installs tools, scans user configuration, or evaluates shell substitutions.
package formatters

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

type Entry struct {
	Name       string            `json:"name"`
	Extensions []string          `json:"extensions"`
	Enabled    bool              `json:"enabled"`
	Available  bool              `json:"available"`
	Command    string            `json:"command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Timeout    int               `json:"timeout_seconds"`
	Reason     string            `json:"reason,omitempty"`
}

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var validEnv = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func Validate(config map[string]core.FormatterConfig) error {
	if len(config) > 32 {
		return fmt.Errorf("at most 32 formatters are supported")
	}
	for name, c := range config {
		if !validName.MatchString(name) {
			return fmt.Errorf("invalid formatter name %q", name)
		}
		if c.Disabled {
			continue
		}
		if (c.Command == "" && name != "gofmt") || len(c.Command) > 4096 || strings.ContainsRune(c.Command, 0) {
			return fmt.Errorf("formatter %s needs a valid command", name)
		}
		if len(c.Args) > 64 || len(c.Env) > 64 || c.Timeout < 0 || c.Timeout > 120 {
			return fmt.Errorf("formatter %s exceeds argument/environment/timeout limits", name)
		}
		if len(c.Extensions) > 32 || (len(c.Extensions) == 0 && name != "gofmt") {
			return fmt.Errorf("formatter %s needs extensions", name)
		}
		for _, ext := range c.Extensions {
			if len(ext) > 64 || !strings.HasPrefix(ext, ".") || strings.ContainsAny(ext, "/\\\x00") {
				return fmt.Errorf("invalid formatter extension %q", ext)
			}
		}
		total := len(c.Command)
		for _, arg := range c.Args {
			total += len(arg)
			if len(arg) > 8192 || strings.ContainsRune(arg, 0) {
				return fmt.Errorf("invalid formatter argument")
			}
		}
		for k, v := range c.Env {
			total += len(k) + len(v)
			if !validEnv.MatchString(k) || len(v) > 8192 || strings.ContainsRune(v, 0) {
				return fmt.Errorf("invalid formatter environment")
			}
		}
		if total > 64<<10 {
			return fmt.Errorf("formatter %s command/environment exceeds 64KiB", name)
		}
	}
	return nil
}
func Discover(config map[string]core.FormatterConfig, project string) ([]Entry, error) {
	if err := Validate(config); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(config))
	for name := range config {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		c := config[name]
		e := Entry{Name: name, Extensions: append([]string{}, c.Extensions...), Enabled: !c.Disabled, Timeout: c.Timeout, Env: c.Env, Command: c.Command, Args: append([]string{}, c.Args...)}
		if name == "gofmt" {
			if len(e.Extensions) == 0 {
				e.Extensions = []string{".go"}
			}
			if e.Command == "" {
				e.Command = "gofmt"
				if len(e.Args) == 0 {
					e.Args = []string{"-w", "$FILE"}
				}
			}
		}
		if e.Timeout == 0 {
			e.Timeout = 30
		}
		if e.Enabled {
			if strings.ContainsAny(e.Command, "/\\") && !filepath.IsAbs(e.Command) {
				e.Command = filepath.Join(project, e.Command)
			}
			resolved, err := exec.LookPath(e.Command)
			if err != nil {
				e.Reason = err.Error()
			} else if !filepath.IsAbs(resolved) {
				e.Reason = "formatter executable must resolve to an absolute path"
			} else {
				e.Available = true
				e.Command = resolved
			}
		}
		out = append(out, e)
	}
	return out, nil
}
func Matching(entries []Entry, path string) []Entry {
	out := []Entry{}
	ext := filepath.Ext(path)
	for _, e := range entries {
		if !e.Enabled {
			continue
		}
		for _, x := range e.Extensions {
			if ext == x {
				out = append(out, e)
				break
			}
		}
	}
	return out
}
func (e Entry) Arguments(absoluteFile string) []string {
	args := append([]string{}, e.Args...)
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "$FILE", absoluteFile)
	}
	return args
}
