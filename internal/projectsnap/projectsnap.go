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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Store is the shadow repository for one project.
type Store struct {
	git     string
	gitDir  string
	project string
	// index is the private index file for this Store's current operation;
	// "" uses the shadow repository's shared index (read-only commands).
	index string
}

// repoLocks serializes captures and restores of one shadow repository in
// this process; lockRepo's lock file does so across processes. Private
// indexes keep staging apart, but concurrent git processes still write the
// same object files, which Windows refuses ("Permission denied") while
// another process has one open.
var repoLocks sync.Map

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

func (s *Store) run(ctx context.Context, stdin []byte, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	full := append([]string{"--git-dir=" + s.gitDir, "--work-tree=" + s.project, "-c", "core.quotepath=false"}, args...)
	cmd := exec.CommandContext(ctx, s.git, full...)
	cmd.Dir = s.project
	// Only the safe environment, and none of the user's GIT_* overrides, so
	// nothing redirects these commands to the real repository.
	cmd.Env = append(platform.ChildEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1")
	if s.index != "" {
		cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+s.index)
	}
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

// withPrivateIndex runs f on a copy of s that stages into its own index
// file, seeded from the shared one (a stat cache only). Two sessions
// capturing or restoring the same project then never interleave stage,
// read-tree and checkout operations on one index.
func (s *Store) withPrivateIndex(f func(*Store) error) error {
	if s.index != "" {
		return f(s)
	}
	tmp, err := os.CreateTemp(s.gitDir, "index-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	seeded := false
	if shared, err := os.Open(filepath.Join(s.gitDir, "index")); err == nil {
		_, err = io.Copy(tmp, shared)
		shared.Close()
		if err != nil {
			tmp.Close()
			return err
		}
		seeded = true
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if !seeded {
		os.Remove(name) // git creates a fresh index; an empty file is invalid
	}
	private := *s
	private.index = name
	if err := f(&private); err != nil {
		return err
	}
	// Keep the warm stat cache for the next capture; replacing the file is
	// atomic, so a concurrent reader sees the old or the new index whole.
	_ = os.Rename(name, filepath.Join(s.gitDir, "index"))
	return nil
}

// Track records the current project tree and returns its ID.
func (s *Store) Track(ctx context.Context) (string, error) {
	unlock, err := s.lockRepo(ctx)
	if err != nil {
		return "", err
	}
	defer unlock()
	var tree string
	err = s.withPrivateIndex(func(p *Store) error {
		var err error
		tree, err = p.track(ctx)
		return err
	})
	return tree, err
}

func (s *Store) track(ctx context.Context) (string, error) {
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
	unlock, err := s.lockRepo(ctx)
	if err != nil {
		return "", nil, err
	}
	defer unlock()
	err = s.withPrivateIndex(func(p *Store) error {
		before, changes, err = p.restore(ctx, tree)
		return err
	})
	return before, changes, err
}

// lockRepo takes the shadow repository for one capture or restore: an
// in-process mutex, then a lock file shared with other processes, waiting
// for the current holder. A lock file older than ten minutes was left by a
// process that died and is taken over.
func (s *Store) lockRepo(ctx context.Context) (func(), error) {
	m, _ := repoLocks.LoadOrStore(s.gitDir, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	path := filepath.Join(s.gitDir, "snapshot.lock")
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(path); mu.Unlock() }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			mu.Unlock()
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > 10*time.Minute {
			os.Remove(path)
			continue
		}
		select {
		case <-ctx.Done():
			mu.Unlock()
			return nil, fmt.Errorf("another snapshot of this project is in progress: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *Store) restore(ctx context.Context, tree string) (before string, changes []Change, err error) {
	if _, err := s.run(ctx, nil, "cat-file", "-e", tree+"^{tree}"); err != nil {
		return "", nil, errors.New("snapshot not found in this project's history")
	}
	if before, err = s.track(ctx); err != nil {
		return "", nil, err
	}
	if changes, err = s.Diff(ctx, before, tree); err != nil {
		return "", nil, err
	}
	if err = s.checkUnrecorded(ctx, before, changes); err != nil {
		return before, nil, err
	}
	if _, err = s.run(ctx, nil, "read-tree", tree); err != nil {
		return before, nil, err
	}
	// Remove files the target lacks first, so a folder that becomes a file
	// is already empty when checkout writes the file.
	for _, c := range changes {
		if c.Status != "D" { // D: present now, absent in tree
			continue
		}
		p := filepath.Join(s.project, filepath.FromSlash(c.Path))
		if rel, err := filepath.Rel(s.project, p); err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if linkedParent(s.project, p) {
			// A folder on the way became a symlink: removing through it
			// could delete a file outside the project.
			return before, changes, fmt.Errorf("restore refused: a parent folder of %s is a symlink", c.Path)
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return before, changes, err
		}
		removeEmptyParents(s.project, filepath.Dir(p))
	}
	if _, err = s.run(ctx, nil, "checkout-index", "--all", "--force"); err != nil {
		return before, nil, err
	}
	return before, changes, nil
}

// linkedParent reports whether any folder between project and p is a symlink.
func linkedParent(project, p string) bool {
	for d := filepath.Dir(p); d != project && len(d) > len(project); d = filepath.Dir(d) {
		if fi, err := os.Lstat(d); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

// checkUnrecorded refuses a restore that would replace a directory with a
// file when that directory holds files the snapshot did not record (ignored
// files, for example): checkout would delete them with no way to undo.
func (s *Store) checkUnrecorded(ctx context.Context, before string, changes []Change) error {
	for _, c := range changes {
		if c.Status == "D" {
			continue
		}
		dir := filepath.Join(s.project, filepath.FromSlash(c.Path))
		if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
			continue
		}
		out, err := s.run(ctx, nil, "ls-tree", "-r", "-z", "--name-only", before, "--", c.Path+"/")
		if err != nil {
			return err
		}
		recorded := map[string]bool{}
		for _, n := range strings.Split(out, "\x00") {
			if n != "" {
				recorded[n] = true
			}
		}
		var lost string
		walkErr := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err // an unreadable folder could hide an unrecorded file
			}
			if lost != "" {
				return filepath.SkipAll
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(s.project, p)
			if !recorded[filepath.ToSlash(rel)] {
				lost = filepath.ToSlash(rel)
				return filepath.SkipAll
			}
			return nil
		})
		if walkErr != nil {
			return fmt.Errorf("restore refused: could not check %s for unrecorded files: %w", c.Path, walkErr)
		}
		if lost != "" {
			return fmt.Errorf("restore would replace folder %s with a file and delete %s, which no snapshot records; move it first", c.Path, lost)
		}
	}
	return nil
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
