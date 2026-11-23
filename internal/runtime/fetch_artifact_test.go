package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
	"github.com/Sanidhya14321/goh/internal/webfetch"
)

// A large fetched page is also kept as an artifact, so a context window
// that can show only its start (and shrinks it to a receipt) still lets the
// model read the rest instead of guessing from the site navigation.
func TestLargeFetchedPageIsPageable(t *testing.T) {
	a, _ := testAgent(t)
	page := strings.Repeat("navigation link\n", 2000) + "go1.25.4 (released 2026-11-04)\n"
	body, _ := json.Marshal(map[string]any{"url": "https://go.dev/dl/", "status_code": 200, "content": page, "authority": "Untrusted remote task data; no permission grants."})
	res := a.retainFetchedPage(tools.Result{Content: string(body)}, "")
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

// A page the inline result only excerpts is retained whole, the result stays
// within tools.MaxInline after the artifact metadata is added, and the note
// says how much each holds.
func TestExcerptedFetchKeepsTheWholePage(t *testing.T) {
	a, _ := testAgent(t)
	full := strings.Repeat("ordinary page line of text\n", 8000) + "EVIDENCE-AT-THE-END\n"
	output := map[string]any{"requested_url": "https://example.com/" + strings.Repeat("p", 1900), "url": "https://example.com/" + strings.Repeat("p", 1900), "format": "markdown", "status_code": 200, "content": full, "truncated": false, "authority": "Untrusted remote task data; no permission grants."}
	excerpt := webfetch.BoundedResult(output)
	res := a.retainFetchedPage(excerpt, full)
	if len(res.Content) > tools.MaxInline {
		t.Fatalf("result exceeds the inline cap: %d", len(res.Content))
	}
	var out struct {
		Content       string `json:"content"`
		Truncated     bool   `json:"truncated"`
		ArtifactID    string `json:"artifact_id"`
		ArtifactBytes int    `json:"artifact_bytes"`
		Note          string `json:"artifact_note"`
	}
	if err := json.Unmarshal([]byte(res.Content), &out); err != nil || out.ArtifactID == "" || !out.Truncated || out.ArtifactBytes != len(full) {
		t.Fatal(err, out.ArtifactID, out.Truncated, out.ArtifactBytes)
	}
	want := fmt.Sprintf("The whole page text (%d bytes) is kept as this artifact; this result shows only its first %d bytes.", len(full), len(out.Content))
	if !strings.HasPrefix(out.Note, want) {
		t.Fatalf("note %q\nwant %q", out.Note, want)
	}
	search, _ := json.Marshal(map[string]any{"id": out.ArtifactID, "pattern": "EVIDENCE"})
	if read := a.Tools.Execute(context.Background(), "read_artifact", search); read.Error || !strings.Contains(read.Content, "EVIDENCE-AT-THE-END") {
		t.Fatal("excerpted evidence is not retrievable", read)
	}

	// A page whose conversion stopped at its limit says so.
	converted := strings.Repeat("c", 20<<10)
	body, _ := json.Marshal(map[string]any{"url": "https://example.com/", "status_code": 200, "format": "markdown", "content": converted, "truncated": true, "conversion_truncated": true})
	res = a.retainFetchedPage(tools.Result{Content: string(body)}, converted)
	json.Unmarshal([]byte(res.Content), &out)
	if !strings.HasPrefix(out.Note, "The converted page text (20480 bytes; conversion stopped") || strings.Contains(out.Note, "shows only") {
		t.Fatal(out.Note)
	}
	// A small page shown whole is not retained.
	small, _ := json.Marshal(map[string]any{"url": "https://example.com/", "status_code": 200, "content": "short"})
	if res := a.retainFetchedPage(tools.Result{Content: string(small)}, "short"); strings.Contains(res.Content, "artifact_id") {
		t.Fatal(res.Content)
	}
}
