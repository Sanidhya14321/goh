package localapi

import (
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
	"github.com/Sanidhya14321/goh/internal/session"
)

const testToken = "fixture-token-12345678901234567890123456"

type fixture struct {
	t       *testing.T
	server  *httptest.Server
	config  core.Config
	project string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []core.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) == 0 {
			http.Error(w, "invalid fixture request", 400)
			return
		}
		last := request.Messages[len(request.Messages)-1]
		content := map[string]any{"content": "fixture complete"}
		finish := "stop"
		name, arguments := "", ""
		switch last.Content {
		case "write":
			name, arguments = "create_file", `{"path":"approved.txt","content":"approved"}`
		case "cancel":
			name, arguments = "create_file", `{"path":"cancelled.txt","content":"unexpected"}`
		case "ask":
			name, arguments = "question", `{"questions":[{"header":"Color","question":"Choose a color","options":[]}]}`
		}
		if last.Role == "user" && name != "" {
			content = map[string]any{"tool_calls": []any{map[string]any{
				"index": 0, "id": "call_" + last.Content, "type": "function",
				"function": map[string]string{"name": name, "arguments": arguments},
			}}}
			finish = "tool_calls"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, choice := range []map[string]any{
			{"index": 0, "delta": content},
			{"index": 0, "delta": map[string]any{}, "finish_reason": finish},
		} {
			data, _ := json.Marshal(map[string]any{"choices": []any{choice}})
			_, _ = io.WriteString(w, "data: "+string(data)+"\n\n")
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(provider.Close)
	config := core.Defaults()
	config.Endpoint, config.Model, config.ContextWindow = provider.URL, "fixture", 131072
	config.APIKeyEnv = "UNSET_LOCAL_API_FIXTURE_KEY"
	config.SessionDir = t.TempDir()
	project := t.TempDir()
	handler, err := New(config, project, testToken)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	server.Client().Timeout = 8 * time.Second
	t.Cleanup(server.Close)
	return fixture{t: t, server: server, config: config, project: handler.project}
}

func (f fixture) request(method, path, body string, status int, headers map[string]string) *http.Response {
	f.t.Helper()
	req, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		f.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	for key, value := range headers {
		if key == "Host" {
			req.Host = value
		} else {
			req.Header.Set(key, value)
		}
	}
	res, err := f.server.Client().Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	if res.StatusCode != status {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		f.t.Fatalf("%s %s: got %d, want %d (%s)", method, path, res.StatusCode, status, b)
	}
	return res
}

func (f fixture) create() string {
	f.t.Helper()
	res := f.request("POST", "/v1/sessions", "", 201, nil)
	defer res.Body.Close()
	var info session.Info
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil || info.ID == "" || info.Project != f.project {
		f.t.Fatal(info, err)
	}
	return info.ID
}

type frame struct {
	Event *core.Event `json:"event"`
	Input *inputFrame `json:"input"`
}

func readInput(t *testing.T, d *json.Decoder, kind string) string {
	t.Helper()
	for {
		var v frame
		if err := d.Decode(&v); err != nil {
			t.Fatal("input challenge missing:", err)
		}
		if v.Input != nil {
			if v.Input.Kind != kind || v.Input.ID == "" {
				t.Fatal(v.Input)
			}
			return v.Input.ID
		}
	}
}

func finish(t *testing.T, d *json.Decoder, want string) {
	t.Helper()
	var terminal string
	for {
		var v frame
		err := d.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if v.Event != nil && v.Event.Type == "run_finished" {
			if terminal != "" {
				t.Fatal("duplicate terminal event")
			}
			data, ok := v.Event.Data.(map[string]any)
			if !ok {
				t.Fatal(v.Event)
			}
			terminal, _ = data["status"].(string)
		}
	}
	if terminal != want {
		t.Fatalf("terminal status %q, want %q", terminal, want)
	}
}

func TestStreamingApprovalQuestionAndCancellation(t *testing.T) {
	f := newFixture(t)
	id := f.create()
	path := "/v1/sessions/" + id + "/run"
	for _, test := range []struct {
		prompt, kind, reply, status string
	}{
		{"write", "permission", `{"allow":true}`, "completed"},
		{"ask", "question", `{"answers":[["blue"]]}`, "completed"},
		{"cancel", "permission", "", "cancelled"},
	} {
		res := f.request("POST", path, `{"prompt":"`+test.prompt+`"}`, 200, nil)
		d := json.NewDecoder(res.Body)
		input := readInput(t, d, test.kind)
		runPath := "/v1/runs/" + res.Header.Get("X-Harness-Run-ID")
		f.request("POST", path, `{"prompt":"overlap"}`, 409, nil).Body.Close()
		inputPath := runPath + "/inputs/" + input
		f.request("POST", runPath+"/inputs/stale", `{"allow":true}`, 404, nil).Body.Close()
		f.request("POST", inputPath, `{}`, 400, nil).Body.Close()
		if test.reply == "" {
			f.request("POST", runPath+"/cancel", "", 202, nil).Body.Close()
		} else {
			f.request("POST", inputPath, test.reply, 204, nil).Body.Close()
			f.request("POST", inputPath, test.reply, 404, nil).Body.Close()
		}
		finish(t, d, test.status)
		res.Body.Close()
	}
	content, err := os.ReadFile(filepath.Join(f.project, "approved.txt"))
	if err != nil || string(content) != "approved" {
		t.Fatal(string(content), err)
	}
	if _, err := os.Stat(filepath.Join(f.project, "cancelled.txt")); !os.IsNotExist(err) {
		t.Fatal("cancelled mutation ran", err)
	}
}

func TestServerAuthenticationProjectScopeAndRecovery(t *testing.T) {
	f := newFixture(t)
	f.request("GET", "/v1/sessions", "", 401, map[string]string{"Authorization": ""}).Body.Close()
	f.request("GET", "/v1/sessions", "", 403, map[string]string{"Host": "elsewhere.example"}).Body.Close()
	f.request("GET", "/v1/sessions", "", 403, map[string]string{"Origin": "https://elsewhere.example"}).Body.Close()
	foreign, err := session.New(f.config.SessionDir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foreign.Close()
	path := filepath.Join(f.config.SessionDir, foreign.ID+".jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("torn"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	id := f.create()
	f.create()
	res := f.request("GET", "/v1/sessions?limit=1", "", 200, nil)
	var page struct {
		Sessions []session.Info `json:"sessions"`
		Next     string         `json:"next_after"`
	}
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil || len(page.Sessions) != 1 || page.Sessions[0].Project != f.project || page.Next != page.Sessions[0].ID {
		t.Fatal(page, err)
	}
	res.Body.Close()
	f.request("POST", "/v1/sessions/"+foreign.ID+"/run", `{"prompt":"foreign"}`, 404, nil).Body.Close()
	body, err := os.ReadFile(path)
	if err != nil || !strings.HasSuffix(string(body), "torn") {
		t.Fatal("foreign journal was repaired", err)
	}

	store, err := session.Open(f.config.SessionDir, id)
	if err != nil {
		t.Fatal(err)
	}
	runPath := "/v1/sessions/" + id + "/run"
	f.request("POST", runPath, `{"prompt":"locked"}`, 409, nil).Body.Close()
	if err := store.Append("tool_start", map[string]string{"id": "uncertain", "name": "shell"}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	f.request("POST", runPath, `{"prompt":"blocked"}`, 409, nil).Body.Close()
	res = f.request("POST", runPath, `{"prompt":"continue","acknowledge_uncertain":true}`, 200, nil)
	finish(t, json.NewDecoder(res.Body), "completed")
	res.Body.Close()
}

func TestDisconnectCancelsPendingPermission(t *testing.T) {
	f := newFixture(t)
	id := f.create()
	res := f.request("POST", "/v1/sessions/"+id+"/run", `{"prompt":"cancel"}`, 200, nil)
	readInput(t, json.NewDecoder(res.Body), "permission")
	res.Body.Close()

	deadline := time.Now().Add(5 * time.Second)
	for {
		store, err := session.Open(f.config.SessionDir, id)
		if err == nil {
			defer store.Close()
			records := store.Records()
			last := records[len(records)-1]
			var terminal struct {
				Status string `json:"status"`
			}
			if last.Type != "run_finished" || json.Unmarshal(last.Data, &terminal) != nil || terminal.Status != "cancelled" || len(store.Pending()) != 0 {
				t.Fatal("disconnect left an unexpected journal state", last)
			}
			break
		}
		if !errors.Is(err, session.ErrBusy) || time.Now().After(deadline) {
			t.Fatal("server did not release the session", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(f.project, "cancelled.txt")); !os.IsNotExist(err) {
		t.Fatal("unapproved file was created", err)
	}
}
