package core

import (
	"errors"
	"testing"
)

func TestWithDetectedLSP(t *testing.T) {
	look := func(name string) (string, error) {
		if name == "gopls" || name == "pyright-langserver" || name == "clangd" {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("missing")
	}
	c := Config{LSP: map[string]LSPConfig{"mypy": {Command: "pylsp", Extensions: []string{".py"}}}}
	if got := c.WithDetectedLSP(look); len(got.LSP) != 1 {
		t.Fatal("detection ran while lsp_auto is off")
	}
	c.LSPAuto = true
	got := c.WithDetectedLSP(look)
	if _, ok := got.LSP["gopls"]; !ok {
		t.Fatal("gopls not added")
	}
	if _, ok := got.LSP["pyright"]; ok {
		t.Fatal("pyright added although .py is already configured")
	}
	if _, ok := got.LSP["typescript"]; ok {
		t.Fatal("missing binary added")
	}
	if _, ok := got.LSP["clangd"]; !ok || len(c.LSP) != 1 {
		t.Fatal("clangd missing or input map mutated")
	}
}
