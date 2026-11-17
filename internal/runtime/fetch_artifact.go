package runtime

import (
	"encoding/json"

	"github.com/Sanidhya14321/goh/internal/tools"
)

// fetchArtifactMin is the page size from which a fetched page is also kept
// as an artifact. A small context window (a provider's 8k per-request cap)
// can only show the start of a large page, which on most sites is the
// navigation; with the artifact the model can read on instead of guessing.
const fetchArtifactMin = 16 << 10

func (a *Agent) retainFetchedPage(res tools.Result) tools.Result {
	if res.Error || len(res.Content) < fetchArtifactMin || a.Tools == nil {
		return res
	}
	var out map[string]json.RawMessage
	if json.Unmarshal([]byte(res.Content), &out) != nil {
		return res
	}
	var page string
	if json.Unmarshal(out["content"], &page) != nil || len(page) < fetchArtifactMin {
		return res
	}
	id, err := a.Tools.SaveArtifact("fetch", page)
	if err != nil {
		return res // the page itself is still returned
	}
	out["artifact_id"], _ = json.Marshal(id)
	out["artifact_note"], _ = json.Marshal("The page text is also kept as this artifact; if this result is cut short, search it with read_artifact (id, pattern) or read a part (id, offset, limit) instead of fetching again.")
	b, err := json.Marshal(out)
	if err != nil {
		return res
	}
	res.Content = string(b)
	return res
}
