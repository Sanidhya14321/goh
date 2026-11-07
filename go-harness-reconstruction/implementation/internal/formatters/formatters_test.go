package formatters

import (
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitDiscoveryMatchingAndLiteralArguments(t *testing.T) {
	testutil.RequireUnixShell(t)
	root := t.TempDir()
	tool := filepath.Join(root, "formatter with spaces")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]core.FormatterConfig{
		"z":   {Command: "./formatter with spaces", Args: []string{"--file=$FILE", "$(touch nope)", "$FILE$FILE"}, Extensions: []string{".go"}},
		"a":   {Command: "/missing-explicit-formatter", Extensions: []string{".go"}},
		"off": {Disabled: true},
	}
	entries, err := Discover(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[0].Name != "a" || entries[0].Available || !entries[2].Available {
		t.Fatal(entries)
	}
	matching := Matching(entries, "folder/a.go")
	if len(matching) != 2 || len(Matching(entries, "a.GO")) != 0 {
		t.Fatal(matching)
	}
	args := matching[1].Arguments(filepath.Join(root, "a 'quoted'.go"))
	if args[1] != "$(touch nope)" || strings.Count(args[2], "a 'quoted'.go") != 2 {
		t.Fatal(args)
	}
}
func TestFormatterValidationAndGofmtDefaults(t *testing.T) {
	for _, cfg := range []map[string]core.FormatterConfig{
		{"x": {}}, {"bad:name": {Disabled: true}}, {"x": {Command: "a", Extensions: []string{"go"}}}, {"x": {Command: "a", Extensions: []string{".go"}, Env: map[string]string{"bad=name": "x"}}}, {"x": {Command: "a", Extensions: []string{".go"}, Timeout: 121}},
	} {
		if Validate(cfg) == nil {
			t.Fatal("accepted", cfg)
		}
	}
	entries, err := Discover(map[string]core.FormatterConfig{"gofmt": {}}, t.TempDir())
	if err != nil || len(entries) != 1 || entries[0].Args[0] != "-w" || entries[0].Timeout != 30 {
		t.Fatal(entries, err)
	}
}
