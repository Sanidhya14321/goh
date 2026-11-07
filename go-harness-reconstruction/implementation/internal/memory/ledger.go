// Package memory stores operator-authored, project-scoped notes with file evidence.
package memory

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/platform"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/independent-harness/internal/tools"
)

const (
	MaxNoteBytes = 1200
	maxNotes     = 256
	maxDatabase  = 1 << 20
)

type Evidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Note struct {
	ID        string     `json:"id"`
	Text      string     `json:"text"`
	CreatedAt time.Time  `json:"created_at"`
	Evidence  []Evidence `json:"evidence"`
}

type Ledger struct {
	directory string
	key       string
	files     *tools.Engine
}

func New(sessionDir, project string) (*Ledger, error) {
	path, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	files, err := tools.New(path, "")
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(path))
	return &Ledger{directory: filepath.Join(sessionDir, "memory-v1"), key: hex.EncodeToString(key[:]), files: files}, nil
}

func (l *Ledger) directoryRoot(create bool) (*os.Root, error) {
	if create {
		if err := os.MkdirAll(l.directory, 0700); err != nil {
			return nil, err
		}
	}
	info, err := os.Lstat(l.directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("memory directory must be a private directory")
	}
	return os.OpenRoot(l.directory)
}

func (l *Ledger) read(root *os.Root) ([]Note, error) {
	file, err := root.OpenFile(l.key+".json", os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if os.IsNotExist(err) {
		return []Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > maxDatabase {
		return nil, errors.New("memory ledger must be a bounded private regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDatabase+1))
	if err != nil {
		return nil, err
	}
	var notes []Note
	if len(data) > maxDatabase || json.Unmarshal(data, &notes) != nil || notes == nil || len(notes) > maxNotes {
		return nil, errors.New("invalid or oversized memory ledger")
	}
	ids := make(map[string]bool, len(notes))
	for _, note := range notes {
		if len(note.ID) < 16 || len(note.ID) > 128 || ids[note.ID] || note.CreatedAt.IsZero() || !validNoteText(note.Text) ||
			len(note.Evidence) == 0 || len(note.Evidence) > 4 {
			return nil, errors.New("invalid memory note")
		}
		ids[note.ID] = true
		for _, file := range note.Evidence {
			digest, err := hex.DecodeString(file.SHA256)
			if !validPath(file.Path) || err != nil || len(digest) != sha256.Size {
				return nil, errors.New("invalid memory evidence")
			}
		}
	}
	return notes, nil
}

func (l *Ledger) List() ([]Note, error) {
	root, err := l.directoryRoot(false)
	if os.IsNotExist(err) {
		return []Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return l.read(root)
}

func (l *Ledger) update(edit func([]Note) ([]Note, error)) error {
	root, err := l.directoryRoot(true)
	if err != nil {
		return err
	}
	defer root.Close()
	lock, err := root.OpenFile(l.key+".lock", os.O_CREATE|os.O_RDWR|platform.NoFollow|platform.NonBlock, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("memory lock must be a regular file")
	}
	if err := platform.TryLock(lock); err != nil {
		return errors.New("memory ledger already has an active writer")
	}
	notes, err := l.read(root)
	if err != nil {
		return err
	}
	notes, err = edit(notes)
	if err != nil {
		return err
	}
	data, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	if len(data) > maxDatabase {
		return errors.New("memory ledger exceeds 1 MiB")
	}
	name := ".memory-" + rand.Text()
	temp, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(name)
	_, writeErr := temp.Write(data)
	if writeErr == nil {
		writeErr = temp.Sync()
	}
	if err := errors.Join(writeErr, temp.Close()); err != nil {
		return err
	}
	if err := root.Rename(name, l.key+".json"); err != nil {
		return err
	}
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (l *Ledger) hash(path string) (string, error) {
	value, err := l.files.FileDigest(path)
	if err != nil {
		return "", err
	}
	hash, ok := value["sha256"].(string)
	if !ok {
		return "", errors.New("evidence file is missing")
	}
	return hash, nil
}

func validNoteText(text string) bool {
	return len(text) <= MaxNoteBytes && utf8.ValidString(text) && strings.TrimSpace(text) != ""
}

func validPath(path string) bool {
	return len(path) <= 192 && utf8.ValidString(path) && filepath.IsLocal(path) &&
		filepath.Clean(path) == path && path != "."
}

func (l *Ledger) Add(text string, paths []string) (Note, error) {
	text = strings.TrimSpace(text)
	encoded, _ := json.Marshal(text)
	if !validNoteText(text) || len(encoded) > 2048 {
		return Note{}, errors.New("note must contain bounded UTF-8 text")
	}
	if len(paths) == 0 || len(paths) > 4 {
		return Note{}, errors.New("supply 1 to 4 project-relative evidence paths")
	}
	note := Note{ID: rand.Text(), Text: text, CreatedAt: time.Now().UTC()}
	seen := make(map[string]bool)
	for _, path := range paths {
		encoded, _ := json.Marshal(path)
		if !validPath(path) || len(encoded) > 384 || seen[path] {
			return Note{}, errors.New("evidence paths must be unique canonical project-relative paths")
		}
		seen[path] = true
		hash, err := l.hash(path)
		if err != nil {
			return Note{}, err
		}
		note.Evidence = append(note.Evidence, Evidence{Path: path, SHA256: hash})
	}
	err := l.update(func(notes []Note) ([]Note, error) {
		if len(notes) >= maxNotes {
			return nil, errors.New("memory ledger is full")
		}
		return append(notes, note), nil
	})
	return note, err
}

func (l *Ledger) Delete(id string) error {
	return l.update(func(notes []Note) ([]Note, error) {
		for i, note := range notes {
			if note.ID == id {
				return slices.Delete(notes, i, i+1), nil
			}
		}
		return nil, errors.New("memory note not found")
	})
}

func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func ValidateQuery(query string) error {
	if len(query) > 256 || !utf8.ValidString(query) || len(words(query)) == 0 {
		return errors.New("query must contain searchable text up to 256 bytes")
	}
	return nil
}

// Recall ranks lexical matches, then checks live file hashes only for candidates
// it may return. A query reads at most 12 distinct evidence files.
func (l *Ledger) Recall(ctx context.Context, query string) ([]Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateQuery(query); err != nil {
		return nil, err
	}
	notes, err := l.List()
	if err != nil {
		return nil, err
	}
	terms := make(map[string]bool)
	for _, term := range words(query) {
		terms[term] = true
	}
	type match struct {
		note  Note
		score int
	}
	var ranked []match
	for _, note := range notes {
		matched := make(map[string]bool)
		matchText := func(text string) {
			for _, word := range words(text) {
				if terms[word] {
					matched[word] = true
				}
			}
		}
		matchText(note.Text)
		for _, file := range note.Evidence {
			matchText(file.Path)
		}
		if len(matched) > 0 {
			ranked = append(ranked, match{note: note, score: len(matched)})
		}
	}
	slices.SortStableFunc(ranked, func(a, b match) int {
		if a.score != b.score {
			return b.score - a.score
		}
		return b.note.CreatedAt.Compare(a.note.CreatedAt)
	})
	hashes := make(map[string]string)
	found := make([]Note, 0, 3)
	for _, item := range ranked {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fresh := len(item.note.Evidence) > 0
		for _, file := range item.note.Evidence {
			hash, checked := hashes[file.Path]
			if !checked && len(hashes) < 12 {
				hash, _ = l.hash(file.Path)
				hashes[file.Path] = hash
			}
			if hash == "" || hash != file.SHA256 {
				fresh = false
				break
			}
		}
		if fresh {
			found = append(found, item.note)
			if len(found) == 3 {
				break
			}
		}
	}
	return found, ctx.Err()
}
