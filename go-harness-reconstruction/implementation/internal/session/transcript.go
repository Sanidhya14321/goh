package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"example.com/independent-harness/internal/core"
)

// WriteTranscript exports the saved conversation, including prompts that
// context compaction removed from the active model context.
func (s *Store) WriteTranscript(out io.Writer, includeTools bool) error {
	s.mu.Lock()
	records := append([]Record(nil), s.records...)
	s.mu.Unlock()

	w := transcriptWriter{out: bufio.NewWriter(out)}
	w.add("# GoHarness session\n\nSession ID: `", s.ID, "`\n\n")
	for i, record := range records {
		if w.err != nil {
			return w.err
		}
		if record.Type != "message" {
			continue
		}
		var message core.Message
		if err := json.Unmarshal(record.Data, &message); err != nil {
			return fmt.Errorf("transcript message %d: %w", i+1, err)
		}
		switch message.Role {
		case "user":
			w.add("## User\n\n", message.Content, "\n\n")
		case "assistant":
			if message.Content == "" && (!includeTools || len(message.ToolCalls) == 0) {
				continue
			}
			w.add("## Assistant\n\n")
			if message.Content != "" {
				w.add(message.Content, "\n\n")
			}
			if includeTools {
				for _, call := range message.ToolCalls {
					w.add("### Tool call\n\n")
					w.code(call.Function.Name)
					w.code(call.Function.Arguments)
				}
			}
		case "tool":
			if includeTools {
				w.add("### Tool result\n\n")
				w.code(message.Content)
			}
		}
	}
	if w.err != nil {
		return w.err
	}
	return w.out.Flush()
}

type transcriptWriter struct {
	out  *bufio.Writer
	size int
	err  error
}

func (w *transcriptWriter) add(parts ...string) {
	for _, part := range parts {
		if w.err != nil {
			return
		}
		if len(part) > maxExportBytes-w.size {
			w.err = errors.New("Markdown transcript exceeds 64 MiB limit")
			return
		}
		var n int
		n, w.err = w.out.WriteString(part)
		w.size += n
	}
}

func (w *transcriptWriter) code(text string) {
	// Indentation keeps literal fence markers in tool output inside the block.
	for w.err == nil {
		line, rest, more := strings.Cut(text, "\n")
		w.add("    ", line, "\n")
		if !more {
			break
		}
		text = rest
	}
	w.add("\n")
}
