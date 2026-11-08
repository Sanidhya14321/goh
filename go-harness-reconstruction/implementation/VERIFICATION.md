# Verification record — 7 November 2026

This is a working development alpha. Full release acceptance in the supplied plan
is not complete. The observed host is macOS 26.5.1, ARM64, Apple M5 Pro, Go 1.27.3.
Live inference was tested on loopback. Source milestones are published to Sanidhya14321/goharness.

## RC23 editor and Windows filesystem

Native CI at RC22 (run 42386722466): `go vet` and `go test ./...` passed on
`ubuntu-latest` and `macos-latest`. On `windows-latest` the build and vet
passed and tests failed, mostly from directory fsync ("Access is denied") and
permission-bit privacy checks; a formatter test also depends on Unix scripts.
RC23 addresses the first two. Locally `1.0.0-rc.23` passed `go vet`, the race
suite, seven CLI scenarios and seventeen PTY scenarios twice. The new
`editor_handoff` PTY scenario sets `$VISUAL` to a script, opens it with Ctrl+X E,
and checks the journal records the rewritten prompt.

## Windows session delete (after RC33)

Native CI for RC32 and RC33 failed on Windows only: renaming a journal while
holding its lock handle is not allowed there, so trashing a session failed.
Commit `29bf905` keeps the lock held through the move on Unix and closes the
handle just before the journal moves on Windows. CI on `29bf905` passed all six
jobs.

## RC50 image attachments

`1.0.0-rc.50` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Runtime tests: an `@shot.png` mention reaches
the model as an image with the prompt text unchanged, the context estimate
counts it as a fixed cost rather than its base64 length, and summaries get a
path note instead of image data. Refusals: content not matching the
extension, over 5 MiB, outside the project, symlinked (Unix), missing, more
than four, and a failed run before any model call. Encoder tests check
`image_url` (Chat Completions), `input_image` (Responses) and base64 `image`
blocks (Anthropic), and that messages without images are unchanged. Live
vision acceptance is unverified.

## RC49 reasoning effort

`1.0.0-rc.49` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests: Chat Completions sends
`reasoning_effort`, Responses sends `reasoning.effort` only when configured,
validation accepts the four levels and refuses others and Anthropic.
Live acceptance of the parameter by OpenAI models is unverified.

## RC48 job visibility

`1.0.0-rc.48` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. The jobs TUI test now checks that an exited job
is not counted, a running one appears in the sidebar count and in the
`/status` section, using `ping` instead of `sleep` on Windows.

## RC47 /init

`1.0.0-rc.47` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests: `/init` expands with arguments, a user
command replaces it, and built-ins never push the command table past 256.
RC46 native CI (`b4013fc`) passed all six jobs.

## RC46 shell command rules

`1.0.0-rc.46` passed vet (macOS and Windows), the race suite, seven CLI
scenarios and twenty-three PTY scenarios (three consecutive runs after the
harness change below). Matcher tests cover allow and deny precedence, a later
allow carving an exception from a deny, and bypass attempts through `;`,
`&&`, pipes, backticks, `$(…)`, redirects, newlines and subshells. Runtime
tests check that rules decide without prompting and are journaled with
source `shell_rules`, that `ask` prompts even with a session grant, that
`job_start` and `child:shell` follow the rules, and that compound commands
fall back to normal approval. Known limit (documented): patterns match text,
so `rm *` misses `FOO=1 rm` or `/bin/rm`.

One PTY run failed in `editor_handoff` because a redraw split the reply on
screen. All scenarios that wait for a model reply now accept it from the
screen or the journal (`wait_reply`).

## RC45 loop guard

`1.0.0-rc.45` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. With a model that repeats one `read_file` call:
a headless run executes it twice, then stops with `needs_permission`;
declining blocks every later repeat while the run completes; a session
approval asks once and allows all repeats. Changing arguments resets the
count.

## RC44 instruction files

`1.0.0-rc.44` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests isolate the user config folder and check
that user and project guidance both reach the system message, that
`AGENTS.md` replaces `CLAUDE.md`, that project text stays labelled untrusted,
and that an oversized personal file is skipped. RC42 native CI (`cd052cd`)
passed all six jobs, including background job tests on Windows.

## RC43 commented configuration

`1.0.0-rc.43` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. JSONC tests cover line and block comments,
comment-like text and escapes inside strings, `//` in URLs, trailing commas
in objects and arrays, unterminated comments, preserved line numbers and
continued rejection of unknown fields. A new fuzz target
(`FuzzStripJSONCIdentityOnJSON`, 2.4 million executions in 10 s locally)
checks that valid JSON passes through byte-for-byte; it is in `make fuzz`.

## RC42 background jobs

`1.0.0-rc.42` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios (two consecutive runs). The new `background_job`
scenario approves a scripted `job_start`, confirms the job's grandchild
process is alive while the chat is open and gone after GoHarness exits.
Tool tests cover output capture and exit codes, stopping a job together with
a grandchild (Unix), the eight-job limit, `StopJobs`, argument and
directory validation, and that only `job_output` skips approval. Policy
tests: plan mode may read but not start or stop jobs; subagents cannot
start them. A TUI test lists a job, shows its output under its own title,
keeps `r` from replacing it and stops the job. Windows job stopping is
covered only by vet and the shared process-group code, not a native test.
A hard kill of GoHarness (SIGKILL) cannot run cleanup and may orphan jobs.

## RC41 typed shell commands

`1.0.0-rc.41` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-two PTY scenarios (two consecutive runs). The new `user_shell`
scenario types `!echo pty-shell-$((40+2))` and checks the shell-evaluated
output, a `user_typed` permission record, a journaled tool result and that
no model reply was requested. Runtime tests check the user/call/result
history, that the next turn's model input contains the output, that no
session grant is created, and refusals for empty commands and plan mode.

## RC40 command files and find highlighting

`1.0.0-rc.40` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). Discovery tests cover
frontmatter, CRLF, reserved and invalid names, empty and unterminated
frontmatter, oversized files, non-Markdown files, subfolders, symlinks and
duplicate names across folders; merge tests cover config precedence and the
256-command limit. A runtime test expands a project file command into the
exact prompt the model receives. Find tests check the marked match row,
that non-matches stay unmarked and that marks clear on Esc. A first race run
failed because file commands were merged once at startup, missing commands
configured later; they are now merged at use.

## RC39 transcript search and CLI revert

`1.0.0-rc.39` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). Find tests use a transcript
whose matches start off screen and check on the rendered view that the
newest match, then an older one 400 lines up, become visible; that keys wrap
between matches, typing never edits the draft, and Esc returns to the
bottom. `RevertTurn` rejects turn 0 and nonexistent turns and reverts turn 1
of 2 with both files restored.

## RC38 revert to an earlier turn

`1.0.0-rc.38` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). The new `timeline_revert`
scenario creates a file in a turn, opens the timeline, reverts with Ctrl+U
and checks the file is gone and the prompt is back. Rewind tests revert turn
2 of 3 (files from turns 2 and 3 removed, turn 1's kept, branch ends before
turn 2), redo restores both turns, an unknown prompt is refused without
changes, and repeated prompts select the right occurrence. Undo PTY
scenarios now use per-case session folders.

## RC37 task-state placement

`1.0.0-rc.37` passed vet (macOS and Windows), the race suite, seven CLI
scenarios and twenty PTY scenarios (three consecutive runs). New tests: all
earlier request messages are byte-identical across todo updates, task state
is never journaled or placed after a pending tool call, and the Anthropic and
Responses encoders accept the trailing state message after tool results.
One PTY run failed in `queued_prompt` because a redraw split "Second reply."
on screen; the scenario now confirms the reply from the journal. This may
also explain the unidentified RC34 flake. Cache-hit effect and model
behavior are unverified until a live run.

## RC36 record times and timestamps

`1.0.0-rc.36` passed vet (macOS and Windows), the race suite, seven CLI and
twenty PTY scenarios (two consecutive runs). The new `timestamps` scenario
toggles `/timestamps` in the real binary, checks a rendered turn time and
that every journal record is version 2 with a time. Unit tests: a version 1
journal opens, accepts version 2 appends and forks keeping both the original
times and their absence; inconsistent version/time pairs are rejected; NUL
in content cannot forge a timestamp line; repeated prompts get times in
order. A first PTY run failed because the hidden marker added a blank line
that changed the layout; fixed before commit.

## RC35 copy picker

`1.0.0-rc.35` passed vet (macOS and Windows), the race suite, seven CLI and
nineteen PTY scenarios. Unit tests check that system and tool content is never
offered, empty replies are skipped, order is newest first, filtering and Enter
copy the exact content, the draft is untouched and Esc closes the picker.
Timestamps were deferred: journal records carry no time and salvage rejects
unknown fields, so adding them is a journal-format change.

## RC34 status diagnostics

`1.0.0-rc.34` passed vet (macOS and Windows), the race suite, seven CLI
scenarios, nineteen PTY scenarios and the install test. One PTY suite run
failed and two immediate reruns passed; the failing scenario was not
captured, so it is recorded as an unexplained flake. New tests check the
validation reason, failover summary, theme line and integration states on
`/status`, and that `DegradedIntegrations` returns a copy.

## RC33 queued prompts

`1.0.0-rc.33` passed vet (macOS and Windows), the race suite, seven CLI and
nineteen PTY scenarios. The new `queued_prompt` scenario types a second prompt
while a 2.5 s reply is in flight, checks the `[queued]` marker, waits for the
second reply and confirms both prompts are journaled in order. Unit tests
cover slash-command and queue-limit refusals that keep the draft, blocked
dialog shortcuts during a run, the return of queued prompts after a cancelled
run, and scheduling after a completed one.

## RC32 session management

`1.0.0-rc.32` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. RC30 and RC31 native CI passed all six jobs. New tests
cover title cleaning (control and bidi characters removed, whitespace kept,
120-character limit) and clearing; trash and restore of journal, title and
artifacts; refusal to trash an open session and to overwrite on restore; the
browser's rename, two-step delete and current-session refusal; and the CLI
commands.

## RC30 credentials and cancellation under load

`1.0.0-rc.30` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. `TestParentChildCancellationUnderLoad` ran 15 times
with `-race` (90 parent/child cancellations): each returned within the 6 s
bound, each child shell process was gone (checked by PID; `pgrep` found no
leftover `sleep 30` afterwards), and each follow-up run refused with the
acknowledgement requirement without calling the model. The test is Unix-only
(it uses POSIX shell and signals); Windows process-tree cleanup is covered by
the native shell timeout test. `TestCorruptCredentialsAreReplacedBySignIn`
covers corrupt and world-readable credential files.

## RC29 damaged-session recovery

`1.0.0-rc.29` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. RC28 native CI (run 42391210737) passed all six jobs.
New tests: a damaged header no longer hides a healthy session; salvage keeps
the turns before mid-journal damage and leaves the file byte-identical; an
unanswered tool call at the cut is dropped; intact and headerless journals are
refused; the CLI path classifies damage, lists, and salvages.

## RC28 themes

`1.0.0-rc.28` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. The new `theme_light` PTY scenario starts chat with
`"theme": "light"` and checks the terminal output carries the light background
and not the dark one. Unit tests cover the picker (Esc keeps the theme, Enter
applies and drops cached rows) and rejection of unknown theme names.

## RC27 transcript navigation

`1.0.0-rc.27` passed vet (macOS and Windows), the race suite, seven CLI and
seventeen PTY scenarios. A new test renders a four-turn transcript and checks
the visible top row after previous, previous, next and first jumps, and that
last returns to the bottom. Clipboard delivery depends on the terminal's OSC 52
support and was not tested in a real terminal.

## Native Windows shell (after RC26)

New Windows-only tests run the `cmd.exe` shell path natively: output and
stderr, verbatim quoting, nonzero exit, and timeout cleanup. The first run
(42390079332) found that a non-forced `taskkill` cannot stop console process
trees, so every Windows timeout reported `cleanup_incomplete`; commit
`fa7c660` forces the tree kill. CI on `fa7c660` passed all six jobs (tests and
installers on Ubuntu, macOS and Windows). The installers now also refuse a
checksum-valid binary that does not run, keeping the existing install
(local test; idea from the Harvest installer).

## RC26 shortcuts and first all-green native CI

Native CI run 42389704720 (RC25): `go vet` and `go test ./...` passed on
`ubuntu-latest`, `macos-latest` and `windows-latest`, and the installer job
passed on all three. This is the first fully native pass on Windows; some
Windows tests skip POSIX-shell fixtures and permission-bit assertions, so the
Windows shell tool and ACL-based privacy still lack dedicated native tests.
Windows CI is no longer allowed to fail. Locally `1.0.0-rc.26` passed vet
(macOS and Windows), the race suite, seven CLI and seventeen PTY scenarios;
new tests cover keymap overrides, rejection rules and remapped keys in the UI.

## RC25 edit feedback

Native CI run 42389421084 (RC24): tests passed on Ubuntu and macOS; the
installer job passed on Ubuntu, macOS and Windows (clean install, tampered
binary refused, nothing installed). Windows tests had one remaining
permission-bit assertion, guarded in RC25. Locally `1.0.0-rc.25` passed vet
(macOS and Windows), the race suite, seven CLI and seventeen PTY scenarios.
New tests: errors from a running fixture server appear as `lsp_errors` with
warnings filtered out, and an edit with no running server neither requests
`lsp_start` nor attaches errors.

## RC24 installers and portability

Native CI run 42388824277 (portability fixes): macOS passed; Windows had four
remaining failures in internal/tools (permission-bit assertions, 8.3 temp
paths, control-character filenames); Ubuntu hit an LSP refusal race that is
now fixed in the client (0 failures in 60 local `-race` runs). Those were
addressed for RC24. Locally `1.0.0-rc.24` passed vet for macOS and Windows,
the race suite, seven CLI and seventeen PTY scenarios, and
`scripts/test_install.sh` (clean install; tampered binary refused with
nothing installed). The RC24 native CI run is the Windows and installer check.

RC23 native CI (run 42387111774): Ubuntu and macOS passed. Windows still
fails in cmd/harness, core, formatters, runtime, session, skills, snapshots
and tools. The directory-sync errors are gone; remaining causes include path
separators in glob/skill matching, Unix shell test fixtures, and file mode
assertions. Windows is not yet a supported platform.

## RC22 Windows build

`1.0.0-rc.22` passed `go vet`, the race suite, seven CLI fixture scenarios,
sixteen PTY scenarios (three consecutive runs) and the ACP suite on this Mac.
`go vet` (including test compilation) passed for windows/amd64,
windows/arm64, linux/amd64, linux/arm64 and darwin/amd64. `make release`
produced six binaries; `file` confirms PE32+ and ELF outputs, and SHA256SUMS
self-verifies. Not yet exercised natively: Linux and Windows runtime, the
Windows shell, lock and process cleanup. The CI workflow is the first native
check. The RC20 undo PTY scenario was timing-sensitive and is fixed here.

## RC21 compaction masking

`1.0.0-rc.21` passed `go vet`, the race suite, seven CLI fixture scenarios and
sixteen PTY scenarios. New tests cover compaction by masking alone with no
model call, placeholder identity, recent groups left verbatim, the
`observation_masking` journal record and its restoration on resume, the
fall-back to summarizing a masked view, and manual compaction still
summarizing. The existing summarizer-path test now disables masking
explicitly. Live providers were not used.

## RC20 undo and redo

`1.0.0-rc.20` passed `go vet`, the race suite, seven CLI fixture scenarios and
sixteen PTY scenarios. The new PTY scenario approves a file creation, types
`/undo`, confirms the file is gone and the prompt is back, then selects Redo in
the palette and confirms the file is restored. Fixture tests cover a two-turn
round trip, refusal after an external edit with rollback of the already
reverted file, shell effects reported as unrestorable, redo refused after a
new prompt, and forking and continuing an undo branch. Live providers were not
used.

## RC19 integration degradation

`1.0.0-rc.19` passed `go vet`, the race suite, seven CLI fixture scenarios and
fifteen PTY scenarios. New fixture tests start a healthy, a broken and a
missing MCP server together and invoke the healthy tool; run an agent whose
only MCP server fails discovery to completion with nothing pending; and run an
LSP query against a missing executable, then a follow-up turn that is not
blocked. The existing bad-initialize LSP test now expects completion with the
server stopped. No third-party servers were tested.

## RC18 provider failover

`1.0.0-rc.18` passed `go vet`, the full race suite, seven CLI fixture scenarios
and fifteen PTY scenarios on this Mac. New tests cover failover after 503,
connection and 429 failures; chained fallbacks; refusal to reroute after
authentication errors, partial streams, or with no list configured; config
validation; and a loopback HTTP run where three 503 attempts on the primary
route are followed by a streamed reply from the fallback endpoint. Failover was
not exercised against live providers.

## V1 release candidate

The installed macOS ARM64 `goh` binary reports `1.0.0-rc.17` and matches the
local build byte for byte. The race suite, `go vet`, seven CLI fixture scenarios,
fifteen PTY scenarios, and compiled ACP integration suite passed on this
candidate. Eleven bounded fuzz targets passed on RC1. A TUI regression covers a
narrow viewport with wide text, tabbed previews, and long input.

RC17 evidence is under `research/evidence/v1-chatgpt-*`. The live
OpenAI test used a user-approved GoHarness ChatGPT grant and the account's
`gpt-6.1-sol` model. It verified streamed text, a permission-gated file edit,
read-back, durable resume, and a real model turn through the terminal
`/connect` picker. Fixture tests cover PKCE and callback state, signed identity,
private storage, refresh rotation, exact endpoint binding, and refusal to
execute tool calls from a partial Responses stream.

RC16 evidence is under `research/evidence/v1-audit-*`. Compiled setup tests
cover provider draft recovery and the gateway fallback. The shell cancellation
terminal test uses Esc. Unit checks cover credential routing, JSON reply bounds
and malformed replies. The native OpenAI probe reached HTTP 401 without a key.
A separate signed-in Codex CLI probe completed. The
[end-to-end audit](END_TO_END_AUDIT.md) records the remaining account,
integration, UI and platform checks.

RC15 evidence is under `research/evidence/v1-ui-export-*`. The terminal setup
flow tests cancellation, successful export, private file mode, collision
handling, unchanged journal bytes and request counts, and matching CLI Markdown.
Focused tests verify optional tool details, history retained after compaction,
and cleanup after a failed write. The paired PTY capture also saves a transcript
with tool details enabled and verifies the resulting tool calls and results.

RC14 evidence is under `research/evidence/v1-ui-fork-*`. The terminal setup
flow cancels a fork without losing the source draft, branches before a numbered
turn, verifies the separate journal and unchanged source, and sends an edited
draft from the new chat. It also exercises the CLI boundary before the first
turn. Focused journal tests verify repeated prompts, retained compaction state,
branch provenance, and invalid boundaries. Paired PTY captures show both fork
menus.

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
