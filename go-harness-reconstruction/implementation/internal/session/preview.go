package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"example.com/independent-harness/internal/core"
)

const previewBytes = 64 << 10

type PreviewInfo struct {
	Info
	Title     string    `json:"title"`
	User      string    `json:"user,omitempty"`
	Assistant string    `json:"assistant,omitempty"`
	Status    string    `json:"status,omitempty"`
	Modified  time.Time `json:"modified"`
	Limited   bool      `json:"limited"`
}

// Preview reads complete records from a bounded journal prefix without taking
// writer ownership, repairing a torn tail, or replaying any effects.
func Preview(dir, id string) (PreviewInfo, error) {
	result := PreviewInfo{}
	if !validID.MatchString(id) {
		return result, errors.New("invalid session ID")
	}
	file, err := os.OpenFile(filepath.Join(dir, id+".jsonl"), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
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
		if err := json.Unmarshal(raw, &record); err != nil || record.Version != Version {
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
			result.Status = "unfinished run"
		case "message":
			var message core.Message
			if err := json.Unmarshal(record.Data, &message); err != nil {
				return result, errors.New("session preview encountered an invalid message")
			}
			switch message.Role {
			case "user":
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
			var finished struct{ Status string }
			if json.Unmarshal(record.Data, &finished) != nil {
				return result, errors.New("invalid run status in preview")
			}
			result.Status = finished.Status
		}
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
