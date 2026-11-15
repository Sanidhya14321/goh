package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Workspace is a project folder the user chose to trust. Trusting a folder
// says the user means to work in it: goh then opens there directly. Edits and
// commands still ask for approval as configured; trust grants no tool
// permissions.
type Workspace struct {
	Path    string    `json:"path"`
	Trusted time.Time `json:"trusted"`
	Opened  time.Time `json:"opened"`
}

const maxWorkspaces = 200

func workspacesPath() (string, error) {
	path, err := UserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "workspaces.json"), nil
}

// Workspaces returns the trusted folders, most recently opened first. A
// missing or unreadable list is empty: the worst case is asking again.
func Workspaces() []Workspace {
	path, err := workspacesPath()
	if err != nil {
		return nil
	}
	b, err := readConfig(path)
	if err != nil {
		return nil
	}
	var list struct {
		Folders []Workspace `json:"folders"`
	}
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	var out []Workspace
	for _, w := range list.Folders {
		if filepath.IsAbs(w.Path) {
			out = append(out, w)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Opened.After(out[j].Opened) })
	return out
}

// IsTrustedWorkspace reports whether dir, or a folder that contains it, was
// trusted.
func IsTrustedWorkspace(dir string) bool {
	dir = cleanWorkspace(dir)
	for _, w := range Workspaces() {
		if rel, err := filepath.Rel(w.Path, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// OpenWorkspace records that dir was opened, trusting it if it was not.
func OpenWorkspace(dir string) error {
	if err := CheckProjectCredentialBoundary(dir); err != nil {
		return err
	}
	dir = cleanWorkspace(dir)
	now := time.Now().UTC().Truncate(time.Second)
	list := Workspaces()
	found := false
	for i := range list {
		if list[i].Path == dir {
			list[i].Opened, found = now, true
		}
	}
	if !found {
		list = append(list, Workspace{Path: dir, Trusted: now, Opened: now})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Opened.After(list[j].Opened) })
	if len(list) > maxWorkspaces {
		list = list[:maxWorkspaces]
	}
	path, err := workspacesPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Folders []Workspace `json:"folders"`
	}{list}, "", "  ")
	if err != nil {
		return err
	}
	return writePrivate(path, string(b)+"\n")
}

// ForgetWorkspace removes dir from the trusted folders.
func ForgetWorkspace(dir string) error {
	dir = cleanWorkspace(dir)
	var keep []Workspace
	for _, w := range Workspaces() {
		if w.Path != dir {
			keep = append(keep, w)
		}
	}
	path, err := workspacesPath()
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(struct {
		Folders []Workspace `json:"folders"`
	}{keep}, "", "  ")
	return writePrivate(path, string(b)+"\n")
}

func cleanWorkspace(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return filepath.Clean(dir)
}

// ProjectKind names what a folder looks like, for the folder browser: "git",
// "Go", "Node" and so on, or "" for a plain folder.
func ProjectKind(dir string) string {
	var kinds []string
	for _, marker := range []struct{ file, kind string }{
		{"go.mod", "Go"}, {"package.json", "Node"}, {"pyproject.toml", "Python"}, {"requirements.txt", "Python"},
		{"Cargo.toml", "Rust"}, {"pom.xml", "Java"}, {"build.gradle", "Java"}, {"Gemfile", "Ruby"},
		{"composer.json", "PHP"}, {"Package.swift", "Swift"}, {"CMakeLists.txt", "C/C++"}, {"Makefile", "make"},
	} {
		if _, err := os.Stat(filepath.Join(dir, marker.file)); err == nil && !containsString(kinds, marker.kind) {
			kinds = append(kinds, marker.kind)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		kinds = append([]string{"git"}, kinds...)
	}
	if len(kinds) > 3 {
		kinds = kinds[:3]
	}
	return strings.Join(kinds, " · ")
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
