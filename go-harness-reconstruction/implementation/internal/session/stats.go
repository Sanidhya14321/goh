package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"example.com/independent-harness/internal/platform"
)

// ModelUsage totals reported tokens for one model.
type ModelUsage struct {
	Model      string `json:"model"`
	Replies    int    `json:"replies"`
	Prompt     int64  `json:"prompt_tokens"`
	Completion int64  `json:"completion_tokens"`
	Total      int64  `json:"total_tokens"`
}

// Stats summarizes saved sessions (like `opencode stats`).
type Stats struct {
	Sessions int          `json:"sessions"`
	Prompts  int          `json:"prompts"`
	Models   []ModelUsage `json:"models"`
	Damaged  int          `json:"damaged_sessions,omitempty"`
}

// CollectStats reads journals without locking them, so it works while chats
// are open. since filters by journal modification time (zero for all);
// project, if set, keeps only that project's sessions. Usage exists only for
// replies recorded by releases that journal it, and only when the provider
// reported it.
func CollectStats(dir, project string, since time.Time) (Stats, error) {
	var st Stats
	infos, damaged, err := ListAll(dir)
	if err != nil {
		return st, err
	}
	st.Damaged = len(damaged)
	byModel := map[string]*ModelUsage{}
	for _, info := range infos {
		if project != "" && info.Project != project {
			continue
		}
		path := filepath.Join(dir, info.ID+".jsonl")
		fi, err := os.Stat(path)
		if err != nil || (!since.IsZero() && fi.ModTime().Before(since)) {
			continue
		}
		f, err := os.OpenFile(path, os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
		if err != nil {
			continue
		}
		records, _, _ := readPrefix(f)
		f.Close()
		st.Sessions++
		for _, r := range records {
			switch r.Type {
			case "message":
				var m struct{ Role string }
				if json.Unmarshal(r.Data, &m) == nil && m.Role == "user" {
					st.Prompts++
				}
			case "usage":
				var u struct {
					Model      string `json:"model"`
					Prompt     int64  `json:"prompt_tokens"`
					Completion int64  `json:"completion_tokens"`
					Total      int64  `json:"total_tokens"`
				}
				if json.Unmarshal(r.Data, &u) != nil || u.Prompt < 0 || u.Completion < 0 || u.Total < 0 {
					continue
				}
				mu := byModel[u.Model]
				if mu == nil {
					mu = &ModelUsage{Model: u.Model}
					byModel[u.Model] = mu
				}
				mu.Replies++
				mu.Prompt += u.Prompt
				mu.Completion += u.Completion
				mu.Total += u.Total
			}
		}
	}
	for _, mu := range byModel {
		st.Models = append(st.Models, *mu)
	}
	sort.Slice(st.Models, func(i, j int) bool { return st.Models[i].Total > st.Models[j].Total })
	return st, nil
}
