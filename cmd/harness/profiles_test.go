package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIProfileSelectionAndExplicitModelOverride(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			http.Error(w, "bad request", 400)
			return
		}
		requests <- body.Model
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"fixture complete\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	config := filepath.Join(t.TempDir(), "config.json")
	data, err := json.Marshal(map[string]any{"profiles": map[string]any{"local": map[string]any{"endpoint": server.URL, "model": "profile-model", "context_window": 1048576}}, "agents": map[string]any{"reviewer": map[string]any{"mode": "plan", "profile": "local", "instructions": "Review carefully without changing files"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		flags []string
		model string
	}{{[]string{"--agent", "reviewer"}, "profile-model"}, {[]string{"--profile", "local", "--model", "cli-model"}, "cli-model"}} {
		args := []string{"run", "--config", config, "--project", t.TempDir(), "--session-dir", t.TempDir(), "--prompt", "Read-only fixture request", "--format", "json"}
		args = append(args, test.flags...)
		if err := run(args); err != nil {
			t.Fatal(err)
		}
		if actual := <-requests; actual != test.model {
			t.Fatalf("requested model %s expected %s", actual, test.model)
		}
	}
}
