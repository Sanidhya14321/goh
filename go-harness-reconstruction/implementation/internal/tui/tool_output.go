package tui

import (
	"encoding/json"
	"fmt"
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
		Command     string `json:"command"`
		Description string `json:"description"`
		Pattern     string `json:"pattern"`
		Query       string `json:"query"`
	}
	if json.Unmarshal(raw, &args) != nil {
		return string(raw)
	}
	var parts []string
	for _, value := range []string{args.Path, args.Command, args.Description, args.Pattern, args.Query} {
		if value != "" {
			parts = append(parts, strings.ReplaceAll(value, "\n", " "))
		}
	}
	if len(parts) == 0 {
		return string(raw)
	}
	return strings.Join(parts, " · ")
}

// Journals retain the full result. The transcript favors the actual file or
// command output over transport envelopes and internal artifact metadata.
func toolResultText(raw string) string {
	var result struct {
		Content  *string `json:"content"`
		Output   *string `json:"output"`
		Path     string  `json:"path"`
		Bytes    *int    `json:"bytes"`
		ExitCode *int    `json:"exit_code"`
	}
	if json.Unmarshal([]byte(raw), &result) != nil {
		return raw
	}
	switch {
	case result.Output != nil && result.ExitCode != nil:
		return fmt.Sprintf("exit %d\n%s", *result.ExitCode, *result.Output)
	case result.Content != nil:
		return result.Path + "\n" + *result.Content
	case result.Path != "" && result.Bytes != nil:
		return fmt.Sprintf("%s · %d bytes", result.Path, *result.Bytes)
	default:
		return raw
	}
}
