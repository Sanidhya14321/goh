package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestFileRulesDecideEdits(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.FileRules = []core.ShellRule{{Match: "*.env", Action: "deny"}, {Match: "notes/*", Action: "allow"}, {Match: "go.mod", Action: "ask"}}
	asked := 0
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		asked++
		return core.Decision{Allow: true}, nil
	}
	perm := func(tool string, args any) bool {
		b, _ := json.Marshal(args)
		d, err := a.Permission(context.Background(), core.Permission{Tool: tool, Arguments: b})
		if err != nil {
			t.Fatal(err)
		}
		return d.Allow
	}
	if perm("create_file", map[string]string{"path": "prod.env", "content": "x"}) || perm("child:create_file", map[string]string{"path": "a/b.env", "content": "x"}) {
		t.Fatal(".env edits must be denied")
	}
	if !perm("create_file", map[string]string{"path": "notes/a.md", "content": "x"}) || asked != 0 {
		t.Fatal("allowed path should run without asking", asked)
	}
	patch := "*** Begin Patch\n*** Add File: notes/b.md\n+x\n*** Add File: secret.env\n+y\n*** End Patch"
	if perm("apply_patch", map[string]any{"patchText": patch}) {
		t.Fatal("a patch touching a denied file must be denied")
	}
	a.Grants["create_file"] = true
	perm("create_file", map[string]string{"path": "go.mod", "content": "x"})
	perm("create_file", map[string]string{"path": "main.go", "content": "x"})
	if asked != 1 {
		t.Fatal("ask rule must prompt despite a grant; other granted paths must not", asked)
	}
}

func TestWebsearchOnlyWhenConfigured(t *testing.T) {
	a, _ := testAgent(t)
	has := func() bool {
		for _, d := range a.localDefinitions() {
			if d.Function.Name == "websearch" {
				return true
			}
		}
		return false
	}
	if has() {
		t.Fatal("websearch offered without websearch_url")
	}
	a.Config.WebsearchURL = "https://search.example.com"
	if !has() {
		t.Fatal("websearch missing when configured")
	}
	if _, side, err := a.prepare("websearch", []byte(`{"query":"go"}`)); err != nil || !side {
		t.Fatal("websearch must need approval", err)
	}
}

type fakeSearcher struct {
	modelFunc
	supported bool
	queries   []string
}

func (f *fakeSearcher) SupportsWebSearch() bool { return f.supported }
func (f *fakeSearcher) WebSearchJSON(_ context.Context, q string) (json.RawMessage, error) {
	f.queries = append(f.queries, q)
	return json.RawMessage(`{"answer":"yes","sources":[]}`), nil
}

func TestProviderWebsearch(t *testing.T) {
	a, _ := testAgent(t)
	searcher := &fakeSearcher{}
	a.Provider = searcher
	has := func() bool {
		for _, d := range a.localDefinitions() {
			if d.Function.Name == "websearch" {
				return true
			}
		}
		return false
	}
	a.Config.Websearch = "provider"
	if has() {
		t.Fatal("offered for a provider without hosted search")
	}
	searcher.supported = true
	if !has() {
		t.Fatal("missing with websearch: provider")
	}
	preview, side, err := a.prepare("websearch", []byte(`{"query":"go release"}`))
	if err != nil || !side || !strings.Contains(preview, "go release") {
		t.Fatal("provider search must need approval", preview, err)
	}
	var call core.Call
	call.Function.Name, call.Function.Arguments = "websearch", `{"query":"go release"}`
	if r := a.execute(context.Background(), call); r.Error || r.Content != `{"answer":"yes","sources":[]}` || len(searcher.queries) != 1 {
		t.Fatalf("%+v %v", r, searcher.queries)
	}
}
