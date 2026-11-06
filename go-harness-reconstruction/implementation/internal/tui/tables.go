package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func tableCells(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "|") {
		return nil, false
	}
	var cells []string
	var cell strings.Builder
	codeTicks := 0
	lastPipe := false
	for i := 0; i < len(line); {
		lastPipe = false
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i += 2
		case line[i] == 96:
			end := i + 1
			for end < len(line) && line[end] == 96 {
				end++
			}
			count := end - i
			if codeTicks == 0 {
				codeTicks = count
			} else if codeTicks == count {
				codeTicks = 0
			}
			cell.WriteString(line[i:end])
			i = end
		case line[i] == '|' && codeTicks == 0:
			lastPipe = true
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			i++
		default:
			cell.WriteByte(line[i])
			i++
		}
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	if strings.HasPrefix(line, "|") {
		cells = cells[1:]
	}
	if lastPipe {
		cells = cells[:len(cells)-1]
	}
	return cells, len(cells) > 0
}

func tableAlignment(line string, columns int) ([]string, bool) {
	cells, ok := tableCells(line)
	if !ok || len(cells) != columns {
		return nil, false
	}
	align := make([]string, columns)
	for i, cell := range cells {
		dashes := strings.TrimSuffix(strings.TrimPrefix(cell, ":"), ":")
		if len(dashes) < 3 || strings.Trim(dashes, "-") != "" {
			return nil, false
		}
		if strings.HasSuffix(cell, ":") {
			align[i] = "right"
			if strings.HasPrefix(cell, ":") {
				align[i] = "center"
			}
		}
	}
	return align, true
}

func renderTable(data [][]string, align []string, width int) []transcriptRow {
	columns := len(data[0])
	for _, row := range data {
		for i := range row {
			row[i] = plainInline(row[i])
		}
	}
	available := width - 3*columns - 1
	var rows []transcriptRow
	if available < columns*3 {
		// Stack columns in narrow panes, keeping all headers and values.
		for _, row := range data[1:] {
			for i, value := range row {
				for _, line := range terminalRows(data[0][i]+": "+value, width) {
					rows = append(rows, transcriptRow{text: line, fg: foreground, bg: background})
				}
			}
			rows = append(rows, transcriptRow{fg: muted, bg: background})
		}
		if len(data) == 1 {
			for _, header := range data[0] {
				for _, line := range terminalRows(header, width) {
					rows = append(rows, transcriptRow{text: line, fg: accent, bg: background})
				}
			}
		}
		return rows
	}
	widths, desired := make([]int, columns), make([]int, columns)
	for i := range widths {
		widths[i] = 3
		for _, row := range data {
			desired[i] = max(desired[i], ansi.StringWidth(row[i]))
		}
	}
	for extra := available - 3*columns; extra > 0; extra-- {
		best := 0
		for i := range widths {
			if desired[i]-widths[i] > desired[best]-widths[best] {
				best = i
			}
		}
		if desired[best] <= widths[best] {
			break
		}
		widths[best]++
	}
	border := func(left, middle, right string) {
		parts := make([]string, columns)
		for i, size := range widths {
			parts[i] = strings.Repeat("─", size+2)
		}
		rows = append(rows, transcriptRow{text: left + strings.Join(parts, middle) + right, fg: muted, bg: background})
	}
	border("┌", "┬", "┐")
	for index, dataRow := range data {
		wrapped := make([][]string, columns)
		height := 1
		for i, value := range dataRow {
			wrapped[i] = terminalRows(value, widths[i])
			height = max(height, len(wrapped[i]))
		}
		for line := 0; line < height; line++ {
			var out strings.Builder
			out.WriteString("│")
			for i, parts := range wrapped {
				value := ""
				if line < len(parts) {
					value = parts[line]
				}
				padding := widths[i] - ansi.StringWidth(value)
				left := 0
				if align[i] == "right" {
					left = padding
				} else if align[i] == "center" {
					left = padding / 2
				}
				out.WriteString(" " + strings.Repeat(" ", left) + value + strings.Repeat(" ", padding-left) + " │")
			}
			color := foreground
			if index == 0 {
				color = accent
			}
			rows = append(rows, transcriptRow{text: out.String(), fg: color, bg: background})
		}
		if index == 0 {
			border("├", "┼", "┤")
		}
	}
	border("└", "┴", "┘")
	return rows
}
