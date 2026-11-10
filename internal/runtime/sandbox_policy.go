package runtime

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
)

// sandboxPolicy turns the user's sandbox setting into a policy: writes only
// in the project, temporary folders and listed paths; GoHarness's own config,
// credentials and session state are unreadable.
func sandboxPolicy(c core.Config, project string) *sandbox.Policy {
	if c.Sandbox == nil || !c.Sandbox.Enabled && c.Sandbox.Container == "" {
		return nil
	}
	if c.Sandbox.Container != "" {
		return &sandbox.Policy{Container: c.Sandbox.Container}
	}
	p := &sandbox.Policy{Network: c.Sandbox.Network, Writable: append([]string{project}, sandbox.TempDirs()...)}
	home, _ := os.UserHomeDir()
	for _, w := range c.Sandbox.Writable {
		if home != "" && (w == "~" || strings.HasPrefix(w, "~/")) {
			w = filepath.Join(home, strings.TrimPrefix(w, "~"))
		}
		p.Writable = append(p.Writable, w)
	}
	if cfg, err := core.UserConfigPath(); err == nil {
		p.Protected = append(p.Protected, filepath.Dir(cfg))
	}
	if c.SessionDir != "" {
		p.Protected = append(p.Protected, filepath.Dir(c.SessionDir))
	}
	// Never hide a folder that contains the project itself.
	kept := p.Protected[:0]
	for _, d := range p.Protected {
		if rel, err := filepath.Rel(d, project); err != nil || strings.HasPrefix(rel, "..") {
			kept = append(kept, d)
		}
	}
	p.Protected = kept
	return p
}
