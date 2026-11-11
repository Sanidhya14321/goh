package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Damaged describes a session journal that cannot be opened.
type Damaged struct {
	ID      string `json:"id"`
	Problem string `json:"problem"`
}

// ListAll returns readable sessions and, separately, journals that could not
// be read. One damaged journal never hides the others.
func ListAll(dir string) ([]Info, []Damaged, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Info{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	infos := []Info{} // an empty list, not null, in JSON output
	var damaged []Damaged
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".jsonl")
		if !validID.MatchString(id) {
			continue
		}
		info, ok, err := readHeader(dir, id)
		switch {
		case err != nil:
			damaged = append(damaged, Damaged{ID: id, Problem: err.Error()})
		case ok:
			infos = append(infos, info)
		}
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })
	sort.Slice(damaged, func(i, j int) bool { return damaged[i].ID < damaged[j].ID })
	return infos, damaged, nil
}

// readHeader validates the first record. ok is false for a journal still
// being created (no committed header yet).
func readHeader(dir, id string) (Info, bool, error) {
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
	if err != nil {
		return Info{}, false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), maxRecordBytes)
	if !scanner.Scan() {
		return Info{}, false, scanner.Err()
	}
	var record Record
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		return Info{}, false, fmt.Errorf("header corruption: %w", err)
	}
	if !validVersion(record) {
		return Info{}, false, fmt.Errorf("unsupported journal version %d", record.Version)
	}
	var info Info
	if record.Type != "session_start" || json.Unmarshal(record.Data, &info) != nil || info.ID != id || !filepath.IsAbs(info.Project) {
		return Info{}, false, errors.New("invalid identity")
	}
	return info, true, nil
}

// SalvageReport says what a salvage kept.
type SalvageReport struct {
	Source       string `json:"source"`
	Session      string `json:"session"`
	KeptRecords  int    `json:"kept_records"`
	DamagedAt    int64  `json:"damaged_at_byte"`
	Problem      string `json:"problem"`
	DroppedGroup int    `json:"dropped_trailing_records,omitempty"`
}

// Salvage copies the readable conversation before the first damaged record
// into a new session. The damaged journal is left untouched for inspection.
// Like fork, it never replays tools or carries over permission grants. If the
// readable part ends inside an incomplete tool group, that tail is dropped.
func Salvage(dir, id string) (*Store, SalvageReport, error) {
	report := SalvageReport{Source: id}
	if !validID.MatchString(id) {
		return nil, report, errors.New("invalid session ID")
	}
	f, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
	if err != nil {
		return nil, report, err
	}
	records, offset, problem := readPrefix(f)
	f.Close()
	report.DamagedAt, report.Problem = offset, problem
	if problem == "" {
		return nil, report, errors.New("journal is not damaged; open it normally")
	}
	if len(records) == 0 {
		return nil, report, errors.New("the session header is damaged, so its project cannot be recovered")
	}
	var info Info
	if records[0].Type != "session_start" || json.Unmarshal(records[0].Data, &info) != nil || info.ID != id || !filepath.IsAbs(info.Project) {
		return nil, report, errors.New("the session header is damaged, so its project cannot be recovered")
	}
	kept := portableRecords(records)
	for len(kept) > 0 && validatePortableRecords(kept) != nil {
		kept = kept[:len(kept)-1]
		report.DroppedGroup++
	}
	if len(kept) == 0 {
		return nil, report, errors.New("no complete conversation could be recovered before the damage")
	}
	store, err := derive(dir, info.Project, kept, "session_salvaged", id, 0)
	if err != nil {
		return nil, report, err
	}
	report.Session, report.KeptRecords = store.ID, len(kept)
	return store, report, nil
}

// readPrefix returns valid records up to the first damaged one, the byte
// offset of the damage, and a description ("" if the journal is intact apart
// from an unterminated tail).
func readPrefix(r io.Reader) ([]Record, int64, string) {
	reader := bufio.NewReaderSize(r, 64<<10)
	var records []Record
	var offset int64
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > maxRecordBytes {
			return records, offset, "record exceeds 16 MiB"
		}
		if err == io.EOF {
			return records, offset, "" // unterminated tail: normal crash repair
		}
		if err != nil {
			return records, offset, err.Error()
		}
		var record Record
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&record) != nil || decoder.Decode(new(any)) != io.EOF ||
			!validVersion(record) || strings.TrimSpace(record.Type) == "" || len(record.Data) == 0 {
			return records, offset, fmt.Sprintf("unreadable record at byte %d", offset)
		}
		records = append(records, record)
		offset += int64(len(line))
	}
}

// ListForPicker returns readable sessions followed by damaged ones (with
// Damaged set to the problem), so pickers can offer recovery.
func ListForPicker(dir string) ([]Info, error) {
	infos, damaged, err := ListAll(dir)
	if err != nil {
		return nil, err
	}
	for _, d := range damaged {
		problem := d.Problem
		if problem == "" {
			problem = "unreadable"
		}
		infos = append(infos, Info{ID: d.ID, Damaged: problem})
	}
	return infos, nil
}
