package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// Review R6: a file_rules deny must cover reads, search, listing and
// attachments, not only edits.
func TestFileRulesDenyCoversReadsSearchAndMentions(t *testing.T) {
	a, root := testAgent(t)
	a.Config.FileRules = []core.ShellRule{{Match: ".env", Action: "deny"}, {Match: "secrets/**", Action: "deny"}}
	a.Tools.DenyPath = func(rel string) bool { return a.Config.FileDecision([]string{rel}) == "deny" }
	os.WriteFile(filepath.Join(root, ".env"), []byte("API_TOKEN=synthetic-secret-1\n"), 0o600)
	os.MkdirAll(filepath.Join(root, "secrets"), 0o755)
	os.WriteFile(filepath.Join(root, "secrets", "key.txt"), []byte("synthetic-secret-2\n"), 0o600)
	os.WriteFile(filepath.Join(root, "main.go"), []byte("package main // API_TOKEN usage\n"), 0o644)
	var seen strings.Builder
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		for _, m := range h {
			seen.WriteString(m.Content)
		}
		if turn++; turn == 1 {
			return answer(
				toolCall("1", "read_file", map[string]any{"path": ".env"}),
				toolCall("2", "grep", map[string]any{"pattern": "synthetic-secret|API_TOKEN"}),
				toolCall("3", "glob", map[string]any{"pattern": "**/*"}),
				toolCall("4", "list_files", map[string]any{}),
				toolCall("5", "read_file", map[string]any{"path": "secrets/key.txt"}),
			), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "look at @.env and @secrets/key.txt"); status != "completed" {
		t.Fatal(status, err)
	}
	all := seen.String()
	if strings.Contains(all, "synthetic-secret") {
		t.Fatalf("denied content reached the model:\n%s", all)
	}
	if !strings.Contains(all, "main.go") {
		t.Fatal("allowed files disappeared from search results")
	}
	if strings.Contains(all, "secrets/key.txt\"") && strings.Contains(all, `"files"`) {
		// listing a denied path name is also blocked
		for _, line := range strings.Split(all, ",") {
			if strings.Contains(line, "secrets/key.txt") && !strings.Contains(line, "@secrets") && !strings.Contains(line, "path") {
				t.Fatalf("denied path listed: %s", line)
			}
		}
	}
}

// Review R7: no tool may create or change files inside .git (hooks there run
// code), whatever the grants or snapshot mode.
func TestWritesIntoGitMetadataAreRefused(t *testing.T) {
	a, root := testAgent(t)
	os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755)
	a.Grants["create_file"], a.Grants["write_file"], a.Grants["apply_patch"] = true, true, true
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			return answer(
				toolCall("1", "create_file", map[string]any{"path": ".git/hooks/pre-commit", "content": "#!/bin/sh\necho pwned\n"}),
				toolCall("2", "apply_patch", map[string]any{"patchText": "*** Begin Patch\n*** Add File: sub/.git/config\n+x\n*** End Patch"}),
			), nil
		}
		return answer(), nil
	})
	a.Run(context.Background(), "go")
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err == nil {
		t.Fatal("created a git hook")
	}
	if _, err := os.Stat(filepath.Join(root, "sub", ".git", "config")); err == nil {
		t.Fatal("patched nested .git metadata")
	}
}
