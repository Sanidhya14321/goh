package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// A resumed session runs in its own project: --file reads from there, and a
// different --project is refused instead of mixing two projects in one run.
func TestResumedSessionKeepsItsProject(t *testing.T) {
	bodies := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies <- string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	config := filepath.Join(t.TempDir(), "config.json")
	data, _ := json.Marshal(map[string]any{"endpoint": server.URL, "model": "m", "context_window": 131072})
	os.WriteFile(config, data, 0o600)
	a, b, sessions := testutil.TempDir(t), testutil.TempDir(t), t.TempDir()
	os.WriteFile(filepath.Join(a, "target.txt"), []byte("FROM A\n"), 0o600)
	os.WriteFile(filepath.Join(b, "target.txt"), []byte("FROM B\n"), 0o600)
	store, err := session.New(sessions, a)
	if err != nil {
		t.Fatal(err)
	}
	id := store.ID
	store.Close()
	base := []string{"run", "--config", config, "--session-dir", sessions, "--session", id, "--prompt", "look", "--file", "target.txt", "--format", "json"}
	err = run(append(base, "--project", b))
	if err == nil || !strings.Contains(err.Error(), "belongs to project") {
		t.Fatalf("mixed projects accepted: %v", err)
	}
	// Without --project the session's own project supplies the file.
	wd, _ := os.Getwd()
	os.Chdir(b)
	defer os.Chdir(wd)
	if err := run(base); err != nil {
		t.Fatal(err)
	}
	if body := <-bodies; !strings.Contains(body, "FROM A") || strings.Contains(body, "FROM B") {
		t.Fatalf("attachment from the wrong project: %s", body)
	}
	if err := run(append(base, "--project", a)); err != nil {
		t.Fatal("the session's own project refused:", err)
	}
}
