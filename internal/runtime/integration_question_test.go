package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestAskForIntegration(t *testing.T) {
	a, _ := testAgent(t)
	request := core.QuestionRequest{Questions: []core.Question{{Question: "Region?", Header: "MCP ops", Options: []core.QuestionOption{}}}}
	if _, err := a.AskForIntegration(context.Background(), request); !errors.Is(err, ErrNeedsInput) {
		t.Fatal("headless must refuse", err)
	}
	a.Ask = func(context.Context, core.QuestionRequest) (core.QuestionReply, error) {
		return core.QuestionReply{Answers: [][]string{{"eu"}}}, nil
	}
	if r, err := a.AskForIntegration(context.Background(), request); err != nil || r.Answers[0][0] != "eu" {
		t.Fatal(r, err)
	}
	a.Ask = func(context.Context, core.QuestionRequest) (core.QuestionReply, error) {
		return core.QuestionReply{Answers: [][]string{{"a"}, {"b"}}}, nil
	}
	if _, err := a.AskForIntegration(context.Background(), request); err == nil {
		t.Fatal("mismatched answers accepted")
	}
}
