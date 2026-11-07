package lspclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// maxEditErrors caps errors reported for one edited file.
const maxEditErrors = 20

// editErrorTimeout bounds how long an edit waits for diagnostics.
const editErrorTimeout = 3 * time.Second

// EditErrors returns error-severity diagnostics for a file the agent just
// edited, as "line:col message" strings. It only uses a language server that
// is already running in this run: it never starts one, so the lsp_start
// approval boundary is unchanged. ok is false when no running server covers
// the file or diagnostics could not be read; edits never fail because of it.
func (m *Manager) EditErrors(ctx context.Context, path string) (errs []string, more int, ok bool) {
	name, _, err := m.selectServer(path)
	if err != nil {
		return nil, 0, false
	}
	m.mu.Lock()
	s := m.servers[name]
	closed := m.closed
	m.mu.Unlock()
	if closed || s == nil || s.broken {
		return nil, 0, false
	}
	ctx, cancel := context.WithTimeout(ctx, editErrorTimeout)
	defer cancel()
	input, _ := json.Marshal(map[string]any{"operation": "diagnostics", "filePath": path, "line": 1, "character": 1})
	r := m.Execute(ctx, "lsp", input)
	if r.Error {
		return nil, 0, false
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal([]byte(r.Content), &envelope) != nil {
		return nil, 0, false
	}
	items := diagnosticItems(envelope.Result)
	for _, d := range items {
		if d.Severity != 1 {
			continue
		}
		if len(errs) == maxEditErrors {
			more++
			continue
		}
		errs = append(errs, fmt.Sprintf("%d:%d %s", d.Range.Start.Line+1, d.Range.Start.Character+1, d.Message))
	}
	return errs, more, true
}

type diagnostic struct {
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Range    struct {
		Start struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
	} `json:"range"`
}

// diagnosticItems accepts both a pull report ({"items": [...]}) and the
// cached push form (a plain array).
func diagnosticItems(raw json.RawMessage) []diagnostic {
	var report struct {
		Items []diagnostic `json:"items"`
	}
	if json.Unmarshal(raw, &report) == nil && report.Items != nil {
		return report.Items
	}
	var list []diagnostic
	json.Unmarshal(raw, &list)
	return list
}
