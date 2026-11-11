// Package testutil contains independently authored protocol fixtures. It is not
// imported by the harness executable and does not reuse the client frame parser.
package testutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func RunLSP() {
	reader := bufio.NewReader(os.Stdin)
	mode := os.Getenv("LSP_FIXTURE_MODE")
	var writeMu sync.Mutex
	write := func(v any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		b, _ := json.Marshal(v)
		fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n", len(b))
		for len(b) > 0 {
			n := 7
			if len(b) < n {
				n = len(b)
			}
			os.Stdout.Write(b[:n])
			b = b[n:]
		}
	}
	for {
		size := 0
		for {
			line, e := reader.ReadString('\n')
			if e != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			name, value, ok := strings.Cut(line, ":")
			if ok && strings.EqualFold(name, "Content-Length") {
				size, _ = strconv.Atoi(strings.TrimSpace(value))
			}
		}
		if size < 1 || size > 2<<20 {
			return
		}
		b := make([]byte, size)
		if _, e := io.ReadFull(reader, b); e != nil {
			return
		}
		var q struct {
			ID     any            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
			Error  any            `json:"error"`
		}
		if json.Unmarshal(b, &q) != nil {
			return
		}
		if log := os.Getenv("LSP_FIXTURE_LOG"); log != "" {
			f, e := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e == nil {
				f.Write(append(b, '\n'))
				f.Close()
			}
		}
		if q.Method == "exit" {
			return
		}
		if q.Method == "textDocument/didOpen" || q.Method == "textDocument/didChange" {
			doc, _ := q.Params["textDocument"].(map[string]any)
			text, _ := doc["text"].(string)
			if changes, ok := q.Params["contentChanges"].([]any); ok && len(changes) > 0 {
				if c, ok := changes[len(changes)-1].(map[string]any); ok {
					text, _ = c["text"].(string)
				}
			}
			diagnostics := []any{map[string]any{"range": map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": 1}}, "severity": 2, "message": "Fixture diagnostic"}}
			// Documents containing BROKEN also get an error, for edit-feedback tests.
			if strings.Contains(text, "BROKEN") {
				diagnostics = append(diagnostics, map[string]any{"range": map[string]any{"start": map[string]int{"line": 1, "character": 4}, "end": map[string]int{"line": 1, "character": 5}}, "severity": 1, "message": "Fixture error: BROKEN"})
			}
			publish := map[string]any{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": map[string]any{"uri": doc["uri"], "diagnostics": diagnostics}}
			switch mode {
			case "silent-diagnostics":
			case "delayed-diagnostics": // like a server that analyses in the background
				go func() { time.Sleep(300 * time.Millisecond); write(publish) }()
			default:
				write(publish)
			}
			continue
		}
		if q.ID == nil {
			continue
		}
		if q.Method == "initialize" {
			if mode == "bad-initialize" {
				write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": map[string]any{}})
				continue
			}
			caps := map[string]any{"typeDefinitionProvider": true, "callHierarchyProvider": true, "hoverProvider": true, "definitionProvider": true, "referencesProvider": true, "documentSymbolProvider": true, "workspaceSymbolProvider": true, "implementationProvider": true, "positionEncoding": "utf-16"}
			if mode == "bad-encoding" {
				caps["positionEncoding"] = "utf-8"
			}
			write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": map[string]any{"capabilities": caps}})
			continue
		}
		if q.Method == "shutdown" {
			write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": nil})
			continue
		}
		if mode == "disconnect" {
			return
		}
		if mode == "hang" {
			time.Sleep(time.Minute)
			return
		}
		if mode == "malformed" {
			fmt.Fprint(os.Stdout, "Content-Length: 8\r\n\r\nnot-json")
			return
		}
		if mode == "request-edit" {
			write(map[string]any{"jsonrpc": "2.0", "id": "server-edit", "method": "workspace/applyEdit", "params": map[string]any{"edit": map[string]any{"changes": map[string]any{"file:///forbidden": []any{}}}}})
		}
		var result any
		switch q.Method {
		case "textDocument/hover":
			position := q.Params["position"]
			result = map[string]any{"contents": map[string]any{"kind": "markdown", "value": "Fixture hover"}, "fixture_position": position}
			if mode == "large" {
				result = map[string]any{"contents": strings.Repeat("X", 70<<10)}
			}
		case "textDocument/documentSymbol", "workspace/symbol":
			result = []any{map[string]any{"name": "FixtureSymbol", "kind": 12}}
		case "textDocument/prepareCallHierarchy":
			doc, _ := q.Params["textDocument"].(map[string]any)
			result = []any{map[string]any{"name": "FixtureFunc", "kind": 12, "uri": doc["uri"], "range": map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": 1}}, "selectionRange": map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": 1}}}}
		case "callHierarchy/incomingCalls", "callHierarchy/outgoingCalls":
			item, _ := q.Params["item"].(map[string]any)
			key := "from"
			if q.Method == "callHierarchy/outgoingCalls" {
				key = "to"
			}
			result = []any{map[string]any{key: map[string]any{"name": "Caller of " + fmt.Sprint(item["name"]), "kind": 12}, "fromRanges": []any{}}}
		case "textDocument/definition", "textDocument/references", "textDocument/implementation", "textDocument/typeDefinition":
			doc, _ := q.Params["textDocument"].(map[string]any)
			result = []any{map[string]any{"uri": doc["uri"], "range": map[string]any{"start": map[string]int{"line": 0, "character": 0}, "end": map[string]int{"line": 0, "character": 1}}}}
		default:
			write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "error": map[string]any{"code": -32601, "message": "Fixture method unsupported"}})
			continue
		}
		write(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
	}
}
