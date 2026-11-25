package runtime

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// sandboxSupported and nestedEnforced are variables so tests can stand in
// for the platform.
var (
	sandboxSupported = sandbox.Supported
	nestedEnforced   = sandbox.NestedRestrictionsEnforced
	tempDirs         = sandbox.TempDirs
)

// canonical resolves symlinks (where the path exists) for comparing folders.
func canonical(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// within reports whether child is parent or inside it.
func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// tempOverlap reports whether a temporary folder the sandbox keeps writable
// would let a command write in the project outside allowed (allowed ""
// means nothing in the project may be written): a temporary folder that
// contains the project, or one inside it but outside allowed.
func tempOverlap(root, allowed string) bool {
	root = canonical(root)
	if allowed != "" {
		allowed = canonical(allowed)
	}
	for _, t := range tempDirs() {
		t = canonical(t)
		if within(t, root) {
			return true
		}
		if within(root, t) && (allowed == "" || !within(allowed, t)) {
			return true
		}
	}
	return false
}

// readOnlyConfine marks, in a confine folder, a command allowed only because
// it reads: it runs with no writable project folder.
const readOnlyConfine = "\x00read-only:"

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
//
// The screen asks for the access level on every frame, and resolving costs a
// system call per path component, so the answer is reused for a moment.
func (a *Agent) projectRoot() string {
	project := a.Store.Project
	a.rootMu.Lock()
	defer a.rootMu.Unlock()
	if a.rootFor == project && time.Since(a.rootAt) < projectRootTTL {
		return a.root
	}
	root := project
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	a.root, a.rootFor, a.rootAt = filepath.Clean(root), project, time.Now()
	return a.root
}

const projectRootTTL = 2 * time.Second

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
			dir = filepath.Join(root, filepath.FromSlash(args.Cwd))
			if filepath.IsAbs(args.Cwd) {
				dir = filepath.Clean(args.Cwd) // the tools accept one inside the project
			}
			if r, err := filepath.EvalSymlinks(dir); err == nil {
				dir = r
			}
			if rel, err := filepath.Rel(root, dir); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				return false, "", "" // outside the project: ask
			}
		}
		level, folder := a.levelAt(dir)
		container := a.Config.Sandbox != nil && a.Config.Sandbox.Container != ""
		switch {
		case level == core.AccessFull:
			return true, "", level
		case container:
			// A configured container runs commands with whatever its own
			// mounts allow: goh cannot show that the project is read-only
			// there, or that stricter folders are protected, so below full
			// access commands ask.
			return false, "", ""
		case name == "shell" && level.Rank() >= core.AccessEdits.Rank() && readOnlyCommand(args.Command):
			// The read-only check is a heuristic, so the command also runs
			// where it cannot write: a sandbox with no writable project
			// folder. Without one, or when a writable temporary folder
			// overlaps the project, it asks.
			if sandboxSupported() && !tempOverlap(root, "") {
				return true, readOnlyConfine + root, level
			}
		case level == core.AccessAuto && sandboxSupported():
			// Folders inside this one with a stricter level, and goh's own
			// state, must stay unwritable; where the sandbox cannot restrict
			// a folder inside a writable one, or a writable temporary folder
			// reaches outside this one, the command asks instead.
			if !nestedEnforced() && len(a.nestedRestrictions(folder)) > 0 {
				return false, "", ""
			}
			if tempOverlap(root, folder) {
				return false, "", ""
			}
			return true, folder, level
		}
		return false, "", ""
	case name == "submit_evidence" || name == "recall_observation":
		// Experiment bookkeeping and data this session already holds.
		level, _ = a.levelAt(root)
		return true, "", level
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
	if root, ok := strings.CutPrefix(folder, readOnlyConfine); ok {
		// Allowed because it only reads: nothing in the project is writable.
		sb.Writable = nil
		p := sandboxPolicy(c, root)
		p.Writable = sandbox.TempDirs()
		return p
	}
	p := sandboxPolicy(c, folder)
	p.ReadOnly = append(p.ReadOnly, a.stricterFolders(folder)...)
	return p
}

// stricterFolders are folders inside folder whose saved level is below auto:
// a command allowed by auto in folder must not write there.
func (a *Agent) stricterFolders(folder string) []string {
	a.accessMu.Lock()
	rules := append([]core.FolderAccess(nil), a.accessRules...)
	a.accessMu.Unlock()
	var out []string
	for _, r := range rules {
		rel, err := filepath.Rel(folder, r.Path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if r.Level.Rank() < core.AccessAuto.Rank() {
			out = append(out, r.Path)
		}
	}
	return out
}

// nestedRestrictions are the paths inside folder that a command confined to
// folder must not write (stricter folders) or read (goh's own state).
func (a *Agent) nestedRestrictions(folder string) []string {
	out := a.stricterFolders(folder)
	c := a.Config
	c.Sandbox = &core.SandboxConfig{Enabled: true}
	if p := sandboxPolicy(c, folder); p != nil {
		for _, d := range p.Protected {
			if rel, err := filepath.Rel(folder, d); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				out = append(out, d)
			}
		}
	}
	return out
}
