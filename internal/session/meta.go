package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Meta is user-editable session metadata kept beside the journal, so listing
// stays cheap and the journal stays append-only.
type Meta struct {
	Title string `json:"title,omitempty"`
}

const maxTitle = 120

func metaPath(dir, id string) string { return filepath.Join(dir, id+".meta.json") }

// sidecars are the per-session paths besides the journal.
func sidecars(id string) []string {
	return []string{id + ".meta.json", id + "-artifacts", id + "-snapshots"}
}

// ReadMeta returns saved metadata; a missing file is not an error.
func ReadMeta(dir, id string) (Meta, error) {
	var m Meta
	if !validID.MatchString(id) {
		return m, errors.New("invalid session ID")
	}
	f, err := os.OpenFile(metaPath(dir, id), os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	defer f.Close()
	if err := json.NewDecoder(&limited{f, 4096}).Decode(&m); err != nil {
		return Meta{}, fmt.Errorf("session metadata unreadable: %w", err)
	}
	return m, nil
}

type limited struct {
	f *os.File
	n int
}

func (l *limited) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, errors.New("metadata exceeds 4 KiB")
	}
	if len(p) > l.n {
		p = p[:l.n]
	}
	n, err := l.f.Read(p)
	l.n -= n
	return n, err
}

// CleanTitle trims a title, removes control characters and bounds its length.
func CleanTitle(title string) string {
	title = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' ' // tabs and newlines separate words
		case unicode.IsControl(r) || r == '‮' || r == '‭':
			return -1
		}
		return r
	}, title)
	title = strings.Join(strings.Fields(title), " ")
	if r := []rune(title); len(r) > maxTitle {
		title = string(r[:maxTitle])
	}
	return title
}

// SetTitle saves a session title; an empty title restores the default
// (derived from the first prompt).
func SetTitle(dir, id, title string) error {
	if !validID.MatchString(id) {
		return errors.New("invalid session ID")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".jsonl")); err != nil {
		return fmt.Errorf("session %s not found", id)
	}
	b, err := json.Marshal(Meta{Title: CleanTitle(title)})
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".meta-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), metaPath(dir, id))
}

// ErrSessionOpen means another process (or this one) holds the session.
var ErrSessionOpen = errors.New("session is open in a running GoHarness; close it first")

func trashDir(dir string) string { return filepath.Join(dir, "trash") }

// Trash moves a session's journal, metadata, artifacts and snapshots into the
// trash folder, from which Restore can bring it back. A session open in any
// process is refused, using the same writer lock as Open.
func Trash(dir, id string) error {
	if !validID.MatchString(id) {
		return errors.New("invalid session ID")
	}
	journal := filepath.Join(dir, id+".jsonl")
	f, err := os.OpenFile(journal, os.O_RDWR|platform.NoFollow|platform.NonBlock, 0)
	if err != nil {
		return fmt.Errorf("session %s not found", id)
	}
	if err := lock(f); err != nil {
		f.Close()
		return ErrSessionOpen
	}
	// Unix: hold the lock until every path has moved. Windows cannot rename a
	// file with an open handle, so the handle is closed just before the journal
	// moves; if another process opens it in that window, Windows refuses the
	// rename (file in use), which is still a safe refusal.
	held := f
	defer func() {
		if held != nil {
			held.Close()
		}
	}()
	if err := os.MkdirAll(trashDir(dir), 0o700); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(trashDir(dir), id+".jsonl")); err == nil {
		return fmt.Errorf("session %s is already in the trash", id)
	}
	// Sidecars first, journal last: an interrupted delete leaves a session
	// that still opens, never an orphaned journal in the trash.
	for _, name := range sidecars(id) {
		if err := os.Rename(filepath.Join(dir, name), filepath.Join(trashDir(dir), name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if runtime.GOOS == "windows" {
		held.Close()
		held = nil
	}
	if err := os.Rename(journal, filepath.Join(trashDir(dir), id+".jsonl")); err != nil {
		return fmt.Errorf("could not move the journal (is the session open?): %w", err)
	}
	return nil
}

// Restore moves a trashed session back. It refuses to overwrite anything.
func Restore(dir, id string) error {
	if !validID.MatchString(id) {
		return errors.New("invalid session ID")
	}
	src := filepath.Join(trashDir(dir), id+".jsonl")
	if _, err := os.Lstat(src); err != nil {
		return fmt.Errorf("session %s is not in the trash", id)
	}
	for _, name := range append([]string{id + ".jsonl"}, sidecars(id)...) {
		if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
			return fmt.Errorf("%s already exists; nothing restored", name)
		}
	}
	// Journal first, so an interrupted restore leaves an openable session.
	if err := os.Rename(src, filepath.Join(dir, id+".jsonl")); err != nil {
		return err
	}
	for _, name := range sidecars(id) {
		if err := os.Rename(filepath.Join(trashDir(dir), name), filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
