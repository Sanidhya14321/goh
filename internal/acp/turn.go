package acp

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/mcpclient"
	"github.com/Sanidhya14321/goh/internal/provider"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/tools"
)

type turn struct {
	server    *server
	sessionID string
	cancel    context.CancelFunc

	// The runtime owns event delivery and tool tracking for the turn.
	calls     map[string]*toolCall
	candidate *toolCall
	sequence  int
	runID     string
	// streamed is the reply text sent to the editor since the last model
	// call began, to reconcile it with the final reply.
	streamed strings.Builder

	// Guarded by the server mutex. The reader delivers one reply.
	permission *permissionRequest
}

type toolCall struct {
	id, name, kind string
}

type permissionRequest struct {
	id    string
	reply chan string
}

func textBlock(text string) map[string]string {
	return map[string]string{"type": "text", "text": text}
}

func toolContent(text string) []any {
	return []any{map[string]any{"type": "content", "content": textBlock(text)}}
}

func (t *turn) update(update map[string]any) {
	t.server.send(map[string]any{
		"method": "session/update",
		"params": map[string]any{"sessionId": t.sessionID, "update": update},
	})
}

func toolKind(name string) string {
	switch name {
	case "read_file", "read_artifact":
		return "read"
	case "list_files", "glob", "grep", "repo_map":
		return "search"
	case "create_file", "write_file", "edit_file", "apply_patch":
		return "edit"
	case "shell":
		return "execute"
	case "webfetch":
		return "fetch"
	default:
		return "other"
	}
}

func (t *turn) newCall(name string, arguments json.RawMessage) *toolCall {
	t.sequence++
	call := &toolCall{
		id: t.runID + "-" + strconv.Itoa(t.sequence), name: name, kind: toolKind(name),
	}
	update := map[string]any{
		"sessionUpdate": "tool_call", "toolCallId": call.id, "title": name,
		"kind": call.kind, "status": "pending",
	}
	if json.Valid(arguments) {
		update["rawInput"] = arguments
	}
	t.update(update)
	return call
}

func (t *turn) callUpdate(call *toolCall, status, content string) {
	update := map[string]any{
		"sessionUpdate": "tool_call_update", "toolCallId": call.id, "status": status,
	}
	if content != "" {
		update["content"] = toolContent(content)
	}
	t.update(update)
}

func (t *turn) event(event core.Event) {
	data, _ := event.Data.(map[string]any)
	switch event.Type {
	case "run_started":
		t.runID = event.RunID
	case "assistant_updated":
		if text, ok := data["text"].(string); ok && text != "" {
			t.streamed.WriteString(text)
			t.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": textBlock(text)})
		}
	case "stream_retry":
		// ACP cannot take back text already sent, so say it was discarded.
		if t.streamed.Len() > 0 {
			t.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": textBlock("\n\n[The connection dropped; the partial reply above was discarded. Retrying.]\n\n")})
		}
		t.streamed.Reset()
	case "assistant_reset":
		if t.streamed.Len() > 0 {
			t.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": textBlock("\n\n[The final reply differs from the text above; it follows in full.]\n\n")})
		}
		t.streamed.Reset()
	case "assistant_completed":
		// Send any final text the stream did not deliver as deltas.
		if m, ok := data["message"].(core.Message); ok {
			if sent := t.streamed.String(); strings.HasPrefix(m.Content, sent) && len(m.Content) > len(sent) {
				t.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": textBlock(m.Content[len(sent):])})
			}
		}
		t.streamed.Reset()
	case "reasoning_updated":
		if text, ok := data["text"].(string); ok && text != "" {
			t.update(map[string]any{"sessionUpdate": "agent_thought_chunk", "content": textBlock(text)})
		}
	case "tool_requested":
		call, ok := event.Data.(core.Call)
		if ok {
			t.candidate = t.newCall(call.Function.Name, json.RawMessage(call.Function.Arguments))
			t.calls[call.ID] = t.candidate
		}
	case "tool_started":
		id, _ := data["id"].(string)
		call := t.calls[id]
		if call == nil {
			name, _ := data["name"].(string)
			args, _ := data["arguments"].(json.RawMessage)
			call = t.newCall(name, args)
			t.calls[id] = call
		}
		t.callUpdate(call, "in_progress", "")
	case "tool_finished":
		id, _ := data["id"].(string)
		if call := t.calls[id]; call != nil {
			result, _ := data["result"].(tools.Result)
			status := "completed"
			if result.Error {
				status = "failed"
			}
			t.callUpdate(call, status, result.Content)
			if t.candidate == call {
				t.candidate = nil
			}
			delete(t.calls, id)
		}
	}
}

func (t *turn) finishOpenCalls() {
	for _, call := range t.calls {
		t.callUpdate(call, "failed", "Tool did not complete. See the session journal for its last recorded state.")
	}
}

func (t *turn) approve(ctx context.Context, permission core.Permission) (core.Decision, error) {
	call := t.candidate
	synthetic := call == nil || call.name != permission.Tool
	if synthetic {
		call = t.newCall(permission.Tool+" permission", permission.Arguments)
	}
	request := &permissionRequest{id: "permission-" + agent.ID(), reply: make(chan string, 1)}
	s := t.server
	s.mu.Lock()
	t.permission = request
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		t.permission = nil
		s.mu.Unlock()
	}()
	tool := map[string]any{
		"toolCallId": call.id, "title": call.name, "kind": toolKind(permission.Tool),
		"status": "pending", "content": toolContent(permission.Preview),
	}
	if json.Valid(permission.Arguments) {
		tool["rawInput"] = permission.Arguments
	}
	s.send(map[string]any{
		"id": request.id, "method": "session/request_permission",
		"params": map[string]any{
			"sessionId": t.sessionID, "toolCall": tool,
			"options": []map[string]string{
				{"optionId": "allow-once", "name": "Allow once", "kind": "allow_once"},
				{"optionId": "reject-once", "name": "Reject", "kind": "reject_once"},
			},
		},
	})
	wait, stop := context.WithTimeout(ctx, 2*time.Minute)
	defer stop()
	var decision core.Decision
	var err error
	select {
	case <-wait.Done():
		err = wait.Err()
	case reply := <-request.reply:
		if reply == "cancelled" {
			t.cancel()
		}
		err = wait.Err()
		decision.Allow = reply == "allow-once" && err == nil
	}
	if synthetic {
		if decision.Allow {
			t.callUpdate(call, "completed", "Permission granted")
		} else {
			t.callUpdate(call, "failed", "Permission withheld")
		}
	}
	return decision, err
}

func (s *server) receivePermission(m message) {
	var id string
	if json.Unmarshal(m.ID, &id) != nil {
		return
	}
	var reply struct {
		Outcome struct {
			Type   string `json:"outcome"`
			Option string `json:"optionId"`
		} `json:"outcome"`
	}
	decision := "reject-once"
	if len(m.Error) == 0 && json.Unmarshal(m.Result, &reply) == nil {
		switch reply.Outcome.Type {
		case "selected":
			if reply.Outcome.Option == "allow-once" {
				decision = "allow-once"
			}
		case "cancelled":
			decision = "cancelled"
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil && s.active.permission != nil && s.active.permission.id == id {
		request := s.active.permission
		s.active.permission = nil
		request.reply <- decision
	}
}

func stopReason(status string, err error) (int, string) {
	switch status {
	case "completed":
		return 0, "end_turn"
	case "cancelled":
		return 0, "cancelled"
	case "budget_exhausted":
		if err != nil && err.Error() == "model turn limit reached" {
			return 0, "max_turn_requests"
		}
		return 0, "max_tokens"
	case "needs_input", "needs_permission":
		return 0, "refusal"
	}
	if errors.Is(err, mcpclient.ErrPermission) {
		return 0, "refusal"
	}
	var finish *provider.FinishError
	if errors.As(err, &finish) && finish.Reason == "content_filter" {
		return 0, "refusal"
	}
	if err != nil {
		return -32603, err.Error()
	}
	return -32603, "Prompt failed"
}
