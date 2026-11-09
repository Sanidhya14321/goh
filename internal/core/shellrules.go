package core

import (
	"fmt"
	"strings"
)

// ShellRule decides a shell command by glob pattern, where * matches any
// text (including spaces). Action is allow, deny or ask. The last matching
// rule wins, as in OpenCode's permission patterns.
type ShellRule struct {
	Match  string `json:"match"`
	Action string `json:"action"`
}

// shellOperators separate or redirect commands. A command containing any of
// them is never auto-allowed, because a prefix like "git status" could be
// followed by anything ("git status; rm -rf ~").
var shellOperators = []string{";", "&", "|", "`", "$(", ">", "<", "\n", "\r"}

func (c Config) ValidateShellRules() error {
	if len(c.ShellRules) > 256 {
		return fmt.Errorf("at most 256 shell_rules are supported")
	}
	for i, r := range c.ShellRules {
		if strings.TrimSpace(r.Match) == "" || len(r.Match) > 512 {
			return fmt.Errorf("shell_rules[%d]: match must be 1 to 512 characters", i)
		}
		switch r.Action {
		case "allow", "deny", "ask":
		default:
			return fmt.Errorf("shell_rules[%d]: action must be allow, deny or ask", i)
		}
	}
	return nil
}

// ShellDecision returns "allow", "deny", "ask" or "" (no rule applies).
// Each string is decided by its last matching rule. The command is denied if
// the whole command or any segment between operators is denied, so a denied
// command cannot hide inside a compound one. Allow applies only to a simple
// command without operators; for a compound command no rule applies and the
// normal approval flow (including session grants) decides.
func (c Config) ShellDecision(command string) string {
	command = strings.TrimSpace(command)
	for _, s := range append([]string{command}, splitShellSegments(command)...) {
		if c.lastShellRule(s) == "deny" {
			return "deny"
		}
	}
	decision := c.lastShellRule(command)
	if decision == "allow" && hasShellOperator(command) {
		return "" // not auto-allowed; the normal approval flow decides
	}
	return decision
}

func (c Config) lastShellRule(s string) string {
	decision := ""
	for _, r := range c.ShellRules {
		if globMatch(r.Match, s) {
			decision = r.Action
		}
	}
	return decision
}

func hasShellOperator(s string) bool {
	for _, op := range shellOperators {
		if strings.Contains(s, op) {
			return true
		}
	}
	return false
}

func splitShellSegments(s string) []string {
	parts := []string{s}
	for _, op := range []string{"&&", "||", ";", "|", "&", "\n", "`", "$("} {
		var next []string
		for _, p := range parts {
			next = append(next, strings.Split(p, op)...)
		}
		parts = next
	}
	var out []string
	for _, p := range parts {
		p = strings.Trim(strings.TrimSpace(p), "()")
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// globMatch matches * as any run of characters; everything else literally.
func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for _, p := range parts[1 : len(parts)-1] {
		i := strings.Index(s, p)
		if i < 0 {
			return false
		}
		s = s[i+len(p):]
	}
	return strings.HasSuffix(s, parts[len(parts)-1])
}

// FileDecision applies file_rules to the project-relative paths one edit
// touches (forward slashes; * matches any text including /). It returns
// "deny" if any path is denied, "allow" only if every path is allowed, "ask"
// if any path must be asked about, and "" when no rule decides.
func (c Config) FileDecision(paths []string) string {
	if len(c.FileRules) == 0 || len(paths) == 0 {
		return ""
	}
	allowed, ask := 0, false
	for _, p := range paths {
		p = strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "./")
		decision := ""
		for _, r := range c.FileRules {
			if globMatch(r.Match, p) {
				decision = r.Action
			}
		}
		switch decision {
		case "deny":
			return "deny"
		case "allow":
			allowed++
		case "ask":
			ask = true
		}
	}
	if ask {
		return "ask"
	}
	if allowed == len(paths) {
		return "allow"
	}
	return ""
}
