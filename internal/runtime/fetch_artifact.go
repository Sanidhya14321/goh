package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/Sanidhya14321/goh/internal/tools"
	"github.com/Sanidhya14321/goh/internal/webfetch"
)

// fetchArtifactMin is the page size from which a fetched page is also kept
// as an artifact. A small context window (a provider's 8k per-request cap)
// can only show the start of a large page, which on most sites is the
// navigation; with the artifact the model can read on instead of guessing.
const fetchArtifactMin = 16 << 10

// retainFetchedPage keeps the page text as an artifact when it is large or
// the inline result only excerpts it, labels what the artifact holds, and
// re-bounds the result with that metadata to tools.MaxInline.
func (a *Agent) retainFetchedPage(res tools.Result, full string) tools.Result {
	if res.Error || a.Tools == nil {
		return res
	}
	var out map[string]any
	if json.Unmarshal([]byte(res.Content), &out) != nil {
		return res
	}
	inline, _ := out["content"].(string)
	if full == "" {
		full = inline
	}
	if len(full) < fetchArtifactMin && len(inline) >= len(full) {
		return res // small and shown whole
	}
	id, err := a.Tools.SaveArtifact("fetch", full)
	if err != nil {
		return res // the excerpt itself is still returned
	}
	kept := min(len(full), tools.MaxArtifact)
	out["artifact_id"] = id
	out["artifact_bytes"] = kept
	out["artifact_note"] = fetchArtifactNote(out, kept, len(full))
	// Re-bound with the metadata; the note names how much the result shows,
	// so it is rewritten until it matches.
	bounded := res
	for range 3 {
		next := webfetch.BoundedResult(out)
		if next.Error {
			return res
		}
		bounded = next
		var check map[string]any
		if json.Unmarshal([]byte(next.Content), &check) != nil {
			return res
		}
		note := fetchArtifactNote(check, kept, len(full))
		if note == check["artifact_note"] {
			break
		}
		check["artifact_note"] = note
		out = check
	}
	return bounded
}

// fetchArtifactNote says exactly what the artifact holds relative to the
// result: the whole page text, or only its start when the conversion or the
// artifact limit cut it.
func fetchArtifactNote(out map[string]any, kept, total int) string {
	shown, _ := out["content"].(string)
	what := fmt.Sprintf("The whole page text (%d bytes) is kept as this artifact", kept)
	switch {
	case kept < total:
		what = fmt.Sprintf("The first %d of %d bytes of page text are kept as this artifact", kept, total)
	case out["conversion_truncated"] == true:
		what = fmt.Sprintf("The converted page text (%d bytes; conversion stopped at its size limit, the rest of the page was not converted) is kept as this artifact", kept)
	}
	if len(shown) < kept {
		what += fmt.Sprintf("; this result shows only its first %d bytes", len(shown))
	}
	return what + ". Search it with read_artifact (id, pattern) or read a part (id, offset, limit) instead of fetching again."
}
