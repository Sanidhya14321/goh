package runtime

import (
	"context"
	"example.com/independent-harness/internal/core"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputArtifactRetrievalAndRepoMapAcrossRun(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "main.go"), []byte("package example\nfunc Entry() string { return \"hidden body\" }\n"), 0600)
	a.Grants["shell"] = true
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			return answer(toolCall("shell", "shell", map[string]any{"command": "printf 'command output'", "timeout_seconds": 3})), nil
		case 2:
			r := toolResult(t, h[len(h)-1])
			return answer(toolCall("artifact", "read_artifact", map[string]any{"id": r["artifact_id"]}), toolCall("map", "repo_map", map[string]any{"query": "Entry"})), nil
		default:
			page := toolResult(t, h[len(h)-2])
			mapping := toolResult(t, h[len(h)-1])
			if page["content"] != "command output" || len(mapping["symbols"].([]any)) != 1 {
				t.Fatal(page, mapping)
			}
			return answer(), nil
		}
	})
	status, err := a.Run(context.Background(), "inspect output and Go structure")
	if err != nil || status != "completed" {
		t.Fatal(status, err)
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal(a.Store.Pending())
	}
	a.Config.AgentMode = "plan"
	for _, name := range []string{"repo_map", "read_artifact"} {
		if err = a.checkMode(name); err != nil {
			t.Fatal(err)
		}
	}
}
