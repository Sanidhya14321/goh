package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"example.com/independent-harness/internal/core"
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
