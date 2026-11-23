package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

type transcriptRow struct {
	text, fg, bg string
	styled       string
}

// fenceOpening returns the delimiter run that opens a fenced block on line
// ("```", "~~~~"), or "".
func fenceOpening(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 || trimmed[0] != '`' && trimmed[0] != '~' {
		return ""
	}
	n := markerRun(trimmed, 0, trimmed[0])
	if n < 3 || trimmed[0] == '`' && strings.Contains(trimmed[n:], "`") {
		return ""
	}
	return trimmed[:n]
}

// fenceCloses reports whether line closes a block opened by opening: a run of
// the same character at least as long, and nothing else.
func fenceCloses(line, opening string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= len(opening) && strings.Trim(trimmed, opening[:1]) == ""
}

// fenceContext returns the opening of an unfinished block. Starting there
// preserves multiline lexer state when a viewport shows only its final rows.
func fenceContext(text string) (string, int) {
	fence, opening, offset := "", 0, 0
	for len(text) > 0 {
		line, rest, found := strings.Cut(text, "\n")
		if fence == "" {
			if marker := fenceOpening(line); marker != "" {
				fence, opening = marker, offset
			}
		} else if fenceCloses(line, fence) {
			fence = ""
		}
		if !found {
			break
		}
		offset += len(line) + 1
		text = rest
	}
	return fence, opening
}

// closeFence is the line that ends a block a message left open, so the block
// cannot swallow the tool output and messages after it; "" when none is open.
func closeFence(message string) string {
	fence, _ := fenceContext(message)
	if fence == "" {
		return ""
	}
	if strings.HasSuffix(message, "\n") {
		return fence + "\n"
	}
	return "\n" + fence + "\n"
}

func plainInline(line string) string {
	for _, delimiter := range []string{"**", "\x60"} {
		if count := strings.Count(line, delimiter); count > 0 && count%2 == 0 {
			line = strings.ReplaceAll(line, delimiter, "")
		}
	}
	return line
}

func transcriptRows(text string, width int, fence string, review bool) []transcriptRow {
	var rows []transcriptRow
	lines := strings.Split(Safe(text), "\n")
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		row := transcriptRow{fg: foreground, bg: background}
		hang := ""
		if verbatim, ok := strings.CutPrefix(line, verbatimMark); ok && !review && fence == "" {
			switch {
			case strings.HasPrefix(verbatim, "[error]"):
				row.fg = red
			case strings.HasPrefix(verbatim, "✓"):
				row.fg = green
			default:
				row.fg = muted
			}
			for _, wrapped := range terminalRows(strings.ReplaceAll(verbatim, "\t", "    "), width) {
				row.text = wrapped
				rows = append(rows, row)
			}
			continue
		}
		if !review && (fence != "" || fenceOpening(line) != "") {
			marker, language := fence, ""
			start := index
			if marker == "" {
				marker = fenceOpening(line)
				language = strings.TrimSpace(strings.TrimSpace(line)[len(marker):])
				label := language
				if label == "" {
					label = "code"
				}
				for _, header := range terminalRows("┌─ "+label, width) {
					rows = append(rows, transcriptRow{text: header, fg: accent, bg: panel})
				}
				start++
			}
			end := start
			for end < len(lines) && !fenceCloses(lines[end], marker) {
				end++
			}
			code := strings.ReplaceAll(strings.Join(lines[start:end], "\n"), "\t", "    ")
			if end > start {
				highlighted := highlightCode(code, language)
				for _, physical := range strings.Split(highlighted, "\n") {
					style := ""
					for _, wrapped := range strings.Split(ansi.Hardwrap(physical, width, true), "\n") {
						rows = append(rows, transcriptRow{text: ansi.Strip(wrapped), styled: style + wrapped, fg: foreground, bg: panel})
						if last := strings.LastIndex(wrapped, "\x1b[38;2;"); last >= 0 {
							if end := strings.IndexByte(wrapped[last:], 'm'); end >= 0 {
								style = wrapped[last : last+end+1]
							}
						}
					}
				}
			}
			if end < len(lines) {
				rows = append(rows, transcriptRow{text: "└─", fg: accent, bg: panel})
			}
			index, fence = end, ""
			continue
		}
		if !review && index+1 < len(lines) {
			if header, ok := tableCells(line); ok {
				if align, ok := tableAlignment(lines[index+1], len(header)); ok {
					data := [][]string{header}
					end := index + 2
					for end < len(lines) {
						cells, ok := tableCells(lines[end])
						if !ok || len(cells) != len(header) {
							break
						}
						data = append(data, cells)
						end++
					}
					rows = append(rows, renderTable(data, align, width)...)
					index = end - 1
					continue
				}
			}
		}
		switch {
		case review:
			if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
				row.fg = muted // file header, not a change
			} else if strings.HasPrefix(line, "@@ ") {
				row.fg = blue
			} else if strings.HasPrefix(line, "+") {
				row.fg = green
			} else if strings.HasPrefix(line, "-") {
				row.fg = red
			}
		case strings.HasPrefix(line, "You:"):
			row.fg, row.bg = blue, panel
		case strings.HasPrefix(line, "[error]"):
			row.fg = red
		case strings.HasPrefix(line, "▸"), bracketTag(line):
			row.fg = muted
		case strings.HasPrefix(line, "✓"):
			row.fg = green
		default:
			trimmed := strings.TrimSpace(line)
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			if heading, rest, ok := strings.Cut(trimmed, " "); ok && len(heading) <= 6 && strings.Trim(heading, "#") == "" {
				line, row.fg = renderInline(rest), accent
			} else if trimmed == "---" || trimmed == "***" || trimmed == "___" {
				line, row.fg = strings.Repeat("─", min(width, 40)), muted
			} else if strings.HasPrefix(line, "> ") {
				line, row.fg = "│ "+renderInline(line[2:]), muted
				hang = "│ "
			} else if item, ok := listItem(trimmed); ok {
				marker := trimmed[:len(trimmed)-len(item)]
				if marker == "- " || marker == "* " || marker == "+ " {
					marker = "• "
				}
				line = indent + marker + renderInline(item)
				hang = strings.Repeat(" ", ansi.StringWidth(indent+marker))
			} else {
				line = renderInline(line)
			}
		}
		if review {
			for _, wrapped := range terminalRows(line, width) {
				row.text = wrapped
				rows = append(rows, row)
			}
			continue
		}
		for _, wrapped := range proseRows(line, width, hang) {
			row.text = wrapped
			rows = append(rows, row)
		}
	}
	return rows
}

// bracketTag reports a transcript notice ("[error] ...", "[tool] read")
// rather than prose that starts with a Markdown link ("[goh](url) is ...").
func bracketTag(line string) bool {
	if !strings.HasPrefix(line, "[") {
		return false
	}
	end := strings.IndexByte(line, ']')
	if end < 2 || end > 40 {
		return false
	}
	for _, c := range line[1:end] {
		if !(c == '_' || c == '-' || c == ' ' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	next := line[end+1:]
	return !strings.HasPrefix(next, "(") && !strings.HasPrefix(next, "[")
}

// listItem returns a bullet or numbered item's text.
func listItem(trimmed string) (string, bool) {
	for _, bullet := range []string{"- ", "* ", "+ "} {
		if item, ok := strings.CutPrefix(trimmed, bullet); ok {
			return item, true
		}
	}
	digits := 0
	for digits < len(trimmed) && digits < 9 && trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits+1 < len(trimmed) && (trimmed[digits] == '.' || trimmed[digits] == ')') && trimmed[digits+1] == ' ' {
		return trimmed[digits+2:], true
	}
	return "", false
}

// proseRows wraps text at spaces to rows of at most width display cells. A
// word wider than a row is broken across rows. Continuation rows start with
// hang (the indent under a list item's text) when it leaves room for words.
func proseRows(text string, width int, hang string) []string {
	text = strings.TrimRight(strings.ReplaceAll(Safe(text), "\t", "    "), " ")
	width = max(1, width)
	if ansi.StringWidth(text) <= width {
		return []string{text}
	}
	if ansi.StringWidth(hang)*2 > width {
		hang = ""
	}
	var rows []string
	row := text[:len(text)-len(strings.TrimLeft(text, " "))]
	if ansi.StringWidth(row)*2 > width {
		row = ""
	}
	used, empty := ansi.StringWidth(row), true
	flush := func() {
		rows = append(rows, row)
		row, used, empty = hang, ansi.StringWidth(hang), true
	}
	for _, word := range strings.Split(strings.TrimLeft(text, " "), " ") {
		if word == "" {
			continue
		}
		w := ansi.StringWidth(word)
		if !empty && used+1+w > width {
			flush()
		}
		if used+w <= width {
			if !empty {
				row += " "
				used++
			}
			row += word
			used += w
			empty = false
			continue
		}
		// Too wide for any row: break it, starting on a fresh row, in one
		// pass over its grapheme clusters.
		if !empty {
			flush()
		}
		for rest := word; rest != ""; {
			cluster, cw := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
			if cluster == "" {
				break
			}
			if used+cw > width {
				if !empty {
					flush()
				}
				if used+cw > width {
					row, used = "", 0 // the hang leaves no room for this cluster
				}
			}
			row += cluster
			used += cw
			empty = false
			rest = rest[len(cluster):]
		}
	}
	if !empty || len(rows) == 0 {
		rows = append(rows, row)
	}
	for i := range rows {
		rows[i] = ansi.Truncate(rows[i], width, "")
	}
	return rows
}

// renderInline is inline Markdown as a reader sees it: emphasis, strike and
// code markers removed, a link as its label followed by its address, an image
// as its alt text, and backslash escapes resolved. Unmatched markers stay
// literal ("**kwargs", "2 * 3").
func renderInline(s string) string {
	if !strings.ContainsAny(s, "*_~`[]<\\") {
		return s
	}
	var b strings.Builder
	inlineInto(&b, s, 0)
	return b.String()
}

func inlineInto(b *strings.Builder, s string, depth int) {
	if depth > 8 {
		b.WriteString(s)
		return
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && strings.IndexByte("\\`*_{}[]()#+-.!<>~|", s[i+1]) >= 0:
			b.WriteByte(s[i+1])
			i += 2
			continue
		case c == '`':
			n := markerRun(s, i, '`')
			if end := closingRun(s, i+n, '`', n, false); end >= 0 {
				code := s[i+n : end]
				if len(code) >= 2 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.TrimSpace(code) != "" {
					code = code[1 : len(code)-1]
				}
				b.WriteString(code)
				i = end + n
				continue
			}
			b.WriteString(s[i : i+n])
			i += n
			continue
		case c == '!' && strings.HasPrefix(s[i:], "!["):
			if label, _, next, ok := parseLink(s, i+1); ok {
				inlineInto(b, label, depth+1)
				i = next
				continue
			}
		case c == '[':
			if label, target, next, ok := parseLink(s, i); ok {
				var text strings.Builder
				inlineInto(&text, label, depth+1)
				b.WriteString(text.String())
				if target != "" && target != text.String() && !strings.HasPrefix(target, "#") {
					b.WriteString(" (" + target + ")")
				}
				i = next
				continue
			}
		case c == '<':
			if end := strings.IndexByte(s[i:], '>'); end > 0 {
				inner := s[i+1 : i+end]
				if (strings.HasPrefix(inner, "https://") || strings.HasPrefix(inner, "http://") || strings.HasPrefix(inner, "mailto:")) && !strings.ContainsAny(inner, " <") {
					b.WriteString(inner)
					i += end + 1
					continue
				}
			}
		case c == '*' || c == '_' || c == '~':
			n := markerRun(s, i, c)
			if (c != '~' || n == 2) && n <= 3 && opensEmphasis(s, i, n, c) {
				if end := closingRun(s, i+n, c, n, true); end >= 0 {
					inlineInto(b, s[i+n:end], depth+1)
					i = end + n
					continue
				}
			}
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+size])
		i += size
	}
}

func markerRun(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
}

func wordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

// opensEmphasis: the run is followed by text, and "_" does not sit inside a
// word (snake_case_name).
func opensEmphasis(s string, i, n int, c byte) bool {
	if i+n >= len(s) || s[i+n] == ' ' {
		return false
	}
	return c != '_' || i == 0 || !wordByte(s[i-1])
}

// closingRun finds a run of exactly n c's after from that closes a span: not
// preceded by a space (for emphasis) and, for "_", not followed by a word
// character. Code spans in between are skipped when emphasis is true.
func closingRun(s string, from int, c byte, n int, emphasis bool) int {
	for j := from; j < len(s); {
		if emphasis && s[j] == '\\' {
			j += 2 // an escaped marker closes nothing
			continue
		}
		if emphasis && s[j] == '`' && c != '`' {
			m := markerRun(s, j, '`')
			if end := closingRun(s, j+m, '`', m, false); end >= 0 {
				j = end + m
				continue
			}
			j += m
			continue
		}
		if s[j] != c {
			j++
			continue
		}
		m := markerRun(s, j, c)
		if m == n && j > from && (!emphasis || s[j-1] != ' ') && (c != '_' || j+m >= len(s) || !wordByte(s[j+m])) {
			return j
		}
		j += m
	}
	return -1
}

// parseLink reads "[label](target)" at s[i] == '['.
func parseLink(s string, i int) (label, target string, next int, ok bool) {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			if j+1 >= len(s) || s[j+1] != '(' {
				return "", "", 0, false
			}
			parens := 0
			for k := j + 1; k < len(s); k++ {
				switch s[k] {
				case '(':
					parens++
				case ')':
					parens--
					if parens == 0 {
						target = strings.TrimSpace(s[j+2 : k])
						if before, _, found := strings.Cut(target, " \""); found {
							target = strings.TrimSpace(before)
						}
						target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
						return s[i+1 : j], target, k + 1, true
					}
				}
			}
			return "", "", 0, false
		}
	}
	return "", "", 0, false
}

func transcriptCell(row transcriptRow, width int) string {
	if row.styled == "" {
		return cell("  "+row.text, width, row.fg, row.bg)
	}
	// styled is generated solely by highlightCode from sanitized text.
	content := ansi.Truncate(row.styled, max(0, width-2), "")
	return paint("  "+content+strings.Repeat(" ", max(0, width-2-ansi.StringWidth(content))), row.fg, row.bg)
}
