package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type transcriptRow struct {
	text, fg, bg string
	styled       string
}

func fenceMarker(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "\x60\x60\x60") || strings.HasPrefix(trimmed, "~~~") {
		return trimmed[:3]
	}
	return ""
}

// fenceContext returns the opening of an unfinished block. Starting there
// preserves multiline lexer state when a viewport shows only its final rows.
func fenceContext(text string) (string, int) {
	fence, opening, offset := "", 0, 0
	for len(text) > 0 {
		line, rest, found := strings.Cut(text, "\n")
		if marker := fenceMarker(line); marker != "" {
			if fence == "" {
				fence, opening = marker, offset
			} else if fence == marker {
				fence = ""
			}
		}
		if !found {
			break
		}
		offset += len(line) + 1
		text = rest
	}
	return fence, opening
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
		if !review && (fence != "" || fenceMarker(line) != "") {
			marker, language := fence, ""
			start := index
			if marker == "" {
				marker = fenceMarker(line)
				language = strings.TrimSpace(strings.TrimSpace(line)[3:])
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
			for end < len(lines) && fenceMarker(lines[end]) != marker {
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
			if strings.HasPrefix(line, "+") {
				row.fg = green
			} else if strings.HasPrefix(line, "-") {
				row.fg = red
			}
		case strings.HasPrefix(line, "You:"):
			row.fg, row.bg = blue, panel
		case strings.HasPrefix(line, "[error]"):
			row.fg = red
		case strings.HasPrefix(line, "▸"), strings.HasPrefix(line, "["):
			row.fg = muted
		case strings.HasPrefix(line, "✓"):
			row.fg = green
		default:
			trimmed := strings.TrimSpace(line)
			if heading, rest, ok := strings.Cut(trimmed, " "); ok && len(heading) <= 6 && strings.Trim(heading, "#") == "" {
				line, row.fg = rest, accent
			} else if strings.HasPrefix(line, "> ") {
				line, row.fg = "│ "+line[2:], muted
			} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
				line = "• " + line[2:]
			}
			line = plainInline(line)
		}
		for _, wrapped := range terminalRows(line, width) {
			row.text = wrapped
			rows = append(rows, row)
		}
	}
	return rows
}

func transcriptCell(row transcriptRow, width int) string {
	if row.styled == "" {
		return cell("  "+row.text, width, row.fg, row.bg)
	}
	// styled is generated solely by highlightCode from sanitized text.
	content := ansi.Truncate(row.styled, max(0, width-2), "")
	return paint("  "+content+strings.Repeat(" ", max(0, width-2-ansi.StringWidth(content))), row.fg, row.bg)
}
