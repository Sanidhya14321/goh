package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
)

type questionResponse struct {
	reply core.QuestionReply
	err   error
}
type questionMsg struct {
	request  core.QuestionRequest
	response chan questionResponse
}
type questionState struct {
	message       questionMsg
	index, cursor int
	selected      map[int]bool
	answers       [][]string
	custom        bool
	text          string
}

func (m *model) questionKey(v tea.KeyPressMsg) {
	q := m.question
	if q == nil {
		return
	}
	key := v.String()
	if key == "esc" && q.custom && len(q.message.request.Questions[q.index].Options) > 0 {
		q.custom, q.text = false, "" // back to the options, not dismiss everything
		return
	}
	if key == "esc" {
		q.message.response <- questionResponse{err: agent.ErrQuestionDismissed}
		m.question = nil
		return
	}
	current := q.message.request.Questions[q.index]
	if key == "enter" && (m.reviewHidden || !m.questionCursorVisible()) {
		// Enter answers what is highlighted, so it must be on screen.
		m.status = "Enlarge the terminal or scroll to see the answer before submitting (Esc dismisses)"
		return
	}
	defer m.followQuestionCursor()
	if q.custom {
		switch key {
		case "pgdown":
			m.reviewOffset += 5
		case "pgup":
			m.reviewOffset = max(0, m.reviewOffset-5)
		case "enter":
			if strings.TrimSpace(q.text) != "" {
				q.finish(m, []string{q.text})
			}
		case "ctrl+u":
			q.text = ""
		case "backspace":
			r := []rune(q.text)
			if len(r) > 0 {
				q.text = string(r[:len(r)-1])
			}
		default:
			if v.Text != "" && len(q.text) < 4096 {
				q.text += Safe(v.Text)
			}
		}
		return
	}
	switch key {
	case "up":
		if q.cursor > 0 {
			q.cursor--
		}
	case "down":
		if q.cursor+1 < len(current.Options) {
			q.cursor++
		}
	case "pgdown":
		m.reviewOffset += 5
	case "pgup":
		m.reviewOffset -= 5
		if m.reviewOffset < 0 {
			m.reviewOffset = 0
		}
	case "c":
		q.custom = true
		q.text = ""
		m.reviewOffset = 1 << 30 // the input is at the end; the view clamps
	case "space", " ":
		if current.Multiple && len(current.Options) > 0 {
			q.selected[q.cursor] = !q.selected[q.cursor]
		}
	case "enter":
		if len(current.Options) == 0 {
			q.custom = true
			m.reviewOffset = 1 << 30
			return
		}
		var answers []string
		if current.Multiple {
			for i, o := range current.Options {
				if q.selected[i] {
					answers = append(answers, o.Label)
				}
			}
			if len(answers) == 0 {
				return
			}
		} else {
			answers = []string{current.Options[q.cursor].Label}
		}
		q.finish(m, answers)
	}
}

// followQuestionCursor scrolls the question so the selected option stays on
// screen after the cursor moves (Enter answers what is highlighted, so it
// must be visible).
func (m *model) followQuestionCursor() {
	q := m.question
	if q == nil || q.custom || m.reviewHeight <= 0 {
		return
	}
	row, ok := m.questionCursorRow()
	if !ok {
		return
	}
	switch {
	case row < m.reviewOffset:
		m.reviewOffset = row
	case row >= m.reviewOffset+m.reviewHeight:
		m.reviewOffset = row - m.reviewHeight + 1
	}
}

// questionCursorRow is the row of the highlighted option (or the custom
// answer line) in the question as the review pane wraps it.
func (m *model) questionCursorRow() (int, bool) {
	q := m.question
	if q == nil {
		return 0, false
	}
	text, line := q.render()
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return 0, false
	}
	width := m.wrappedWidth
	if width <= 0 {
		width = max(1, m.width-4)
	}
	if line == 0 {
		return 0, true
	}
	return len(transcriptRows(strings.Join(lines[:line], "\n"), width, "", true)), true
}

// questionCursorVisible reports whether the last render showed the row
// Enter would answer with.
func (m *model) questionCursorVisible() bool {
	row, ok := m.questionCursorRow()
	return ok && m.reviewHeight > 0 && row >= m.reviewOffset && row < m.reviewOffset+m.reviewHeight
}

func (q *questionState) finish(m *model, answer []string) {
	q.answers = append(q.answers, answer)
	q.index++
	if q.index == len(q.message.request.Questions) {
		q.message.response <- questionResponse{reply: core.QuestionReply{Answers: q.answers}}
		m.question = nil
		return
	}
	q.cursor = 0
	q.selected = map[int]bool{}
	q.custom = false
	q.text = ""
	m.reviewOffset = 0
}
func (q *questionState) view() string {
	text, _ := q.render()
	return text
}

// render returns the question text and the line index of the highlighted
// option, or of the custom answer while one is typed.
func (q *questionState) render() (string, int) {
	current := q.message.request.Questions[q.index]
	var b strings.Builder
	fmt.Fprintf(&b, "Question %d/%d: %s\n\n%s\n\n", q.index+1, len(q.message.request.Questions), Safe(current.Header), Safe(current.Question))
	lineOf := func() int { return strings.Count(b.String(), "\n") }
	if q.custom {
		back := "Esc dismiss"
		if len(current.Options) > 0 {
			back = "Esc back to options"
		}
		line := lineOf()
		b.WriteString("Custom answer: " + Safe(q.text) + "▏\nEnter submit • " + back + "\n")
		return b.String(), line
	}
	cursorLine := -1
	for i, o := range current.Options {
		cursor := "  "
		if i == q.cursor {
			cursor = "> "
			cursorLine = lineOf()
		}
		check := ""
		if current.Multiple {
			check = "[ ] "
			if q.selected[i] {
				check = "[x] "
			}
		}
		line := cursor + check + Safe(o.Label)
		if d := strings.TrimSpace(o.Description); d != "" {
			line += " — " + Safe(d)
		}
		b.WriteString(line + "\n")
	}
	if current.Multiple {
		b.WriteString("\n↑/↓ choose • Space toggle • Enter submit (pick at least one) • c custom • Esc dismiss\n")
	} else {
		b.WriteString("\n↑/↓ choose • Enter submit • c custom • Esc dismiss\n")
	}
	return b.String(), cursorLine
}
