// Package session provides a single-writer, crash-recoverable JSONL journal.
// A tool start must be synchronized before execution; unfinished starts are
// reported as uncertain effects and are never replayed by this package.
package session

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const Version = 1
const maxRecordBytes = 16 << 20

var validID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

type Record struct {
	Version int             `json:"version"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}
type Info struct {
	ID      string `json:"id"`
	Project string `json:"project"`
}
type Store struct {
	ID      string
	Project string
	mu      sync.Mutex
	file    *os.File
	records []Record
	failed  error
}

func New(dir, project string) (*Store, error) {
	root, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("project: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("project must be an existing directory")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	var entropy [12]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return nil, err
	}
	id := time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(entropy[:])
	path := filepath.Join(dir, id+".jsonl")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	if err = lock(f); err != nil {
		f.Close()
		return nil, err
	}
	s := &Store{ID: id, Project: root, file: f}
	if err := s.Append("session_start", Info{ID: id, Project: root}); err != nil {
		s.Close()
		return nil, err
	}
	// Persist the directory entry as well as the journal contents.
	d, err := os.Open(dir)
	if err != nil {
		s.Close()
		return nil, err
	}
	err = d.Sync()
	d.Close()
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("sync session directory: %w", err)
	}
	return s, nil
}

var ErrBusy = errors.New("session already has an active writer")
var ErrProjectMismatch = errors.New("session belongs to another project")

func lock(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("%w: %w", ErrBusy, err)
	}
	return nil
}

func Open(dir, id string) (*Store, error) {
	return open(dir, id, "")
}

// OpenProject rejects journals from other projects before repairing or changing them.
// project must be the canonical absolute project directory.
func OpenProject(dir, id, project string) (*Store, error) {
	if !filepath.IsAbs(project) {
		return nil, errors.New("project must be absolute")
	}
	return open(dir, id, project)
}

func open(dir, id, project string) (*Store, error) {
	if !validID.MatchString(id) {
		return nil, errors.New("invalid session ID")
	}
	path := filepath.Join(dir, id+".jsonl")
	// O_NOFOLLOW prevents replacing the journal with a symlink into another path.
	f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	if info, statErr := f.Stat(); statErr != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("session journal must be a regular file")
	}
	if err := lock(f); err != nil {
		f.Close()
		return nil, err
	}
	fail := func(err error) (*Store, error) { f.Close(); return nil, err }
	records, validBytes, torn, err := readRecords(f)
	if err != nil {
		return fail(err)
	}
	if len(records) == 0 || records[0].Type != "session_start" {
		return fail(errors.New("journal missing session_start"))
	}
	var header Info
	if err := json.Unmarshal(records[0].Data, &header); err != nil || header.ID != id || !filepath.IsAbs(header.Project) {
		return fail(errors.New("invalid session identity"))
	}
	if project != "" && header.Project != project {
		return fail(ErrProjectMismatch)
	}
	if err := f.Chmod(0600); err != nil {
		return fail(err)
	}
	if torn {
		if err := f.Truncate(validBytes); err != nil {
			return fail(err)
		}
		if err := f.Sync(); err != nil {
			return fail(err)
		}
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fail(err)
	}
	s := &Store{ID: id, Project: header.Project, file: f, records: records}
	if torn {
		if err := s.Append("recovery", map[string]any{"truncated_tail": true, "valid_bytes": validBytes}); err != nil {
			return fail(err)
		}
	}
	return s, nil
}

// readRecords accepts only a valid prefix followed by an unterminated tail.
// A newline-terminated malformed record, including the last one, is corruption.
func readRecords(r io.Reader) ([]Record, int64, bool, error) {
	reader := bufio.NewReaderSize(r, 64<<10)
	var records []Record
	var offset int64
	for {
		line, err := reader.ReadSlice('\n')
		var full []byte
		for err == bufio.ErrBufferFull {
			full = append(full, line...)
			if len(full) > maxRecordBytes {
				return nil, offset, false, errors.New("journal record exceeds 16 MiB")
			}
			line, err = reader.ReadSlice('\n')
		}
		if len(full) > 0 {
			line = append(full, line...)
		}
		if len(line) > maxRecordBytes {
			return nil, offset, false, errors.New("journal record exceeds 16 MiB")
		}
		if err == io.EOF {
			return records, offset, len(line) > 0, nil
		}
		if err != nil {
			return nil, offset, false, err
		}
		var record Record
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, offset, false, fmt.Errorf("journal corruption at byte %d: %w", offset, err)
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return nil, offset, false, fmt.Errorf("journal corruption at byte %d: trailing content", offset)
		}
		if record.Version != Version {
			return nil, offset, false, fmt.Errorf("unsupported journal version %d at byte %d", record.Version, offset)
		}
		if strings.TrimSpace(record.Type) == "" || len(record.Data) == 0 {
			return nil, offset, false, fmt.Errorf("invalid journal record at byte %d", offset)
		}
		records = append(records, record)
		offset += int64(len(line))
	}
}

func (s *Store) Append(kind string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendLocked(kind, data)
}

func (s *Store) appendLocked(kind string, data any) error {
	if s.file == nil {
		return errors.New("session closed")
	}
	if s.failed != nil {
		return fmt.Errorf("session journal unavailable after prior write failure: %w", s.failed)
	}
	if strings.TrimSpace(kind) == "" {
		return errors.New("empty record type")
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	record := Record{Version: Version, Type: kind, Data: encoded}
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if len(line)+1 > maxRecordBytes {
		return errors.New("journal record exceeds 16 MiB")
	}
	line = append(line, '\n')
	n, err := s.file.Write(line)
	if err == nil && n != len(line) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = s.file.Sync()
	}
	if err != nil {
		s.failed = err
		return fmt.Errorf("durable journal write: %w", err)
	}
	s.records = append(s.records, record)
	return nil
}

func (s *Store) Records() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyRecords := make([]Record, len(s.records))
	for i, r := range s.records {
		copyRecords[i] = r
		copyRecords[i].Data = append(json.RawMessage(nil), r.Data...)
	}
	return copyRecords
}

func (s *Store) Pending() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingLocked()
}

func (s *Store) pendingLocked() []string {
	pending := map[string]bool{}
	var order []string
	for _, record := range s.records {
		switch record.Type {
		case "recovery_ack":
			pending = map[string]bool{}
			order = nil
		case "tool_start", "tool_result":
			var effect struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(record.Data, &effect) != nil || effect.ID == "" {
				continue
			}
			if record.Type == "tool_start" {
				if !pending[effect.ID] {
					order = append(order, effect.ID)
				}
				pending[effect.ID] = true
			} else {
				delete(pending, effect.ID)
			}
		}
	}
	var out []string
	for _, id := range order {
		if pending[id] {
			out = append(out, id)
			delete(pending, id)
		}
	}
	return out
}

// Acknowledge records that a caller explicitly reviewed all uncertain effects.
// The caller, never this package, decides whether to permit continued execution.
func (s *Store) Acknowledge() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendLocked("recovery_ack", map[string]any{"uncertain_tool_ids": s.pendingLocked(), "time": time.Now().UTC()})
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	// Closing the descriptor releases its flock atomically with writer ownership.
	err := s.file.Close()
	s.file = nil
	return err
}

func List(dir string) ([]Info, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Info{}, nil
	}
	if err != nil {
		return nil, err
	}
	var result []Info
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".jsonl")
		if !validID.MatchString(id) {
			continue
		}
		f, err := os.OpenFile(filepath.Join(dir, entry.Name()), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 4096), maxRecordBytes)
		ok := scanner.Scan()
		scanErr := scanner.Err()
		first := append([]byte(nil), scanner.Bytes()...)
		f.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if !ok {
			continue
		} // An in-progress creation has not committed metadata yet.
		var record Record
		if err := json.Unmarshal(first, &record); err != nil {
			return nil, fmt.Errorf("session %s header corruption: %w", id, err)
		}
		if record.Version != Version {
			return nil, fmt.Errorf("session %s: unsupported journal version %d", id, record.Version)
		}
		var info Info
		if record.Type != "session_start" || json.Unmarshal(record.Data, &info) != nil || info.ID != id || !filepath.IsAbs(info.Project) {
			return nil, fmt.Errorf("session %s: invalid identity", id)
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
