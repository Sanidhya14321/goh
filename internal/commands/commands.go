// Package commands expands explicitly configured slash-command text without
// executing processes, reading attachments, or changing runtime authority.
package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
)

const MaxTemplate = 64 << 10
const MaxPrompt = 1 << 20

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var placeholders = regexp.MustCompile(`\$ARGUMENTS|\$[0-9]+`)
var reserved = map[string]bool{"compact": true, "todos": true, "tasks": true, "commands": true, "skills": true}

// Expand returns the substituted prompt and selected command name. Non-command
// text and reserved runtime commands pass through with an empty command name.
// Arguments are data; their placeholder-looking text is not expanded again.
func Expand(prompt string, definitions map[string]core.CommandConfig) (string, string, error) {
	if len(prompt) > MaxPrompt {
		return "", "", fmt.Errorf("prompt exceeds %d-byte limit", MaxPrompt)
	}
	if !utf8.ValidString(prompt) {
		return "", "", fmt.Errorf("prompt must be valid UTF-8")
	}
	trimmed := strings.TrimSpace(prompt)
	if !strings.HasPrefix(trimmed, "/") {
		return prompt, "", nil
	}
	command, raw := splitCommand(trimmed[1:])
	if reserved[command] {
		return prompt, "", nil
	}
	if !namePattern.MatchString(command) {
		return "", "", fmt.Errorf("invalid slash command name %q", command)
	}
	definition, ok := definitions[command]
	if !ok {
		hint := ""
		if near := closestCommand(command, definitions); near != "" {
			hint = "did you mean /" + near + "? "
		}
		return "", command, fmt.Errorf("unknown slash command /%s; %s/commands lists the available ones, and .opencode/commands/%s.md would define it", command, hint, command)
	}
	if err := Validate(definitions); err != nil {
		return "", command, err
	}
	args, err := parseArguments(raw)
	if err != nil {
		return "", command, fmt.Errorf("/%s arguments: %w", command, err)
	}
	matches := placeholders.FindAllStringIndex(definition.Template, -1)
	highest := 0
	for _, span := range matches {
		token := definition.Template[span[0]:span[1]]
		if token == "$ARGUMENTS" {
			continue
		}
		number, err := strconv.Atoi(token[1:])
		if err != nil || number < 1 || number > 10000 {
			return "", command, fmt.Errorf("/%s positional placeholder must be $1..$10000", command)
		}
		if number > highest {
			highest = number
		}
	}
	var builder strings.Builder
	appendBounded := func(text string) error {
		if builder.Len()+len(text) > MaxPrompt {
			return fmt.Errorf("expanded command exceeds %d-byte limit", MaxPrompt)
		}
		builder.WriteString(text)
		return nil
	}
	offset := 0
	for _, span := range matches {
		if err := appendBounded(definition.Template[offset:span[0]]); err != nil {
			return "", command, err
		}
		token := definition.Template[span[0]:span[1]]
		value := ""
		if token == "$ARGUMENTS" {
			value = raw
		} else {
			number, _ := strconv.Atoi(token[1:])
			if number <= len(args) {
				if number == highest {
					value = strings.Join(args[number-1:], " ")
				} else {
					value = args[number-1]
				}
			}
		}
		if err := appendBounded(value); err != nil {
			return "", command, err
		}
		offset = span[1]
	}
	if err := appendBounded(definition.Template[offset:]); err != nil {
		return "", command, err
	}
	if len(matches) == 0 && strings.TrimSpace(raw) != "" {
		if err := appendBounded("\n\n" + raw); err != nil {
			return "", command, err
		}
	}
	return strings.TrimSpace(builder.String()), command, nil
}
func Validate(definitions map[string]core.CommandConfig) error {
	if len(definitions) > 256 {
		return fmt.Errorf("at most 256 custom commands are supported")
	}
	for name, definition := range definitions {
		if !namePattern.MatchString(name) || reserved[name] {
			return fmt.Errorf("command name %q is invalid or reserved", name)
		}
		if strings.TrimSpace(definition.Template) == "" || len(definition.Template) > MaxTemplate || !utf8.ValidString(definition.Template) {
			return fmt.Errorf("/%s template must be nonempty UTF-8 of at most %d bytes", name, MaxTemplate)
		}
		if len(definition.Description) > 8192 || !utf8.ValidString(definition.Description) {
			return fmt.Errorf("/%s description exceeds 8192 bytes or contains invalid UTF-8", name)
		}
	}
	return nil
}
func splitCommand(input string) (string, string) {
	for index, r := range input {
		if unicode.IsSpace(r) {
			return input[:index], strings.TrimSpace(input[index:])
		}
	}
	return input, ""
}

// parseArguments supports whitespace separation, single/double quote grouping,
// empty quoted arguments and backslash escaping. No shell, environment, glob,
// substitution, or file-reference interpretation takes place.
func parseArguments(raw string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped, started := false, false
	for _, r := range raw {
		if escaped {
			current.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
				if len(args) > 10000 {
					return nil, fmt.Errorf("at most 10000 arguments are supported")
				}
			}
			continue
		}
		current.WriteRune(r)
		started = true
	}
	if escaped {
		return nil, fmt.Errorf("trailing backslash requires an escaped character")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}
	if started {
		args = append(args, current.String())
	}
	if len(args) > 10000 {
		return nil, fmt.Errorf("at most 10000 arguments are supported")
	}
	return args, nil
}

// closestCommand returns the defined name nearest to typo (edit distance at
// most 2, ties broken alphabetically), or "".
func closestCommand(typo string, definitions map[string]core.CommandConfig) string {
	best, bestDistance := "", 3
	for name := range definitions {
		d := editDistance(typo, name)
		if d < bestDistance || (d == bestDistance && name < best) {
			best, bestDistance = name, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}
