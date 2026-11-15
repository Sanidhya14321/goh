package core

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
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
	for _, s := range denyCandidates(command, 0) {
		if c.lastShellRule(s) == "deny" {
			return "deny"
		}
	}
	decision := c.lastShellRule(command)
	if decision == "allow" && HasShellOperator(command) {
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

// HasShellOperator reports whether s separates or redirects commands, which
// makes it compound: such a command is never auto-allowed by a rule or
// covered by a session approval of another command.
func HasShellOperator(s string) bool {
	for _, op := range shellOperators {
		if strings.Contains(s, op) {
			return true
		}
	}
	return false
}

// commandWrappers run the command that follows them.
var commandWrappers = map[string]bool{"command": true, "builtin": true, "exec": true, "nohup": true, "time": true,
	"nice": true, "sudo": true, "doas": true, "env": true, "xargs": true, "timeout": true}

// shells run the script given with -c.
var shells = map[string]bool{"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "fish": true}

// denyCandidates lists the texts a deny rule is checked against: the whole
// command, each segment between operators, and each segment rewritten the
// way it actually runs, so trivial re-spellings do not slip past a rule
// such as "rm *": "/bin/rm -rf x", "command rm -rf x", "FOO=1 rm -rf x" and
// "sh -c 'rm -rf x'" all also check as "rm -rf x". Deny rules remain a
// guardrail on text, not a sandbox.
func denyCandidates(command string, depth int) []string {
	out := []string{command}
	for _, seg := range splitShellSegments(command) {
		out = append(out, seg)
		words := strings.Fields(seg)
		for len(words) > 0 && (commandWrappers[words[0]] || isAssignment(words[0])) {
			words = words[1:]
		}
		if len(words) == 0 {
			continue
		}
		words[0] = baseName(words[0])
		canonical := strings.Join(words, " ")
		out = append(out, canonical)
		if shells[words[0]] && depth < 2 {
			for i := 1; i+1 < len(words); i++ {
				if words[i] == "-c" {
					script := strings.Trim(strings.Join(words[i+1:], " "), `'"`)
					out = append(out, denyCandidates(script, depth+1)...)
					break
				}
			}
		}
	}
	return out
}

// isAssignment reports a VAR=value prefix such as FOO=1 in "FOO=1 rm x".
func isAssignment(w string) bool {
	name, _, ok := strings.Cut(w, "=")
	if !ok || name == "" {
		return false
	}
	for i, r := range name {
		if r != '_' && !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

func baseName(w string) string {
	if i := strings.LastIndexAny(w, `/\`); i >= 0 && i+1 < len(w) {
		return w[i+1:]
	}
	return w
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
	return c.FileDecisionFold(paths, false)
}

// FileDecisionFold is FileDecision for a filesystem that folds names: with
// fold set, rules and paths compare with case folded and Unicode
// normalized, as the filesystem compares them.
func (c Config) FileDecisionFold(paths []string, fold bool) string {
	if len(c.FileRules) == 0 || len(paths) == 0 {
		return ""
	}
	key := func(s string) string { return s }
	if fold {
		key = func(s string) string { return strings.ToLower(norm.NFC.String(s)) }
	}
	allowed, ask := 0, false
	for _, p := range paths {
		p = key(strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "./"))
		decision := ""
		for _, r := range c.FileRules {
			if globMatch(key(r.Match), p) {
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
