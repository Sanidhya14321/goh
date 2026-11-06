package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
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
	if key == "esc" {
		q.message.response <- questionResponse{err: agent.ErrQuestionDismissed}
		m.question = nil
		return
	}
	current := q.message.request.Questions[q.index]
	if q.custom {
		switch key {
		case "enter":
			if strings.TrimSpace(q.text) != "" {
				q.finish(m, []string{q.text})
			}
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
	case "space", " ":
		if current.Multiple && len(current.Options) > 0 {
			q.selected[q.cursor] = !q.selected[q.cursor]
		}
	case "enter":
		if len(current.Options) == 0 {
			q.custom = true
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
	current := q.message.request.Questions[q.index]
	var b strings.Builder
	fmt.Fprintf(&b, "Question %d/%d: %s\n\n%s\n\n", q.index+1, len(q.message.request.Questions), Safe(current.Header), Safe(current.Question))
	if q.custom {
		b.WriteString("Custom answer: " + Safe(q.text) + "\nEnter submit • Esc dismiss all\n")
		return b.String()
	}
	for i, o := range current.Options {
		cursor := "  "
		if i == q.cursor {
			cursor = "> "
		}
		check := ""
		if current.Multiple {
			check = "[ ] "
			if q.selected[i] {
				check = "[x] "
			}
		}
		b.WriteString(cursor + check + Safe(o.Label) + " — " + Safe(o.Description) + "\n")
	}
	b.WriteString("\n↑/↓ choose • Space toggle multiple • Enter submit • c custom • Esc dismiss\n")
	return b.String()
}
