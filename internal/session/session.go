// Package session provides a single-writer, crash-recoverable JSONL journal.
// A tool start must be synchronized before execution; unfinished starts are
// reported as uncertain effects and are never replayed by this package.
package session

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Version 2 records carry Time (Unix milliseconds); version 1 records, from
// older releases, have none. Both are read; new records are written as 2.
const Version = 2

// validVersion accepts a version 1 record without a time or a version 2 record
// with one.
func validVersion(r Record) bool {
	return (r.Version == 1 && r.Time == 0) || (r.Version == 2 && r.Time > 0)
}

const maxRecordBytes = 16 << 20

var validID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

type Record struct {
	Version int             `json:"version"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Time    int64           `json:"time,omitempty"`
}
type Info struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	// Damaged is set only by ListForPicker for journals that cannot be read.
	Damaged string `json:"damaged,omitempty"`
}
type Store struct {
	ID      string
	Project string
	mu      sync.Mutex
	file    *os.File
	records []Record
	failed  error
	// readOnly marks a view opened with OpenReadOnly: another process
	// owns the journal, and every write is refused.
	readOnly bool
}

// ErrReadOnly is returned for writes to a read-only session view.
var ErrReadOnly = errors.New("session is open read-only here; it is written by the GoHarness server it is attached to")

// OpenReadOnly reads a journal that another process may be writing (an
// attached `goh serve`), without taking the writer lock or repairing it. A
// torn tail is ignored rather than truncated. The view never writes.
func OpenReadOnly(dir, id, project string) (*Store, error) {
	if !validID.MatchString(id) {
		return nil, errors.New("invalid session ID")
	}
	reconcileMoves(dir) // an interrupted trash or restore is settled first
	f, err := openJournal(dir, id, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("session journal must be a regular file")
	}
	records, _, _, err := readRecords(f)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDamaged, err)
	}
	if len(records) == 0 || records[0].Type != "session_start" {
		return nil, errors.New("journal missing session_start")
	}
	var header Info
	if err := json.Unmarshal(records[0].Data, &header); err != nil || header.ID != id || !filepath.IsAbs(header.Project) {
		return nil, errors.New("invalid session identity")
	}
	if project != "" && header.Project != project {
		return nil, ErrProjectMismatch
	}
	return &Store{ID: id, Project: header.Project, records: records, readOnly: true}, nil
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
	err = platform.SyncDir(d)
	d.Close()
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("sync session directory: %w", err)
	}
	return s, nil
}

var ErrBusy = errors.New("session already has an active writer")
var ErrProjectMismatch = errors.New("session belongs to another project")

// ErrDamaged marks a journal whose records cannot be read beyond a torn tail.
// Salvage can recover the readable conversation into a new session.
var ErrDamaged = errors.New("session journal is damaged")

// ErrNotFound is returned when no journal exists for a session ID.
var ErrNotFound = errors.New("no saved session with this ID (goh sessions list shows them)")

// openJournal opens a journal file without following a final symlink and
// reports a missing journal as ErrNotFound, naming the ID.
func openJournal(dir, id string, flags int) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), flags|platform.NoFollow|platform.NonBlock, 0600)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return f, err
}

func lock(f *os.File) error {
	if err := platform.TryLock(f); err != nil {
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
	// Every entry point settles interrupted trash and restore moves first,
	// so a session never opens with its title or artifacts left behind.
	reconcileMoves(dir)
	// O_NOFOLLOW prevents replacing the journal with a symlink into another path.
	f, err := openJournal(dir, id, os.O_RDWR)
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
		return fail(fmt.Errorf("%w: %w", ErrDamaged, err))
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
// The journal is read in one buffer and each record's Data is a slice of it.
func readRecords(r io.Reader) ([]Record, int64, bool, error) {
	var buf bytes.Buffer
	if f, ok := r.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.Size() > 0 {
			buf.Grow(int(info.Size()) + bytes.MinRead)
		}
	}
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, 0, false, err
	}
	content := buf.Bytes()
	records := make([]Record, 0, bytes.Count(content, []byte{'\n'}))
	var offset int64
	for len(content) > 0 {
		end := bytes.IndexByte(content, '\n')
		if end < 0 {
			if len(content) > maxRecordBytes {
				return nil, offset, false, errors.New("journal record exceeds 16 MiB")
			}
			return records, offset, true, nil
		}
		if end+1 > maxRecordBytes {
			return nil, offset, false, errors.New("journal record exceeds 16 MiB")
		}
		line := content[: end+1 : end+1]
		record, ok := fastRecord(line[:end])
		if !ok {
			decoder := json.NewDecoder(bytes.NewReader(line))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&record); err != nil {
				return nil, offset, false, fmt.Errorf("journal corruption at byte %d: %w", offset, err)
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return nil, offset, false, fmt.Errorf("journal corruption at byte %d: trailing content", offset)
			}
		}
		if !validVersion(record) {
			return nil, offset, false, fmt.Errorf("unsupported journal version %d at byte %d", record.Version, offset)
		}
		if strings.TrimSpace(record.Type) == "" || len(record.Data) == 0 {
			return nil, offset, false, fmt.Errorf("invalid journal record at byte %d", offset)
		}
		records = append(records, record)
		offset += int64(end + 1)
		content = content[end+1:]
	}
	return records, offset, false, nil
}

// fastRecord decodes a line in exactly the form appendAt writes,
// {"version":N,"type":"T","data":D[,"time":N]}, without a decoder per line.
// Anything else (escapes in the type, other key orders, spacing) reports
// false and goes through the strict decoder, which accepts the same records.
func fastRecord(line []byte) (Record, bool) {
	var r Record
	rest, ok := bytes.CutPrefix(line, []byte(`{"version":`))
	if !ok || len(rest) == 0 || (rest[0] != '1' && rest[0] != '2') {
		return r, false
	}
	r.Version = int(rest[0] - '0')
	if rest, ok = bytes.CutPrefix(rest[1:], []byte(`,"type":"`)); !ok {
		return r, false
	}
	quote := bytes.IndexByte(rest, '"')
	if quote < 0 {
		return r, false
	}
	for _, c := range rest[:quote] {
		if c < 0x20 || c >= 0x7f || c == '\\' {
			return r, false
		}
	}
	r.Type = string(rest[:quote])
	if rest, ok = bytes.CutPrefix(rest[quote+1:], []byte(`,"data":`)); !ok {
		return r, false
	}
	if rest, ok = bytes.CutSuffix(rest, []byte("}")); !ok {
		return r, false
	}
	if at := bytes.LastIndex(rest, []byte(`,"time":`)); at >= 0 {
		digits := rest[at+len(`,"time":`):]
		if t, ok := parseTime(digits); ok {
			r.Time, rest = t, rest[:at]
		}
	}
	if len(rest) == 0 || isJSONSpace(rest[0]) || isJSONSpace(rest[len(rest)-1]) || !json.Valid(rest) {
		return r, false
	}
	r.Data = json.RawMessage(rest[:len(rest):len(rest)])
	return r, true
}

func isJSONSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// parseTime reads a positive JSON integer with no leading zero.
func parseTime(b []byte) (int64, bool) {
	if len(b) == 0 || len(b) > 18 || b[0] == '0' {
		return 0, false
	}
	var t int64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		t = t*10 + int64(c-'0')
	}
	return t, true
}

func (s *Store) Append(kind string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendLocked(kind, data)
}

func (s *Store) appendLocked(kind string, data any) error {
	return s.appendAt(kind, data, time.Now().UnixMilli())
}

// appendAt writes a record with the given time; 0 writes a version 1 record
// (used when copying records from an older journal, keeping their absence of
// a time honest rather than inventing one).
func (s *Store) appendAt(kind string, data any, at int64) error {
	if s.readOnly {
		return ErrReadOnly
	}
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
	record := Record{Version: Version, Type: kind, Data: encoded, Time: at}
	if at <= 0 {
		record.Version, record.Time = 1, 0
	}
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

// List returns readable sessions. Damaged journals are skipped (see ListAll)
// so one bad file never hides the rest.
func List(dir string) ([]Info, error) {
	infos, _, err := ListAll(dir)
	return infos, err
}
