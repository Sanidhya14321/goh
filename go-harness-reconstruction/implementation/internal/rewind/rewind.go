// Package rewind implements turn-level undo and redo for chat. Undo restores
// the file checkpoints recorded during the latest user turn and continues in a
// new branch that ends before that turn; the source journal is never edited.
// Redo reapplies those checkpoints and returns to the source session, and is
// available only until a new prompt is sent from the branch.
package rewind

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/snapshots"
)

// originRecord is appended to an undo branch so redo can find its source.
const originRecord = "undo_origin"

type origin struct {
	Source      string   `json:"source"`
	Turn        int      `json:"turn"`
	LastTurn    int      `json:"last_turn,omitempty"`
	Checkpoints []string `json:"checkpoints"`
}

// Result describes what an undo or redo changed and what it could not.
type Result struct {
	Session string `json:"session"` // session chat should continue in
	Prompt  string `json:"prompt,omitempty"`
	Turn    int    `json:"turn"`
	// LastTurn is set when a revert covered later turns too (Turn..LastTurn).
	LastTurn int `json:"last_turn,omitempty"`
	// Files lists project paths restored from checkpoints.
	Files []string `json:"files,omitempty"`
	// Unrestorable names effectful tools from the turn that have no checkpoint
	// (shell, MCP, unchecked edits). Their effects remain in the workspace.
	Unrestorable []string `json:"unrestorable,omitempty"`
}

func snapshotDir(dir, id string) string { return filepath.Join(dir, id+"-snapshots") }

type turnState struct {
	turn, last   int
	prompt       string
	checkpoints  []string // capture order
	unrestorable []string
}

// lastTurn locates the latest user turn and the effects recorded after it.
func lastTurn(records []session.Record) (turnState, error) { return turnFrom(records, 0) }

// userPrompts lists the journal's user prompts with their record indexes.
func userPrompts(records []session.Record) (idx []int, prompts []string) {
	for i, r := range records {
		if r.Type != "message" {
			continue
		}
		var m struct{ Role, Content string }
		if json.Unmarshal(r.Data, &m) == nil && m.Role == "user" {
			idx, prompts = append(idx, i), append(prompts, m.Content)
		}
	}
	return idx, prompts
}

// turnFrom locates user turn n (1-based; 0 means the latest) and every effect
// recorded from it to the end of the journal, so reverting it also reverts
// all later turns.
func turnFrom(records []session.Record, n int) (turnState, error) {
	var st turnState
	idx, prompts := userPrompts(records)
	if len(idx) == 0 {
		return st, errors.New("nothing to undo: this session has no prompts yet")
	}
	if n == 0 {
		n = len(idx)
	}
	if n < 1 || n > len(idx) {
		return st, fmt.Errorf("turn %d does not exist", n)
	}
	st.turn, st.last, st.prompt = n, len(idx), prompts[n-1]
	start := idx[n-1]
	captured := map[string]bool{}
	started := []struct{ id, name string }{}
	for _, r := range records[start+1:] {
		var d struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ToolCallID string `json:"tool_call_id"`
		}
		json.Unmarshal(r.Data, &d)
		switch r.Type {
		case "snapshot_captured":
			st.checkpoints = append(st.checkpoints, d.ID)
			captured[d.ToolCallID] = true
		case "tool_start":
			started = append(started, struct{ id, name string }{d.ID, d.Name})
		}
	}
	seen := map[string]bool{}
	for _, s := range started {
		if !captured[s.id] && !seen[s.name] {
			seen[s.name] = true
			st.unrestorable = append(st.unrestorable, s.name)
		}
	}
	sort.Strings(st.unrestorable)
	return st, nil
}

func restoredFiles(rs []snapshots.Record) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range rs {
		for _, e := range r.Before {
			if !seen[e.Path] {
				seen[e.Path] = true
				out = append(out, e.Path)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Undo reverts the latest turn of session id. Checkpoints are undone newest
// first; if any refuses (for example a file changed outside GoHarness), the
// ones already undone are redone so files and conversation stay consistent.
func Undo(ctx context.Context, dir, id string) (Result, error) {
	return undoFrom(ctx, dir, id, func(records []session.Record) (turnState, error) { return lastTurn(records) })
}

// RevertTo reverts the turn whose prompt is prompt and every later turn,
// as OpenCode's "revert to message". occurrence picks among identical prompts,
// counted from the latest (0 is the latest such prompt). Matching by content,
// not position, means a stale or compacted UI view can never select the wrong
// turn: a prompt that no longer exists is refused. Redo restores everything.
func RevertTo(ctx context.Context, dir, id, prompt string, occurrence int) (Result, error) {
	return undoFrom(ctx, dir, id, func(records []session.Record) (turnState, error) {
		_, prompts := userPrompts(records)
		seen := 0
		for i := len(prompts) - 1; i >= 0; i-- {
			if prompts[i] != prompt {
				continue
			}
			if seen == occurrence {
				return turnFrom(records, i+1)
			}
			seen++
		}
		return turnState{}, errors.New("that prompt is not in this session's journal; nothing reverted")
	})
}

func undoFrom(ctx context.Context, dir, id string, locate func([]session.Record) (turnState, error)) (Result, error) {
	source, err := session.Open(dir, id)
	if err != nil {
		return Result{}, err
	}
	if len(source.Pending()) > 0 {
		source.Close()
		return Result{}, errors.New("cannot undo while tool effects are uncertain; inspect and acknowledge them first")
	}
	st, err := locate(source.Records())
	project := source.Project
	source.Close()
	if err != nil {
		return Result{}, err
	}
	var undone []snapshots.Record
	if len(st.checkpoints) > 0 {
		manager, err := snapshots.Open(project, snapshotDir(dir, id))
		if err != nil {
			return Result{}, fmt.Errorf("open file checkpoints: %w", err)
		}
		defer manager.Close()
		for i := len(st.checkpoints) - 1; i >= 0; i-- {
			r, err := manager.Undo(ctx, st.checkpoints[i])
			if err != nil {
				if rollbackErr := redoAll(ctx, manager, undone); rollbackErr != nil {
					return Result{}, fmt.Errorf("undo stopped at checkpoint %s (%v) and restoring the earlier files also failed: %w; inspect with `goh changes list %s`", st.checkpoints[i], err, rollbackErr, id)
				}
				return Result{}, fmt.Errorf("undo refused, no files changed: %w", err)
			}
			undone = append(undone, r)
		}
	}
	branch, prompt, err := session.ForkBefore(dir, id, st.turn)
	if err == nil {
		err = branch.Append(originRecord, origin{Source: id, Turn: st.turn, LastTurn: max(st.last, st.turn), Checkpoints: st.checkpoints})
		if err == nil {
			err = branch.Close()
		} else {
			branch.Close()
		}
	}
	if err != nil {
		if len(undone) > 0 {
			manager, openErr := snapshots.Open(project, snapshotDir(dir, id))
			if openErr == nil {
				openErr = redoAll(ctx, manager, undone)
				manager.Close()
			}
			if openErr != nil {
				return Result{}, fmt.Errorf("conversation branch failed (%v) after files were restored, and reapplying them failed: %w", err, openErr)
			}
		}
		return Result{}, fmt.Errorf("undo failed, files reapplied: %w", err)
	}
	result := Result{Session: branch.ID, Prompt: prompt, Turn: st.turn, Files: restoredFiles(undone), Unrestorable: st.unrestorable}
	if st.last > st.turn {
		result.LastTurn = st.last
	}
	return result, nil
}

// redoAll reapplies undone checkpoints in their original capture order.
func redoAll(ctx context.Context, m *snapshots.Manager, undone []snapshots.Record) error {
	for i := len(undone) - 1; i >= 0; i-- {
		if _, err := m.Redo(ctx, undone[i].ID); err != nil {
			return err
		}
	}
	return nil
}

// Redo returns from an undo branch to its source session and reapplies the
// files. It refuses once the branch has new prompts, since redo would then
// discard work done in the branch.
func Redo(ctx context.Context, dir, branchID string) (Result, error) {
	branch, err := session.Open(dir, branchID)
	if err != nil {
		return Result{}, err
	}
	records := branch.Records()
	branch.Close()
	var o *origin
	newPrompt := false
	for _, r := range records {
		if r.Type == originRecord {
			o = &origin{}
			if err := json.Unmarshal(r.Data, o); err != nil {
				return Result{}, fmt.Errorf("undo record is damaged: %w", err)
			}
			newPrompt = false
			continue
		}
		if o != nil && r.Type == "message" {
			var m struct{ Role string }
			if json.Unmarshal(r.Data, &m) == nil && m.Role == "user" {
				newPrompt = true
			}
		}
	}
	if o == nil {
		return Result{}, errors.New("nothing to redo: this session was not created by /undo")
	}
	if newPrompt {
		return Result{}, fmt.Errorf("redo is unavailable after a new prompt; the original conversation is still in /sessions (%s)", o.Source)
	}
	source, err := session.Open(dir, o.Source)
	if err != nil {
		return Result{}, fmt.Errorf("original session unavailable: %w", err)
	}
	project := source.Project
	source.Close()
	var redone []snapshots.Record
	if len(o.Checkpoints) > 0 {
		manager, err := snapshots.Open(project, snapshotDir(dir, o.Source))
		if err != nil {
			return Result{}, fmt.Errorf("open file checkpoints: %w", err)
		}
		defer manager.Close()
		for i, id := range o.Checkpoints {
			r, err := manager.Redo(ctx, id)
			if err != nil {
				// Put back what this redo reapplied, newest first.
				for j := len(redone) - 1; j >= 0; j-- {
					if _, undoErr := manager.Undo(ctx, redone[j].ID); undoErr != nil {
						return Result{}, fmt.Errorf("redo stopped at checkpoint %s (%v) and reverting earlier files failed: %w", o.Checkpoints[i], err, undoErr)
					}
				}
				return Result{}, fmt.Errorf("redo refused, no files changed: %w", err)
			}
			redone = append(redone, r)
		}
	}
	result := Result{Session: o.Source, Turn: o.Turn, Files: restoredFiles(redone)}
	if o.LastTurn > o.Turn {
		result.LastTurn = o.LastTurn
	}
	return result, nil
}

// Summary is the chat notice for a result.
func (r Result) Summary(undo bool) string {
	verb := "Redid"
	if undo {
		verb = "Undid"
	}
	text := fmt.Sprintf("%s turn %d.", verb, r.Turn)
	if r.LastTurn > r.Turn {
		text = fmt.Sprintf("%s turns %d–%d.", verb, r.Turn, r.LastTurn)
	}
	switch n := len(r.Files); n {
	case 0:
		text += " No checkpointed files changed."
	case 1:
		text += " Restored " + r.Files[0] + "."
	default:
		text += fmt.Sprintf(" Restored %d files.", n)
	}
	if len(r.Unrestorable) > 0 {
		text += fmt.Sprintf(" Not restorable (no checkpoint): %v; those effects remain in the workspace.", r.Unrestorable)
	}
	if undo {
		text += " The prompt is back in the composer. /redo returns until you send a new prompt."
	}
	return text
}
