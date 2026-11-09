// Package projectsnap records whole-project snapshots in a private shadow Git
// repository, so changes made by any means (edits, shell commands, code
// generators) can be listed, diffed and reverted per turn.
//
// The shadow repository lives outside the project. Every command runs with
// an explicit --git-dir and --work-tree, so the user's own .git directory,
// index, branches, commits and hooks are never read or changed. Files ignored
// by the project's .gitignore are not recorded.
package projectsnap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Store is the shadow repository for one project.
type Store struct {
	git     string
	gitDir  string
	project string
}

var treeID = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// ErrNoGit reports that git is not installed.
var ErrNoGit = errors.New("project snapshots need git on PATH")

// Open creates or opens the shadow repository for project under dataDir.
func Open(dataDir, project string) (*Store, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrNoGit
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	if abs, err = filepath.EvalSymlinks(abs); err != nil {
		return nil, err
	}
	if dataDir, err = resolveExisting(dataDir); err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(abs))
	s := &Store{git: git, gitDir: filepath.Join(dataDir, hex.EncodeToString(sum[:12])+".git"), project: abs}
	if rel, err := filepath.Rel(abs, s.gitDir); err == nil && !strings.HasPrefix(rel, "..") {
		return nil, errors.New("snapshot storage must be outside the project")
	}
	if _, err := os.Stat(filepath.Join(s.gitDir, "HEAD")); err != nil {
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return nil, err
		}
		if _, err := s.run(context.Background(), nil, "init", "--quiet"); err != nil {
			return nil, err
		}
		for _, kv := range [][2]string{{"core.autocrlf", "false"}, {"core.symlinks", "true"}, {"gc.auto", "0"}, {"core.fsmonitor", "false"}, {"core.hooksPath", os.DevNull}} {
			if _, err := s.run(context.Background(), nil, "config", kv[0], kv[1]); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

// Project returns the resolved project directory.
func (s *Store) Project() string { return s.project }

func (s *Store) run(ctx context.Context, stdin []byte, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	full := append([]string{"--git-dir=" + s.gitDir, "--work-tree=" + s.project, "-c", "core.quotepath=false"}, args...)
	cmd := exec.CommandContext(ctx, s.git, full...)
	cmd.Dir = s.project
	// Only the safe environment, and none of the user's GIT_* overrides, so
	// nothing redirects these commands to the real repository.
	cmd.Env = append(platform.ChildEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// Track records the current project tree and returns its ID.
func (s *Store) Track(ctx context.Context) (string, error) {
	if _, err := s.run(ctx, nil, "add", "--all", "--", "."); err != nil {
		return "", err
	}
	out, err := s.run(ctx, nil, "write-tree")
	if err != nil {
		return "", err
	}
	tree := strings.TrimSpace(out)
	if !treeID.MatchString(tree) {
		return "", fmt.Errorf("unexpected tree id %q", tree)
	}
	return tree, nil
}

// Change is one file that differs between two snapshots.
type Change struct {
	Status string `json:"status"` // A added, M modified, D deleted (relative to from)
	Path   string `json:"path"`
}

// Diff lists files that differ between two trees.
func (s *Store) Diff(ctx context.Context, from, to string) ([]Change, error) {
	if !treeID.MatchString(from) || !treeID.MatchString(to) {
		return nil, errors.New("invalid snapshot id")
	}
	out, err := s.run(ctx, nil, "diff-tree", "-r", "--no-renames", "--name-status", "-z", from, to)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var changes []Change
	for i := 0; i+1 < len(parts); i += 2 {
		changes = append(changes, Change{Status: parts[i][:1], Path: parts[i+1]})
	}
	return changes, nil
}

// Patch returns a unified diff between two trees, bounded to limit bytes.
func (s *Store) Patch(ctx context.Context, from, to string, limit int) (string, bool, error) {
	if !treeID.MatchString(from) || !treeID.MatchString(to) {
		return "", false, errors.New("invalid snapshot id")
	}
	out, err := s.run(ctx, nil, "diff-tree", "-r", "-p", "--no-renames", "--no-color", from, to)
	if err != nil {
		return "", false, err
	}
	if len(out) > limit {
		return out[:limit], true, nil
	}
	return out, false, nil
}

// Restore makes the project match tree. It first records the current state
// and returns that tree, so the restore can itself be undone. Files the
// project gained since tree are removed; ignored files are left alone.
func (s *Store) Restore(ctx context.Context, tree string) (before string, changes []Change, err error) {
	if !treeID.MatchString(tree) {
		return "", nil, errors.New("invalid snapshot id")
	}
	if _, err := s.run(ctx, nil, "cat-file", "-e", tree+"^{tree}"); err != nil {
		return "", nil, errors.New("snapshot not found in this project's history")
	}
	if before, err = s.Track(ctx); err != nil {
		return "", nil, err
	}
	if changes, err = s.Diff(ctx, before, tree); err != nil {
		return "", nil, err
	}
	if _, err = s.run(ctx, nil, "read-tree", tree); err != nil {
		return before, nil, err
	}
	if _, err = s.run(ctx, nil, "checkout-index", "--all", "--force"); err != nil {
		return before, nil, err
	}
	for _, c := range changes {
		if c.Status != "D" { // D: present now, absent in tree
			continue
		}
		p := filepath.Join(s.project, filepath.FromSlash(c.Path))
		if rel, err := filepath.Rel(s.project, p); err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return before, changes, err
		}
		removeEmptyParents(s.project, filepath.Dir(p))
	}
	return before, changes, nil
}

func removeEmptyParents(root, dir string) {
	for dir != root && strings.HasPrefix(dir, root) {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// resolveExisting makes dir absolute and resolves symlinks in its longest
// existing prefix, so containment checks compare like with like.
func resolveExisting(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	rest := ""
	for p := abs; ; p = filepath.Dir(p) {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(r, rest), nil
		}
		if filepath.Dir(p) == p {
			return abs, nil
		}
		rest = filepath.Join(filepath.Base(p), rest)
	}
}
