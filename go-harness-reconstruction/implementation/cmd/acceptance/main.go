// acceptance runs the built product against a credential-free local fixture.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"example.com/independent-harness/internal/core"
)

func main() {
	h := flag.String("harness", "bin/goh", "built GoHarness path")
	f := flag.String("fixture", "bin/scripted-model", "built fixture path")
	flag.Parse()
	if e := verify(*h, *f); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func verify(harness, fixture string) error {
	harness, e := filepath.Abs(harness)
	if e != nil {
		return e
	}
	fixture, e = filepath.Abs(fixture)
	if e != nil {
		return e
	}
	root, e := os.MkdirTemp("", "harness-acceptance-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(root)
	type report struct {
		Scenario string `json:"scenario"`
		Status   string `json:"status"`
		Events   int    `json:"events"`
		Exit     int    `json:"exit_code"`
	}
	var reports []report
	for _, scenario := range []string{"text", "multi", "edit", "deny", "drop", "malformed", "http-error"} {
		project := filepath.Join(root, scenario)
		if e = os.MkdirAll(project, 0700); e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(project, "note.txt"), []byte("alpha\n"), 0600); e != nil {
			return e
		}
		ln, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			return e
		}
		address := ln.Addr().String()
		ln.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		server := exec.CommandContext(ctx, fixture, "--listen", address, "--scenario", scenario)
		var logs bytes.Buffer
		server.Stdout = &logs
		server.Stderr = &logs
		if e = server.Start(); e != nil {
			cancel()
			return e
		}
		endpoint := "http://" + address + "/v1/chat/completions"
		client := http.Client{Timeout: time.Second}
		ready := false
		for i := 0; i < 100; i++ {
			r, x := client.Get(endpoint)
			if x == nil {
				r.Body.Close()
				ready = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !ready {
			cancel()
			server.Wait()
			return fmt.Errorf("fixture startup: %s", logs.String())
		}
		args := []string{"run", "--project", project, "--prompt", "Perform the fixture task", "--format", "json", "--endpoint", endpoint, "--model", "scripted", "--context-window", "131072", "--session-dir", filepath.Join(root, "sessions")}
		if scenario == "edit" {
			args = append(args, "--allow", "edit_file,shell")
		}
		cmd := exec.CommandContext(ctx, harness, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, runErr := cmd.Output()
		cancel()
		server.Wait()
		exit := 0
		if runErr != nil {
			if x, ok := runErr.(*exec.ExitError); ok {
				exit = x.ExitCode()
			} else {
				return runErr
			}
		}
		var events []core.Event
		decoder := json.NewDecoder(bytes.NewReader(out))
		for decoder.More() {
			var event core.Event
			if e = decoder.Decode(&event); e != nil {
				return fmt.Errorf("%s invalid JSON: %w", scenario, e)
			}
			events = append(events, event)
		}
		if len(events) == 0 {
			return fmt.Errorf("%s missing events: %s", scenario, stderr.String())
		}
		terminal := 0
		status := ""
		starts := map[string]bool{}
		for i, event := range events {
			if event.Version != 1 || event.Sequence != i+1 || event.SessionID == "" || event.RunID == "" {
				return fmt.Errorf("%s event envelope: %+v", scenario, event)
			}
			b, _ := json.Marshal(event.Data)
			var d struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			json.Unmarshal(b, &d)
			switch event.Type {
			case "tool_started":
				starts[d.ID] = true
			case "tool_finished":
				delete(starts, d.ID)
			case "run_finished":
				terminal++
				status = d.Status
			}
		}
		if terminal != 1 || events[len(events)-1].Type != "run_finished" || len(starts) != 0 {
			return fmt.Errorf("%s terminal/pairing error", scenario)
		}
		wantStatus, wantExit := "completed", 0
		switch scenario {
		case "deny":
			wantStatus = "needs_permission"
			wantExit = 3
		case "drop", "malformed", "http-error":
			wantStatus = "failed"
			wantExit = 1
		}
		if status != wantStatus || exit != wantExit {
			return fmt.Errorf("%s: got %s/%d want %s/%d (%s)", scenario, status, exit, wantStatus, wantExit, stderr.String())
		}
		b, e := os.ReadFile(filepath.Join(project, "note.txt"))
		if e != nil {
			return e
		}
		want := "alpha\n"
		if scenario == "edit" {
			want = "beta\n"
		}
		if string(b) != want {
			return fmt.Errorf("%s unexpected file effects: %q", scenario, b)
		}
		if scenario == "drop" && strings.Contains(string(out), `"type":"tool_started"`) {
			return fmt.Errorf("interrupted reply executed tool")
		}
		reports = append(reports, report{scenario, status, len(events), exit})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(reports)
}
