package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestToolSchemasCompileAsDeliveredOverJSON(t *testing.T) {
	for _, d := range Definitions() {
		t.Run(d.Name, func(t *testing.T) {
			raw, err := json.Marshal(d.Schema)
			if err != nil {
				t.Fatal(err)
			}
			var delivered any
			if err = json.Unmarshal(raw, &delivered); err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			url := "https://fixtures.invalid/" + d.Name
			if err = compiler.AddResource(url, delivered); err != nil {
				t.Fatal(err)
			}
			if _, err = compiler.Compile(url); err != nil {
				t.Fatalf("invalid provider tool schema: %s: %v", string(raw), err)
			}
		})
	}
}
func TestLargePermissionDiffIncludesBothSidesWithinLimit(t *testing.T) {
	before := []byte(strings.Repeat("old", 50000))
	after := []byte(strings.Repeat("new", 50000))
	preview := diffPreview("fixture.txt", before, after, false)
	if len(preview) > MaxInline {
		t.Fatalf("unbounded permission preview: %d", len(preview))
	}
	if !strings.Contains(preview, "-old") || !strings.Contains(preview, "+new") || !strings.Contains(preview, "truncated") {
		t.Fatal("permission preview must show bounded excerpts from both before and after contents")
	}
}
