package commands

import "example.com/independent-harness/internal/core"

// Builtins are default commands; a command file or configured command with
// the same name replaces them. They only expand to a prompt: the model then
// uses ordinary tools under the ordinary approval rules.
var Builtins = map[string]core.CommandConfig{
	"init": {
		Description: "Study this project and write AGENTS.md guidance",
		Template: `Create or update AGENTS.md at the project root so future coding sessions start with the right context.

First investigate with read-only tools: the build, test and lint commands (and how to run a single test), the top-level layout and the role of each main directory, the architecture a newcomer would need several files to understand, and any existing guidance files (README, CONTRIBUTING, CLAUDE.md, .cursor rules, editor configs).

Then write AGENTS.md that is concise and specific to this repository:
- commands for building, testing, linting and running one test;
- the architecture and the non-obvious conventions;
- nothing generic (no "write clean code"), no invented facts, no exhaustive file listings.

If AGENTS.md already exists, improve it instead of replacing what is still correct. Extra focus from the user, if any: $ARGUMENTS`,
	},
}

// WithBuiltins returns commands with built-in defaults underneath.
func WithBuiltins(commands map[string]core.CommandConfig) map[string]core.CommandConfig {
	out := make(map[string]core.CommandConfig, len(commands)+len(Builtins))
	for k, v := range commands {
		out[k] = v
	}
	for k, v := range Builtins {
		if _, taken := out[k]; !taken && len(out) < 256 {
			out[k] = v
		}
	}
	return out
}
