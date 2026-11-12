package remote

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/localapi"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

const token = "remote-client-fixture-token-0123456789abcdef"

// server starts a scripted model and a real local API server. Prompts
// "write" and "ask" make the model call create_file and question.
func server(t *testing.T) (*Client, localapi.Info) {
	t.Helper()
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []core.Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		last := request.Messages[len(request.Messages)-1]
		delta := map[string]any{"content": "server reply"}
		finish := "stop"
		name, args := "", ""
		switch last.Content {
		case "write":
			name, args = "create_file", `{"path":"made.txt","content":"remote"}`
		case "ask":
			name, args = "question", `{"questions":[{"header":"Color","question":"Pick","options":[{"label":"red","description":"r"}]}]}`
		case "slow":
			time.Sleep(3 * time.Second)
		}
		if last.Role == "user" && name != "" {
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call_" + last.Content, "type": "function", "function": map[string]string{"name": name, "arguments": args}}}}
			finish = "tool_calls"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, choice := range []map[string]any{{"index": 0, "delta": delta}, {"index": 0, "delta": map[string]any{}, "finish_reason": finish}} {
			b, _ := json.Marshal(map[string]any{"choices": []any{choice}})
			io.WriteString(w, "data: "+string(b)+"\n\n")
		}
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(model.Close)
	c := core.Defaults()
	c.Endpoint, c.Model, c.ContextWindow = model.URL, "fixture", 131072
	c.APIKeyEnv = "UNSET_REMOTE_FIXTURE_KEY"
	c.SessionDir = t.TempDir()
	handler, err := localapi.New(c, testutil.TempDir(t), token)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := New(srv.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	info, err := client.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return client, info
}

func TestNewRefusesNonLoopbackAndMissingToken(t *testing.T) {
	for _, bad := range []string{"http://example.com:4096", "https://127.0.0.1:4096", "http://user@127.0.0.1:1", "http://127.0.0.1:1/x"} {
		if _, err := New(bad, token); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
	if _, err := New("127.0.0.1:4096", ""); err == nil {
		t.Fatal("empty token accepted")
	}
	if _, err := New("127.0.0.1:4096", token); err != nil {
		t.Fatal(err)
	}
}

func TestRunRelaysEventsApprovalsAndQuestions(t *testing.T) {
	client, info := server(t)
	ctx := context.Background()
	id, err := client.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var events []string
	approvals, questions := 0, 0
	h := Handlers{
		Emit: func(e core.Event) { events = append(events, e.Type) },
		Approve: func(_ context.Context, p core.Permission) (core.Decision, error) {
			approvals++
			if p.Tool != "create_file" {
				t.Errorf("permission for %q", p.Tool)
			}
			return core.Decision{Allow: true}, nil
		},
		Ask: func(_ context.Context, q core.QuestionRequest) (core.QuestionReply, error) {
			questions++
			return core.QuestionReply{Answers: [][]string{{"red"}}}, nil
		},
	}
	status, err := client.Run(ctx, id, "write", false, h)
	if status != "completed" || err != nil {
		t.Fatal(status, err, events)
	}
	if b, err := os.ReadFile(filepath.Join(info.Project, "made.txt")); err != nil || string(b) != "remote" {
		t.Fatalf("approved write not applied: %q %v", b, err)
	}
	if status, err := client.Run(ctx, id, "ask", false, h); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if approvals != 1 || questions != 1 || !strings.Contains(strings.Join(events, ","), "run_finished") {
		t.Fatalf("approvals %d questions %d events %v", approvals, questions, events)
	}
	// The read-only view sees what the server wrote and cannot write.
	store, err := session.OpenReadOnly(info.SessionDir, id, info.Project)
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.New(core.Config{Model: "m", SessionDir: info.SessionDir, ContextWindow: 131072, OutputBudget: 4096}, store)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, m := range a.History {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "server reply") || !strings.Contains(joined, "write") {
		t.Fatalf("history not read: %q", joined)
	}
	if err := store.Append("message", core.Message{Role: "user", Content: "x"}); !errors.Is(err, session.ErrReadOnly) {
		t.Fatal("read-only view wrote:", err)
	}
	if list, err := client.Sessions(ctx); err != nil || len(list) != 1 || list[0].ID != id {
		t.Fatal(list, err)
	}
}

func TestDeniedApprovalAndCancellation(t *testing.T) {
	client, info := server(t)
	ctx := context.Background()
	id, _ := client.Create(ctx)
	deny := Handlers{Approve: func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{}, nil }}
	client.Run(ctx, id, "write", false, deny)
	if _, err := os.Stat(filepath.Join(info.Project, "made.txt")); err == nil {
		t.Fatal("denied write applied")
	}
	cctx, cancel := context.WithCancel(ctx)
	h := Handlers{Emit: func(e core.Event) {
		if e.Type == "run_started" {
			cancel()
		}
	}}
	start := time.Now()
	status, _ := client.Run(cctx, id, "slow", false, h)
	if status != "cancelled" || time.Since(start) > 10*time.Second {
		t.Fatalf("status %q after %s", status, time.Since(start))
	}
}

func TestWrongTokenIsReported(t *testing.T) {
	client, _ := server(t)
	client.token = strings.Repeat("x", 40)
	if _, err := client.Info(context.Background()); err == nil || !strings.Contains(err.Error(), "refused the token") {
		t.Fatal(err)
	}
}

func TestCancelBeforeResponseHeaders(t *testing.T) {
	release := make(chan struct{})
	gone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // as the real server decodes the prompt
		select {                    // then hold the headers until the client goes away
		case <-r.Context().Done():
			close(gone)
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)
	client, err := New(srv.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	status, err := client.Run(ctx, "s", "hi", false, Handlers{})
	if status != "cancelled" || err != nil || time.Since(start) > 5*time.Second {
		t.Fatalf("%q %v after %s", status, err, time.Since(start))
	}
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the server never saw the request end")
	}
}
