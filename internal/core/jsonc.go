package core

import "errors"

// StripJSONC converts JSON with comments (// and /* */) and trailing commas,
// as accepted by OpenCode's config, into plain JSON. Comments become spaces
// (newlines kept) so decoder error positions still match the user's file.
// Text inside strings is never changed.
func StripJSONC(src string) (string, error) {
	b := []byte(src)
	out := make([]byte, 0, len(b))
	inString, escaped := false, false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				out = append(out, ' ')
				i++
			}
			if i < len(b) {
				out = append(out, '\n')
			}
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			end := -1
			for j := i + 2; j+1 < len(b); j++ {
				if b[j] == '*' && b[j+1] == '/' {
					end = j + 1
					break
				}
			}
			if end < 0 {
				return "", errors.New("configuration has an unterminated /* comment")
			}
			for ; i <= end; i++ {
				if b[i] == '\n' {
					out = append(out, '\n')
				} else {
					out = append(out, ' ')
				}
			}
			i--
		default:
			out = append(out, c)
		}
	}
	// Trailing commas: a comma followed only by whitespace before } or ].
	result := make([]byte, 0, len(out))
	inString, escaped = false, false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			result = append(result, c)
			continue
		}
		if c == '"' {
			inString = true
		}
		if c == ',' {
			j := i + 1
			for j < len(out) && (out[j] == ' ' || out[j] == '\t' || out[j] == '\n' || out[j] == '\r') {
				j++
			}
			if j < len(out) && (out[j] == '}' || out[j] == ']') {
				result = append(result, ' ')
				continue
			}
		}
		result = append(result, c)
	}
	return string(result), nil
}

// HasJSONCSyntax reports whether text uses comments or trailing commas, which
// a JSON round trip would erase.
func HasJSONCSyntax(src string) bool {
	plain, err := StripJSONC(src)
	return err != nil || plain != src
}
