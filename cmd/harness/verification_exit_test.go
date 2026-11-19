package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// scriptedEndpoint answers Chat Completions requests with the given tool
// calls in order (one call per reply), then with a final text reply.
func scriptedEndpoint(t *testing.T, calls ...[2]string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		mu.Lock()
		i := n
		n++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if i < len(calls) {
			args, _ := json.Marshal(calls[i][1])
			fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"c%d\",\"type\":\"function\",\"function\":{\"name\":%q,\"arguments\":%s}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n", i, calls[i][0], args)
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"All done.\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func lastRunFinished(t *testing.T, dir string) map[string]any {
	t.Helper()
	infos, err := session.List(dir)
	if err != nil || len(infos) != 1 {
		t.Fatal(infos, err)
	}
	s, err := session.Open(dir, infos[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var d map[string]any
	for _, r := range s.Records() {
		if r.Type == "run_finished" {
			json.Unmarshal(r.Data, &d)
		}
	}
	return d
}

// The required regression slice through the normal headless entry point:
// an edit followed only by pwd is unverified; a valid check verifies; a
// failing configured check with no repairs exits 6 and says so in text; a
// failing check repaired on the next turn exits 0 as verified.
func TestHeadlessVerificationSlice(t *testing.T) {
	testutil.RequireUnixShell(t)
	edit := [2]string{"create_file", `{"path":"note.txt","content":"beta\n"}`}
	for name, tc := range map[string]struct {
		calls    [][2]string
		verify   map[string]any
		wantExit int
		outcome  string
	}{
		"pwd only":      {[][2]string{edit, {"shell", `{"command":"pwd"}`}}, nil, 0, "unverified"},
		"valid check":   {[][2]string{edit, {"shell", `{"command":"grep -q beta note.txt"}`}}, nil, 0, "verified"},
		"failing check": {[][2]string{edit}, map[string]any{"command": "grep -q gamma note.txt", "repair_attempts": 0}, 6, "failed_verification"},
		"repaired": {[][2]string{edit, {"create_file", `{"path":"fixed.txt","content":"gamma\n"}`}},
			map[string]any{"command": "test -f fixed.txt", "repair_attempts": 1}, 0, "verified"},
	} {
		t.Run(name, func(t *testing.T) {
			// Each call is one reply; the "repaired" case's second edit is
			// the model's answer to the handed-back failure.
			srv := scriptedEndpoint(t, tc.calls...)
			cfg := map[string]any{"endpoint": srv.URL, "model": "m", "context_window": 131072}
			if tc.verify != nil {
				cfg["verify"] = tc.verify
			}
			config := filepath.Join(t.TempDir(), "config.json")
			b, _ := json.Marshal(cfg)
			os.WriteFile(config, b, 0o600)
			project, sessions := testutil.TempDir(t), t.TempDir()
			stderr := captureStderr(t, func() {
				err := run([]string{"run", "--config", config, "--project", project, "--session-dir", sessions, "--allow", "create_file,shell", "--prompt", "edit the note"})
				code := 0
				var x *exitError
				if errors.As(err, &x) {
					code = x.code
				} else if err != nil {
					t.Fatal(err)
				}
				if code != tc.wantExit {
					t.Fatalf("exit %d want %d (%v)", code, tc.wantExit, err)
				}
			})
			d := lastRunFinished(t, sessions)
			if d["outcome"] != tc.outcome || d["status"] != "completed" {
				t.Fatalf("journal %v", d)
			}
			if tc.wantExit == exitFailedVerification && !strings.Contains(stderr, "Verification failed: grep -q gamma note.txt (exit 1)") {
				t.Fatalf("text output does not show the failure:\n%s", stderr)
			}
			if tc.outcome == "unverified" && !strings.Contains(stderr, "no check (test or build command) succeeded") {
				t.Fatalf("missing unverified note:\n%s", stderr)
			}
		})
	}
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	defer func() { os.Stderr = old }()
	f()
	w.Close()
	os.Stderr = old
	return <-done
}
