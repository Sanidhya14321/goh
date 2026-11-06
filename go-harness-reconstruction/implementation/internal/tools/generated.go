package tools

import (
	"bytes"
	"fmt"
)

const generatedHeaderLimit = 8 << 10

// protectGenerated uses the bytes already read for the edit's hash check.
func (e *Engine) protectGenerated(path string, data []byte) error {
	if !e.AllowGeneratedEdits && generatedHeader(data) {
		return fmt.Errorf("%q has a generated-file header. Edit the generator input and regenerate it", path)
	}
	return nil
}

func generatedHeader(data []byte) bool {
	truncated := len(data) > generatedHeaderLimit
	data = bytes.TrimPrefix(data[:min(len(data), generatedHeaderLimit)], []byte("\xef\xbb\xbf"))
	for len(data) > 0 {
		data = bytes.TrimLeft(data, " \t\r\n")
		var comment []byte
		switch {
		case bytes.HasPrefix(data, []byte("//")), bytes.HasPrefix(data, []byte("#")):
			line, rest, newline := bytes.Cut(data, []byte("\n"))
			if !newline && truncated {
				return false
			}
			comment = bytes.TrimLeft(line, "/#")
			data = rest
		case bytes.HasPrefix(data, []byte("/*")):
			body, rest, closed := bytes.Cut(data[2:], []byte("*/"))
			if !closed {
				return false
			}
			comment, data = body, rest
		default:
			return false
		}
		comment = bytes.TrimSpace(comment)
		if bytes.Equal(comment, []byte("@generated")) ||
			(bytes.HasPrefix(comment, []byte("Code generated ")) && bytes.HasSuffix(comment, []byte(" DO NOT EDIT."))) {
			return true
		}
	}
	return false
}
