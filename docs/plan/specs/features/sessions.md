# Session list, portable export/import and fork

## Reference questions and observed contracts

Reference: OpenCode v1.19.5 / commit `810c0eba1a7348036f8bdf61e6632d0c36c6876c`. The existing reference manifest binds the matching official macOS ARM64 executable. Source-only assistance is saved separately in `research/reference/features/session-cli.ts`, `export-cli.ts`, `import-cli.ts`, `session-service.ts`.

Actual REA process observations, under explicit isolated HOME/XDG/config settings:

| Question | Evidence | Observed answer |
|---|---|---|
| List shape | `session-reference-list.json`, `ev_b2a229fc31ec568b7b19aa2be36ffd8aeb43cdb9522c325e4de54f0451b576c0` | `session list --format json --max-count 3` exits0 and returns three objects with id, title, created/updated times, projectId and directory. |
| Export shape | `session-reference-export.json`, `ev_7c3606e69fe0ac1b55cdfa95ec38d676a46c89d78c139f23596c6a8c1ea5f2da` | `export ID` exits0, prints a diagnostic plus `{info,messages}` JSON in the PTY capture. This fixture has four saved messages. |
| Import identity | `session-reference-import.json`, `ev_6c30420ccdf6474890b19b1b34d3eba6d849163108daedf3d2e1ce3ea99431b1` | Import into a separate disposable reference data directory exits0 and reports the original source session ID. Source confirms project rebinding and identity-preserving upsert. |
| Fork identity and inherited context | `session-reference-fork.json`, `ev_5d683886d13886cbb87f2c98ae4b00010231980a78ee2a201392fd224a16f42e` | `run --session ID --fork` exits0, emits a new session ID, and calls only the local scripted model. `session-fork-model-requests.jsonl` records inherited conversation plus the new request. Source confirms copying messages/parts with fresh IDs. |

PTY capture merges stdout/stderr and does not establish independent stream separation. These bounded observations do not cover every export part variant, sanitized export, arbitrary malformed reference imports, or interrupted reference forks. OpenCode fork is a run flag; its source does not expose a standalone `session fork` command.

## Original Go contract

Commands:

```text
harness sessions list [--session-dir DIR]
harness sessions show ID [--session-dir DIR]
harness sessions export ID [--output NEW_FILE] [--session-dir DIR]
harness sessions import FILE [--project EXISTING_DIR] [--session-dir DIR]
harness sessions fork ID [--session-dir DIR]
```

Session storage defaults may be overridden by `HARNESS_SESSION_DIR`; the explicit flag wins. List/show retain the original Go JSON interfaces. Export defaults to JSON stdout or publishes a new owner-readable file without overwriting an existing file or symlink.

The portable format is explicitly identified as `independent-harness/session`, version1. It preserves completed conversation, `context_compacted` and ordered `todos_updated` records. It includes the source identity, project path, export time and any uncertain effect IDs. It excludes permission decisions, executable integrations, provider credentials, tool-start/result effect ledgers and transient deltas. Exported transcript/tool-result text is unredacted and may contain data previously read by the agent.

Import accepts only this format, enforces a 64MiB archive limit and strict known fields/record kinds, validates complete tool-call/result relationships, and rejects uncertain effects. OpenCode `{info,messages}` exports receive an actionable unsupported-format error; no OpenCode import compatibility is claimed. The source project must be an absolute existing directory, or the user can explicitly relocate it using `--project`. Import always creates a fresh identity and restores no grants. Runtime replay re-anchors trusted current system framing and treats historical summaries as untrusted.

Fork creates a fresh identity without a model call. It copies the same reviewed durable state and never executes tools. It refuses pending effect IDs, unresolved assistant tool groups, active writers, corruption and unsupported journal versions. If a prior side effect was explicitly acknowledged but its transcript remains incomplete, source recovery must repair that transcript before copying. Lineage is retained as a non-authorizing journal record.

Both import and fork fully validate state before creation, prepare the complete new journal in a private staging directory, then publish it with an exclusive hard link. The writer flock stays attached to the same inode; partially copied sessions are absent from list/open. Existing journals are not overwritten. Journal and export file modes are0600; staging directories are0700. Filesystems must support local hard links and directory synchronization. There is no network import, automatic permission restoration, file restoration, or tool replay.

## Compatibility differences and deferred questions

Go deliberately uses a separate portable format and new import identities. OpenCode import preserves IDs and can update existing session associations. Go fork is an explicit local management operation; OpenCode combines fork with continuation. Go list currently returns id/project and does not reproduce reference titles, update sorting or table pagination. Reference sanitize export and message-boundary fork selection remain unimplemented. Artifact stdout retention and file snapshots are bounded REA observations, not an equivalence proof.

## Verification

Package tests verify export/import/fork state retention, fresh identities, writer lock retention after publication, private permissions, relocation, authority exclusion, invalid schema/version/project rejection without partial publication, uncertain-effect refusal, and unresolved tool-group refusal. CLI tests verify private export files, non-overwrite behavior, import/fork commands and extra-argument rejection. Test commands and Go process-capture comparisons are reported in the implementation checkpoint.

## Go runtime capture

`research/evidence/go-session-portable.json`,
`ev_8e5f06b10142eddb01133be6d11fb9a361c51f84ea519e5b95302476e7c1d7f0`,
captures an original Node driver of the Go binary (no model requests). It exports
a real fixture session containing task state and manual compaction, imports it,
and forks it. All three identities differ; the 29 portable state records compare
exactly, permission records are absent from derived journals, and export mode is
0600. This proves the selected original portable contract, not OpenCode format
compatibility. Original reference export/import/fork captures above remain the
basis for documented interface and identity differences.
