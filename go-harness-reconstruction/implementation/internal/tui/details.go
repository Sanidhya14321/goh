package tui

import "strings"

// Tool results are written twice behind markers: a one-line summary and the
// full output. /details picks which is shown, like OpenCode's tool-details
// toggle. Safe() replaces NUL in all content, so markers cannot be forged.
const (
	summaryMark = "\x00H"
	detailMark  = "\x00R"
)

func toolResultLines(mark, content string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	summary := mark + oneLine(lines[0], 80)
	if len(lines) > 1 {
		summary += " (+" + itoa(len(lines)-1) + " lines)"
	}
	var b strings.Builder
	b.WriteString(summaryMark + summary + "\n")
	for i, line := range lines {
		if i == 0 {
			line = mark + line
		}
		b.WriteString(detailMark + line + "\n")
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

// withDetails keeps either the full tool output or its summaries.
func withDetails(log string, show bool) string {
	if !strings.Contains(log, summaryMark) {
		return log
	}
	keep, drop := detailMark, summaryMark
	if !show {
		keep, drop = summaryMark, detailMark
	}
	lines := strings.Split(log, "\n")
	out := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(line, drop) {
			continue
		}
		out = append(out, strings.TrimPrefix(line, keep))
	}
	return strings.Join(out, "\n")
}

func (m *model) toggleDetails() {
	m.hideDetails = !m.hideDetails
	m.status = "Tool details shown"
	if m.hideDetails {
		m.status = "Tool details hidden (one line per result)"
	}
}

// displayLog is the transcript as rendered (timestamps and details applied).
func (m *model) displayLog() string { return withDetails(withStamps(m.log, m.timestamps), !m.hideDetails) }
