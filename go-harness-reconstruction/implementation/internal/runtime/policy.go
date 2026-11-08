package runtime

import (
	"fmt"
	"strings"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/webfetch"
)

func (a *Agent) isPlan() bool { return a.Config.AgentMode == "plan" || a.Config.Agent == "plan" }
func (a *Agent) allowedInPlan(name string) bool {
	switch name {
	case "memory_recall", "repo_map", "read_artifact", "formatter_status", "snapshot_list", "snapshot_diff", "list_files", "read_file", "search", "glob", "grep", "skill", "question", "task", "job_output":
		return true
	}
	return false
}
func (a *Agent) checkMode(name string) error {
	base := strings.TrimPrefix(name, "child:")
	if a.depth >= 1 && name == "task" {
		return fmt.Errorf("nested subagent delegation is disabled")
	}
	if a.depth >= 1 && (base == "job_start" || base == "job_stop") {
		return fmt.Errorf("subagents cannot manage background jobs; jobs belong to the main chat")
	}
	if a.isPlan() && !a.allowedInPlan(base) {
		return fmt.Errorf("tool %q is disabled by read-only plan mode; grants and instructions cannot enable it", name)
	}
	return nil
}
func (a *Agent) localDefinitions() []core.Tool {
	defs := []core.Tool{todoDefinition(), a.skillDefinition(), questionDefinition(), webfetch.Definition()}
	if a.depth == 0 {
		defs = append(defs, a.taskDefinition())
	}
	defs = append(defs, a.changeDefinitions()...)
	if a.Config.Memory {
		defs = append(defs, memoryDefinition())
	}
	for _, d := range a.Tools.Definitions() {
		defs = append(defs, core.NewTool(d.Name, d.Description, d.Schema))
	}
	if !a.isPlan() {
		return defs
	}
	reads := make([]core.Tool, 0, len(defs))
	for _, d := range defs {
		if a.allowedInPlan(d.Function.Name) {
			reads = append(reads, d)
		}
	}
	return reads
}
