package core

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// AccessLevel is how much goh may do in a folder without asking.
type AccessLevel string

const (
	// AccessAsk asks before every edit and command (session approvals still
	// apply). It is the default.
	AccessAsk AccessLevel = "ask"
	// AccessEdits makes file edits inside the folder without asking;
	// commands still ask.
	AccessEdits AccessLevel = "edits"
	// AccessAuto also runs commands without asking, inside the OS sandbox:
	// they can write only inside the folder and temporary folders. Where no
	// sandbox is available, commands still ask.
	AccessAuto AccessLevel = "auto"
	// AccessFull runs everything without asking and without a sandbox.
	AccessFull AccessLevel = "full"
)

// AccessLevels lists the levels from least to most access.
var AccessLevels = []AccessLevel{AccessAsk, AccessEdits, AccessAuto, AccessFull}

// ParseAccessLevel accepts a level name ("" is ask).
func ParseAccessLevel(s string) (AccessLevel, error) {
	switch l := AccessLevel(strings.ToLower(strings.TrimSpace(s))); l {
	case "":
		return AccessAsk, nil
	case AccessAsk, AccessEdits, AccessAuto, AccessFull:
		return l, nil
	}
	return "", fmt.Errorf("unknown access level %q (use ask, edits, auto or full)", s)
}

// Rank orders levels: ask 0, edits 1, auto 2, full 3.
func (l AccessLevel) Rank() int {
	for i, v := range AccessLevels {
		if v == l {
			return i
		}
	}
	return 0
}

// Label is the level's name in the interface.
func (l AccessLevel) Label() string {
	switch l {
	case AccessEdits:
		return "Auto-edit"
	case AccessAuto:
		return "Auto"
	case AccessFull:
		return "Full access"
	}
	return "Ask"
}

// Describe says what the level allows without asking.
func (l AccessLevel) Describe() string {
	switch l {
	case AccessEdits:
		return "Edits files without asking; commands still ask"
	case AccessAuto:
		return "Edits and runs commands without asking; commands can only write inside the folder (sandboxed)"
	case AccessFull:
		return "Everything without asking and without a sandbox; only for folders you can afford to lose"
	}
	return "Asks before every edit and command"
}

// Next is the level after l, wrapping from full back to ask.
func (l AccessLevel) Next() AccessLevel {
	return AccessLevels[(l.Rank()+1)%len(AccessLevels)]
}

// FolderAccess is an access level for a folder and everything inside it.
type FolderAccess struct {
	Path  string      `json:"path"`
	Level AccessLevel `json:"level"`
}

// AccessFor returns the rule for the deepest folder in rules that contains
// path, if any.
func AccessFor(rules []FolderAccess, path string) (FolderAccess, bool) {
	var best FolderAccess
	found := false
	for _, r := range rules {
		if !pathWithin(r.Path, path) {
			continue
		}
		if !found || len(r.Path) > len(best.Path) {
			best, found = r, true
		}
	}
	return best, found
}

// pathWithin reports whether path is dir or inside it.
func pathWithin(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// SavedAccess returns the access levels saved for folders, deepest first.
// A missing or unreadable list is empty: the worst case is asking.
func SavedAccess() []FolderAccess {
	path, err := workspacesPath()
	if err != nil {
		return nil
	}
	b, err := readConfig(path)
	if err != nil {
		return nil
	}
	var file struct {
		Access []FolderAccess `json:"access"`
	}
	if json.Unmarshal(b, &file) != nil {
		return nil
	}
	var out []FolderAccess
	for _, r := range file.Access {
		if level, err := ParseAccessLevel(string(r.Level)); err == nil && filepath.IsAbs(r.Path) {
			out = append(out, FolderAccess{Path: filepath.Clean(r.Path), Level: level})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Path) > len(out[j].Path) })
	return out
}

// SaveFolderAccess stores level for dir and its subfolders. Saving ask
// removes dir's own entry, so a parent folder's level applies again.
func SaveFolderAccess(dir string, level AccessLevel) error {
	if err := CheckProjectCredentialBoundary(dir); err != nil {
		return err
	}
	dir = cleanWorkspace(dir)
	var keep []FolderAccess
	for _, r := range SavedAccess() {
		if r.Path != dir {
			keep = append(keep, r)
		}
	}
	if level != AccessAsk {
		keep = append(keep, FolderAccess{Path: dir, Level: level})
	}
	return updateWorkspacesFile(func(file map[string]json.RawMessage) error {
		b, err := json.Marshal(keep)
		if err != nil {
			return err
		}
		if len(keep) == 0 {
			delete(file, "access")
		} else {
			file["access"] = b
		}
		return nil
	})
}

// updateWorkspacesFile rewrites workspaces.json through edit, keeping the
// fields it does not change.
func updateWorkspacesFile(edit func(map[string]json.RawMessage) error) error {
	path, err := workspacesPath()
	if err != nil {
		return err
	}
	file := map[string]json.RawMessage{}
	if b, err := readConfig(path); err == nil {
		if json.Unmarshal(b, &file) != nil {
			file = map[string]json.RawMessage{}
		}
	}
	if err := edit(file); err != nil {
		return err
	}
	b, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return writePrivate(path, string(b)+"\n")
}
