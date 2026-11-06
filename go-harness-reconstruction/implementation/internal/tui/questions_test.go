package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
)

func TestQuestionsMultipleCustomAndDismissal(t *testing.T) {
	ch := make(chan questionResponse, 1)
	m := &model{question: &questionState{message: questionMsg{request: core.QuestionRequest{Questions: []core.Question{{Header: "Checks", Question: "Choose checks", Multiple: true, Options: []core.QuestionOption{{Label: "unit"}, {Label: "race"}}}, {Header: "Notes", Question: "More context", Options: []core.QuestionOption{}}}}, response: ch}, selected: map[int]bool{}}}
	m.questionKey(tea.KeyPressMsg{Code: ' ', Text: " "})
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyDown})
	m.questionKey(tea.KeyPressMsg{Code: ' ', Text: " "})
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.questionKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	for _, r := range "custom notes" {
		m.questionKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m.questionKey(tea.KeyPressMsg{Code: tea.KeyEnter})
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
