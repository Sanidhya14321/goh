package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestResponsesReasoningEffort(t *testing.T) {
	cfg := core.Defaults()
	cfg.Model = "fixture"
	msgs := []core.Message{{Role: "system", Content: "s"}, {Role: "user", Content: "u"}}
	b, err := encodeResponses(cfg, msgs, nil)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	json.Unmarshal(b, &body)
	if _, ok := body["reasoning"]; ok {
		t.Fatal("default must not send reasoning")
	}
	cfg.ReasoningEffort = "high"
	b, _ = encodeResponses(cfg, msgs, nil)
	body = nil
	json.Unmarshal(b, &body)
	if r, _ := body["reasoning"].(map[string]any); r["effort"] != "high" {
		t.Fatal(string(b))
	}
}

func TestChatReasoningEffortSent(t *testing.T) {
	var got map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		io.WriteString(w, textFrame("ok")+finish("stop")+"data: [DONE]\n\n")
	}))
	defer s.Close()
	c := New(core.Config{Endpoint: s.URL, Model: "fixture", ReasoningEffort: "low"})
	if _, err := c.Complete(context.Background(), []core.Message{{Role: "user", Content: "hi"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got["reasoning_effort"] != "low" {
		t.Fatal(got)
	}
}
