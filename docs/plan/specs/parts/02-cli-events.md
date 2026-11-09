# Part2: CLI, configuration and events

The original CLI shares its coordinator with the TUI. Commands, precedence,
credential indirection and exact flags are documented in the root README.md.
Event envelopes contain version1, session/run identities, monotonic sequence,
UTC timestamp, type and payload. Every started run has exactly one terminal event.
Configuration errors before a run begins are stderr diagnostics and nonzero exit.

Reference text: research/evidence/reference-text.json. Go capture:
research/evidence/go-text.json. Selected semantic outcomes match; event schemas
are intentionally original. REA raw changed results are preserved. The built
executable driver validates envelopes, event terminal uniqueness, exit codes,
permission outcomes and effects across seven deterministic scenarios.

Validation: `go test ./internal/core ./internal/runtime ./internal/tui`,
`go run ./cmd/acceptance` and retained REA comparisons. Live endpoint compatibility
is unverified. Doctor performs actual local configuration/tool diagnostics without
network requests or integration startup. API-key values are never serialized from
configuration into journals; user prompts and project/tool data are not scrubbed.
