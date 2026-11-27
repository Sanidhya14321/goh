package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestMCPAddAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte(`{"model":"kept","websearch":"provider"}`), 0o600)

	name := "fixture-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe" // LookPath needs an executable extension there
	}
	bin := filepath.Join(t.TempDir(), name)
	os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	var out bytes.Buffer
	run := func(args ...string) error { out.Reset(); return runMCPEdit(&out, args) }
	if err := run("add", "files", "--env", "ROOT=/tmp", "--", "fixture-mcp", "--root", "."); err != nil {
		t.Fatal(err)
	}
	if err := run("add", "docs", "--url", "https://mcp.example.com/sse", "--sse", "--header", "Authorization=DOCS_TOKEN"); err != nil {
		t.Fatal(err)
	}
	c, err := core.LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	files, docs := c.MCP["files"], c.MCP["docs"]
	if !filepath.IsAbs(files.Command) || filepath.Base(files.Command) != name || strings.Join(files.Args, " ") != "--root ." || files.Env["ROOT"] != "/tmp" {
		t.Fatalf("local %+v", files)
	}
	if docs.URL != "https://mcp.example.com/sse" || docs.Transport != "sse" || docs.HeadersEnv["Authorization"] != "DOCS_TOKEN" {
		t.Fatalf("remote %+v", docs)
	}
	if c.Model != "kept" || c.Websearch != "provider" {
		t.Fatal("other settings lost", c.Model, c.Websearch)
	}
	for _, bad := range [][]string{
		{"add", "a__b", "--", "fixture-mcp"},
		{"add", "x"},
		{"add", "x", "--url", "https://a.example", "--", "fixture-mcp"},
		{"add", "x", "--", "no-such-program-goh-test"},
		{"add", "x", "--header", "A=B", "--", "fixture-mcp"},
		{"add", "x", "--url", "https://a.example", "--env", "A=B"},
		{"remove", "missing"},
	} {
		if run(bad...) == nil {
			t.Errorf("accepted %v", bad)
		}
	}
	if err := run("remove", "files"); err != nil {
		t.Fatal(err)
	}
	if err := run("remove", "docs"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "mcp") || !strings.Contains(string(raw), "kept") {
		t.Fatalf("after removing both: %s", raw)
	}
}
