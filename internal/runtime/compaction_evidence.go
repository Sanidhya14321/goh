package runtime

import (
	"encoding/hex"
	"encoding/json"
	"path/filepath"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// compactionEvidence extracts a bounded set of observations from discarded
// turns. It preserves identities and failure codes without replaying raw output.
func compactionEvidence(messages []core.Message) []session.CompactionEvidence {
	messages = messages[max(0, len(messages)-128):]
	results := make(map[string]string)
	seenPaths := make(map[string]bool)
	var evidence []session.CompactionEvidence
	failedShell := false
	addFile := func(tool, path, digest string) {
		if len(evidence) >= 6 || len(path) > 192 || !filepath.IsLocal(path) || seenPaths[path] {
			return
		}
		seenPaths[path] = true
		hash, err := hex.DecodeString(digest)
		if err == nil && len(hash) == 32 {
			evidence = append(evidence, session.CompactionEvidence{Tool: tool, Path: path, FileSHA256: digest})
		}
	}
	for i := len(messages) - 1; i >= 0 && len(evidence) < 6; i-- {
		message := messages[i]
		if message.Role == "tool" {
			results[message.ToolCallID] = message.Content
			continue
		}
		for j := len(message.ToolCalls) - 1; j >= 0 && len(evidence) < 6; j-- {
			call := message.ToolCalls[j]
			raw, ok := results[call.ID]
			delete(results, call.ID)
			if !ok {
				continue
			}
			switch call.Function.Name {
			case "shell", "read_file", "write_file", "edit_file", "create_file", "apply_patch":
			default:
				continue
			}
			var result tools.Result
			if json.Unmarshal([]byte(raw), &result) != nil {
				continue
			}
			type file struct {
				Path        string `json:"path"`
				SHA256      string `json:"sha256"`
				Destination string `json:"destination"`
			}
			var metadata struct {
				Path       string `json:"path"`
				SHA256     string `json:"sha256"`
				ExitCode   *int   `json:"exit_code"`
				Files      []file `json:"files"`
				FinalFiles []file `json:"final_files"`
			}
			if json.Unmarshal([]byte(result.Content), &metadata) != nil {
				continue
			}
			if call.Function.Name == "shell" {
				if !failedShell && metadata.ExitCode != nil && *metadata.ExitCode != 0 {
					evidence = append(evidence, session.CompactionEvidence{Tool: "shell", ExitCode: metadata.ExitCode})
					failedShell = true
				}
				continue
			}
			if result.Error {
				continue
			}
			if call.Function.Name == "apply_patch" {
				for _, item := range metadata.Files {
					if item.Destination != "" || item.SHA256 == "" {
						seenPaths[item.Path] = true
					}
				}
				files := metadata.Files
				if len(metadata.FinalFiles) > 0 {
					files = metadata.FinalFiles
				}
				for _, item := range files {
					path := item.Path
					if item.Destination != "" {
						path = item.Destination
					}
					addFile(call.Function.Name, path, item.SHA256)
				}
			} else {
				addFile(call.Function.Name, metadata.Path, metadata.SHA256)
			}
		}
	}
	return evidence
}
