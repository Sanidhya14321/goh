package main

import (
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/tui"
)

func TestModelListCache(t *testing.T) {
	forgetModelLists()
	t.Cleanup(forgetModelLists)
	key := modelCacheKey("saved", "groq", "openai", "groq", "https://api.groq.com/openai/v1", "")
	if _, _, ok := cachedModelList(key); ok {
		t.Fatal("empty cache answered")
	}
	storeModelList(key, []tui.Option{{Value: "a"}, {Value: "b"}}, map[string]int{"a": 8000})
	options, windows, ok := cachedModelList(key)
	if !ok || len(options) != 2 || windows["a"] != 8000 {
		t.Fatal("cached list not returned", options, windows)
	}
	options[0].Value = "changed by a caller"
	if again, _, _ := cachedModelList(key); again[0].Value != "a" {
		t.Fatal("callers share the cached slice")
	}
	if _, _, ok := cachedModelList(modelCacheKey("saved", "groq", "openai", "groq", "https://other.example/v1", "")); ok {
		t.Fatal("another route answered from this route's list")
	}
	modelCache.Lock()
	entry := modelCache.entries[key]
	entry.at = time.Now().Add(-modelListTTL - time.Second)
	modelCache.entries[key] = entry
	modelCache.Unlock()
	if _, _, ok := cachedModelList(key); ok {
		t.Fatal("an expired list was used")
	}
	storeModelList(key, []tui.Option{{Value: "a"}}, nil)
	forgetModelLists()
	if _, _, ok := cachedModelList(key); ok {
		t.Fatal("a provider change must clear the cache")
	}
}
