package runtime

import (
	"sort"
	"strings"
)

// subagentNames are the types the task tool accepts.
func (a *Agent) subagentNames() map[string]bool {
	names := map[string]bool{"explore": true, "general": true}
	for name := range a.Config.Agents {
		names[name] = true
	}
	return names
}

// withAgentMentions handles "@general ..." style prompts, as OpenCode does:
// a mention of a known subagent adds a note asking the model to delegate
// through the task tool. It grants nothing; the task tool keeps its usual
// approval. Child agents cannot delegate, so they never get the note.
func (a *Agent) withAgentMentions(prompt string) (string, []string) {
	if a.depth > 0 {
		return prompt, nil
	}
	known := a.subagentNames()
	seen := map[string]bool{}
	var found []string
	for _, field := range strings.Fields(prompt) {
		name, ok := strings.CutPrefix(field, "@")
		name = strings.TrimRight(name, ".,;:!?)\"'")
		if ok && known[name] && !seen[name] {
			seen[name] = true
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return prompt, nil
	}
	sort.Strings(found)
	note := "\n\n[The user mentioned the " + strings.Join(found, ", ") + " subagent"
	if len(found) > 1 {
		note += "s"
	}
	note += ": delegate the relevant work with the task tool using that subagent_type.]"
	return prompt + note, found
}
