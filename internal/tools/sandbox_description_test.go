package tools

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/sandbox"
)

func TestShellDescriptionReflectsSandbox(t *testing.T) {
	shell := func(e *Engine) string {
		for _, d := range e.Definitions() {
			if d.Name == "shell" {
				return d.Description
			}
		}
		t.Fatal("no shell definition")
		return ""
	}
	if d := shell(&Engine{}); !strings.Contains(d, "not sandboxed") {
		t.Fatalf("unsandboxed description = %q", d)
	}
	if d := shell(&Engine{Sandbox: &sandbox.Policy{}}); strings.Contains(d, "not sandboxed") || !strings.Contains(d, "OS sandbox") {
		t.Fatalf("sandboxed description = %q", d)
	}
}
