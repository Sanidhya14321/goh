# Formatter investigation: pinned reference contract

This is reference research, not a claim that all formatter behavior is
implemented by the Go product. Source identity is OpenCode v1.19.5,
810c0eba1a7348036f8bdf61e6632d0c36c6876c; host artifact/digest/license are in
research/reference-manifest.json. Normal source tools inspected complete source;
actual runtime facts below use rea-agents6.8.0 process captures. The existing
REA skill/runtime/process guides were read. No upstream implementation or prompt
was copied into Go code by this investigation.

## Source-established interface and boundaries

Relevant sources under research/reference/source:

- packages/opencode/src/format/index.ts: formatter selection and execution.
- packages/opencode/src/format/formatter.ts: built-in formatter detection.
- packages/core/src/v1/config/formatter.ts: configuration schema.
- packages/opencode/src/tool/write.ts, edit.ts, apply_patch.ts: invocation after
  file mutation, before edited-event/LSP diagnostics processing.

Configuration `formatter` is a boolean or map. A custom entry may include
`disabled`, argv `command`, `environment` string map, and `extensions` string
array. Disabled entries are removed. Built-in defaults may be overridden and
custom entries added. Ruff/uv disabling is linked by the reference. Effective
configuration defaults and all built-in auto-detection paths were not runtime
probed here.

File selection compares the exact file extension against each enabled entry.
Every selected formatter runs sequentially. Each argv string replaces its first
`$FILE` occurrence with the full file path. Commands use project cwd, inherit
host environment, overlay the configured environment and ignore stdin/stdout/
stderr. The service logs spawn and nonzero-exit failures but returns true when
at least one formatter was selected. No separate formatter permission request
exists inside this service. It is invoked after an admitted file mutation.
Built-ins inspect executables/project dependencies; no automatic installation
or universal availability is inferred from source declarations.

## Actual controlled observations

Original loopback-only research/formatter-model.mjs drives three `write` calls
against a disposable project. Original research/formatter-command.mjs uppercases
the file, appends an invocation record and optionally exits7. No live model or
remote endpoint was used. LSP was explicitly disabled; plugin/model fetch and
reference configuration/data/cache/state were isolated.

Enabled configuration: research/formatter-enabled.json.
Scenario: research/formatter-enabled-scenario.json.
Capture: research/evidence/formatter-enabled.json,
`ev_787e9dd0319a18f94bc0f1145a195524b0c24b0fdcefe39f26abf2fc6636f8b9`.

- Writing `normal.fmtcase` persisted `RAW INPUT\n` after the configured formatter.
- Writing `failure.failcase` persisted the same formatted contents even though
  its formatter exited7.
- Both write calls returned `Wrote file successfully.` The failing formatter
  was not converted into a tool failure in this fixture.
- `unmatched.other` remained `raw input\n` and did not invoke a formatter.
- Retained invocation JSON verifies exact absolute target paths, project cwd,
  and per-formatter `FORMATTER_FIXTURE_MODE` environment values.

Disabled configuration: research/formatter-disabled.json (`formatter:false`).
Scenario: research/formatter-disabled-scenario.json.
Capture: research/evidence/formatter-disabled.json,
`ev_1c2846f4b64bb346d5d95e1a558fbf52c9120e5a415b373bd5169ba99421607c`.
All three writes persisted raw text; no invocation log appeared. Tool calls
still succeeded.

Provider request/result messages and final file snapshots are retained as
research/evidence/formatter-{enabled,disabled}-requests.jsonl and matching
-final.json files. `python3 research/formatter-verify.py` passes assertions for
selection, disabled behavior, formatted output, environment/cwd and tolerated
nonzero exit. It emits formatter-semantic-report.json.

## Coverage limits and reconstruction implications

Both captures exited0; REA verified the owned process group and removed its
temporary root. Both report incomplete shutdown process sampling and therefore
truncated overall observations. Complete Evidence retains the precise scope.
Filesystem snapshots establish before/after states, not syscall traces. Host
inherited environment is unrecorded; executable identity is a prelaunch sample,
not atomic OS image binding. The CLI reports artifact identity unavailable for
this observation; the reference-manifest checksum separately identifies acquired
bytes. No universal formatter latency, cancellation or cleanup claim follows.

Unverified: missing executable/spawn failure, overlapping multiple formatter
selection on one extension, long-running formatter timeout/cancellation,
formatter modification of other files, builtin package discovery and all host
variants. A Go implementation should keep command execution independently
approved, process-group-supervised and durably checkpointed, record changes
made before formatter failure, and explicitly document any stricter failure or
permission behavior. Raw write approval does not imply authority for an
arbitrary repository-selected executable. No product code was changed here.

All fixture listeners owned by this investigation were stopped; ports18784–18786
were checked with no listeners after captures.

## Original Go implementation and selected comparison

Config.Formatters maps explicit names to command/args/extensions/env/disabled/
timeout_seconds. An empty gofmt entry opts into PATH detection and default -w
argv. There is no automatic built-in catalog, project package scan or installation.
Discovery is side-effect-free; matching is exact extension, sorted by name.
All $FILE occurrences are replaced literally, deliberately differing from the
reference's first occurrence. All argv/env values are quoted as literals by the
shared supervised process runner; they do not evaluate shell substitutions.

File mutation tools run formatting only after the admitted edit. Each formatter
independently requests format:NAME permission, including environment variable
names without their values. Parent write grants cannot admit formatter execution;
child delegation uses child:format:NAME. Missing/headless authority skips formatting
with a report and preserves the edit. Denial and spawn/nonzero errors are separate
formatter outcomes. Plan mode refuses execution. Commands inherit the host env,
use project cwd, ignore stdin, drain bounded output and retain private artifacts.
Timeout/cancellation keeps durable uncertain starts and does not replay effects.
These are authorized local processes, not sandboxes; unrelated effects are not
claimed to be captured or rolled back.

At most32 configured entries,64 args/env entries,64KiB command/env total per entry,
1–120 seconds per invocation (default30), and16 invocations per edit. Skipped
matches are reported. Outputs contain final edited-file hashes, bounded formatter
previews and retained artifact identities. Opt-in snapshots capture before mutation
and seal only after formatting; formatter failures still preserve final bytes.

Go capture research/evidence/go-formatters.json:
ev_82834b10376ec6eec517c709cd747a261d23cdb6326f931d1d860d538e782a7f.
Original scripts/capture_changes_batch.py confirms the same selected normal,
exit7 and unmatched-extension file states as the retained reference fixture.
Runtime tests additionally cover authority, stale conflict prevention, failed
formatter mutation, final identities and actual cancellation. Three added PTY
scenarios approve, deny and cancel a running formatter with restored terminal
settings. REA owns the harness process group, but incomplete shutdown sampling
and unverified ambient-process identities remain in Evidence. This selected
semantic projection is not canonical REA equivalence or complete formatter parity.
