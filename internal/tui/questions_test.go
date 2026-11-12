package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
)

func TestQuestionsMultipleCustomAndDismissal(t *testing.T) {
	ch := make(chan questionResponse, 1)
	m := &model{question: &questionState{message: questionMsg{request: core.QuestionRequest{Questions: []core.Question{{Header: "Checks", Question: "Choose checks", Multiple: true, Options: []core.QuestionOption{{Label: "unit"}, {Label: "race"}}}, {Header: "Notes", Question: "More context", Options: []core.QuestionOption{}}}}, response: ch}, selected: map[int]bool{}}, width: 100, height: 30}
	key := func(k tea.KeyPressMsg) { m.View(); m.questionKey(k) } // keys act on what is shown
	key(tea.KeyPressMsg{Code: ' ', Text: " "})
	key(tea.KeyPressMsg{Code: tea.KeyDown})
	key(tea.KeyPressMsg{Code: ' ', Text: " "})
	key(tea.KeyPressMsg{Code: tea.KeyEnter})
	key(tea.KeyPressMsg{Code: 'c', Text: "c"})
	for _, r := range "custom notes" {
		key(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	key(tea.KeyPressMsg{Code: tea.KeyEnter})
	reply := <-ch
	if len(reply.reply.Answers) != 2 || len(reply.reply.Answers[0]) != 2 || reply.reply.Answers[1][0] != "custom notes" || m.question != nil {
		t.Fatal(reply)
	}
	m.question = &questionState{message: questionMsg{response: ch}}
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if result := <-ch; !errors.Is(result.err, agent.ErrQuestionDismissed) {
		t.Fatal(result)
	}
}

// Enter answers only when the highlighted option is on screen, even if the
// question's heading is; a Markdown quote in the question must not be taken
// for the highlighted option when scrolling to it.
func TestQuestionAnswerMustBeVisible(t *testing.T) {
	ch := make(chan questionResponse, 1)
	question := core.Question{Header: "Pick", Question: "> quoted line\n> another quoted line\nWhich one?", Options: []core.QuestionOption{{Label: "alpha"}, {Label: "beta"}, {Label: "gamma"}, {Label: "delta"}}}
	m := &model{question: &questionState{message: questionMsg{request: core.QuestionRequest{Questions: []core.Question{question}}, response: ch}, selected: map[int]bool{}}, width: 64, height: 4}
	m.View()
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case r := <-ch:
		t.Fatalf("answered %v with only the heading visible", r.reply.Answers)
	default:
	}
	m.height = 18
	for range 3 {
		m.View()
		m.questionKey(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	m.View()
	if !m.questionCursorVisible() {
		t.Fatalf("highlighted option scrolled out of view (offset %d height %d)", m.reviewOffset, m.reviewHeight)
	}
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if r := <-ch; len(r.reply.Answers) != 1 || r.reply.Answers[0][0] != "delta" {
		t.Fatal(r)
	}
}
