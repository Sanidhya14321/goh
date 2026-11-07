package tools

import (
	"context"
	"encoding/json"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"testing"
)

func TestLiteralFormatterProcessArguments(t *testing.T) {
	testutil.RequireUnixShell(t)
	root := t.TempDir()
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(root, "tool ' spaces")
	if err = os.WriteFile(tool, []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" \"$VALUE\" > result\n"), 0700); err != nil {
		t.Fatal(err)
	}
	value := "$(touch INJECTED) `touch OTHER` ' spaced"
	r := e.RunCommand(context.Background(), tool, []string{value}, map[string]string{"VALUE": value}, 5)
	if r.Error {
		t.Fatal(r)
	}
	b, err := os.ReadFile(filepath.Join(root, "result"))
	if err != nil || string(b) != value+"\n"+value+"\n" {
		t.Fatal(string(b), err)
	}
	for _, p := range []string{"INJECTED", "OTHER"} {
		if _, err = os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Fatal("executed substituted argument", p)
		}
	}
}
func TestMutationPathInventory(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"patchText": "*** Begin Patch\n*** Update File: before\n*** Move to: after\n@@\n-a\n+b\n*** Add File: new\n+c\n*** End Patch"})
	paths, err := e.MutationPaths("apply_patch", raw)
	if err != nil || len(paths) != 3 || paths[0] != "after" || paths[1] != "before" {
		t.Fatal(paths, err)
	}
	if paths, err = e.MutationPaths("shell", nil); err != nil || len(paths) != 0 {
		t.Fatal(paths, err)
	}
}
