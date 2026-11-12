package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

const PortableFormat = "independent-harness/session"
const PortableVersion = 1
const maxExportBytes = 64 << 20

type Portable struct {
	Format          string    `json:"format"`
	Version         int       `json:"version"`
	SourceSessionID string    `json:"source_session_id"`
	Project         string    `json:"project"`
	ExportedAt      time.Time `json:"exported_at"`
	Pending         []string  `json:"pending_effects,omitempty"`
	Records         []Record  `json:"records"`
}

// portableRecords intentionally excludes permissions, executable integration
// configuration, process starts and their effect records. No exported record
// grants authority. Pending effects are reported separately and cannot import.
func portableRecords(records []Record) []Record {
	var out []Record
	for _, record := range records {
		switch record.Type {
		case "message":
			out = append(out, repairToolArguments(record))
		case "context_compacted", "todos_updated":
			out = append(out, record)
		}
	}
	return out
}

// repairToolArguments replaces tool-call arguments that are not a JSON
// object with "{}", as the runtime has done before journaling since rc.171.
// Sessions saved earlier can hold a malformed call (never executed); without
// this they export but cannot be forked, imported or salvaged.
func repairToolArguments(record Record) Record {
	var message core.Message
	if json.Unmarshal(record.Data, &message) != nil || len(message.ToolCalls) == 0 {
		return record
	}
	changed := false
	for i, call := range message.ToolCalls {
		var object map[string]json.RawMessage
		if json.Unmarshal([]byte(call.Function.Arguments), &object) == nil && object != nil {
			continue
		}
		message.ToolCalls[i].Function.Arguments = "{}"
		changed = true
	}
	if !changed {
		return record
	}
	data, err := json.Marshal(message)
	if err != nil {
		return record
	}
	record.Data = data
	return record
}

// Export snapshots a session under its writer lock. Content is unredacted.
// An interrupted session can be exported for inspection but cannot be imported
// or forked until its uncertain effects and transcript are explicitly resolved.
func Export(dir, id string, w io.Writer) error {
	store, err := Open(dir, id)
	if err != nil {
		return err
	}
	defer store.Close()
	portable := Portable{Format: PortableFormat, Version: PortableVersion, SourceSessionID: id, Project: store.Project, ExportedAt: time.Now().UTC(), Pending: store.Pending(), Records: portableRecords(store.Records())}
	data, err := json.MarshalIndent(portable, "", "  ")
	if err != nil {
		return err
	}
	if len(data)+1 > maxExportBytes {
		return errors.New("portable session exceeds 64 MiB limit")
	}
	data = append(data, '\n')
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

// Import accepts only this project's explicitly versioned portable format.
// Every imported session receives a fresh identity. An explicit project override
// permits moving an archive to a different existing local project directory.
func Import(dir string, r io.Reader, projectOverride string) (*Store, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxExportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxExportBytes {
		return nil, errors.New("portable session exceeds 64 MiB limit")
	}
	var portable Portable
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&portable); err != nil {
		return nil, fmt.Errorf("invalid portable session (OpenCode exports are not supported): %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("portable session has trailing content")
	}
	if portable.Format != PortableFormat || portable.Version != PortableVersion {
		return nil, fmt.Errorf("unsupported portable session format/version; expected %s version %d", PortableFormat, PortableVersion)
	}
	if !validID.MatchString(portable.SourceSessionID) {
		return nil, errors.New("invalid source session identity")
	}
	if len(portable.Pending) > 0 {
		return nil, errors.New("archive contains uncertain effects; resolve the source session before importing")
	}
	for i, record := range portable.Records {
		if record.Type == "message" {
			portable.Records[i] = repairToolArguments(record)
		}
	}
	if err := validatePortableRecords(portable.Records); err != nil {
		return nil, err
	}
	project := portable.Project
	if projectOverride != "" {
		project = projectOverride
	}
	if !filepath.IsAbs(project) {
		return nil, errors.New("import project must be an absolute existing directory; use --project to relocate")
	}
	return derive(dir, project, portable.Records, "session_imported", portable.SourceSessionID, 0)
}

// Fork copies completed conversation, compaction and task state. It never
// replays tools or transfers permission decisions, and refuses uncertain effects.
func Fork(dir, id string) (*Store, error) {
	store, _, err := ForkBefore(dir, id, 0)
	return store, err
}

// ForkBefore branches immediately before a one-based journal user turn and
// returns that turn's prompt. Zero copies the full session.
func ForkBefore(dir, id string, beforeTurn int) (*Store, string, error) {
	if beforeTurn < 0 {
		return nil, "", errors.New("fork turn must be positive, or zero for the full session")
	}
	source, err := Open(dir, id)
	if err != nil {
		return nil, "", err
	}
	defer source.Close()
	if len(source.Pending()) > 0 {
		return nil, "", errors.New("cannot fork uncertain tool effects; inspect and acknowledge the source session first")
	}
	records, prompt := source.records, ""
	if beforeTurn > 0 {
		turn := 0
		for i, record := range records {
			if !isUserRecord(record) {
				continue
			}
			turn++
			if turn == beforeTurn {
				prompt = userRecordContent(record)
				records = records[:i]
				break
			}
		}
		if turn < beforeTurn {
			return nil, "", fmt.Errorf("no user turn %d in source session", beforeTurn)
		}
	}
	records = portableRecords(records)
	if err := validatePortableRecords(records); err != nil {
		return nil, "", fmt.Errorf("source transcript cannot be forked: %w", err)
	}
	store, err := derive(dir, source.Project, records, "session_forked", id, beforeTurn)
	return store, prompt, err
}

func validatePortableRecords(records []Record) error {
	if len(records) > 100000 {
		return errors.New("portable session exceeds 100000 state records")
	}
	var history []core.Message
	for i, record := range records {
		if !validVersion(record) {
			return fmt.Errorf("record %d has unsupported journal version", i)
		}
		if len(record.Data) > maxRecordBytes {
			return fmt.Errorf("record %d exceeds record limit", i)
		}
		switch record.Type {
		case "message":
			var message core.Message
			if err := strictJSON(record.Data, &message); err != nil {
				return fmt.Errorf("record %d message: %w", i, err)
			}
			if message.Role == "system" {
				return fmt.Errorf("record %d: standalone system messages cannot be imported", i)
			}
			if err := validateMessage(message); err != nil {
				return fmt.Errorf("record %d: %w", i, err)
			}
			history = append(history, message)
		case "context_compacted":
			var state Compaction
			if err := strictJSON(record.Data, &state); err != nil {
				return fmt.Errorf("record %d context: %w", i, err)
			}
			if err := state.VerifyReceipt(); err != nil {
				return fmt.Errorf("record %d context: %w", i, err)
			}
			if len(state.Messages) == 0 || state.Messages[0].Role != "system" {
				return fmt.Errorf("record %d: compaction requires system framing", i)
			}
			for _, message := range state.Messages {
				if err := validateMessage(message); err != nil {
					return fmt.Errorf("record %d context: %w", i, err)
				}
			}
			if err := validateGroups(history); err != nil {
				return fmt.Errorf("record %d preceding context: %w", i, err)
			}
			if err := validateGroups(state.Messages); err != nil {
				return fmt.Errorf("record %d compacted context: %w", i, err)
			}
			history = state.Messages
		case "todos_updated":
			var state struct {
				Todos []struct {
					Content  string `json:"content"`
					Status   string `json:"status"`
					Priority string `json:"priority"`
				} `json:"todos"`
			}
			if err := strictJSON(record.Data, &state); err != nil {
				return fmt.Errorf("record %d task state: %w", i, err)
			}
			if state.Todos == nil {
				return fmt.Errorf("record %d task state requires a todos array", i)
			}
			if len(state.Todos) > 1000 {
				return fmt.Errorf("record %d task state exceeds 1000 items", i)
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(record.Data, &envelope); err != nil {
				return err
			}
			if len(envelope["todos"]) > 60<<10 {
				return fmt.Errorf("record %d task state exceeds 60KiB", i)
			}
			var tasks []map[string]json.RawMessage
			if err := json.Unmarshal(envelope["todos"], &tasks); err != nil {
				return err
			}
			for _, task := range tasks {
				if len(task) != 3 {
					return fmt.Errorf("record %d each task requires content, status and priority", i)
				}
				for _, key := range []string{"content", "status", "priority"} {
					raw, ok := task[key]
					var value string
					if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil {
						return fmt.Errorf("record %d task %s must be a string", i, key)
					}
				}
			}
		default:
			return fmt.Errorf("record %d type %q cannot be imported as state", i, record.Type)
		}
	}
	return validateGroups(history)
}
func strictJSON(data []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}
func validateMessage(message core.Message) error {
	switch message.Role {
	case "user", "assistant", "tool", "system":
	default:
		return fmt.Errorf("unsupported message role %q", message.Role)
	}
	if message.Role != "assistant" && len(message.ToolCalls) > 0 {
		return errors.New("only assistant messages may contain tool calls")
	}
	if message.Role == "tool" && message.ToolCallID == "" {
		return errors.New("tool result has no call ID")
	}
	if message.Role != "tool" && message.ToolCallID != "" {
		return errors.New("tool_call_id on a non-tool message")
	}
	seen := map[string]bool{}
	for _, call := range message.ToolCalls {
		if call.ID == "" || seen[call.ID] || call.Type != "function" || call.Function.Name == "" || !json.Valid([]byte(call.Function.Arguments)) {
			return errors.New("invalid or duplicate tool call")
		}
		seen[call.ID] = true
	}
	return nil
}
func validateGroups(history []core.Message) error {
	pending := map[string]bool{}
	for _, message := range history {
		if message.Role == "tool" {
			if !pending[message.ToolCallID] {
				return fmt.Errorf("unpaired tool result %q", message.ToolCallID)
			}
			delete(pending, message.ToolCallID)
			continue
		}
		if len(pending) > 0 {
			return errors.New("incomplete tool-call group; resolve interrupted source history before copying")
		}
		for _, call := range message.ToolCalls {
			pending[call.ID] = true
		}
	}
	if len(pending) > 0 {
		return errors.New("incomplete tool-call group; resolve interrupted source history before copying")
	}
	return nil
}

// derive builds the complete new journal privately, then publishes a hard link
// without overwriting an existing destination. The writer lock stays attached
// to the same inode throughout publication, and no partially copied session is
// visible to List or Open.
func derive(dir, project string, records []Record, kind, sourceID string, beforeTurn int) (result *Store, err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(dir, ".session-stage-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	store, err := New(staging, project)
	if err != nil {
		return nil, err
	}
	published := ""
	defer func() {
		if err != nil {
			store.Close()
			if published != "" {
				os.Remove(published)
			}
		}
	}()
	origin := map[string]any{"source_session_id": sourceID, "time": time.Now().UTC()}
	if beforeTurn > 0 {
		origin["before_turn"] = beforeTurn
	}
	if err = store.Append(kind, origin); err != nil {
		return nil, err
	}
	for _, record := range records {
		store.mu.Lock()
		err = store.appendAt(record.Type, record.Data, record.Time)
		store.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	destination := filepath.Join(dir, store.ID+".jsonl")
	if err = os.Link(filepath.Join(staging, store.ID+".jsonl"), destination); err != nil {
		return nil, fmt.Errorf("publish derived session: %w", err)
	}
	published = destination
	directory, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	err = platform.SyncDir(directory)
	directory.Close()
	if err != nil {
		return nil, err
	}
	return store, nil
}
