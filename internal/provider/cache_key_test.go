package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// C5: Responses requests carry one prompt_cache_key per conversation.
func TestResponsesPromptCacheKey(t *testing.T) {
	c := core.Config{Model: "gpt-6.1-sol", OutputBudget: 100}
	msgs := []core.Message{{Role: "system", Content: "s"}, {Role: "user", Content: "hi"}}
	b, err := encodeResponses(c, msgs, nil)
	if err != nil || strings.Contains(string(b), "prompt_cache_key") {
		t.Fatalf("key without a session: %s %v", b, err)
	}
	c.CacheKey = "20261109T120000-abc"
	b, err = encodeResponses(c, msgs, nil)
	if err != nil || !strings.Contains(string(b), `"prompt_cache_key":"20261109T120000-abc"`) {
		t.Fatalf("%s %v", b, err)
	}
}

// C5: if an endpoint names prompt_cache_key as unsupported, the request is
// resent once without it (as the ChatGPT preview does for other fields).
func TestResponsesDropsRejectedCacheKeyOnce(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if strings.Contains(string(b), "prompt_cache_key") {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"param":"prompt_cache_key","code":"unsupported_parameter"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: "+`{"type":"response.output_text.delta","delta":"ok"}`+"\n\n")
		io.WriteString(w, "data: "+`{"type":"response.completed","response":{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}}`+"\n\n")
	}))
	defer server.Close()
	t.Setenv("FIXTURE_API_KEY", "k")
	c := New(core.Config{Endpoint: server.URL + "/v1/responses", Protocol: "openai_responses", Model: "m", APIKeyEnv: "FIXTURE_API_KEY", OutputBudget: 64, CacheKey: "session-1"})
	reply, err := c.Complete(context.Background(), []core.Message{{Role: "user", Content: "hi"}}, nil, nil)
	if err != nil || len(bodies) != 2 || strings.Contains(bodies[1], "prompt_cache_key") || !strings.Contains(reply.Message.Content, "ok") {
		t.Fatalf("err %v bodies %d reply %+v", err, len(bodies), reply.Message)
	}
}
