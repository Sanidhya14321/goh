# Explicit custom slash-command text expansion

## Reference question and evidence

The selected OpenCode v1.19.5 source for command registration and expansion is saved in `research/reference/features/commands.ts` and `session-prompt.ts`. It defines user command templates, positional placeholders, raw `$ARGUMENTS`, command-selected agent/model and MCP/skill sources. Expansion additionally recognizes shell and file syntax; those latter operations are distinct from pure template substitution.

An independently written loopback SSE endpoint `research/command-model.mjs` logs the actual provider requests and returns a fixed response. Every reference invocation is a real REA process capture with the pinned artifact and explicit isolated HOME/XDG/config settings; no live credentials/model calls were used.

| Scenario | Evidence ID | Observed provider request/outcome |
|---|---|---|
| Separate argv alpha / beta gamma / delta | `ev_bf59dc47cd15e35a515cda350bea36815dd1413ed1accec7bdcb195d2e7b8546` | `first=<alpha> second=<beta gamma delta> all=<alpha "beta gamma" delta>`, exit0. |
| Missing second argument | `ev_4a5d14414e55e8f4878f5ad2c23d859913ab1559c8d0bd65c81e72bd83803f07` | `first=<only> second=<> all=<only>`, exit0. |
| No placeholders, separate argv one/two | `ev_98b1111ec557131a43ac9ca2c2af62d88e9639685047a1f730a5754f8e6c68f1` | Template followed by a blank line and `one two`, exit0. |
| Unknown command | `ev_0b9d6f78f6ee21c952655ea9f32daee4ae8e187a2066b05e98a3eb519bb69b86` | Exit1 and an error event with generic server-error wording. Source has a more specific unknown-command error internally; that detail was not the observed CLI wording. |
| Argument containing harmless executable syntax | `ev_aed98d9983f65da0a10f928329b3f35fab5872f9d8925135c7ba379608247eb1` | `!` backtick-wrapped `printf ignored` was replaced with `ignored` before the request reached the model; exit0. This bounded probe wrote no files and ran no network command. Source confirms shell substitution happens after argument expansion. |

Capture records are `research/evidence/command-reference-*.json`; provider request ledger is `command-model-requests.jsonl`. Initial `quoted` and `append` captures deliberately retained reveal a CLI boundary: a single argv containing spaces is re-quoted/escaped by `run` before substitution. Follow-up `quoted-tokens` and `append-tokens` use separate argv tokens to observe the intended argument grouping. The Go API starts from a slash prompt string and does not reproduce this argv re-quoting layer.

Source shows `@file` template resolution can attach existing files/agent references. The literal probe included `@message.txt`, but its placement inside `<...>` does not establish successful file attachment behavior. That boundary is source-observed, not runtime-verified. Reference unmatched quotes, escaped quotes, `$0`, huge position indexes, nested placeholder injection, MCP prompts and command-selected agent/model remain unverified.

## Original implementation contract

`core.Config.Commands` is an explicit user-config map of `core.CommandConfig{Description, Template}`. `commands.Expand(prompt, definitions)` returns expanded text and selected command name; it performs no I/O, shell execution, file inclusion, provider calls, agent switches or permission changes. `commands.Validate` allows startup validation independently of expansion.

- Non-command text passes through unchanged, with an empty returned command name.
- Names use letters/digits/underscore/hyphen, start alphanumeric, and are at most64 bytes. Unknown slash names return an actionable error. `/compact`, `/todos` and `/tasks` pass through for runtime built-ins; definitions cannot override them.
- Numbered placeholders start at `$1`. The highest numbered placeholder receives that argument and all remaining parsed arguments joined by spaces; earlier positions receive one argument. Missing positions become empty. Valid numeric indexes are1–10000.
- `$ARGUMENTS` receives the original trimmed argument text, retaining quotes. With no placeholders, nonempty arguments are appended after a blank line.
- The original parser supports single/double quotes, empty quoted values and backslash escapes outside single quotes. Unterminated quotes/trailing escapes fail explicitly. This is stricter and more coherent than the reference regex's loosely tokenized malformed quoting; exact malformed-input parity is not claimed.
- Expansion is a single pass. Placeholder-looking text inside arguments stays literal, including `$ARGUMENTS` introduced by a positional value. Arguments cannot create additional expansion passes.
- Backticks, `!` command syntax, `$()` and `@file` remain literal text. They are not rejected merely because they appear in a coding request, and are never executed/read by this package. This intentional difference closes the implicit-execution boundary observed in the reference. Any future executable command feature must pass through the shared runtime approval/effect journal rather than expanding here.
- Template limit64KiB, description8KiB, input/output1MiB, at most256 definitions and10000 parsed arguments. Text must be valid UTF-8. Output growth is checked before each append, so repeated raw-argument placeholders cannot grow output without bound.

Example:

```json
{"commands":{"review":{"description":"Review selected files","template":"Review $1. Additional context: $2"}}}
```

`/review parser.go "edge cases" streaming` becomes `Review parser.go. Additional context: edge cases streaming`. The model may subsequently request ordinary tools; the expansion itself grants no authority.

## Verification and implementation boundary

`go test -race ./internal/commands ./internal/core` passed. Tests verify observed grouping, missing args, remainder consumption, suffix appending, normal/reserved/unknown handling, nonrecursive injection resistance, quote escapes, no filesystem effects from executable syntax, malformed syntax, invalid definitions and input/output limits. `go vet ./internal/commands ./internal/core` passed. Bounded fuzzing passed22078 executions over roughly3.5 seconds.

This slice supplies pure expansion plus config definitions. Runtime/TUI integration, command catalog presentation and Go process captures belong to the shared-runtime follow-up; no CLI slash-command execution is claimed solely from package tests. Built-in reference init/review templates are not copied.

## Shared-runtime integration

Explicit Config.Commands is validated before starting a session runtime. CLI user
prompts and TUI submissions pass through Expand exactly once before being durably
committed as user task data. The original command name/request is journaled as
command_invoked; command_expanded event reports its name and output size. No
transformed text is interpreted as another slash command or permission grant.
Unknown commands fail before model/integration startup. Local TUI /commands lists
configured descriptions; /skills is also reserved along with prior built-ins.

Go REA captures: go-command-quoted.json
(`ev_7639176132b63bf7e866df741800e52a23afb34b529141fcd63ea91b3e1428f4`)
checks the exact selected reference expansion; go-command-literal.json
(`ev_1a7ff5e888e076c1a942373f96a57b71c449708b51c399a0575dc5f353ad26dd`)
completes without creating the fixture file or starting a tool. This deliberately
differs from the reference executable-substitution boundary. Runtime tests prove
single expansion, durable provenance, no direct effects and unknown-command
refusal. Model/agent override, Markdown/project-command discovery, MCP prompts and
actual file/shell substitutions remain future work rather than implied parity.
