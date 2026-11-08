package core

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Project folders holding NAME.md agent definitions, as in OpenCode
// (.opencode/agent) and Claude Code (.claude/agents).
var agentDirs = []string{".opencode/agent", ".opencode/agents", ".claude/agents", ".harness/agents"}

// SkippedAgentFile explains an agent file that was not loaded.
type SkippedAgentFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// WithProjectAgents adds agents defined in project Markdown files. Configured
// agents win. Frontmatter supplies description and mode ("plan" for read-only,
// anything else is build); the body becomes the instructions. A file cannot
// choose a model profile, so repository content never selects where prompts
// are sent. Only regular top-level *.md files are read, each at most 64 KiB.
func (c Config) WithProjectAgents(project string) (Config, []SkippedAgentFile) {
	var skipped []SkippedAgentFile
	found := map[string]AgentProfile{}
	for _, sub := range agentDirs {
		dir := filepath.Join(project, sub)
		if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			skipped = append(skipped, SkippedAgentFile{dir, err.Error()})
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), ".md")
			if !ok || !e.Type().IsRegular() {
				continue
			}
			path := filepath.Join(dir, e.Name())
			switch {
			case !profileName.MatchString(name) || name == "build" || name == "plan" || name == "general" || name == "explore":
				skipped = append(skipped, SkippedAgentFile{path, "invalid or reserved agent name"})
				continue
			case found[name] != (AgentProfile{}):
				skipped = append(skipped, SkippedAgentFile{path, "duplicate of an earlier agent folder"})
				continue
			}
			profile, reason := readAgentFile(path)
			if reason != "" {
				skipped = append(skipped, SkippedAgentFile{path, reason})
				continue
			}
			found[name] = profile
		}
	}
	if len(found) > 0 {
		merged := make(map[string]AgentProfile, len(c.Agents)+len(found))
		for k, v := range found {
			merged[k] = v
		}
		for k, v := range c.Agents {
			merged[k] = v // configured agents win
		}
		c.Agents = merged
	}
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Path < skipped[j].Path })
	return c, skipped
}

func readAgentFile(path string) (AgentProfile, string) {
	f, err := os.Open(path)
	if err != nil {
		return AgentProfile{}, err.Error()
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if err != nil {
		return AgentProfile{}, err.Error()
	}
	if len(b) > 64<<10 || !utf8.Valid(b) {
		return AgentProfile{}, "larger than 64 KiB or not UTF-8"
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	p := AgentProfile{Mode: "build"}
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		front, body, closed := strings.Cut(rest, "\n---")
		if !closed {
			return p, "unterminated frontmatter"
		}
		for _, line := range strings.Split(front, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			switch strings.TrimSpace(key) {
			case "description":
				p.Description = value
			case "mode":
				if value == "plan" {
					p.Mode = "plan"
				}
			}
		}
		text = strings.TrimLeft(body, "\n")
	}
	p.Instructions = strings.TrimSpace(text)
	if p.Instructions == "" {
		return p, "no instructions"
	}
	if len(p.Description) > 8192 {
		return p, "description too long"
	}
	return p, ""
}
