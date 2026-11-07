# Verification record — 7 November 2026

This is a working development alpha. Full release acceptance in the supplied plan
is not complete. The observed host is macOS 26.5.1, ARM64, Apple M5 Pro, Go 1.27.3.
Live inference was tested on loopback. Source milestones are published to Sanidhya14321/goharness.

## V1 release candidate

The installed macOS ARM64 `goh` binary reports `1.0.0-rc.13` and matches the
local build byte for byte. The race suite, `go vet`, seven CLI fixture scenarios,
fifteen PTY scenarios, and compiled ACP integration suite passed on this
candidate. Eleven bounded fuzz targets passed on RC1. A TUI regression covers a
narrow viewport with wide text, tabbed previews, and long input.

RC13 evidence is under `research/evidence/v1-ui-files-*`. The terminal setup
flow finds a file with spaces using a fuzzy abbreviation. It checks that
selection leaves provider request counts and journal bytes unchanged, then
sends the quoted relative path with a separate Enter. The provider fixture
receives the exact prompt text. Focused checks cover Unicode, cursor placement,
narrow terminals, draft preservation, and hidden and symlink exclusions.
Paired PTY captures show the GoHarness and OpenCode composer menus.

RC12 evidence is under `research/evidence/v1-ui-timeline-*`. The executable
setup flow searches a prompt from a resumed conversation, opens its preview,
and recalls it. Request counts and journal byte comparisons verify that browsing
and recall do not send or persist a turn; a separate Enter sends the exact selected
prompt. Unit regressions cover draft/cursor preservation, newest-first ordering,
multiline and Unicode search/recall, empty results, the 200-prompt bound, terminal
control sanitization, and viewport bounds down to 8×4. Captures include the Go
list and preview and the reference OpenCode timeline.

RC11 evidence is under `research/evidence/v1-ui-status-*`. The PTY flow opens
status before provider setup, uses its provider shortcut, then inspects and
refreshes status after a run. Provider request counts and journal byte comparisons
verify that opening and refreshing the snapshot create no model request or
journal mutation. Unit regressions check key/path/query/environment redaction,
draft preservation, Plan-mode labels, terminal sizes, and scrolling.
Resumed tool-message regressions check readable saved envelopes, preserved error
labels, and legacy payloads. The same tool formatter serves live and resumed
results.

RC10 evidence is under `research/evidence/v1-ui-sessions-*`. The PTY flow
searches an older conversation by title, resumes its original journal, and
verifies that a second journal is untouched. It also cancels the browser back
to chat and invokes the CLI preview while the chat writer is active.
Focused tests check prefix limits, incomplete tails, symlink/traversal refusal,
unfinished-run status, workspace scope, empty results, and terminal dimensions.
Preview byte comparisons verify that it leaves the journal contents intact.

RC9 evidence is under `research/evidence/v1-ui-pickers-*`. Its executable PTY
test selects a named model profile, observes the changed model ID in an actual
provider request, returns to the default profile, and verifies the original
model ID. It selects Plan and checks that shell and mutation tools are absent
from the provider request. It also cycles to Build and verifies /new creates a
second journal while retaining the original. Race, vet, CLI, ACP, and the
isolated 15-case terminal suite pass.

`goh-ui-models.png` and `goh-ui-agents.png` capture the new selectors. Source,
test evidence, and project stage documents are published through the project
Git repository. Local toolchains and reference checkouts are excluded by the
repository ignore rules.

RC8 evidence is under `research/evidence/v1-ui-rich-*`. The race suite, vet,
CLI, ACP, setup, and isolated 15-case PTY run passed. One PTY run concurrent
with reference captures returned `cleanup_incomplete` in child cancellation.
The subsequent isolated pass leaves the cause and reliability under load
unresolved. See `research/evidence/ui-rich-cancellation-note.txt`.

Streaming regressions check every byte prefix of a mixed code/table response,
12/24/60/90-column panes, escaped and inline-code pipes, column alignment,
multiline comments, wrapped strings, unknown-language fallback, NO_COLOR, and
terminal-control sanitization. Approval previews bypass Markdown interpretation.
The OpenCode executable was also sent the same rich-text fixture response.
`opencode-ui-session.png` and `goh-ui-session.png` retain the rendered captures.

The `v1-chat-first-*` evidence includes a first-run PTY test. With a fresh
home, it opens chat before configuration, verifies blocked submissions make no
requests, saves a masked provider key, recovers from HTTP 401 by updating the
key in the UI, and verifies that the session is retained. It also checks saved
credentials in CLI requests, settings return to chat, and terminal restoration. The
`goh` and `harness` commands resolve to the same installed build and session
storage.

The UI redesign is covered by `research/evidence/v1-opencode-ui-*`. Layout
regressions cover 24×10, 60×18, 80×24, and 120×36 terminals, wide input, permission
previews, and command palette navigation. Actual PTY frames rendered into PNGs
are retained as `goh-ui-home.png`, `goh-ui-palette.png`,
`goh-ui-session.png`, and `goh-ui-approval.png`. OpenCode v1.19.5 was run
with an isolated home and a loopback provider configuration for the startup
comparison and `opencode-ui-home.png`. These frame captures use a terminal
emulator and a fixed font rather than an OS screenshot.

The `v1-ui-composer-*` records cover RC7, including the full race suite and
CLI, ACP, and terminal integration checks. The setup PTY test checks that
editing inside a prompt changes the actual provider request and that recalled
history can be resubmitted. Focused regressions cover Unicode input, multiline
cursor movement, draft restoration, configured command completion, literal
code/diff preservation, and code styling when scrolling into a fenced block.
`goh-ui-slash.png` and `goh-ui-composer.png` capture the composer changes.

RC2 streamed a real Qwen3-0.6B response from a loopback llama.cpp server at a
16,384-token context. The CLI base URL override and a custom bearer token
environment variable succeeded. The same server rejected a wrong token.
The 8,192-token trial stopped before inference because the default tool schemas
exceeded the harness's conservative budget. See
`research/evidence/v1-live-local-provider.json` and the `v1-base-url-*` logs.
The Windows AMD64 cross-build currently fails on Unix-specific filesystem and
subprocess operations. See
`research/evidence/v1-windows-crossbuild.txt` for compiler diagnostics.

## Executed checks

| Check | Result | Retained record |
|---|---|---|
| `go test -race ./...` | Pass across all packages, including 47 top-level tests and their subcases | research/evidence/go-test-race.txt |
| `go vet ./...` | Pass, no diagnostics | research/evidence/go-vet.txt |
| Bounded SSE fuzzing | Pass, 96,707 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Bounded tool arguments fuzzing | Pass, 145,879 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Bounded journal fuzzing | Pass, 131,383 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Built executable driver | Seven cases pass: text, parallel reads, read/edit/shell, permission, drop, malformed stream, HTTP failure | research/evidence/cli-acceptance.json |
| Pseudo-terminal integration | Five cases pass: text, refusal, edit approval, active shell cancellation, termination; app remains open after first cancellation | research/evidence/tui-acceptance.json |
| Primary paired semantic comparison | Five selected cases pass | research/evidence/semantic-comparisons.json |
| Extended paired semantic comparison | Nine selected cases pass | research/evidence/extended-semantic-comparisons.json |
| Vulnerability scan | govulncheck 1.9.0 source/symbol scan exits0, no findings in final module graph; database 2026-11-02 | research/evidence/govulncheck.json and dependency-audit.md |
| Cross-builds | CGO_ENABLED=0 for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64 | dist/SHA256SUMS and BUILD_INFO.txt |

Retained paths above are relative to the reconstruction project root except
`dist/`, which is relative to implementation/. Raw REA comparisons are preserved
and report changed/truncated output. Independent semantic checks establish only
the declared observed facts, not universal equivalence.

The terminal integration test compares termios flags, speeds and control
characters before/after. It masks only Darwin's PENDIN flag, which the kernel sets
when restoring canonical mode after raw mode; an independent Python raw/restore
probe reproduced that single-bit change. Resize is covered by state tests and
REA PTY captures. Terminal restore escapes alone are not treated as proof of
termios restoration; the independent test checks the actual slave attributes.

## Acceptance coverage

Parser tests cover one-byte network fragmentation (including UTF-8), split tool
arguments, empty-choice usage chunks, early EOF, bad JSON, duplicate tool IDs,
unknown finish reasons, late content and cancelled requests. Partially consumed
responses are never transparently replayed. Length termination has a separate
budget outcome and cannot execute incomplete tool calls.

File tests cover exact ambiguous/stale edits, intervening mutations, CRLF, Unicode,
mode preservation, new-file collisions, escaped/symlink/FIFO/binary paths,
argument validation, complete tool schemas, bounded two-sided previews and
cancellation during admission. Runtime tests verify parallel read results remain
in call order and mutation/shell operations are barriers.

Process tests cover successful/failing commands, bounded retained output,
large output, timeout, cancellation and children in owned groups. Journals cover
conflicting writers, private permissions, torn tail versus corruption, unsupported
version, and uncertain-effect acknowledgment without replay. Context tests cover
successful/failed/ineffective compaction, active request and recent groups,
unresolved calls and explicit exhaustion. MCP fixtures verify discovery,
validation, references rejection, cancellation, disconnect, bounded results,
cleanup, admission, durable startup and recovery without authority restoration.

## Outstanding release gates

- Linux runtime tests: cross-builds passed, but no Linux execution environment was
  available on this host. No Docker/Podman/QEMU executable was found on PATH.
- macOS AMD64 and Linux architecture runtime coverage; macOS ARM64 is the tested host.
- Live model endpoint/profile compatibility: intentionally untested without an
  explicitly configured endpoint, credentials and authorization for live calls.
- Reference compaction and journal-corruption behavior, plus full interactive
  reference approval/cancellation coverage, remain unverified. Go deterministic
  behavior is tested; this does not imply reference parity.
- Reference shell cancellation returned REA cleanup_incomplete; its orphaned
  experiment child was manually cleaned up and recorded. Go cancellation cleanup
  was verified separately.
- Full comparative performance: only local microbenchmarks and version-process
  startup were measured. No reliability/speed superiority claim is made.
- Public-release ownership, branding, contribution and legal review remain open.

See ../PROGRESS.md, ../research/FEATURE_MAP.md and
../specs/COMPATIBILITY_MATRIX.md for exact scoped evidence and next actions.


## Expanded-feature verification (same pinned reference)

The full reference checkout is detached at 810c0eba1a7348036f8bdf61e6632d0c36c6876c:
6,627 tracked files across 32 top-level packages, indexed under research/. This
is whole-repository acquisition and surface mapping, not an assertion that every
source file or behavior was reviewed.

Current test inventory: 68 top-level tests plus their subcases. The full current
race suite and vet pass (feature-tests-race.txt, feature-vet.txt). Original seven
built-CLI acceptance cases still pass (feature-cli-acceptance.json). Six PTY
scenarios now pass, adding task approval, task overview/local reader and manual
compaction to the prior five cases (feature-tui-acceptance.json). Actual terminal
settings are checked; the inherited Darwin PENDIN exception remains documented.

New reference/Go evidence covers recursive glob/include grep, session task list
snapshots and durable reset, manual compaction, and session portability/fork.
`research/compare_new_features.py` passes its six selected semantic checks;
`grep-semantic-comparison.json` confirms the declared seven-case Go contract,
while documenting different default eligible files and named-file semantics.
`go-session-portable.json` captures fresh identities/equal state/no authority copy.
Glob records retain their truncation limits and changed raw comparison status.
These checks do not imply universal parity. Exact contracts and evidence IDs are
in specs/features/. Full Go snapshot/manifest identity remains in each record.

Security: replay always re-anchors the current trusted system message; historical
compaction systems become untrusted assistant data. Portable archives reject
unsupported formats, uncertain effects, incomplete tool groups, invalid task
schemas and oversized state, and never import permission grants. Manual compaction
starts no integrations. Named todowrite grants authorize task-state updates only.

The expanded whole-repository objective remains unfinished: skills, patch engine,
agent/model profiles, subagents, structured questions, additional providers/auth,
LSP, formatters, plugins, server/ACP, web/desktop and integrations are backlog
items, alongside existing Linux/live-provider/public-release gates. The manual
reference compaction gap is partially resolved; automatic thresholds and pruning
remain unverified. See PROGRESS.md and FEATURE_BACKLOG.md for the next exact slice.

Final added glob-parser fuzz run: 67,442 executions, no failures. Final source/symbol vulnerability scan has 0 findings (feature-govulncheck.json). All four CGO-free platform builds/checksums were regenerated after feature integration.

## Further feature verification

Skills, exact multi-file patches, JSON model/agent profiles, strict build/plan mode,
pure custom commands and structured questions are now integrated over the same
runtime. Detailed contracts/capture identities: ../specs/features/{skills,
apply-patch,profiles,commands,questions}.md. All observed reference data remains in
research; original product prompts and implementation remain separate.

Current full race and vet logs are rest-tests-race.txt/rest-vet.txt. Seven CLI
acceptance scenarios remain passing. Nine PTY scenarios now include actual
structured choices/multiple/custom text, dismissal and cancellation with terminal
restoration, in addition to the earlier task/compaction and shell cases. Runtime
checks cover plan-mode forced-tool refusal despite grants, profile/env/CLI
precedence, metadata-only skill admission, lazy resources, explicit custom-command
provenance without effects, no question-to-permission transfer, and repaired input
relationships on safe continuation. Actual real patch tests cover approval,
denial, intervening writes, CRLF/modes, partial cancellation, durable observations,
uncertain recovery blocking and no replay. New schema tests compile all runtime
tool schemas with the pinned validator.

The patch semantic projection passes 10 bounded reference/Go cases; selected new
skill/plan/command/question checks are in rest-semantic-comparisons.json. Captures
and tests document stricter deliberate matching/authority/input differences.
Source-only surfaces remain distinguished. Skill parser fuzzing passed 270,248
executions; patch grammar fuzzing passed 180,851 in the final worker run. No new
runtime module dependencies were introduced. The renewed source/symbol advisory
scan is retained in rest-govulncheck.json.

Scope remains partial: controlled subagents, webfetch/search, LSP, additional
providers/auth, server/ACP, remote MCP, plugin compatibility and web/desktop
interfaces are among the remaining whole-repository backlog. Existing Linux and
other-architecture runtime/live-provider/public-release gates remain open.


Final batch inventory: 112 top-level tests and their subcases pass under race;
vet passes, seven CLI/nine PTY cases pass, and fresh govulncheck has 0 findings.
All four CGO-free binaries/checksums were refreshed. Existing local-only performance
measurements predate this feature batch and are not performance claims about the
new binaries; benchmark full workloads separately when those gates are reached.

## Foreground child/web/LSP verification

The next slice adds private child sessions/owned continuation with independent
child authority, explicit web fetching and configured local LSP. Current retained
logs are next-tests-race.txt and next-vet.txt; seven CLI acceptance cases and eleven
PTY cases pass. New PTY cases separately approve a child mutation and cancel an
executing child shell (command-entry pid fixture), while preserving terminal
settings and leaving the application open after cancel.

Unit/integration tests cover child ownership/type/limits/grants, strict explore/
parent-plan caps, safe continuation, no recursive delegation, web credential/proxy/
cookie isolation and redirect destination non-contact, response/HTML/result bounds,
LSP initialize/sync/UTF16/diagnostics, malformed/large frames, out-of-order replies,
blocked writes/cancellation, startup admission/failure, server-requested edit
rejection, special files and uncertain recovery without replay. Original helper
servers frame JSON-RPC independently of the client's parser.

Actual reference/Go captures and selected assertions are retained in
next-batch-driver.json and next-semantic-comparisons.json. Child creation/resume,
eleven fetch cases and six LSP operations pass their declared selected contracts;
interface, authority, byte/charset/redirect and Unicode-column differences are
explicit. Webfetch/LSP rendered outputs are truncated; webfetch includes unverified
ambient-process observations despite verified owned-group cleanup. No global
process/network absence or universal equivalence claim follows.

Bounded fuzz runs: 207,743 LSP-frame executions and 40,395 web-argument executions,
no failures. Renewed govulncheck source/symbol scan has no findings after pinning
x/net v0.62.0 and its selected transitive graph; exact 39-module notices/digests were
refreshed. No live providers, production language servers, remote fetch targets,
automatic server downloads or remote publication were used for this acceptance.
Native/advanced LSP, background/nested children, richer web search/auth, all broader
backlog and platform/live-provider/public-release gates remain open.

Final inventory for this batch: 133 top-level tests plus subcases pass under race, clean vet, seven CLI and eleven PTY acceptance cases. Four rebuilt Darwin/Linux AMD64/ARM64 artifacts pass SHA256 verification. Platform cross-builds remain distinct from native runtime acceptance; no whole-repository parity claim follows.

## Explicit formatting and file checkpoint verification

This batch adds original formatter configuration/discovery, literal argv
substitution, separately admitted supervised processes, post-edit final identity,
and opt-in durable before/after checkpoints. Snapshot list/diff/undo/redo are
available through the runtime and `changes` CLI. Tests cover separate write versus
formatter authority, headless skip/denial, failure after mutation, final formatted
bytes, child/plan boundaries, cancelled formatter checkpoints, patch moves and
creation, all-file conflict prevalidation, CLI recovery and concurrent read tools.
The snapshot package additionally checks binary bytes, Git-index preservation,
private storage, corrupt blobs/metadata, modes, bounds, locks, links/FIFO and
partial multi-file cancellation without replay. Restores preserve unrelated files;
full Git-tree snapshots/message-history revert are unimplemented.

Fourteen actual PTY cases now include formatter approval, denial and running
formatter cancellation; terminal settings are restored and cancellation leaves the
application open. Seven existing compiled-CLI scenarios also pass. Logs are
changes-tests-race.txt, changes-integration-race.txt, changes-vet.txt,
changes-cli-acceptance.json and formatter-tui-acceptance.json in research/evidence.
Research-only formatter and snapshot scripts verify real pinned reference
observations, including exit7 preservation and revert/unrevert of externally
modified touched files. Go REA captures go-formatters/go-snapshot-{undo,redo,conflict}
plus changes-semantic-comparisons.json retain selected original assertions.
These projections are not canonical REA equivalence. Process observations have
incomplete shutdown sampling; the formatter capture additionally contains
unverified ambient-process identities. Owned-group cleanup was verified. No
universal absence-of-effects or runtime-platform equivalence claim follows.

Webfetch HTML raw-text handling now also preserves suppression for self-closing
script/style tokens; regression and bounded argument/HTML fuzz checks cover that
parser behavior. Dependencies and their selected39-module graph are unchanged.
All broader repository parity, native/live-provider, platform runtime and public
release review gates remain open. Existing benchmark measurements predate this
batch and are not measurements of these new features.

Final batch inventory:153 top-level tests plus subcases, seven CLI and fourteen
PTY scenarios pass; race/vet are clean. Web argument fuzz60,219 and HTML fuzz26,293
executions pass. The renewed source/symbol scan contains0 findings. All Go REA
captures in this batch have normalized_result.truncated=true because of incomplete
process observations, even when terminal output itself fits the render budget.
Four CGO-free Darwin/Linux AMD64/ARM64 binaries and checksums are regenerated;
platform cross-builds do not establish runtime parity. No new dependencies.

##6 November2026: native protocol, Harvest roles/results and context efficiency

Native Anthropic Messages text/client-tool encoding and streaming, explicit
compaction/task/explore roles, strict optional task JSON results, Go declaration
maps and scoped shell/formatter output retrieval are implemented. Full source
pins/license digests for Harvest/Aider/Goose/Crush are retained under
research/comparisons; their builds/tests were not executed and no upstream runtime
claims follow from source inspection. OpenCode native complete/drop observations
are separately retained and checked by research/compare_native_provider.py.

New tests cover native request history/tool-result identity, fragmented Unicode/
JSON SSE, usage/cache counts, finish/errors/lifecycle/bounds/unsupported blocks,
credential separation, redirect non-contact, HTTP classification/redaction,
backoff cancellation and partial-call no-effect. Runtime tests exercise a real
mixed chat/native child role, preflight invalid schema no-dispatch, strict child
output failures without repeated effects, plan/custom-role precedence and failed
summary admission preserving history. Tool tests cover content-cache invalidation,
concurrent maps, syntax/body exclusion, limits, retained-artifact byte/base64/tail
paging, session ownership, links/FIFO, escaped results and cancellation.

184 top-level tests plus subcases pass under race, with clean vet. Seven existing
CLI acceptance cases, seven additional native/Harvest CLI REA cases and15 actual
PTY cases (including typed child approval/result) are checked independently.
Fuzz runs:19,334 structured output cases and67,105 native SSE cases, no failure.
Renewed govulncheck source/symbol scan reports0 findings in the unchanged selected
39-module graph. No extra module/runtime is linked for repository mapping.
Logs: harvest-tests-race.txt, harvest-vet.txt, harvest-integration-race.txt,
efficiency-integration-race.txt, harvest-tui-acceptance.json, harvest-govulncheck.json,
structured-fuzz.txt, native-provider-fuzz.txt and repomap-benchmark.txt.

Actual Go CLI captures indexed by harvest-batch-driver.json exercise valid/invalid
typed child results, mixed protocol, a dedicated compaction role, native read/text,
partial stream,401 and redirect refusal. harvest-semantic-checks.json and
native-provider-comparison.json are original selected assertions, not canonical
REA equivalence. In the pinned reference dropped stream, the read is interrupted
then the run continues and exits0; Go fails once and commits no incomplete tool.
Each capture retains its own build identity, truncation/cleanup/ambient-process
unknowns; process capture is not network tracing or universal absence-of-effects.

The bounded declaration/output fixture and local microbenchmark are reported in
BENCHMARKS.md, efficiency-report.json and go-efficiency.json. Byte reduction is
fixture-specific, not a tokenizer/live-cost or cross-agent benchmark. Native
live-account acceptance, server/API/ACP, richer protocol content/auth, general
artifact storage/portability/GC, multilanguage/dependency-ranked context and all
other broader compatibility/platform/release gates remain incomplete.

## Local HTTP API verification

The race suite now includes 187 top-level tests. The HTTP tests exercise
permission approval, question validation and retry, stale challenge rejection,
overlapping run rejection, explicit cancellation, disconnect cancellation,
project scope, bearer authentication, and journal recovery. The compiled
Darwin ARM64 smoke test checks startup on an assigned loopback port, streaming
events with matching run IDs, an approved shell command without the server
token in its environment, and graceful shutdown. Vet and all four rebuilt
release checksums passed.

## ACP stdio verification

Run `python3 scripts/test_acp.py bin/harness` from the implementation directory.
The fixture launches the compiled agent with real pipes and verifies version
negotiation, workspace scope, approval before a file write, prompt overlap
rejection, cancellation followed by a late approval, durable resume after a
restart, client MCP startup denial, and EOF and SIGTERM cleanup. The full Go
race suite and vet pass. The same stdio fixture also passes against a binary
built with the race detector. Evidence is under research/evidence/acp-*.

Generated-file protection tests cover preview and direct execution, explicit
configuration override, whole-patch rejection on a generated deletion or move,
and a marker after source code. Before/after exact-edit benchmarks use the
same fixture and are retained in research/evidence/generated-edit-{before,after}.txt.

Compaction receipt tests cover projected file and exit-code evidence, the exact
context estimate, portable export and import, and rejection of a modified
working view. Existing tests continue to cover legacy compaction, preserved
tool groups, and uncertain effects. Logs are retained under
research/evidence/compaction-receipt-*.

Project memory verification covers cross-project isolation, changed evidence,
deletion, symlink rejection, opt-in and explicit recall through the real
runtime loop in plan mode. The CLI smoke test exercises the compiled binary
against a temporary project. Logs are under research/evidence/project-memory-*.
