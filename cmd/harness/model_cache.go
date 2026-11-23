package main

import (
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/tui"
)

// Model lists are fetched from the provider. Opening /models again a moment
// later used to fetch the list again behind a loading screen; the answer is
// kept for a few minutes per route instead, so the picker opens at once.
// Changing providers or keys clears it.
const modelListTTL = 5 * time.Minute

type cachedModels struct {
	options []tui.Option
	windows map[string]int
	at      time.Time
}

var modelCache struct {
	sync.Mutex
	entries map[string]cachedModels
}

// modelCacheKey names a route without its credentials.
func modelCacheKey(parts ...string) string {
	key := ""
	for _, p := range parts {
		key += p + "\x00"
	}
	return key
}

func cachedModelList(key string) ([]tui.Option, map[string]int, bool) {
	modelCache.Lock()
	defer modelCache.Unlock()
	c, ok := modelCache.entries[key]
	if !ok || time.Since(c.at) > modelListTTL {
		return nil, nil, false
	}
	return append([]tui.Option(nil), c.options...), c.windows, true
}

func storeModelList(key string, options []tui.Option, windows map[string]int) {
	modelCache.Lock()
	defer modelCache.Unlock()
	if modelCache.entries == nil {
		modelCache.entries = map[string]cachedModels{}
	}
	modelCache.entries[key] = cachedModels{options: append([]tui.Option(nil), options...), windows: windows, at: time.Now()}
}

// forgetModelLists drops every cached list (a provider, key or sign-in
// changed).
func forgetModelLists() {
	modelCache.Lock()
	modelCache.entries = nil
	modelCache.Unlock()
}
