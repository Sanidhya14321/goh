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
	"review": {
		Description: "Review uncommitted changes, or a commit, branch or path given as the argument",
		Template: `Review code changes for real defects. Do not edit any files; this is a read-only review.

What to review: $ARGUMENTS
If that is empty, review the uncommitted changes (git diff and git diff --staged, plus untracked files). A commit hash means that commit (git show); a branch name means its difference from the main branch (git diff main...BRANCH); a path means the changes under it.

Read enough surrounding code to judge each change in context. Report only concrete problems: bugs, wrong behaviour on edge cases, security issues, races, resource leaks, broken error handling and missing tests for new behaviour. For each, give the file and line, what goes wrong and when, and a suggested fix. Order findings by severity and skip style preferences. If you find nothing worth fixing, say so plainly.`,
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
