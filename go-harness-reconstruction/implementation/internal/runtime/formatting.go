package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/formatters"
	"example.com/independent-harness/internal/snapshots"
	"example.com/independent-harness/internal/tools"
)

func (a *Agent) snapshotDirectory() string {
	return filepath.Join(a.Config.SessionDir, a.Store.ID+"-snapshots")
}
func boundedResult(v any, failed bool) tools.Result {
	b, err := json.Marshal(v)
	if err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	if len(b) > tools.MaxInline {
		return tools.Result{Error: true, Content: `{"status":"interrupted","error":"result metadata exceeds 64KiB; inspect journal and snapshot artifacts"}`}
	}
	return tools.Result{Content: string(b), Error: failed}
}
func (a *Agent) executeMutation(ctx context.Context, c core.Call, paths []string) tools.Result {
	entries, err := formatters.Discover(a.Config.Formatters, a.Store.Project)
	if err != nil {
		return tools.Result{Error: true, Content: err.Error()}
	}
	var manager *snapshots.Manager
	snapshotID, skippedCheckpoint := "", ""
	if a.Config.Snapshots {
		manager, err = snapshots.Open(a.Store.Project, a.snapshotDirectory())
		if err == nil {
			defer manager.Close()
			snapshotID, err = manager.Capture(ctx, paths)
		}
		switch {
		case err == nil:
		case snapshotID != "":
			return boundedResult(map[string]any{"status": "interrupted", "snapshot_id": snapshotID, "error": err.Error()}, true)
		case a.Config.SnapshotsStrict || ctx.Err() != nil:
			return tools.Result{Error: true, Content: err.Error()}
		default:
			// Default (best-effort) checkpoints never block an edit; undo will
			// list this change as not restorable.
			skippedCheckpoint = err.Error()
		}
	}
	if snapshotID != "" {
		data := map[string]any{"id": snapshotID, "paths": paths, "tool_call_id": c.ID}
		if err = a.Store.Append("snapshot_captured", data); err != nil {
			return boundedResult(map[string]any{"status": "interrupted", "snapshot_id": snapshotID, "error": err.Error()}, true)
		}
		a.event("snapshot_captured", data)
	}
	result := a.Tools.Execute(ctx, c.Function.Name, json.RawMessage(c.Function.Arguments))
	var body map[string]any
	if json.Unmarshal([]byte(result.Content), &body) != nil || body == nil {
		body = map[string]any{"content": result.Content}
	}
	if snapshotID != "" {
		body["snapshot_id"] = snapshotID
	}
	if skippedCheckpoint != "" {
		body["checkpoint_skipped"] = skippedCheckpoint
	}
	uncertain := ctx.Err() != nil || body["status"] == "interrupted" || body["status"] == "timed_out" || body["status"] == "cancelled" || body["status"] == "cleanup_incomplete"
	if uncertain {
		body["status"] = "interrupted"
		return boundedResult(body, true)
	}
	if !result.Error {
		reports := []any{}
		invocations, skipped := 0, 0
		for _, path := range paths {
			digest, e := a.Tools.FileDigest(path)
			if e != nil {
				reports = append(reports, map[string]any{"path": path, "error": e.Error()})
				continue
			}
			if digest["exists"] == false {
				continue
			}
			for _, entry := range formatters.Matching(entries, path) {
				if invocations >= 16 {
					skipped++
					continue
				}
				invocations++
				report, interrupted := a.runFormatter(ctx, entry, path)
				reports = append(reports, report)
				if interrupted {
					body["status"] = "interrupted"
					body["formatters"] = reports
					body["mutation_completed"] = true
					return boundedResult(body, true)
				}
			}
		}
		if skipped > 0 {
			reports = append(reports, map[string]any{"status": "skipped", "reason": "16 formatter invocation limit per edit", "count": skipped})
		}
		if len(reports) > 0 {
			body["formatters"] = reports
			final := []any{}
			for _, path := range paths {
				d, e := a.Tools.FileDigest(path)
				if e != nil {
					final = append(final, map[string]any{"path": path, "error": e.Error()})
				} else {
					final = append(final, d)
				}
			}
			body["final_files"] = final
			// Replace the pre-format hash when the original tool describes one file.
			if len(paths) == 1 && len(final) == 1 {
				if d, ok := final[0].(map[string]any); ok {
					if hash, ok := d["sha256"]; ok {
						body["sha256"] = hash
						body["bytes"] = d["bytes"]
					}
				}
			}
		}
	}
	if manager != nil && snapshotID != "" {
		record, e := manager.Seal(ctx, snapshotID)
		if e != nil {
			body["status"] = "interrupted"
			body["snapshot_error"] = e.Error()
			return boundedResult(body, true)
		}
		data := map[string]any{"id": record.ID, "tool_call_id": c.ID}
		if e = a.Store.Append("snapshot_recorded", data); e != nil {
			body["status"] = "interrupted"
			body["snapshot_error"] = e.Error()
			return boundedResult(body, true)
		}
		a.event("snapshot_recorded", data)
	}
	return boundedResult(body, result.Error)
}

func (a *Agent) runFormatter(ctx context.Context, e formatters.Entry, path string) (map[string]any, bool) {
	report := map[string]any{"name": e.Name, "path": path}
	if !e.Available {
		report["status"] = "unavailable"
		report["reason"] = e.Reason
		return report, false
	}
	arguments := e.Arguments(filepath.Join(a.Store.Project, path))
	envNames := make([]string, 0, len(e.Env))
	for name := range e.Env {
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)
	raw, _ := json.Marshal(map[string]any{"path": path, "command": e.Command, "args": arguments, "env_names": envNames, "timeout_seconds": e.Timeout})
	permission := core.Permission{Tool: "format:" + e.Name, Arguments: raw, Preview: fmt.Sprintf("Formatter %s after file edit\nCommand: %s\nArguments: %q\nEnvironment variable names: %v\nDirectory: %s\nTimeout: %ds\nThe configured process can affect files beyond this path; snapshot scope is the edited paths only.", e.Name, e.Command, arguments, envNames, a.Store.Project, e.Timeout)}
	decision, err := a.Permission(ctx, permission)
	if err != nil {
		report["status"] = "skipped"
		report["reason"] = err.Error()
		return report, ctx.Err() != nil || !errors.Is(err, errPermission)
	}
	if !decision.Allow {
		report["status"] = "denied"
		return report, false
	}
	call := core.Call{ID: ID(), Type: "function"}
	call.Function.Name = permission.Tool
	call.Function.Arguments = string(raw)
	if err = a.start(call); err != nil {
		report["status"] = "interrupted"
		report["error"] = err.Error()
		return report, true
	}
	result := a.Tools.RunCommand(ctx, e.Command, arguments, e.Env, e.Timeout)
	var outcome map[string]any
	json.Unmarshal([]byte(result.Content), &outcome)
	if output, ok := outcome["output"].(string); ok && len(output) > 1024 {
		outcome["output"] = output[:1024]
		outcome["formatter_preview_truncated"] = true
	}
	report["result"] = outcome
	report["error"] = result.Error
	report["status"] = "completed"
	state, _ := outcome["status"].(string)
	if ctx.Err() != nil || state == "cancelled" || state == "timed_out" || state == "cleanup_incomplete" {
		report["status"] = "interrupted"
		data := map[string]any{"id": call.ID, "result": result}
		if err = a.Store.Append("tool_observation", data); err != nil {
			report["journal_error"] = err.Error()
		}
		a.event("tool_updated", data)
		return report, true
	}
	if err = a.finish(call, result); err != nil {
		report["status"] = "interrupted"
		report["journal_error"] = err.Error()
		return report, true
	}
	if result.Error {
		report["status"] = "failed"
	}
	return report, false
}
