package tui

import "strings"

// thinkMark prefixes reasoning lines in the log; /thinking shows or hides
// them. Safe() neutralises NUL in content, so the marker cannot be forged.
const thinkMark = "\x00K"

const maxReasoning = 32 << 10

// addReasoning buffers streamed reasoning until the turn moves on, so a
// whole block is written at once and partial lines never interleave with
// the answer.
func (m *model) addReasoning(text string) {
	if m.reasoning.Len()+len(text) <= maxReasoning {
		m.reasoning.WriteString(Safe(text))
	}
	m.status = "Thinking…"
}

// flushReasoning writes buffered reasoning to the log as marked, dimmed
// lines. It runs on any other event, so reasoning lands before the answer
// text or tool call it led to.
func (m *model) flushReasoning() {
	text := strings.TrimSpace(m.reasoning.String())
	m.reasoning.Reset()
	if text == "" {
		return
	}
	var b strings.Builder
	if !strings.HasSuffix(m.log, "\n") && m.log != "" {
		b.WriteString("\n")
	}
	for _, line := range strings.Split(text, "\n") {
		b.WriteString(thinkMark + "  ┊ " + line + "\n")
	}
	m.log += b.String()
}

// withThinking drops reasoning lines when hidden, otherwise unmarks them.
func withThinking(log string, show bool) string {
	if !strings.Contains(log, thinkMark) {
		return log
	}
	lines := strings.Split(log, "\n")
	out := lines[:0]
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(line, thinkMark); ok {
			if show {
				out = append(out, rest)
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func (m *model) toggleThinking() {
	m.hideThinking = !m.hideThinking
	m.status = "Reasoning shown"
	if m.hideThinking {
		m.status = "Reasoning hidden"
	}
}
