package skills

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// This deliberately small frontmatter subset handles string scalars and block
// descriptions, without object tags, aliases, interpolation or executable YAML.
func parse(raw []byte) (name, description, body string, err error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", "", errors.New("skill requires --- frontmatter with a string name")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", "", "", errors.New("unterminated skill frontmatter")
	}
	seen := map[string]bool{}
	for i := 1; i < end; i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return "", "", "", errors.New("unsupported frontmatter; expected key: value")
		}
		key = strings.TrimSpace(key)
		if key != "name" && key != "description" {
			continue
		}
		if seen[key] {
			return "", "", "", fmt.Errorf("duplicate frontmatter %s", key)
		}
		seen[key] = true
		value = strings.TrimSpace(value)
		var parsed string
		if value == "|" || value == ">" || value == "|-" || value == ">-" {
			var parts []string
			for i+1 < end {
				next := lines[i+1]
				if strings.TrimSpace(next) != "" && !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
					break
				}
				i++
				parts = append(parts, strings.TrimSpace(next))
			}
			sep := "\n"
			if strings.HasPrefix(value, ">") {
				sep = " "
			}
			parsed = strings.Join(parts, sep)
		} else {
			parsed, err = scalar(value)
			if err != nil {
				return "", "", "", fmt.Errorf("%s: %w", key, err)
			}
		}
		if key == "name" {
			name = parsed
		} else {
			description = parsed
		}
	}
	if !seen["name"] || name == "" || len(name) > 256 || len(description) > 4096 {
		return "", "", "", errors.New("skill requires a nonempty name <=256 bytes and description <=4096 bytes")
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == '/' || r == '\\' {
			return "", "", "", errors.New("skill name cannot contain controls or path separators")
		}
	}
	return name, description, strings.TrimSpace(strings.Join(lines[end+1:], "\n")), nil
}
func scalar(value string) (string, error) {
	if strings.HasPrefix(value, "\"") {
		v, e := strconv.Unquote(value)
		if e != nil {
			return "", errors.New("invalid quoted string")
		}
		return v, nil
	}
	if strings.HasPrefix(value, "'") {
		if len(value) < 2 || value[len(value)-1] != '\'' {
			return "", errors.New("unterminated single-quoted string")
		}
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), nil
	}
	if comment := strings.Index(value, " #"); comment >= 0 {
		value = value[:comment]
	}
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "null", "~", "true", "false", "yes", "no":
		return "", errors.New("expected a string scalar; quote empty, boolean or null-looking values")
	}
	if _, e := strconv.ParseFloat(value, 64); e == nil {
		return "", errors.New("expected a string, not a number")
	}
	if strings.ContainsAny(value, "[]{}") || strings.HasPrefix(value, "&") || strings.HasPrefix(value, "*") || strings.HasPrefix(value, "!") {
		return "", errors.New("unsupported YAML scalar; quote the text")
	}
	return value, nil
}
