package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/memory"
)

// Review R18: a recalled note is rechecked before every replay; once its
// evidence changes it is withdrawn from what the model sees.
func TestRecalledNotesAreRevalidatedOnReplay(t *testing.T) {
	a, root := testAgent(t)
	a.Config.Memory = true
	os.WriteFile(filepath.Join(root, "config.go"), []byte("package x // port 8080\n"), 0o644)
	ledger, err := memory.New(a.Config.SessionDir, a.Store.Project)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Add("NOTE-777 the server listens on port 8080", []string{"config.go"}); err != nil {
		t.Fatal(err)
	}
	turn := 0
	var third string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			return answer(toolCall("m", "memory_recall", map[string]any{"query": "server port"})), nil
		case 2:
			if !strings.Contains(h[len(h)-1].Content, "NOTE-777") {
				t.Fatalf("note not recalled: %s", h[len(h)-1].Content)
			}
			os.WriteFile(filepath.Join(root, "config.go"), []byte("package x // port 9090\n"), 0o644)
			return answer(toolCall("l", "list_files", map[string]any{})), nil
		}
		for _, m := range h {
			third += m.Content
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "what port?"); status != "completed" {
		t.Fatal(status, err)
	}
	if strings.Contains(third, "NOTE-777") || !strings.Contains(third, "withdrawn") {
		t.Fatalf("stale note replayed: %.400s", third)
	}
}
