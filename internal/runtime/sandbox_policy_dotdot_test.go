package runtime

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// A protected folder that contains the project is never hidden, also when
// the project's folder name starts with two dots.
func TestSandboxNeverHidesAProjectNamedWithTwoDots(t *testing.T) {
	base := testutil.TempDir(t)
	project := filepath.Join(base, "..work")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	c := core.Defaults()
	c.Sandbox = &core.SandboxConfig{Enabled: true}
	c.SessionDir = filepath.Join(base, "sessions")
	p := sandboxPolicy(c, project)
	resolvedBase, _ := filepath.EvalSymlinks(base)
	if slices.Contains(p.Protected, base) || slices.Contains(p.Protected, resolvedBase) {
		t.Fatalf("the folder holding the project is protected: %v", p.Protected)
	}
}
