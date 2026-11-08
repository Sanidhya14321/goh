// scripted-model is an independently authored local endpoint for deterministic tests.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"example.com/independent-harness/internal/core"
)

type step struct {
	Status  int         `json:"status"`
	Text    string      `json:"text"`
	Calls   []core.Call `json:"calls"`
	Raw     string      `json:"raw"`
	DelayMS int         `json:"delay_ms"`
	Drop    bool        `json:"drop"`
	Usage   bool        `json:"usage"`
}

var scriptSteps []step
var scriptIndex atomic.Int64

func main() {
	address := flag.String("listen", "127.0.0.1:18766", "local listen address")
	scenario := flag.String("scenario", "text", "text, edit, deny, shell, drop, malformed, http-error")
	script := flag.String("script", "", "JSON file containing an array of response steps")
	flag.Parse()
	if *script != "" {
		b, e := os.ReadFile(*script)
		if e != nil {
			log.Fatal(e)
		}
		if e = json.Unmarshal(b, &scriptSteps); e != nil {
			log.Fatal(e)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) { serve(w, r, *scenario) })
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"scripted"},{"id":"scripted-large"}]}`))
	})
	server := &http.Server{Addr: *address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("scripted endpoint http://%s/v1/chat/completions scenario=%s", *address, *scenario)
	log.Fatal(server.ListenAndServe())
}
func serve(w http.ResponseWriter, r *http.Request, scenario string) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	var request struct {
		Messages []core.Message `json:"messages"`
	}
	if e := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&request); e != nil {
		http.Error(w, "bad fixture request", 400)
		return
	}
	count := 0
	for _, m := range request.Messages {
		if m.Role == "tool" {
			count++
		}
	}
	s := step{Text: "Fixture response: ready ✓", Usage: true}
	call := func(name string, args any) {
		b, _ := json.Marshal(args)
		c := core.Call{ID: fmt.Sprintf("fixture_%d", count), Type: "function"}
		c.Function.Name = name
		c.Function.Arguments = string(b)
		s.Text = ""
		s.Calls = []core.Call{c}
	}
	switch scenario {
	case "edit", "deny":
		switch count {
		case 0:
			call("read_file", map[string]any{"path": "note.txt"})
		case 1:
			hash := ""
			for _, m := range request.Messages {
				if m.Role == "tool" {
					var envelope struct {
						Content string `json:"content"`
					}
					json.Unmarshal([]byte(m.Content), &envelope)
					var read struct {
						SHA string `json:"sha256"`
					}
					json.Unmarshal([]byte(envelope.Content), &read)
					if read.SHA != "" {
						hash = read.SHA
					}
				}
			}
			call("edit_file", map[string]any{"path": "note.txt", "expected_sha256": hash, "old_text": "alpha", "new_text": "beta"})
		case 2:
			if scenario == "edit" {
				call("shell", map[string]any{"command": "test \"$(cat note.txt)\" = beta", "timeout_seconds": 5})
			}
		default:
			s.Text = "Fixture edit and verification complete"
		}
	case "job":
		if count == 0 {
			call("job_start", map[string]any{"command": "sleep 300 & echo $! > job.pid; echo job-running; wait"})
		} else {
			s.Text = "Background job started"
		}
	case "shell":
		if count == 0 {
			call("shell", map[string]any{"command": "printf fixture-shell", "timeout_seconds": 5})
		}
	case "multi":
		if count == 0 {
			call("read_file", map[string]any{"path": "note.txt"})
			c := core.Call{ID: "fixture_second", Type: "function"}
			c.Function.Name = "list_files"
			c.Function.Arguments = `{}`
			s.Calls = append(s.Calls, c)
		}
	case "delay":
		s.DelayMS = 1500
	case "missing-usage":
		s.Usage = false
	case "drop":
		s.Text = "partial"
		s.Drop = true
	case "malformed":
		s.Raw = "data: {broken}\n\n"
	case "http-error":
		s.Status = 503
	}
	if len(scriptSteps) > 0 {
		i := int(scriptIndex.Add(1) - 1)
		if i < len(scriptSteps) {
			s = scriptSteps[i]
		}
	}
	if s.DelayMS > 0 {
		timer := time.NewTimer(time.Duration(s.DelayMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-r.Context().Done():
			return
		}
	}
	if s.Status != 0 {
		http.Error(w, "fixture failure", s.Status)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	send := func(v any) {
		b, _ := json.Marshal(v)
		frame := "data: " + string(b) + "\n\n"
		for len(frame) > 0 {
			n := 3
			if len(frame) < n {
				n = len(frame)
			}
			io.WriteString(w, frame[:n])
			frame = frame[n:]
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
	chunk := func(delta any, finish any) any {
		return map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}
	}
	if s.Raw != "" {
		io.WriteString(w, s.Raw)
		return
	}
	for _, word := range strings.Split(s.Text, " ") {
		if word != "" {
			send(chunk(map[string]any{"content": word + " "}, nil))
		}
	}
	for i, c := range s.Calls {
		args := c.Function.Arguments
		mid := len(args) / 2
		send(chunk(map[string]any{"tool_calls": []any{map[string]any{"index": i, "id": c.ID, "type": "function", "function": map[string]any{"name": c.Function.Name, "arguments": args[:mid]}}}}, nil))
		send(chunk(map[string]any{"tool_calls": []any{map[string]any{"index": i, "function": map[string]any{"arguments": args[mid:]}}}}, nil))
	}
	if s.Drop {
		return
	}
	reason := "stop"
	if len(s.Calls) > 0 {
		reason = "tool_calls"
	}
	send(chunk(map[string]any{}, reason))
	if s.Usage {
		send(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})
	}
	io.WriteString(w, "data: [DONE]\n\n")
}
