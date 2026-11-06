package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/formatters"
	"example.com/independent-harness/internal/snapshots"
	"example.com/independent-harness/internal/tools"
)

var snapshotIDPattern = regexp.MustCompile(`^[a-f0-9]{24}$`)

func (a *Agent) changeDefinitions() []core.Tool {
	defs := []core.Tool{}
	empty := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
	if len(a.Config.Formatters) > 0 {
		defs = append(defs, core.NewTool("formatter_status", "Inspect explicitly configured local formatters without running them.", empty))
	}
	if a.Config.Snapshots {
		defs = append(defs, core.NewTool("snapshot_list", "List bounded file-edit checkpoints in this session; shell and unrelated formatter effects are not included.", empty))
		for _, item := range []struct{ name, description string }{
			{"snapshot_diff", "Review saved before/after file changes for a checkpoint."},
			{"snapshot_undo", "Restore checkpoint before-states only when all current file hashes/modes still match its after-states. Requires permission."},
			{"snapshot_redo", "Reapply checkpoint after-states only when all current file hashes/modes still match its before-states. Requires permission."},
		} {
			defs = append(defs, core.NewTool(item.name, item.description, map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}, "additionalProperties": false}))
		}
	}
	return defs
}
func parseChange(name string, raw json.RawMessage) (string, error) {
	if len(raw) > 1024 {
		return "", errors.New("change arguments exceed 1KiB")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return "", errors.New("change arguments must be an object")
	}
	if name == "formatter_status" || name == "snapshot_list" {
		if len(obj) != 0 {
			return "", errors.New("no arguments accepted")
		}
		return "", nil
	}
	if len(obj) != 1 || obj["id"] == nil {
		return "", errors.New("snapshot id is required")
	}
	var id string
	if err := json.Unmarshal(obj["id"], &id); err != nil || !snapshotIDPattern.MatchString(id) {
		return "", errors.New("invalid snapshot id")
	}
	return id, nil
}
func (a *Agent) prepareChange(name string, raw json.RawMessage) (string, bool, error) {
	id, err := parseChange(name, raw)
	if err != nil {
		return "", false, err
	}
	if name == "formatter_status" {
		if len(a.Config.Formatters) == 0 {
			return "", false, errors.New("no formatters configured")
		}
		return "Inspect local formatter configuration", false, nil
	}
	if !a.Config.Snapshots {
		return "", false, errors.New("snapshots are disabled")
	}
	side := name == "snapshot_undo" || name == "snapshot_redo"
	if !side {
		return "Inspect saved file checkpoint", false, nil
	}
	m, err := snapshots.Open(a.Store.Project, a.snapshotDirectory())
	if err != nil {
		return "", true, err
	}
	defer m.Close()
	changes, err := m.Diff(id)
	if err != nil {
		return "", true, err
	}
	preview, _ := json.MarshalIndent(changes, "", "  ")
	if len(preview) > tools.MaxInline {
		preview = append(preview[:tools.MaxInline-100], []byte("\n[preview truncated; inspect changes diff before approval]")...)
	}
	return fmt.Sprintf("%s %s: saved files only, with current-hash conflict checks; multiple files publish individually.\n%s", name, id, preview), true, nil
}
func (a *Agent) executeChange(ctx context.Context, name string, raw json.RawMessage) tools.Result {
	if _, _, err := a.prepareChange(name, raw); err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	if name == "formatter_status" {
		entries, err := formatters.Discover(a.Config.Formatters, a.Store.Project)
		if err != nil {
			return tools.Result{Error: true, Content: err.Error()}
		}
		status := []any{}
		for _, e := range entries {
			status = append(status, map[string]any{"name": e.Name, "extensions": e.Extensions, "enabled": e.Enabled, "available": e.Available, "reason": e.Reason})
		}
		return boundedResult(map[string]any{"formatters": status}, false)
	}
	a.snapshotMu.Lock()
	defer a.snapshotMu.Unlock()
	id, _ := parseChange(name, raw)
	m, err := snapshots.Open(a.Store.Project, a.snapshotDirectory())
	if err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	defer m.Close()
	var value any
	switch name {
	case "snapshot_list":
		records, e := m.List()
		err = e
		summaries := []any{}
		for _, r := range records {
			summaries = append(summaries, map[string]any{"id": r.ID, "created": r.Created, "status": r.Status, "files": len(r.Before)})
		}
		// Newest checkpoints stay visible without unbounded model context.
		truncated := len(summaries) > 100
		if truncated {
			summaries = summaries[len(summaries)-100:]
		}
		value = map[string]any{"snapshots": summaries, "truncated": truncated}
	case "snapshot_diff":
		value, err = m.Diff(id)
	case "snapshot_undo":
		value, err = m.Undo(ctx, id)
	case "snapshot_redo":
		value, err = m.Redo(ctx, id)
	}
	if err != nil {
		state := "failed"
		if errors.Is(err, snapshots.ErrUncertain) || ctx.Err() != nil {
			state = "interrupted"
		}
		return boundedResult(map[string]any{"status": state, "snapshot_id": id, "error": err.Error()}, true)
	}
	return boundedResult(map[string]any{"status": "completed", "result": value}, false)
}
func isChangeTool(name string) bool {
	switch name {
	case "formatter_status", "snapshot_list", "snapshot_diff", "snapshot_undo", "snapshot_redo":
		return true
	}
	return false
}
