package tools

import (
	"fmt"
	"strings"
)

// flexibleReplace is the fallback when old_text is not found exactly. It
// compares line by line ignoring leading and trailing whitespace, and applies
// the edit only when exactly one place matches; new text is re-indented by
// the difference between the file's and old_text's indentation. Research on
// edit formats finds that tolerant matching rescues most failed edits but
// misapplies some unless ambiguous matches are refused (arXiv:2609.11957), so
// zero or several matches return an explanatory error instead.
func flexibleReplace(content, old, replacement string) (string, bool, string) {
	oldLines := splitLines(strings.TrimRight(old, "\n"))
	if len(oldLines) == 0 || strings.TrimSpace(old) == "" {
		return "", false, "old_text not found"
	}
	lines := strings.SplitAfter(content, "\n")
	var starts []int
	for i := 0; i+len(oldLines) <= len(lines); i++ {
		ok := true
		for j, ol := range oldLines {
			if strings.TrimSpace(lines[i+j]) != strings.TrimSpace(ol) {
				ok = false
				break
			}
		}
		if ok {
			starts = append(starts, i)
		}
	}
	switch len(starts) {
	case 0:
		return "", false, "old_text not found (exactly or ignoring whitespace)." + nearLines(content, oldLines) + " Re-read the file and copy the exact text."
	case 1:
	default:
		lineNos := make([]string, len(starts))
		for k, st := range starts {
			lineNos[k] = fmt.Sprint(st + 1)
		}
		return "", false, "old_text matches " + fmt.Sprint(len(starts)) + " places when whitespace is ignored (lines " + strings.Join(lineNos, ", ") + "); include more surrounding lines so it is unique"
	}
	start := starts[0]
	fileIndent, oldIndent := firstIndent(lines[start:start+len(oldLines)]), firstIndent(oldLines)
	var b strings.Builder
	for _, l := range lines[:start] {
		b.WriteString(l)
	}
	newText := strings.TrimRight(replacement, "\n")
	if newText != "" || replacement != "" {
		for _, nl := range splitLines(newText) {
			if strings.TrimSpace(nl) != "" {
				nl = fileIndent + strings.TrimPrefix(nl, oldIndent)
			}
			b.WriteString(nl + "\n")
		}
	}
	rest := lines[start+len(oldLines):]
	// Keep the file's final-newline state when the match ended the file.
	if len(rest) == 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
		out := b.String()
		return strings.TrimSuffix(out, "\n"), true, ""
	}
	for _, l := range rest {
		b.WriteString(l)
	}
	return b.String(), true, ""
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func firstIndent(lines []string) string {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		}
	}
	return ""
}

// nearLines points at lines containing old_text's first non-blank line.
func nearLines(content string, oldLines []string) string {
	first := ""
	for _, l := range oldLines {
		if t := strings.TrimSpace(l); t != "" {
			first = t
			break
		}
	}
	if first == "" {
		return ""
	}
	var hits []string
	for i, l := range strings.Split(content, "\n") {
		if strings.Contains(l, first) {
			hits = append(hits, fmt.Sprint(i+1))
			if len(hits) == 5 {
				break
			}
		}
	}
	if len(hits) == 0 {
		return " Its first line does not appear in the file."
	}
	return " Its first line appears at line(s) " + strings.Join(hits, ", ") + "; the lines after it differ."
}

// matchLines lists the line numbers of exact matches (for ambiguity errors).
func matchLines(content, old string) string {
	var nos []string
	idx := 0
	for len(nos) < 8 {
		k := strings.Index(content[idx:], old)
		if k < 0 {
			break
		}
		nos = append(nos, fmt.Sprint(strings.Count(content[:idx+k], "\n")+1))
		idx += k + 1
	}
	if len(nos) < 2 {
		return ""
	}
	return "; matches start at lines " + strings.Join(nos, ", ")
}
