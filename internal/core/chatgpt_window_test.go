package core

import (
	"encoding/json"
	"testing"
)

// The account model catalog reports each model's context window; it must be
// read rather than assuming a small fixed default.
func TestModelCatalogContextWindowParsed(t *testing.T) {
	var cat struct{ Models []ChatGPTModelInfo }
	raw := `{"models":[{"slug":"gpt-6.1-sol","display_name":"GPT-6.1-Sol","visibility":"list","context_window":500000,"max_context_window":872000}]}`
	if err := json.Unmarshal([]byte(raw), &cat); err != nil || cat.Models[0].ContextWindow != 500000 {
		t.Fatalf("%+v %v", cat, err)
	}
}
