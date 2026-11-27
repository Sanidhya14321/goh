package runtime

import (
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/webfetch"
	"github.com/Sanidhya14321/goh/internal/websearch"
)

func (a *Agent) isPlan() bool { return a.Config.AgentMode == "plan" || a.Config.Agent == "plan" }
func (a *Agent) allowedInPlan(name string) bool {
	switch name {
	case "recall_observation", "memory_recall", "repo_map", "read_artifact", "formatter_status", "snapshot_list", "snapshot_diff", "list_files", "read_file", "search", "glob", "grep", "skill", "question", "task", "job_output", "repeated_call", "plan_exit":
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
	defs := []core.Tool{todoDefinition()}
	if len(a.Skills.List()) > 0 {
		defs = append(defs, a.skillDefinition()) // an empty catalog is not worth a tool
	}
	defs = append(defs, questionDefinition(), webfetch.Definition())
	if a.Config.WebsearchURL != "" {
		defs = append(defs, websearch.Definition())
	} else if a.providerSearch() != nil {
		defs = append(defs, websearch.ProviderDefinition())
	}
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
	defs = append(defs, a.experimentDefinitions()...)
	offered := defs[:0]
	for _, d := range defs {
		if !unadvertisedTools[d.Function.Name] {
			offered = append(offered, d)
		}
	}
	defs = offered
	if !a.isPlan() {
		return defs
	}
	if a.depth == 0 {
		defs = append(defs, planExitDefinition())
	}
	reads := make([]core.Tool, 0, len(defs))
	for _, d := range defs {
		if a.allowedInPlan(d.Function.Name) {
			reads = append(reads, d)
		}
	}
	return reads
}

// unadvertisedTools stay callable (older sessions and models that know them
// keep working) but are not offered, because every offered definition is
// sent with every request. search is grep with a limit; write_file creates a
// new file as create_file does. The snapshot tools were called once in 453
// benchmark requests while costing about 300 tokens on each; the user undoes
// and inspects runs with /undo, /redo, /changes and /snapshots.
var unadvertisedTools = map[string]bool{"search": true, "create_file": true,
	"snapshot_list": true, "snapshot_diff": true, "snapshot_undo": true, "snapshot_redo": true}

// providerSearch is the provider's hosted search when "websearch": "provider"
// is set and the provider has one; nil otherwise.
func (a *Agent) providerSearch() websearch.ProviderSearcher {
	if a.Config.Websearch != "provider" {
		return nil
	}
	s, ok := a.Provider.(websearch.ProviderSearcher)
	if !ok || !s.SupportsWebSearch() {
		return nil
	}
	return s
}
