package runtime

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// sandboxSupported is a variable so tests can stand in for the platform.
var sandboxSupported = sandbox.Supported

// SetAccess replaces the folder access levels (the saved ones). It is safe
// to call while a run is active; the next permission check uses them.
func (a *Agent) SetAccess(rules []core.FolderAccess) {
	a.accessMu.Lock()
	defer a.accessMu.Unlock()
	a.accessRules = append([]core.FolderAccess(nil), rules...)
}

// SetSessionAccess sets the level for the whole project for this session,
// in place of a level saved for the project folder itself. Deeper saved
// folders keep their own level.
func (a *Agent) SetSessionAccess(level core.AccessLevel) {
	a.accessMu.Lock()
	defer a.accessMu.Unlock()
	a.sessionLevel = level
}

// ProjectAccess is the level that applies at the project root: the session
// level if one was chosen, else the saved level of the project or a folder
// containing it, else ask.
func (a *Agent) ProjectAccess() core.AccessLevel {
	level, _ := a.levelAt(a.projectRoot())
	return level
}

// projectRoot is the project folder with symlinks resolved, as saved access
// paths are.
func (a *Agent) projectRoot() string {
	root := a.Store.Project
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	return filepath.Clean(root)
}

// levelAt returns the level for path and the folder it was granted on.
func (a *Agent) levelAt(path string) (core.AccessLevel, string) {
	a.accessMu.Lock()
	rules := append([]core.FolderAccess(nil), a.accessRules...)
	session := a.sessionLevel
	a.accessMu.Unlock()
	root := a.projectRoot()
	if session != "" {
		kept := rules[:0]
		for _, r := range rules {
			if r.Path != root {
				kept = append(kept, r)
			}
		}
		rules = append(kept, core.FolderAccess{Path: root, Level: session})
	}
	r, ok := core.AccessFor(rules, path)
	if !ok {
		return core.AccessAsk, ""
	}
	return r.Level, r.Path
}

// accessDecision reports whether the access levels allow p without asking.
// confine is the folder a command allowed by the auto level must stay in.
func (a *Agent) accessDecision(p core.Permission) (allow bool, confine string, level core.AccessLevel) {
	name := strings.TrimPrefix(p.Tool, "child:")
	root := a.projectRoot()
	switch {
	case fileChangeTools[name]:
		if a.Tools == nil {
			return false, "", ""
		}
		paths, err := a.Tools.MutationPaths(name, p.Arguments)
		if err != nil || len(paths) == 0 {
			return false, "", ""
		}
		lowest := core.AccessFull
		for _, rel := range paths {
			// Decide on the stored spelling: a symlinked folder must not
			// carry an edit into a folder with less access.
			abs := filepath.Join(root, filepath.FromSlash(tools.CanonicalRel(a.Store.Project, rel)))
			level, _ = a.levelAt(abs)
			if level.Rank() < core.AccessEdits.Rank() {
				return false, "", ""
			}
			if level.Rank() < lowest.Rank() {
				lowest = level
			}
		}
		return true, "", lowest
	case name == "shell" || name == "job_start":
		var args struct {
			Command string `json:"command"`
			Cwd     string `json:"cwd"`
		}
		if json.Unmarshal(p.Arguments, &args) != nil {
			return false, "", ""
		}
		dir := root
		if args.Cwd != "" {
			if filepath.IsAbs(args.Cwd) {
				return false, "", ""
			}
			dir = filepath.Join(root, filepath.FromSlash(args.Cwd))
			if r, err := filepath.EvalSymlinks(dir); err == nil {
				dir = r
			}
		}
		level, folder := a.levelAt(dir)
		switch {
		case level == core.AccessFull:
			return true, "", level
		case name == "shell" && level.Rank() >= core.AccessEdits.Rank() && readOnlyCommand(args.Command):
			return true, "", level // only reads inside the project
		case level == core.AccessAuto && a.Config.Sandbox != nil && a.Config.Sandbox.Container != "":
			return true, "", level // every command already runs in the configured container
		case level == core.AccessAuto && sandboxSupported():
			return true, folder, level
		}
		return false, "", ""
	case name == "todowrite" || name == "skill" || name == "task" || strings.HasPrefix(name, "format:") || name == "snapshot_undo" || name == "snapshot_redo":
		// Session bookkeeping, subagents (whose own tools are checked
		// again) and changes to files goh itself edited.
		level, _ = a.levelAt(root)
		return level.Rank() >= core.AccessEdits.Rank(), "", level
	case name == "repeated_call":
		// The loop guard always asks: it exists to stop a model that is stuck.
		return false, "", ""
	}
	// Network, MCP tools, starting servers: only with full access.
	level, _ = a.levelAt(root)
	return level == core.AccessFull, "", level
}

// confinedSandbox is the sandbox for a command an access level allowed in
// folder: writes only there and in temporary folders, goh's own settings,
// keys and sessions unreadable, network as configured (on by default).
func (a *Agent) confinedSandbox(folder string) *sandbox.Policy {
	c := a.Config
	sb := core.SandboxConfig{Enabled: true, Network: true}
	if c.Sandbox != nil {
		sb.Network = c.Sandbox.Network || !c.Sandbox.Enabled
		sb.Writable = c.Sandbox.Writable
	}
	c.Sandbox = &sb
	return sandboxPolicy(c, folder)
}
