package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

const previewBytes = 64 << 10

type PreviewInfo struct {
	Info
	Title     string `json:"title"`
	User      string `json:"user,omitempty"`
	Assistant string `json:"assistant,omitempty"`
	Status    string `json:"status,omitempty"`
	// Outcome is the latest run's verification outcome (verified,
	// unverified, no_changes, failed_verification) and RunID the run it
	// belongs to; empty while that run is unfinished or had no outcome.
	Outcome  string    `json:"outcome,omitempty"`
	RunID    string    `json:"run_id,omitempty"`
	Modified time.Time `json:"modified"`
	Limited  bool      `json:"limited"`
}

// Preview reads complete records from a bounded journal prefix without taking
// writer ownership, repairing a torn tail, or replaying any effects.
func Preview(dir, id string) (PreviewInfo, error) {
	result := PreviewInfo{}
	if !validID.MatchString(id) {
		return result, errors.New("invalid session ID")
	}
	file, err := openJournal(dir, id, os.O_RDONLY)
	if err != nil {
		return result, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return result, errors.New("session preview requires a regular file")
	}
	result.Modified = info.ModTime()
	data, err := io.ReadAll(io.LimitReader(file, previewBytes+1))
	if err != nil {
		return result, err
	}
	result.Limited = len(data) > previewBytes
	data = data[:min(len(data), previewBytes)]
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return result, errors.New("session preview has no complete header")
	}
	for index, raw := range bytes.Split(data[:end], []byte{'\n'}) {
		var record Record
		if err := json.Unmarshal(raw, &record); err != nil || !validVersion(record) {
			return result, errors.New("session preview encountered an invalid record")
		}
		if index == 0 {
			if record.Type != "session_start" || json.Unmarshal(record.Data, &result.Info) != nil || result.ID != id || !filepath.IsAbs(result.Project) {
				return result, errors.New("session preview has an invalid identity")
			}
			continue
		}
		switch record.Type {
		case "run_started":
			result.Status, result.Outcome, result.RunID = "unfinished run", "", ""
		case "message":
			var message core.Message
			if err := json.Unmarshal(record.Data, &message); err != nil {
				return result, errors.New("session preview encountered an invalid message")
			}
			switch message.Role {
			case "user":
				if message.Automatic {
					continue
				}
				if result.Title == "" && strings.TrimSpace(message.Content) != "" {
					result.Title = previewExcerpt(strings.Join(strings.Fields(message.Content), " "), 100)
				}
				result.User = previewExcerpt(message.Content, 800)
			case "assistant":
				if message.Content != "" {
					result.Assistant = previewExcerpt(message.Content, 1200)
				}
			}
		case "run_finished":
			var finished struct {
				Status  string
				Outcome string
				RunID   string `json:"run_id"`
			}
			if json.Unmarshal(record.Data, &finished) != nil {
				return result, errors.New("invalid run status in preview")
			}
			result.Status, result.Outcome, result.RunID = finished.Status, finished.Outcome, finished.RunID
		}
	}
	if result.Limited {
		// The prefix ends before the latest run: read the journal's tail
		// for its latest run status so the preview does not show a stale one.
		if st, ok := latestRunInTail(file, info.Size()); ok {
			result.Status, result.Outcome, result.RunID = st.Status, st.Outcome, st.RunID
		}
	}
	if m, err := ReadMeta(dir, id); err == nil && m.Title != "" {
		result.Title = m.Title // a saved (renamed) title, as in session lists
	}
	if result.Title == "" {
		result.Title = "New conversation"
	}
	result.Limited = result.Limited || end != len(data)-1
	return result, nil
}

func previewExcerpt(text string, limit int) string {
	value := []rune(text)
	if len(value) <= limit {
		return text
	}
	return string(value[:limit]) + "…"
}

type tailRun struct {
	Status, Outcome, RunID string
}

// latestRunInTail scans the last previewBytes of the journal for the latest
// run_started or run_finished record. Torn or invalid lines are skipped; it
// never writes.
func latestRunInTail(file *os.File, size int64) (tailRun, bool) {
	start := max(0, size-previewBytes)
	buf := make([]byte, size-start)
	if _, err := file.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
		return tailRun{}, false
	}
	if start > 0 {
		if i := bytes.IndexByte(buf, '\n'); i >= 0 {
			buf = buf[i+1:] // drop the partial first line
		}
	}
	var found tailRun
	ok := false
	for _, raw := range bytes.Split(buf, []byte{'\n'}) {
		var record Record
		if len(raw) == 0 || json.Unmarshal(raw, &record) != nil || !validVersion(record) {
			continue
		}
		switch record.Type {
		case "run_started":
			found, ok = tailRun{Status: "unfinished run"}, true
		case "run_finished":
			var d struct {
				Status  string
				Outcome string
				RunID   string `json:"run_id"`
			}
			if json.Unmarshal(record.Data, &d) == nil {
				found, ok = tailRun{Status: d.Status, Outcome: d.Outcome, RunID: d.RunID}, true
			}
		}
	}
	return found, ok
}
