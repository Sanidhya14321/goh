package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DefaultChatGPTWindow is used only when the model catalog cannot be read.
const DefaultChatGPTWindow = 32768

const windowCacheTTL = 24 * time.Hour

type windowCache struct {
	Fetched time.Time      `json:"fetched"`
	Windows map[string]int `json:"windows"`
}

// ChatGPTContextWindow returns the context window the account's model
// catalog reports for model, cached privately for a day, or
// DefaultChatGPTWindow when it cannot be determined. Benchmarks with
// gpt-6.1-sol showed the old fixed 32K default (about 7% of its 500K window)
// ending short tasks in budget_exhausted.
func ChatGPTContextWindow(ctx context.Context, model string) int {
	path, perr := chatGPTPath("model-windows.json")
	if perr == nil {
		if b, err := os.ReadFile(path); err == nil {
			var c windowCache
			if json.Unmarshal(b, &c) == nil && time.Since(c.Fetched) < windowCacheTTL {
				if w := c.Windows[model]; w > 0 {
					return w
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	models, err := ListChatGPTModels(ctx)
	if err != nil {
		return DefaultChatGPTWindow
	}
	c := windowCache{Fetched: time.Now(), Windows: map[string]int{}}
	for _, m := range models {
		if m.ContextWindow >= 1024 && m.ContextWindow <= 10_000_000 {
			c.Windows[m.Slug] = m.ContextWindow
		}
	}
	if perr == nil {
		if b, err := json.Marshal(c); err == nil {
			_ = os.MkdirAll(filepath.Dir(path), 0o700)
			_ = writePrivate(path, string(b))
		}
	}
	if w := c.Windows[model]; w > 0 {
		return w
	}
	return DefaultChatGPTWindow
}
