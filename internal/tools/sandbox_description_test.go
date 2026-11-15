package tools

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/sandbox"
)

func TestShellDescriptionReflectsSandbox(t *testing.T) {
	for _, tool := range []string{"shell", "job_start"} {
		describe := func(e *Engine) string {
			for _, d := range e.Definitions() {
				if d.Name == tool {
					return d.Description
				}
			}
			t.Fatal("no definition for", tool)
			return ""
		}
		if d := describe(&Engine{}); !strings.Contains(d, "not sandboxed") {
			t.Fatalf("%s unsandboxed description = %q", tool, d)
		}
		if d := describe(&Engine{Sandbox: &sandbox.Policy{}}); strings.Contains(d, "not sandboxed") || !strings.Contains(d, "OS sandbox") {
			t.Fatalf("%s sandboxed description = %q", tool, d)
		}
	}
}
