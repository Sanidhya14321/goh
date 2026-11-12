package core

import (
	"fmt"
	"strconv"
	"strings"
)

// FileBlock frames an attached file's contents for a prompt. The header
// records the content length, so a reader finds the block's end by counting
// rather than by searching for a closing tag the file itself may contain.
func FileBlock(path, content string) string {
	return fmt.Sprintf("<file path=%s bytes=%d>\n%s\n</file>", strconv.Quote(path), len(content), content)
}

// StripFileBlocks replaces each well-formed FileBlock in text with a space,
// leaving what the user wrote. Text that only looks like a block (a wrong
// length, a missing closing tag) is kept, so it can hide nothing.
func StripFileBlocks(text string) string {
	const open = "<file path="
	var out strings.Builder
	for {
		i := strings.Index(text, open)
		if i < 0 {
			out.WriteString(text)
			return out.String()
		}
		end, ok := fileBlockEnd(text[i:])
		if !ok {
			out.WriteString(text[:i+len(open)])
			text = text[i+len(open):]
			continue
		}
		out.WriteString(text[:i])
		out.WriteByte(' ')
		text = text[i+end:]
	}
}

// fileBlockEnd returns the length of the FileBlock at the start of s.
func fileBlockEnd(s string) (int, bool) {
	rest := strings.TrimPrefix(s, "<file path=")
	q, err := strconv.QuotedPrefix(rest)
	if err != nil {
		return 0, false
	}
	rest = rest[len(q):]
	digits, ok := strings.CutPrefix(rest, " bytes=")
	if !ok {
		return 0, false
	}
	n := 0
	for n < len(digits) && n < 12 && digits[n] >= '0' && digits[n] <= '9' {
		n++
	}
	size, err := strconv.Atoi(digits[:n])
	if err != nil || !strings.HasPrefix(digits[n:], ">\n") {
		return 0, false
	}
	body := digits[n+2:]
	if size > len(body) || !strings.HasPrefix(body[size:], "\n</file>") {
		return 0, false
	}
	return len(s) - len(body) + size + len("\n</file>"), true
}
