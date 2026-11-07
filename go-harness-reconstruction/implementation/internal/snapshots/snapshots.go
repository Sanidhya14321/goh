// Package snapshots keeps explicit, bounded file-change checkpoints outside the
// project. It never invokes Git or writes a user's repository index. Multi-file
// restoration is not a transaction: interrupted applications remain uncertain
// until an explicit acknowledgement, and are never automatically replayed.
package snapshots

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/platform"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	MaxPaths    = 64
	MaxFile     = 4 << 20
	MaxBytes    = 16 << 20
	MaxRecords  = 1024
	maxMetadata = 256 << 10
)

var (
	ErrPending   = errors.New("snapshot effects require explicit inspection and acknowledgement")
	ErrConflict  = errors.New("snapshot conflict: file no longer matches the expected state")
	ErrUncertain = errors.New("snapshot application interrupted; inspect applied paths before acknowledgement; no replay")
	idPattern    = regexp.MustCompile(`^[a-f0-9]{24}$`)
	hashPattern  = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Entry struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	SHA256 string `json:"sha256,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
}

type Record struct {
	Version int       `json:"version"`
	ID      string    `json:"id"`
	Project string    `json:"project"`
	Created time.Time `json:"created"`
	Status  string    `json:"status"`
	Before  []Entry   `json:"before"`
	After   []Entry   `json:"after,omitempty"`
	Applied []string  `json:"applied,omitempty"`
}

type Change struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Before    Entry  `json:"before"`
	After     Entry  `json:"after"`
	Binary    bool   `json:"binary"`
	Preview   string `json:"preview,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type Manager struct {
	mu        sync.Mutex
	project   string
	directory string
	lock      *os.File
	root      *os.Root
	store     *os.Root
}

// Open holds an exclusive nonblocking lock until Close. Directory must resolve
// outside project; each session should use its own private snapshot directory.
func Open(project, directory string) (*Manager, error) {
	p, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return nil, err
	}
	d, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	// Reject lexical containment before creating anything inside the project.
	if contained(p, d) || contained(d, p) {
		return nil, errors.New("snapshot directory must be separate from project and its ancestors")
	}
	// Resolve the nearest existing ancestor before MkdirAll, including aliases
	// into the project whose final directory has not yet been created.
	ancestor := d
	var tail []string
	for {
		_, e := os.Lstat(ancestor)
		if e == nil {
			break
		}
		if !os.IsNotExist(e) {
			return nil, e
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return nil, e
		}
		tail = append(tail, filepath.Base(ancestor))
		ancestor = parent
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return nil, err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, tail[i])
	}
	if contained(p, resolved) || contained(resolved, p) {
		return nil, errors.New("snapshot directory resolves into project or an ancestor")
	}
	if err = os.MkdirAll(d, 0700); err != nil {
		return nil, err
	}
	d, err = filepath.EvalSymlinks(d)
	if err != nil {
		return nil, err
	}
	if contained(p, d) || contained(d, p) {
		return nil, errors.New("snapshot directory resolves into project or an ancestor")
	}
	if err = os.Chmod(d, 0700); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(p)
	if err != nil {
		return nil, err
	}
	s, err := os.OpenRoot(d)
	if err != nil {
		r.Close()
		return nil, err
	}
	f, err := s.OpenFile("lock", os.O_RDWR|os.O_CREATE|platform.NoFollow|platform.NonBlock, 0600)
	if err != nil {
		r.Close()
		s.Close()
		return nil, err
	}
	fail := func(e error) (*Manager, error) { f.Close(); r.Close(); s.Close(); return nil, e }
	if st, e := f.Stat(); e != nil || !st.Mode().IsRegular() {
		return fail(errors.New("snapshot lock must be a regular file"))
	}
	if err = platform.TryLock(f); err != nil {
		return fail(fmt.Errorf("snapshots already have an active writer: %w", err))
	}
	if err = s.Mkdir("blobs", 0700); err != nil && !os.IsExist(err) {
		return fail(err)
	}
	if st, e := s.Lstat("blobs"); e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fail(errors.New("snapshot blobs must be a directory without symlinks"))
	}
	m := &Manager{project: p, directory: d, lock: f, root: r, store: s}
	if _, err = m.list(); err != nil {
		m.Close()
		return nil, err
	}
	return m, nil
}

func contained(root, path string) bool {
	rel, e := filepath.Rel(root, path)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lock == nil {
		return nil
	}
	e := m.lock.Close()
	m.lock = nil
	return errors.Join(e, m.root.Close(), m.store.Close())
}
func (m *Manager) ready() error {
	if m.lock == nil {
		return errors.New("snapshot manager is closed")
	}
	return nil
}

func clean(path string) (string, error) {
	if path == "" || len(path) > 4096 || filepath.IsAbs(path) || strings.IndexByte(path, 0) >= 0 {
		return "", errors.New("snapshot path must be a relative project file")
	}
	p := filepath.Clean(path)
	if p == "." || p == ".." || strings.HasPrefix(p, ".."+string(os.PathSeparator)) {
		return "", errors.New("snapshot path escapes project")
	}
	for _, part := range strings.Split(p, string(os.PathSeparator)) {
		if part == ".git" {
			return "", errors.New("repository metadata is excluded from snapshots")
		}
	}
	return p, nil
}

func checked(r *os.Root, path string, missing bool) error {
	parts := strings.Split(path, string(os.PathSeparator))
	p := ""
	for i, part := range parts {
		p = filepath.Join(p, part)
		st, err := r.Lstat(p)
		if err != nil {
			if missing && i == len(parts)-1 && os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("snapshot symlink paths are unsupported")
		}
		if i < len(parts)-1 && !st.IsDir() {
			return errors.New("snapshot parent is not a directory")
		}
	}
	return nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func nonce() (string, error) {
	var b [12]byte
	_, e := rand.Read(b[:])
	return hex.EncodeToString(b[:]), e
}

func (m *Manager) readCurrent(path string) (Entry, []byte, error) {
	e := Entry{Path: path}
	if err := checked(m.root, path, true); err != nil {
		return e, nil, err
	}
	f, err := m.root.OpenFile(path, os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if os.IsNotExist(err) {
		return e, nil, nil
	}
	if err != nil {
		return e, nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return e, nil, err
	}
	if !st.Mode().IsRegular() {
		return e, nil, errors.New("snapshot supports regular files only")
	}
	if st.Size() > MaxFile {
		return e, nil, errors.New("snapshot file exceeds 4MiB")
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxFile+1))
	if err != nil {
		return e, nil, err
	}
	if len(b) > MaxFile {
		return e, nil, errors.New("snapshot file exceeds 4MiB")
	}
	e.Exists = true
	e.SHA256 = digest(b)
	e.Size = int64(len(b))
	e.Mode = uint32(st.Mode().Perm())
	return e, b, nil
}

func (m *Manager) saveBlob(e Entry, b []byte) error {
	if !e.Exists {
		return nil
	}
	path := filepath.Join("blobs", e.SHA256)
	f, err := m.store.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|platform.NoFollow, 0600)
	if os.IsExist(err) {
		_, err = m.readBlob(e)
		return err
	}
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	d, err := m.store.Open("blobs")
	if err != nil {
		return err
	}
	return errors.Join(platform.SyncDir(d), d.Close())
}

func (m *Manager) readBlob(e Entry) ([]byte, error) {
	if !e.Exists {
		return nil, nil
	}
	if !hashPattern.MatchString(e.SHA256) || e.Size < 0 || e.Size > MaxFile {
		return nil, errors.New("invalid snapshot blob identity")
	}
	f, err := m.store.OpenFile(filepath.Join("blobs", e.SHA256), os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return nil, errors.New("snapshot blob must be a regular file")
	}
	if st.Size() != e.Size {
		return nil, errors.New("snapshot blob size mismatch")
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxFile+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != e.Size || digest(b) != e.SHA256 {
		return nil, errors.New("snapshot blob integrity mismatch")
	}
	return b, nil
}

func (m *Manager) writeRecord(r Record) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(b) > maxMetadata {
		return errors.New("snapshot metadata exceeds limit")
	}
	t, err := nonce()
	if err != nil {
		return err
	}
	temp := ".record-" + t
	f, err := m.store.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer m.store.Remove(temp)
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = m.store.Rename(temp, r.ID+".json"); err != nil {
		return err
	}
	d, err := m.store.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(platform.SyncDir(d), d.Close())
}

func (m *Manager) load(id string) (Record, error) {
	var r Record
	if !idPattern.MatchString(id) {
		return r, errors.New("invalid snapshot ID")
	}
	f, err := m.store.OpenFile(id+".json", os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if err != nil {
		return r, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > maxMetadata {
		return r, errors.New("invalid snapshot metadata file")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxMetadata+1))
	if err != nil {
		return r, err
	}
	if len(b) > maxMetadata {
		return r, errors.New("snapshot metadata exceeds limit")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&r); err != nil {
		return r, err
	}
	if d.Decode(new(any)) != io.EOF {
		return r, errors.New("trailing snapshot metadata")
	}
	if r.Version != 1 || r.ID != id || r.Project != m.project || r.Created.IsZero() {
		return r, errors.New("snapshot identity mismatch")
	}
	switch r.Status {
	case "captured", "recorded", "undone", "applying_undo", "applying_redo", "acknowledged", "aborted":
	default:
		return r, errors.New("invalid snapshot status")
	}
	if len(r.Before) == 0 || len(r.Before) > MaxPaths {
		return r, errors.New("invalid snapshot path count")
	}
	if len(r.After) != 0 && len(r.After) != len(r.Before) {
		return r, errors.New("snapshot side length mismatch")
	}
	if r.Status != "captured" && r.Status != "acknowledged" && r.Status != "aborted" && len(r.After) == 0 {
		return r, errors.New("snapshot after state missing")
	}
	for _, side := range [][]Entry{r.Before, r.After} {
		total := int64(0)
		prev := ""
		for _, e := range side {
			p, err := clean(e.Path)
			if err != nil || p != e.Path || p <= prev {
				return r, errors.New("invalid or unordered snapshot path")
			}
			prev = p
			total += e.Size
			if e.Exists {
				if !hashPattern.MatchString(e.SHA256) || e.Size < 0 || e.Size > MaxFile || e.Mode > 0777 {
					return r, errors.New("invalid snapshot entry")
				}
			} else if e.SHA256 != "" || e.Size != 0 || e.Mode != 0 {
				return r, errors.New("invalid missing-file snapshot entry")
			}
		}
		if total > MaxBytes {
			return r, errors.New("snapshot bytes exceed limit")
		}
	}
	for i := range r.After {
		if r.Before[i].Path != r.After[i].Path {
			return r, errors.New("snapshot path pairing mismatch")
		}
	}
	seen := map[string]bool{}
	for _, p := range r.Applied {
		found := false
		for _, e := range r.Before {
			if e.Path == p {
				found = true
				break
			}
		}
		if !found || seen[p] {
			return r, errors.New("invalid snapshot progress")
		}
		seen[p] = true
	}
	return r, nil
}

func (m *Manager) list() ([]Record, error) {
	if err := m.ready(); err != nil {
		return nil, err
	}
	d, err := m.store.Open(".")
	if err != nil {
		return nil, err
	}
	defer d.Close()
	var out []Record
	for {
		entries, err := d.ReadDir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			if len(out) >= MaxRecords {
				return nil, errors.New("snapshot record limit reached")
			}
			r, err := m.load(strings.TrimSuffix(e.Name(), ".json"))
			if err != nil {
				return nil, fmt.Errorf("snapshot %s: %w", e.Name(), err)
			}
			out = append(out, r)
		}
		if err == io.EOF {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Created.Equal(out[j].Created) {
			return out[i].ID < out[j].ID
		}
		return out[i].Created.Before(out[j].Created)
	})
	return out, nil
}

func pending(r Record) bool {
	return r.Status == "captured" || r.Status == "applying_undo" || r.Status == "applying_redo"
}
func (m *Manager) noPending(except string) error {
	rs, err := m.list()
	if err != nil {
		return err
	}
	for _, r := range rs {
		if r.ID != except && pending(r) {
			return fmt.Errorf("%w: %s (%s)", ErrPending, r.ID, r.Status)
		}
	}
	return nil
}
func (m *Manager) List() ([]Record, error) { m.mu.Lock(); defer m.mu.Unlock(); return m.list() }
func (m *Manager) Read(id string) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ready(); err != nil {
		return Record{}, err
	}
	return m.load(id)
}
func (m *Manager) Pending() ([]Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rs, e := m.list()
	if e != nil {
		return nil, e
	}
	out := []Record{}
	for _, r := range rs {
		if pending(r) {
			out = append(out, r)
		}
	}
	return out, nil
}

// Capture durably saves the supplied files before the caller mutates them.
// A caller must Seal or Abort; a crashed capture blocks later modifications.
func (m *Manager) Capture(ctx context.Context, paths []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.noPending(""); err != nil {
		return "", err
	}
	rs, err := m.list()
	if err != nil {
		return "", err
	}
	if len(rs) >= MaxRecords {
		return "", errors.New("snapshot record limit reached")
	}
	if len(paths) == 0 || len(paths) > MaxPaths {
		return "", errors.New("snapshot requires 1..64 paths")
	}
	set := map[string]bool{}
	for _, path := range paths {
		p, err := clean(path)
		if err != nil {
			return "", err
		}
		set[p] = true
	}
	paths = make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	r := Record{Version: 1, Project: m.project, Created: time.Now().UTC(), Status: "captured"}
	r.ID, err = nonce()
	if err != nil {
		return "", err
	}
	total := int64(0)
	for _, path := range paths {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		e, b, err := m.readCurrent(path)
		if err != nil {
			return "", err
		}
		total += e.Size
		if total > MaxBytes {
			return "", errors.New("snapshot exceeds 16MiB")
		}
		if err = m.saveBlob(e, b); err != nil {
			return "", err
		}
		r.Before = append(r.Before, e)
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = m.writeRecord(r); err != nil {
		return r.ID, err
	}
	return r.ID, nil
}

func (m *Manager) Seal(ctx context.Context, id string) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ready(); err != nil {
		return Record{}, err
	}
	r, err := m.load(id)
	if err != nil {
		return r, err
	}
	if r.Status != "captured" {
		return r, errors.New("only a captured snapshot can be sealed")
	}
	if err = m.noPending(id); err != nil {
		return r, err
	}
	total := int64(0)
	for _, before := range r.Before {
		if err = ctx.Err(); err != nil {
			return r, err
		}
		e, b, e2 := m.readCurrent(before.Path)
		if e2 != nil {
			return r, e2
		}
		total += e.Size
		if total > MaxBytes {
			return r, errors.New("snapshot exceeds 16MiB")
		}
		if err = m.saveBlob(e, b); err != nil {
			return r, err
		}
		r.After = append(r.After, e)
	}
	if err = ctx.Err(); err != nil {
		return r, err
	}
	r.Status = "recorded"
	err = m.writeRecord(r)
	return r, err
}

// Abort succeeds only when every current file still matches its before state.
func (m *Manager) Abort(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ready(); err != nil {
		return err
	}
	r, err := m.load(id)
	if err != nil {
		return err
	}
	if r.Status != "captured" {
		return errors.New("only a captured snapshot can be aborted")
	}
	for _, e := range r.Before {
		if err = ctx.Err(); err != nil {
			return err
		}
		current, _, e2 := m.readCurrent(e.Path)
		if e2 != nil {
			return e2
		}
		if current != e {
			return fmt.Errorf("%w: %s", ErrConflict, e.Path)
		}
	}
	r.Status = "aborted"
	return m.writeRecord(r)
}

// Acknowledge preserves the checkpoint and progress but makes it ineligible for
// Undo/Redo. The caller must first explicitly inspect all uncertain file effects.
func (m *Manager) Acknowledge(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ready(); err != nil {
		return err
	}
	r, err := m.load(id)
	if err != nil {
		return err
	}
	if !pending(r) {
		return errors.New("snapshot is not pending")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	r.Status = "acknowledged"
	return m.writeRecord(r)
}

func (m *Manager) Undo(ctx context.Context, id string) (Record, error) {
	return m.apply(ctx, id, false)
}
func (m *Manager) Redo(ctx context.Context, id string) (Record, error) { return m.apply(ctx, id, true) }

func (m *Manager) apply(ctx context.Context, id string, redo bool) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.noPending(""); err != nil {
		return Record{}, err
	}
	r, err := m.load(id)
	if err != nil {
		return r, err
	}
	expected, target := r.After, r.Before
	state, final := "applying_undo", "undone"
	if redo {
		if r.Status != "undone" {
			return r, errors.New("redo requires an undone snapshot")
		}
		expected, target = r.Before, r.After
		state, final = "applying_redo", "recorded"
	} else if r.Status != "recorded" {
		return r, errors.New("undo requires a recorded snapshot")
	}
	// Verify every source, destination and blob before the first publication.
	for i, e := range expected {
		if err = ctx.Err(); err != nil {
			return r, err
		}
		current, _, e2 := m.readCurrent(e.Path)
		if e2 != nil {
			return r, e2
		}
		if current != e {
			return r, fmt.Errorf("%w: %s", ErrConflict, e.Path)
		}
		if _, err = m.readBlob(target[i]); err != nil {
			return r, err
		}
	}
	if err = ctx.Err(); err != nil {
		return r, err
	}
	r.Status = state
	r.Applied = nil
	if err = m.writeRecord(r); err != nil {
		return r, err
	}
	uncertain := func(e error) (Record, error) { r.Status = state; return r, fmt.Errorf("%w: %v", ErrUncertain, e) }
	for i, e := range expected {
		if err = ctx.Err(); err != nil {
			return uncertain(err)
		}
		if e == target[i] {
			continue
		}
		if err = m.publish(ctx, e, target[i]); err != nil {
			return uncertain(err)
		}
		r.Applied = append(r.Applied, e.Path)
		if err = m.writeRecord(r); err != nil {
			return uncertain(err)
		}
	}
	r.Status = final
	if err = m.writeRecord(r); err != nil {
		return uncertain(err)
	}
	return r, nil
}

func (m *Manager) publish(ctx context.Context, expected, target Entry) error {
	b, err := m.readBlob(target)
	if err != nil {
		return err
	}
	var temp string
	if target.Exists {
		n, err := nonce()
		if err != nil {
			return err
		}
		temp = filepath.Join(filepath.Dir(target.Path), ".harness-restore-"+n)
		f, err := m.root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(target.Mode))
		if err != nil {
			return err
		}
		defer m.root.Remove(temp)
		_, err = f.Write(b)
		if err == nil {
			err = f.Chmod(os.FileMode(target.Mode))
		}
		if err == nil {
			err = f.Sync()
		}
		err = errors.Join(err, f.Close())
		if err != nil {
			return err
		}
	}
	current, _, err := m.readCurrent(expected.Path)
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf("%w: %s", ErrConflict, expected.Path)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if !target.Exists {
		err = m.root.Remove(target.Path)
	} else if expected.Exists {
		err = m.root.Rename(temp, target.Path)
	} else {
		err = m.root.Link(temp, target.Path)
	}
	if err != nil {
		return err
	}
	d, err := m.root.Open(filepath.Dir(target.Path))
	if err != nil {
		return err
	}
	return errors.Join(platform.SyncDir(d), d.Close())
}

// Diff returns stored before/after comparisons, never a comparison against
// unsnapshotted working files. Previews share a 32KiB budget; blobs stay private.
func (m *Manager) Diff(id string) ([]Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ready(); err != nil {
		return nil, err
	}
	r, err := m.load(id)
	if err != nil {
		return nil, err
	}
	if len(r.After) == 0 {
		return nil, errors.New("snapshot has no sealed after state")
	}
	changes := []Change{}
	budget := 32 << 10
	for i, before := range r.Before {
		after := r.After[i]
		if before == after {
			continue
		}
		b, err := m.readBlob(before)
		if err != nil {
			return nil, err
		}
		a, err := m.readBlob(after)
		if err != nil {
			return nil, err
		}
		c := Change{Path: before.Path, Status: "modified", Before: before, After: after}
		if !before.Exists {
			c.Status = "added"
		} else if !after.Exists {
			c.Status = "deleted"
		}
		c.Binary = !utf8.Valid(b) || !utf8.Valid(a) || bytes.IndexByte(b, 0) >= 0 || bytes.IndexByte(a, 0) >= 0
		if !c.Binary {
			var out strings.Builder
			for _, side := range []struct {
				prefix string
				body   []byte
			}{{"-", b}, {"+", a}} {
				if len(side.body) == 0 {
					continue
				}
				for line := range strings.SplitSeq(strings.TrimSuffix(string(side.body), "\n"), "\n") {
					need := len(line) + 2
					if need > budget {
						n := budget - 2
						if n < 0 {
							n = 0
						}
						if n > len(line) {
							n = len(line)
						}
						for n > 0 && !utf8.ValidString(line[:n]) {
							n--
						}
						if budget >= 2 {
							out.WriteString(side.prefix)
							out.WriteString(line[:n])
							out.WriteByte('\n')
							budget -= n + 2
						}
						c.Truncated = true
						break
					}
					out.WriteString(side.prefix)
					out.WriteString(line)
					out.WriteByte('\n')
					budget -= need
				}
				if c.Truncated {
					break
				}
			}
			c.Preview = out.String()
		}
		changes = append(changes, c)
	}
	return changes, nil
}
