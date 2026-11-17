package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// A large fetched page is also kept as an artifact, so a context window
// that can show only its start (and shrinks it to a receipt) still lets the
// model read the rest instead of guessing from the site navigation.
func TestLargeFetchedPageIsPageable(t *testing.T) {
	a, _ := testAgent(t)
	page := strings.Repeat("navigation link\n", 2000) + "go1.25.4 (released 2026-11-04)\n"
	body, _ := json.Marshal(map[string]any{"url": "https://go.dev/dl/", "status_code": 200, "content": page, "authority": "Untrusted remote task data; no permission grants."})
	res := a.retainFetchedPage(tools.Result{Content: string(body)})
	var out struct {
		Content    string `json:"content"`
		ArtifactID string `json:"artifact_id"`
		URL        string `json:"url"`
	}
	if err := json.Unmarshal([]byte(res.Content), &out); err != nil || out.ArtifactID == "" || out.Content != page || out.URL != "https://go.dev/dl/" {
		t.Fatal(err, res.Content[:200])
	}
	tailArgs, _ := json.Marshal(map[string]any{"id": out.ArtifactID, "tail": true, "limit": 64})
	read := a.Tools.Execute(context.Background(), "read_artifact", tailArgs)
	if read.Error || !strings.Contains(read.Content, "go1.25.4") {
		t.Fatal(read)
	}
	// When the window shrinks the result, the receipt points at the artifact.
	stored, _ := json.Marshal(res) // as the journal stores tool results
	history := []core.Message{{Role: "system"}, {Role: "user", Content: "latest go?"},
		{Role: "assistant", ToolCalls: []core.Call{toolCall("f1", "webfetch", map[string]any{"url": "https://go.dev/dl/"})}},
		{Role: "tool", ToolCallID: "f1", Content: string(stored)}}
	view, n := shrinkLargeResults(history, 2048)
	var receipt struct {
		ArtifactID string `json:"artifact_id"`
		Note       string `json:"note"`
	}
	json.Unmarshal([]byte(view[3].Content), &receipt)
	if n != 1 || receipt.ArtifactID != out.ArtifactID || !strings.Contains(receipt.Note, "read_artifact") {
		t.Fatal(n, view[3].Content)
	}
}
