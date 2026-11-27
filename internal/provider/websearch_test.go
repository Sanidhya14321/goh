package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

const searchMessage = `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Go 1.27.4 is current.","annotations":[` +
	`{"type":"url_citation","url":"https://go.dev/dl/","title":"Downloads"},{"type":"url_citation","url":"https://go.dev/dl/","title":"again"},` +
	`{"type":"url_citation","url":"javascript:alert(1)","title":"x"}]}]}`

func TestWebSearchResponses(t *testing.T) {
	t.Setenv("FIXTURE_API_KEY", "k")
	for name, stream := range map[string]string{
		// the ChatGPT service: items stream, the completed event has none
		"items": `data: {"type":"response.output_item.done","item":{"type":"web_search_call","status":"completed"}}` + "\n\n" +
			`data: {"type":"response.output_item.done","item":` + searchMessage + `}` + "\n\n" +
			`data: {"type":"response.completed","response":{"status":"completed","output":[]}}` + "\n\n",
		"completed": `data: {"type":"response.completed","response":{"status":"completed","output":[` + searchMessage + `]}}` + "\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			var sent map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				json.Unmarshal(b, &sent)
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, stream)
			}))
			defer server.Close()
			c := New(core.Config{Endpoint: server.URL + "/v1/responses", Protocol: "openai_responses", Model: "m", APIKeyEnv: "FIXTURE_API_KEY"})
			a, err := c.WebSearch(context.Background(), "go version")
			if err != nil {
				t.Fatal(err)
			}
			if a.Answer != "Go 1.27.4 is current." || len(a.Sources) != 1 || a.Sources[0].URL != "https://go.dev/dl/" {
				t.Fatalf("%+v", a)
			}
			tools, _ := json.Marshal(sent["tools"])
			if string(tools) != `[{"type":"web_search"}]` || sent["store"] != false {
				t.Fatalf("request %v", sent)
			}
		})
	}
}

func TestWebSearchAnthropic(t *testing.T) {
	t.Setenv("FIXTURE_API_KEY", "k")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"web_search_20250305"`) || r.Header.Get("x-api-key") != "k" {
			t.Errorf("request %s", b)
		}
		io.WriteString(w, `{"content":[{"type":"server_tool_use","id":"s","name":"web_search"},`+
			`{"type":"web_search_tool_result","content":[{"type":"web_search_result","url":"https://a.example/","title":"A"}]},`+
			`{"type":"text","text":"Answer.","citations":[{"type":"web_search_result_location","url":"https://b.example/","title":"B"}]}]}`)
	}))
	defer server.Close()
	c := New(core.Config{Endpoint: server.URL + "/v1/messages", Protocol: "anthropic_messages", Model: "m", APIKeyEnv: "FIXTURE_API_KEY"})
	a, err := c.WebSearch(context.Background(), "q")
	if err != nil || a.Answer != "Answer." || len(a.Sources) != 2 {
		t.Fatalf("%+v %v", a, err)
	}
}

func TestWebSearchFailures(t *testing.T) {
	t.Setenv("FIXTURE_API_KEY", "k")
	if New(core.Config{Protocol: "chat_completions"}).SupportsWebSearch() {
		t.Fatal("chat completions has no hosted search")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"error":{"message":"secret-echo"}}`)
	}))
	defer server.Close()
	c := New(core.Config{Endpoint: server.URL + "/v1/responses", Protocol: "openai_responses", Model: "m", APIKeyEnv: "FIXTURE_API_KEY"})
	_, err := c.WebSearch(context.Background(), "q")
	if err == nil || strings.Contains(err.Error(), "secret-echo") || !strings.Contains(err.Error(), "400") {
		t.Fatal(err)
	}
	if _, err := webAnswer("  ", nil); err == nil {
		t.Fatal("empty answer accepted")
	}
}
