package tui

import (
	"encoding/json"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/core"
	"io"
	"strings"
)

func historyToolText(raw string) string {
	var envelope struct {
		Content *string
		Error   bool
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) == nil && envelope.Content != nil && decoder.Decode(new(any)) == io.EOF {
		mark := "✓ "
		if envelope.Error {
			mark = "[error] "
		}
		return mark + toolResultText(*envelope.Content)
	}
	return "[result] " + toolResultText(raw)
}

func toolArgumentText(raw json.RawMessage) string {
	var args struct {
		Path        string `json:"path"`
		FilePath    string `json:"filePath"`
		Operation   string `json:"operation"`
		URL         string `json:"url"`
		Name        string `json:"name"`
		Server      string `json:"server"`
		URI         string `json:"uri"`
		Command     string `json:"command"`
		Description string `json:"description"`
		Pattern     string `json:"pattern"`
		Query       string `json:"query"`
		PatchText   string `json:"patchText"`
		ID          string `json:"id"`
		Offset      int    `json:"offset"`
		Limit       int    `json:"limit"`
		Tail        bool   `json:"tail"`
		Todos       []struct {
			Status string `json:"status"`
		} `json:"todos"`
	}
	if json.Unmarshal(raw, &args) != nil {
		return string(raw)
	}
	if args.PatchText != "" {
		// "update a.go, add b.go" from the patch's file headers.
		var files []string
		for _, line := range strings.Split(args.PatchText, "\n") {
			for _, kind := range []string{"Add", "Update", "Delete"} {
				if path, ok := strings.CutPrefix(line, "*** "+kind+" File: "); ok {
					files = append(files, strings.ToLower(kind)+" "+strings.TrimSpace(path))
				}
			}
		}
		if len(files) > 0 {
			return strings.Join(files, ", ")
		}
	}
	if args.Todos != nil {
		done := 0
		for _, todo := range args.Todos {
			if todo.Status == "completed" {
				done++
			}
		}
		return fmt.Sprintf("%d tasks, %d done", len(args.Todos), done)
	}
	var parts []string
	for _, value := range []string{args.Operation, args.Path, args.FilePath, args.URL, args.Name, args.Server, args.URI, args.Command, args.Description, args.Pattern, args.Query} {
		if value != "" {
			parts = append(parts, strings.ReplaceAll(value, "\n", " "))
		}
	}
	if len(parts) > 0 && args.ID != "" {
		parts = append([]string{args.ID}, parts...) // read_artifact with pattern
	}
	if len(parts) == 0 && args.ID != "" {
		// read_artifact and job tools: "fetch-1.log · bytes 4000–8000".
		switch {
		case args.Tail:
			return args.ID + " · end"
		case args.Limit > 0:
			return fmt.Sprintf("%s · bytes %d–%d", args.ID, args.Offset, args.Offset+args.Limit)
		case args.Offset > 0:
			return fmt.Sprintf("%s · from byte %d", args.ID, args.Offset)
		}
		return args.ID
	}
	if len(parts) == 0 {
		return string(raw)
	}
	return strings.Join(parts, " · ")
}

// Journals retain the full result. The transcript favors the actual file or
// command output over transport envelopes and internal artifact metadata.
func toolResultText(raw string) string {
	if listed, ok := listingText(raw); ok {
		return listed
	}
	var result struct {
		Content  *string `json:"content"`
		Output   *string `json:"output"`
		Path     string  `json:"path"`
		Bytes    *int    `json:"bytes"`
		ExitCode *int    `json:"exit_code"`
		URL      string  `json:"url"`
		Status   int     `json:"status_code"`
		// Shell and formatter output: the retained artifact and how much of
		// the output the inline text shows.
		ArtifactID       string `json:"artifact_id"`
		OutputBytes      int64  `json:"output_bytes"`
		RetainedBytes    int64  `json:"retained_bytes"`
		PreviewTruncated bool   `json:"preview_truncated"`
	}
	if json.Unmarshal([]byte(raw), &result) != nil {
		return raw
	}
	switch {
	case result.Output != nil && result.ExitCode != nil:
		head := fmt.Sprintf("exit %d", *result.ExitCode)
		if result.PreviewTruncated && result.ArtifactID != "" {
			// On the first line, so no display cap can cut the pointer off.
			head += fmt.Sprintf(" · %s of output, preview below; retained %s as artifact %s (ask to read_artifact it)", byteSize(result.OutputBytes), byteSize(result.RetainedBytes), result.ArtifactID)
		}
		return head + "\n" + *result.Output
	case result.Content != nil && result.URL != "":
		return fmt.Sprintf("%s · HTTP %d", result.URL, result.Status) + "\n" + *result.Content
	case result.Content != nil:
		return result.Path + "\n" + *result.Content
	case result.Path != "" && result.Bytes != nil:
		return fmt.Sprintf("%s · %d bytes", result.Path, *result.Bytes)
	default:
		return raw
	}
}

// listingText shows search and listing results as plain lines (path:line:
// text, or one path per line) instead of their JSON envelope.
func listingText(raw string) (string, bool) {
	var result struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
			Text string `json:"text"`
		} `json:"matches"`
		Files     []string `json:"files"`
		Truncated bool     `json:"truncated"`
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") || json.Unmarshal([]byte(raw), &result) != nil {
		return "", false
	}
	var lines []string
	switch {
	case result.Matches != nil:
		if len(result.Matches) == 0 {
			return "no matches", true
		}
		lines = append(lines, plural(len(result.Matches), "match", "matches"))
		for _, match := range result.Matches {
			lines = append(lines, fmt.Sprintf("%s:%d: %s", match.Path, match.Line, strings.TrimSpace(match.Text)))
		}
	case result.Files != nil:
		if len(result.Files) == 0 {
			return "no files", true
		}
		lines = append(lines, plural(len(result.Files), "file", "files"))
		lines = append(lines, result.Files...)
	default:
		return "", false
	}
	if result.Truncated {
		lines = append(lines, "[truncated]")
	}
	return strings.Join(lines, "\n"), true
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// ToolLine is a one-line description of a tool call ("read_file  a.go") for
// headless text output.
func ToolLine(name string, arguments json.RawMessage) string {
	line := Safe(name + "  " + strings.ReplaceAll(toolArgumentText(arguments), "\n", " "))
	if len(line) > 160 {
		line = strings.ToValidUTF8(line[:160], "") + "…"
	}
	return line
}

// childEventLine shows a subagent's tool activity as what it did ("grep
// TODO"), not just that an event happened.
func childEventLine(event core.Event) string {
	b, _ := json.Marshal(event.Data)
	switch event.Type {
	case "tool_started":
		var tool struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		json.Unmarshal(b, &tool)
		return "\n  ↳ subagent ▸ " + ToolLine(tool.Name, tool.Arguments) + "\n"
	case "tool_finished":
		var tool struct {
			Result struct {
				Error   bool   `json:"error"`
				Content string `json:"content"`
			} `json:"result"`
		}
		json.Unmarshal(b, &tool)
		if !tool.Result.Error {
			return ""
		}
		first, _, _ := strings.Cut(tool.Result.Content, "\n")
		return "  ↳ subagent [error] " + Safe(oneLine(first, 120)) + "\n"
	default:
		return "  ↳ subagent [error] " + Safe(oneLine(string(b), 120)) + "\n"
	}
}

// StreamRetryText explains a re-sent model request: a broken stream, or a
// reply the provider refused while generating it (an invalid tool call).
func StreamRetryText(data any) string {
	b, _ := json.Marshal(data)
	var retry struct {
		Error string `json:"error"`
		Busy  bool   `json:"busy"`
	}
	json.Unmarshal(b, &retry)
	if retry.Busy {
		return "The provider was overloaded; nothing ran, the request was sent again"
	}
	if strings.Contains(retry.Error, "rejected the model's output") {
		return "The provider rejected the model's tool call; nothing ran, asking the model again"
	}
	return "Stream broke; the partial reply was discarded and the request re-sent"
}

// RequestCapText explains a request resent after the provider stated a
// smaller per-request token cap than the configured context window.
func RequestCapText(data any) string {
	b, _ := json.Marshal(data)
	var learned struct {
		Limit int `json:"limit"`
	}
	json.Unmarshal(b, &learned)
	return fmt.Sprintf("The provider accepts at most %d tokens per request; fitting the context to that and resending", learned.Limit)
}

func byteSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}
