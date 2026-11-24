package tui

import "strings"

// Tool results are written twice behind markers: a one-line summary and the
// full output. /details picks which is shown, like OpenCode's tool-details
// toggle. Safe() replaces NUL in all content, so markers cannot be forged.
const (
	summaryMark = "\x00H"
	detailMark  = "\x00R"
	// previewMark lines are a fetched page's preview: hidden until /details
	// asks for previews, because a page is reference material for the model,
	// not output the user is waiting to read.
	previewMark = "\x00P"
	// verbatimMark (WORD JOINER, zero width) survives Safe() and tells the
	// transcript renderer to show the line as-is: tool output is file and
	// command text, not Markdown, so "# comment" must not become a heading
	// and "**kwargs" must keep its asterisks.
	verbatimMark = "\u2060"
)

// summaryOutputLines is how much output the compact view shows under a
// result's first line; /details shows all of it.
const summaryOutputLines = 4

func toolResultLines(mark, content string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	var b strings.Builder
	b.WriteString(summaryMark + mark + oneLine(lines[0], 120) + "\n")
	rest := lines[1:]
	// Most commands end with what matters (a summary, the failure), so the
	// compact view keeps the last lines.
	if len(rest) > summaryOutputLines {
		b.WriteString(summaryMark + "  … " + itoa(len(rest)-summaryOutputLines+1) + " earlier lines · /details shows all\n")
		rest = rest[len(rest)-summaryOutputLines+1:]
	}
	for _, line := range rest {
		b.WriteString(summaryMark + "  " + strings.TrimRight(line, " \r") + "\n")
	}
	for i, line := range lines {
		if i == 0 {
			line = mark + line
		} else {
			line = "  " + line // output sits under its result line
		}
		b.WriteString(detailMark + line + "\n")
	}
	return b.String()
}

// fetchedPageLines writes a page result as the same line in both modes plus
// its preview.
func fetchedPageLines(mark, summary, preview string) string {
	var b strings.Builder
	b.WriteString(summaryMark + mark + summary + "\n")
	b.WriteString(detailMark + mark + summary + "\n")
	if preview != "" {
		for _, line := range strings.Split(preview, "\n") {
			b.WriteString(previewMark + "  " + line + "\n")
		}
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

// detailsMode is what /details shows; it cycles through the three, skipping
// previews when the transcript has none.
type detailsMode int

const (
	detailsHidden   detailsMode = iota // compact: a result line and the last few output lines (default)
	detailsShown                       // all tool output; fetched pages as one line
	detailsPreviews                    // and fetched-page previews
)

// withDetails keeps the lines the mode shows and drops the markers.
func withDetails(log string, mode detailsMode) string {
	if !strings.Contains(log, summaryMark) {
		return log
	}
	lines := strings.Split(log, "\n")
	out := lines[:0]
	for _, line := range lines {
		var keep bool
		var content string
		switch {
		case strings.HasPrefix(line, summaryMark):
			keep, content = mode == detailsHidden, line[len(summaryMark):]
		case strings.HasPrefix(line, detailMark):
			keep, content = mode != detailsHidden, line[len(detailMark):]
		case strings.HasPrefix(line, previewMark):
			keep, content = mode == detailsPreviews, line[len(previewMark):]
		default:
			out = append(out, line)
			continue
		}
		if keep {
			out = append(out, verbatimMark+content)
		}
	}
	return strings.Join(out, "\n")
}

func (m *model) toggleDetails() {
	switch m.details {
	case detailsHidden:
		m.details = detailsShown
	case detailsShown:
		m.details = detailsHidden
		if strings.Contains(m.log, previewMark) {
			m.details = detailsPreviews
		}
	default:
		m.details = detailsHidden
	}
	switch m.details {
	case detailsPreviews:
		m.status = "Tool details shown, with fetched page previews"
	case detailsHidden:
		m.status = "Tool details compact (last lines of each result)"
	default:
		m.status = "Tool details shown"
	}
}

// displayLog is the transcript as rendered (timestamps and details applied).
func (m *model) displayLog() string {
	return withThinking(withDetails(withStamps(m.log, m.timestamps), m.details), !m.hideThinking)
}
