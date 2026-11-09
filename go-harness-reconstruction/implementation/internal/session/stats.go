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
	// CostUSD is an estimate from configured prices; nil when unpriced.
	CostUSD *float64 `json:"estimated_cost_usd,omitempty"`
}

// Estimate fills CostUSD from per-million-token prices.
func (m *ModelUsage) Estimate(input, output float64) {
	cost := float64(m.Prompt)/1e6*input + float64(m.Completion)/1e6*output
	m.CostUSD = &cost
}

// Stats summarizes saved sessions (like `opencode stats`).
type Stats struct {
	Sessions int          `json:"sessions"`
	Prompts  int          `json:"prompts"`
	Models   []ModelUsage `json:"models"`
	Damaged  int          `json:"damaged_sessions,omitempty"`
	Tools    []ToolUsage  `json:"tools,omitempty"`
}

// ToolUsage counts recorded tool calls by name; Errors counts results the
// tool reported as failed.
type ToolUsage struct {
	Name   string `json:"name"`
	Calls  int    `json:"calls"`
	Errors int    `json:"errors"`
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
	byTool := map[string]*ToolUsage{}
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
		callTool := map[string]string{}
		for _, r := range records {
			switch r.Type {
			case "tool_start":
				var c struct{ ID, Name string }
				if json.Unmarshal(r.Data, &c) != nil || c.Name == "" || len(c.Name) > 128 {
					continue
				}
				callTool[c.ID] = c.Name
				tu := byTool[c.Name]
				if tu == nil {
					tu = &ToolUsage{Name: c.Name}
					byTool[c.Name] = tu
				}
				tu.Calls++
			case "tool_result":
				var res struct {
					ID     string
					Result struct{ Error bool }
				}
				if json.Unmarshal(r.Data, &res) == nil && res.Result.Error {
					if tu := byTool[callTool[res.ID]]; tu != nil {
						tu.Errors++
					}
				}
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
	for _, tu := range byTool {
		st.Tools = append(st.Tools, *tu)
	}
	sort.Slice(st.Tools, func(i, j int) bool {
		if st.Tools[i].Calls != st.Tools[j].Calls {
			return st.Tools[i].Calls > st.Tools[j].Calls
		}
		return st.Tools[i].Name < st.Tools[j].Name
	})
	return st, nil
}
