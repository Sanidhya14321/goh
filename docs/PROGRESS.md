# Implementation progress — 8 November 2026

A working development alpha is implemented at the repository root (`cmd/`, `internal/`). The supplied
plan at `plan/IMPLEMENTATION_PLAN.md` remains unchanged. Full release acceptance
is not complete; observed compatibility and remaining gates are separate below.

## Readiness at RC109 (9 November 2026)

| Subsystem | Estimate | Done locally | Waiting on |
| --- | --- | --- | --- |
| Agent loop and recovery | ~93% | Journaled effects, undo/revert, loop guard, default checkpoints, orphaned-job recovery, cancellation under load on all three OSes, compaction margin (found live), guided salvage of damaged sessions, provider cooldowns, visible retries, ChatGPT 401 renewal | Live provider failover and token-expiry runs |
| Providers | ~82% | Chat Completions, Responses, Sign in with ChatGPT (verified live: streaming, tools, resume, images, reasoning effort), Anthropic incl. extended thinking, model lists, streamed reasoning display | Live check of Responses reasoning summaries; user keys for Anthropic thinking and API-key model lists |
| Tools and integrations | ~90% | Files, patches, shell and file rules, background jobs, local and remote MCP (`goh mcp list`), LSP with post-edit errors verified against real clangd, approved restarts, lsp_auto, gitignore-aware grep, @path attachments, web fetch and search | Third-party MCP services and OAuth; language servers other than clangd |
| Terminal UI | ~90% | Fuzzy palette and pickers, /find /copy /details /timestamps /changes /jobs /rename /thinking /effort /exit, plan handoff, context meter, themes | Visual comparison passes against OpenCode |
| CLI | ~90% | run (words, stdin, --file, --title, --command, --reasoning-effort, --continue), export/import, sessions (table), agents, models, stats, mcp list, /review and /init built-ins | — |
| Platforms | ~85% | Native CI on Ubuntu, macOS and Windows for every commit | Native Windows manual session |
| Release | ~60% | Cross-builds, checksums, installers tested in CI | User approval to publish; release workflow deferred by request |

Not attempted, needing a decision first: custom tool plugins (they would
execute repository code and need a security design), session sharing
(uploads conversations), and MCP OAuth (needs a live server).

## RC149 C4 repository understanding milestone

repo_map covers eight language families.

## RC148 C3 context milestone

Recent-window masking.

## RC147 C2 agent loop milestone

Evidence-gated completion reminder; automatic messages separated from user turns.

## RC146 C1 tool editing milestone

Component plan started (docs/COMPONENT_PLAN.md); C1 tolerant unique-match edits.

## RC145 shared budgets milestone

R21 now open only for the live benchmark suite (needs keys) and cost budgets.

## RC144 verify/repair milestone

R21 partial: verify/repair, max_tool_calls, adversarial canary. Live-model benchmark suite still needs keys.

## RC143 providers and guidance milestone

R19 and R20 fixed (fixture-verified; live model checks pending).

## RC142 constraints and memory milestone

R17 and R18 fixed.

## RC141 scoped grants milestone

R15 fixed. All P0 and P1 review findings are closed.

## RC140 outcomes milestone

R11 (outcome field, empty replies) and R16 fixed. A fuller verify/repair workflow stays on the roadmap (R21).

## RC139 usage attribution milestone

R14 fixed.

## RC138 external review P1 batch 1

R10, R12, R13 fixed.

## RC137 external review P0 batch 2

R6–R9 fixed; all P0 review findings are now closed.

## RC136 external review P0 batch 1

R1–R5 from docs/REVIEW_FINDINGS.md validated and fixed.

## RC135 MCP reconnect milestone

Gap analysis M3 closed. M1 (resources/prompts) and M2 (OAuth) remain open.

## RC134 shell sandbox milestone

Gap analysis S3 closed (opt-in): sandbox-exec on macOS, Landlock on Linux, refusal elsewhere; also stops shell commands reading GoHarness credentials when enabled.

## RC133 whole-project snapshots milestone

Gap analysis A2 closed (opt-in): shadow-Git project snapshots per run with CLI list/diff/restore/undo. TUI integration and default-on remain follow-ups.

## RC132 hooks milestone

Gap analysis A1 closed: user-config hooks that can refuse or add feedback, never grant.

## RC131 prompt caching milestone

Gap analysis P1 closed (fixture-verified; live hit rate unmeasured).

## RC130 environment scrubbing milestone

Gap analysis S1/S2 closed: child processes receive only a safe environment plus `env_passthrough`.

## RC129 repository restructure milestone

Go project at the repository root, docs in `docs/`, plan/specs in `docs/plan/`, research in `research/`; repository and module renamed to `goh`.

## RC128 git worktree milestone

Checkpoint: isolated worktree sessions (`goh worktree NAME|list`). Architecture overview written to `ARCHITECTURE_OVERVIEW.md`.

## RC127 sessions list --project milestone

Backlog R08: per-project session listing.

## RC126 deeper completion milestone

Backlog I11: subcommand and session-ID completion.

## RC125 sessions prune milestone

Backlog R08: bulk housekeeping through the existing restorable trash.

## RC124 tool statistics milestone

Backlog R22: tool usage in `goh stats`, as in OpenCode's stats.

## RC123 piped run input milestone

Backlog I01: headless run reads piped input, as OpenCode does.

## RC122 goh config milestone

Backlog R06: inspect configuration precedence without exposing secrets.

## RC121 command agent milestone

Backlog R05: command-specific agent for user-configured commands (model override still not offered: it could change the provider).

## RC120 goh uninstall milestone

Backlog I11: uninstall, tested only in disposable prefixes; upgrade waits for a published release.

## RC119 goh auth list milestone

Backlog R03: read-only credential overview; storage is unchanged.

## RC118 Ctrl+Z suspend milestone

Backlog I02: terminal suspend (the "suspend" gap in the full-TUI row).

## RC117 goh pr milestone

Backlog I09: PR checkout through the user's gh; GitHub automation (`goh github`) still open.

## RC116 shell completion milestone

Backlog I11: shell completion, generated locally; nothing written to shell configuration.

## RC115 formatter auto-detection milestone

Backlog R18: built-in formatter discovery, opt-in like `lsp_auto`.

## RC114 HTML from /export milestone

TUI parity for the HTML transcript.

## RC113 HTML export milestone

Backlog R10: shareable output without uploading anything; publishing
(share/unshare) still awaits a decision.

## RC112 run --fork milestone

Backlog R09: continue a copy of a session from the CLI.

## RC111 sidebar cost milestone

Live cost estimate beside reported tokens.

## RC110 cost estimates milestone

Backlog R22: user-priced cost estimates in stats; no bundled prices.

## RC109 grep ignore rules milestone

Grep now matches the observed reference by leaving gitignored files out;
glob keeps its observed include-everything behaviour.

## RC108 file mentions milestone

OpenCode-style content attachments for `@path`, bounded and confined to
the project.

## RC107 command suggestions milestone

Helpful failures for mistyped slash commands.

## RC106 skill diagnostics milestone

Skill load problems are readable and listed in /status.

## RC105 provider cooldowns milestone

Failed routes are skipped for a minute while fallbacks exist; the primary
remains the last resort, so a cooldown never causes an outage on its own.

## RC104 retry visibility milestone

Retries are reported through a context-scoped notifier, as reasoning text is.

## RC103 damaged sessions in the picker milestone

Damaged journals are visible and recoverable from the picker.

## RC102 guided salvage milestone

Resume offers recovery of damaged journals, asking first and leaving the
original untouched.

## RC101 slow diagnostics milestone

Real-server CI exposed that slow diagnostics killed healthy servers; fixed
with a race-free wait.

## RC100 language server restart milestone

Closes the audit gap on restarting a disconnected server within a run, with
fresh approval and a per-run cap.

## RC99 real language server milestone

First end-to-end run against a real server (clangd) in the test suite.

## RC98 language server detection milestone

Opt-in discovery of installed language servers; approval boundaries unchanged.

## RC97 compact --focus milestone

CLI parity for steered compaction.

## RC96 compaction focus milestone

User-steerable manual compaction.

## RC95 ChatGPT 401 renewal milestone

Recovery from early token revocation: one forced renewal under the
credential lock, then a resend that is safe because nothing was generated.

## RC94 sidebar title milestone

Session titles are visible in the chat sidebar.

## RC93 automatic titles milestone

Opt-in model-written session titles via the `title` role; best-effort and
bounded to the first prompt.

## RC92 ChatGPT default everywhere milestone

ACP, serve and models now pick the provider the same way as chat and run.

## RC91 ChatGPT default milestone

One provider-default rule for chat, run and doctor. The versioned build now
comes after acceptance, which had been overwriting it.

## RC90 agent completion milestone

`@` completion lists matching subagents before files.

## RC89 agent mentions milestone

Prompt-level `@subagent` delegation hints; authority stays with the task
tool's approval.

## RC88 session table milestone

Human-readable session listing beside the JSON default.

## RC87 reasoning-effort flag milestone

Per-invocation effort override, validated with the config rules.

## RC86 effort command milestone

Session-scoped reasoning effort; the stock provider client is rebuilt
between runs and injected models are left alone.

## RC85 export and import milestone

OpenCode-style top-level export/import over the existing portable format.

## RC84 MCP list milestone

Read-only MCP configuration report with secret-safe output.

## RC83 agents list milestone

`goh agents` shows the agent catalogue the TUI cycles through.

## RC82 review command milestone

Second built-in command beside `/init`; user command files still win.

## RC81 ACP reasoning milestone

Reasoning reaches ACP clients as thought chunks.

## RC80 reasoning display milestone

Providers pass reasoning text through a context-scoped sink, so the model
interface is unchanged. The TUI writes it as one marked block per step
behind the `/thinking` toggle.

## RC79 command help milestone

Described top-level help, tested against the dispatcher; the setup PTY
suite is repaired and joins the per-milestone gates.

## RC78 run --command milestone

Headless runs can invoke commands through the same expansion as the TUI.

## RC77 run --title milestone

Headless runs can name their session through the shared metadata sidecar.

## RC76 run --file milestone

Headless runs can attach project files; reads go through os.Root so the
project boundary holds.

## RC75 transcript redaction milestone

Backlog I14 (first slice): shareable Markdown transcripts mask
recognisable credentials; the importable JSON export is unchanged.

## RC74 exit command milestone

`/exit` and `/quit` follow the Ctrl+D rule: quit when idle, cancel first
during a run.

## RC73 session rename milestone

`/rename` in the chat sets or resets the open session's title through the
same metadata sidecar the session picker uses.

## RC71 web search milestone

Backlog T14: an opt-in, keyless search backend (SearXNG JSON) rather than a
vendor account; queries leave the machine only with explicit configuration
and per-search approval.

## RC70 extended thinking milestone

Backlog R01 (Anthropic reasoning): thinking blocks are journaled on the
assistant message (`anthropic_thinking`) and replayed only to Anthropic while
thinking is enabled, mirroring how Responses reasoning items are scoped.

## RC69 fuzzy pickers milestone

One ranking function serves the palette and every option picker.

## RC68 LSP operations milestone

Backlog T15: type definitions and call hierarchy (prepare, then incoming or
outgoing calls on the first prepared item), gated on server capabilities.

## RC67 plan handoff milestone

Backlog T16: `plan_exit` exists only for the top-level plan agent and always
goes through the permission prompt. Mode never changes inside a run; the UI
restarts the same session with the build agent afterwards.

## RC66 orphaned jobs milestone

A private job registry in session storage (one file per GoHarness process)
survives hard kills. Orphans are only stopped by explicit request and only
after the live command line matches, avoiding PID-reuse accidents.

## RC65 compaction threshold milestone

The early compaction trigger is configurable; the fit-based hard trigger
is unchanged, so a high threshold cannot cause overflow errors.

## RC64 tool details milestone

Tool results render through summary/detail marker lines filtered at display
time; `/find` and rendering share `displayLog`, so search and screen agree.

## RC63 file rules milestone

Backlog R07 (edit patterns): `file_rules` reuse the shell rule matcher on
paths extracted by the same code that chooses checkpoint paths, so rules
see exactly the files an edit or patch touches.

## RC62 context meter milestone

Sidebar context fill from the latest reply's reported prompt and output
tokens; providers that report no usage show no meter rather than a guess.

## RC61 ACP HTTP MCP milestone

ACP `session/new` accepts `type: "http"` MCP servers, mapped to RC57's
remote transport with in-memory headers (`json:"-"`, never persisted).

## RC60 stats milestone

Each reply with reported usage appends a `usage` record (excluded from
replay, export and fork). `goh stats` reads journals without locks, so it
works beside open chats.

## RC59 /models provider list milestone

The launcher carries a per-chat `--model` override chosen from the
provider's catalog, cleared when a profile or provider changes.

## RC58 model list milestone

`goh models` queries the configured provider's model list using the same
endpoint host and credential selection as chat requests. Live provider
catalogs are unverified until keys are added.

## RC57 remote MCP milestone

OpenCode parity for remote MCP using the pinned go-sdk's Streamable HTTP
client (no new dependency). Credentials come only from named environment
variables, redirects are refused so headers cannot leak to another host,
and approval boundaries are unchanged.

## RC56 default checkpoints milestone

Checkpoints default to best effort, matching OpenCode's always-on snapshots
without letting storage limits block edits. Explicit opt-in keeps the strict
guarantee. Undo already reports uncheckpointed effects as not restorable.

## RC55 changes panel milestone

`Agent.Checkpoints`/`CheckpointDiff` expose the snapshot store read-only to
the TUI; `/changes` lists and diffs them. Reverting stays with turn-level
undo so files and conversation remain consistent.

## RC54 continue milestone

`--continue` picks the newest readable session for the canonical project
path (damaged journals skipped); `run` gains `--session`.

## RC53 run ergonomics milestone

`goh run` accepts OpenCode-style positional messages with interleaved flags
and explicit `-` for standard input (never read implicitly, so scripts with
open pipes cannot hang).

## RC52 agent files everywhere milestone

Closes the RC51 gaps: serve and ACP merge project agent files, and skipped
files are visible on `/status`.

## RC51 project agent files milestone

OpenCode/Claude Code-style agent Markdown files merge under configured
agents before profile resolution. As with command files, repository content
cannot select a model profile or provider.

## RC50 image attachments milestone

Backlog R14 (images): `core.Image` on user messages, journaled with the
message so resume, fork and undo keep them. Attachment goes through os.Root
with magic-byte checks. Image bytes are excluded from token estimates and
summarizer input.

## RC49 reasoning effort milestone

Backlog R01 (reasoning, partial): configurable effort for OpenAI protocols.
Anthropic extended thinking stays refused because thinking blocks would need
to round-trip through history.

## RC48 job visibility milestone

Running background jobs are visible in the sidebar and `/status`, so a dev
server started by the agent is never invisible.

## RC47 /init milestone

Built-in default commands sit beneath configured and file commands; the
first is `/init`, an original prompt for writing AGENTS.md. It only expands
text, so the model's file writes still need normal approval.

## RC46 shell rules milestone

Backlog R07 (patterns): `shell_rules` with last-match-wins glob patterns.
Unlike a prefix match, auto-allow refuses any command with shell operators
and deny checks every segment, so compound commands cannot smuggle a denied
command past an allow rule. Rules are checked inside `Permission`, before
session grants.

## RC45 loop guard milestone

Backlog R07 (doom loop): the third consecutive identical call (tool name and
exact argument text) requires a `repeated_call` approval, independent of the
tool's own grant. Motivated by OpenCode's doom-loop permission and
SWE-agent's observation that agents get stuck repeating actions.

## RC44 instruction files milestone

Backlog T01 (instruction loading): CLAUDE.md fallback and a per-user
AGENTS.md beside the config file, both bounded to 64 KiB and framed as
guidance without authority. Files in use appear on `/status`.

## RC43 commented configuration milestone

Backlog R06 (partial): JSONC config with comments and trailing commas,
string-aware and position-preserving. The provider setup's JSON round trip
now refuses commented files instead of silently dropping comments.

## RC42 background jobs milestone

Backlog R15: `job_start`/`job_output`/`job_stop` tools on the tool engine,
reusing the owned process-group termination from `shell`. Jobs are
process-lifetime state, never journaled for replay: every agent owner
(chat, local API, ACP, child) stops its jobs when it is discarded. The
journal still records the approved `job_start` call, so undo reports it as
unrestorable.

## RC41 typed shell milestone

`Agent.RunShell` runs `!command` through the same journaled path as model
shell calls (tool_start/tool_result, so an interruption becomes an uncertain
effect) with the user's typing as the recorded approval. History gets the
user's `!command`, a shell call and its result, so every protocol sees a
valid user → call → result sequence.

## RC40 command files milestone

Backlog R05: Markdown command files load from OpenCode-compatible project
folders. Configured commands always win, and frontmatter `agent`/`model`
overrides are ignored so repository files cannot change the selected mode,
model or provider. The compatibility matrix no longer lists Windows,
subagents and native providers as outside the alpha. `/find` now marks
matching rows.

## RC39 transcript search milestone

`/find` searches the full transcript text rather than the wrapped window, then
scrolls by wrapping only the text after the match, so old turns are reachable
without rendering the whole log. `goh changes revert-turn` exposes RC38's
revert by turn number.

## RC38 revert-to-turn milestone

`rewind.RevertTo` generalizes undo from the latest turn to any turn: it
reverts that turn and all later ones with the same safeguards (refused while
tool effects are uncertain, all-or-nothing file restore, source journal
untouched, redo available). The turn is matched by prompt content and
occurrence rather than position, so a stale UI list cannot pick another turn.

## RC37 task-state placement milestone

Research experiment 3: the todo list moved from the system message to a
trailing, request-only user message framed as state data. The request prefix
now stays stable across todo updates (cache-friendly) and the plan sits at
the recency end of the context. Live cache and behavior checks remain.

## RC36 record times milestone

Journal records now carry a write time as version 2; version 1 records stay
valid, and a record must match its version (v1 without time, v2 with), so
older binaries refuse new journals as an unsupported version instead of
calling them damaged. Fork and salvage copy times rather than restamping.
`/timestamps` toggles OpenCode-style turn times using NUL-prefixed marker
lines that sanitized content cannot produce.

## RC35 copy picker milestone

`/copy` (also in the command palette) lists recent user and assistant
messages and copies the chosen one through OSC 52, extending the RC27
copy-last-reply action to any message. Timestamp display is deferred until
the journal records message times, which needs a versioned format change.

## RC34 status diagnostics milestone

`/status` previously said only "Configuration needs attention". It now shows
the validation error (validators never echo URLs, so no endpoint secret
appears), the theme, explicit failover profiles or "Failover: off", and marks
MCP/LSP servers that failed to start and were stopped in this process, read
through a mutex-guarded copy so the screen can open during a run.

## RC33 queued prompts milestone

The composer previously ignored every key except Esc and paging while a run
was active. It now stays editable, and Enter queues up to five prompts that are
sent in order after completed runs, following OpenCode's queued prompts. Any
other outcome returns the queue to the composer, so a failed, cancelled or
approval-blocked run never triggers further sends. Research follow-up noted:
todo state is appended to the system message, which changes the cached prompt
prefix on every update; moving it needs live checks on all four protocols.

## RC32 session management milestone

The session browser renames sessions with Ctrl+R and deletes them with a
confirmed Ctrl+D, following OpenCode's session list. Deletion is recoverable:
the journal, title, artifacts and checkpoints move to a trash folder, a
session open in any process is refused via its writer lock, and restore never
overwrites. Titles live in a private sidecar so the journal stays append-only.
RC31 made on-screen shortcut hints follow the keymap.

## RC30 credential recovery and cancellation-under-load milestone

Damaged ChatGPT credentials now produce errors that name the file and the
recovery command, and a test shows a new sign-in replaces corrupt or
world-readable files. A load test cancels six concurrent parents while their
child tasks run a long shell command; 15 repetitions under the race detector
(90 cancellations) returned promptly, left no child process alive, and kept
each interrupted effect pending so the next run required acknowledgement.

## RC29 damaged-session recovery milestone

A probe showed that one journal with a damaged header made session listing
fail, hiding every saved session. Listing now skips and reports damaged
journals. `goh sessions salvage ID` copies the readable prefix of a damaged
journal into a new session using the fork validation (no tool replay or
grants; an incomplete tool group is dropped), and leaves the original
untouched. Resume errors for damaged journals name the salvage command.
Torn tails from crashes were already repaired automatically.

## RC28 themes milestone

Three built-in themes with a searchable picker (`/theme`, `<leader>t`, as in
OpenCode's `theme_list`) and a `theme` setting validated at load. Palettes
come from a fixed table, preserving the rule that only trusted sequences
reach the terminal. A new PTY scenario checks the light background is
actually rendered.

## RC27 transcript navigation milestone

Leader and Ctrl+Alt shortcuts jump to the first or last message and between
user turns, scroll half pages, copy the last assistant reply through the
terminal clipboard, and undo or redo the last turn. Defaults follow OpenCode's
`messages_*` actions where the key is free in GoHarness, and every action can
be remapped. A changelog now records release history and ships with releases.

## RC26 configurable shortcuts milestone

`keybinds` remaps the chat-level actions (leader, command palette, sidebar,
timeline, status, external editor), following OpenCode's keybind config and
leader model. Invalid, conflicting or reserved bindings are rejected at load.
Native CI for RC25 passed on Ubuntu, macOS and Windows for both the test
suite and the installers; Windows is now a required CI platform.

## RC25 edit feedback milestone

After a successful built-in edit, GoHarness asks a language server that is
already running in the run for the edited file's diagnostics and adds
error-severity items (capped at 20 per file) to the tool result as
`lsp_errors`, so the model sees breakage in the same turn. It never starts a
server, so `lsp_start` approval is unchanged, and warnings are not reported.
This follows OpenCode's post-edit diagnostics and the lint-gated edit
ablation in SWE-agent recorded in research_notes. Its effect on task success
has not been measured in GoHarness. RC24 CI passed the installers natively on
all three systems.

## RC24 installers, Windows portability and file search milestone

`install.sh` and `install.ps1` install with checksum verification and are
shipped in the release; a CI job tests a clean and a tampered install natively
on all three systems. Windows path handling now works for glob, grep, repo map
and skills (fs.WalkDir requires slash paths), and locks no longer block readers
of an active journal. Native Windows CI went from about 25 failing tests to 4,
then to 1 at RC24 (see VERIFICATION). `@` completion honors
`.gitignore` rules and refreshes its snapshot after runs or 15 seconds. The LSP
client now settles replies to server requests before shutdown, fixing a
flaky refusal check seen on Ubuntu CI.

## RC23 external editor and Windows filesystem milestone

Ctrl+X E and `/editor` hand the draft to `$VISUAL`/`$EDITOR` and load the result
back into the composer, following OpenCode's leader binding and editor
behavior. Windows directory syncs are now no-ops and the Unix permission-bit
privacy checks rely on profile ACLs there, which removes the root causes of
most native Windows test failures in the first CI run. macOS and Ubuntu passed
the full suite natively on GitHub Actions at RC22.

## RC22 Windows build milestone

GoHarness now compiles and vets for Windows amd64 and arm64, alongside macOS
and Linux. Locks, no-follow opens and process-group cleanup go through a new
`internal/platform` package: flock and O_NOFOLLOW on Unix; LockFileEx, a
reparse-point check and `taskkill /T` on Windows. The shell tool uses
`/bin/sh -c` on Unix and `cmd.exe /d /s /c` on Windows. The release script
builds six targets with verified SHA256SUMS. A GitHub Actions workflow runs
vet and tests natively on Ubuntu, macOS and Windows. Windows is cross-compiled
only until that workflow reports; its no-follow check is weaker than
O_NOFOLLOW against a concurrent swap.

## RC21 compaction masking milestone

Automatic compaction masks old tool output before calling a summary model and
skips the model when the masked view fits with headroom. The summarizer, when
still needed, receives the masked view. Full results stay in the journal; manual
`/compact` and `"compaction_masking": false` keep summarization. Based on the
observation-masking evidence recorded in research_notes. A fixture
measurement is in BENCHMARKS.md; solve-rate effects are unmeasured.

## RC20 undo and redo milestone

`/undo` reverts the files the last turn changed through checkpointed tools and
continues in a branch that ends before that turn, with the prompt back in the
composer. `/redo` reapplies them and returns to the original conversation,
until a new prompt is sent. Effects without checkpoints (shell, MCP) are named,
not reverted. Files changed outside GoHarness cause a refusal that leaves files
unchanged. The source journal is never edited. CLI: `goh changes undo-turn` and
`redo-turn`. The behavior follows OpenCode's observed revert workflow; the
implementation is original and reuses GoHarness fork and snapshot code.

## RC19 integration degradation milestone

A failing optional integration no longer fails the run. An approved MCP server
that cannot start, initialize or present valid tools is stopped and skipped;
the other servers and built-in tools stay available. An approved language
server that cannot spawn or initialize is stopped and settled, so it no longer
leaves an uncertain effect that blocks the next run. Cleanup that cannot be
confirmed still stays uncertain. Each case is journaled as
`integration_degraded` and explained in chat. Approval denial is unchanged.

## RC18 provider failover milestone

`fallback_profiles` names an ordered list of configured profiles that receive a
turn or compaction request only when the active route fails before returning
any response (connection, 408, 429, 5xx after bounded retries). Authentication
errors, cancellation and failures after a stream began are never rerouted. Each
attempt or skip is journaled and shown in chat. Verified with fixture models and
a loopback HTTP test; no live multi-provider run yet.

## RC17 OpenAI connection milestone

GoHarness supports browser authorization for an eligible ChatGPT plan, an
account model catalog, and native streamed OpenAI Responses. The OpenCode-style
`/connect` selector also keeps API key and compatible gateway setup available.
The live check verified streamed text, a permission-gated read/edit/verify
cycle, journal resume, and a terminal conversation opened through `/connect`.
The audit lists remaining recovery and parity gaps.

## RC16 recovery milestone

GoHarness preserves drafts while switching provider settings and profiles, and
Esc cancels an active run. The Chat Completions adapter handles compatible
JSON replies and a narrow streaming-option fallback. The audit tracks tested
recovery paths, the native ChatGPT authorization gap, terminal parity and
platform release gates. See [the audit](END_TO_END_AUDIT.md).

## RC15 UI milestone

The Go chat now has an /export dialog for Markdown transcripts. It uses the
durable conversation journal, lets the user edit the filename and opt into
tool details, and publishes a new private file. The CLI shares the Markdown
formatter and the existing portable JSON export remains its default.

## RC14 UI milestone

The Go composer now has a searchable /fork dialog for the full session or an
earlier durable user turn. Branching reuses the portable journal validation
and publishes a fresh session with the selected prefix. It opens the chosen
prompt as an editable draft and shares the source workspace. The CLI accepts
`sessions fork ID --before-turn N` with the same boundary semantics.

## RC13 UI milestone

The composer now opens a project file menu above the draft when the user types
`@`. The Go search ranks names and paths, supports fuzzy abbreviations, and
inserts a quoted relative path when needed. It scans directory entries in the
background on first use and caches the bounded index for the chat. Selecting a
path keeps the draft editable. Fixture tests verify a separate send action and
that the provider receives the path without the file body.

## RC12 UI milestone

The chat now has /timeline and Ctrl+T, a searchable recent-prompt list inspired
by OpenCode's timeline dialog. Enter opens a scrollable preview and Ctrl+R
recalls the prompt into the composer. Cancellation preserves the draft; recall
requires a separate send action. The list reuses the bounded 200-prompt history
and includes only retained runtime history after resume. It does not implement
OpenCode's message revert, file undo, fork, or timeline-to-transcript jump.
Preview wrapping is cached per selection and width. No runtime dependency was
added. Source and isolated terminal captures document the reference and result.

## V1 release candidate

GoHarness `1.0.0-rc.17` is installed as `goh` on the development Mac.
Bare `go` opens chat in the local zsh profile. Incomplete configuration
displays a setup message in chat and blocks prompt submission. Provider,
workspace, session selection, and settings are accessible through chat commands.
Correcting a rejected API key retains the session.
The Go UI now follows the inspected OpenCode layout patterns with a centered
welcome screen, prompt panel, searchable command palette, session sidebar,
transcript scrolling, themed settings, and colored approval diffs.
The composer now supports cursor editing, multiline navigation, persisted-session
prompt recall, draft restoration, and slash completion. Custom commands appear
in the palette. Conversation rendering adds Markdown headings, lists, quotes,
and fenced code panels while preserving literal code and approval contents.
Pipe tables now have aligned grids, wrapping, and a stacked fallback for narrow
panes. Code panels highlight Go, JSON, JavaScript, and TypeScript lexically.
Searchable model-profile and agent selectors are available through /models and
/agents. Tab/Shift+Tab cycle agents, and /new starts a fresh conversation.
Model IDs in provider requests and Plan's read-only tool definitions were checked through
the executable in a disposable terminal.
The session browser now has title/ID/workspace search, current-workspace scope,
newest first ordering, and bounded asynchronous conversation previews.
Narrow terminals can open a full-width preview with Tab. The preview command
reads complete journal-prefix records and reports unfinished runs.
The local status dialog links provider/model/agent/settings configuration and
summarizes context budgets, credential presence, explicit grants, integrations,
and formatter availability. Its snapshot preserves drafts and performs no
provider request or server startup.
Resumed tool-result displays now reuse the live result formatter and retain
error labels while hiding internal transport envelopes.
An intermittent child-cancellation cleanup result observed under concurrent
verification remains a release investigation item.
The previous `harness` command points to the same build. The terminal UI now wraps by
display cells and keeps approval controls visible when the prompt or preview
contains wide characters or long lines. The local fixture checks and remaining
gates are recorded in [Verification](VERIFICATION.md#v1-release-candidate).

The authenticated live Chat Completions smoke test passed on the Mac with
llama.cpp and Qwen3. V1 will ship as one version across macOS, Linux, and Windows.
The remaining release gates are a Windows port that preserves process cleanup
and file safety, native smoke tests on the supported platforms, and
end-to-end tests of the one-command installers against checksummed artifacts.
The coordinated V1 release follows native platform and installer validation.

The public source repository is [GoHarness](https://github.com/Sanidhya14321/goharness).
Git history begins with the RC8 implementation baseline and then records
subsequent UI work. Published V1 acceptance still depends on the gates above.

## Part checkpoints

| Part | Status | Working capability, checks and remaining gate |
|---|---|---|
| 0 Environment/provenance | complete | Official OpenCode v1.19.5 tag/commit verified; publisher artifact digest matches; local Go 1.27.3 checksum verified; REA 6.8.0 actual CLI/version process capture; manifest and origin ledger retained |
| 1 Reference/driver | complete | Original scripted model fixtures, real REA text/read-edit-test/permission/drop captures; original Go fixture service and executable acceptance driver; feature map and contracts |
| 2 CLI/config/events | complete | Buildable text/JSON CLI, actual doctor diagnostics, strict JSON config, credentials by env name, versioned ordered events; actual executable 7-scenario acceptance passes |
| 3 Provider/runtime | complete | Bounded SSE assembly, UTF-8 fragmentation, tool indices/IDs, usage chunks, before-content retries, no partial-response replay, cancellation/turn/invalid-call guards; parser/runtime/race/fuzz checks pass |
| 4 Read tools | complete | Anchored bounded Go listing/read/regex search; full hashes, schemas, up to 4 adjacent reads with ordered results; traversal/special-file/binary tests and paired read evidence |
| 5 Permissions/edits | complete | Headless needs_permission, TUI once/session decisions, exact hash-guarded edits, atomic replacement, no-overwrite creation, mode/CRLF preservation; stale/ambiguity/approval/refusal/cancel-admission tests; paired intentional matching differences |
| 6 Shell/lifecycle | in_progress | Go process groups, concurrent drains, bounded artifacts, TERM/2s/KILL, timeout/cancel/child tests and verified Go REA cleanup on host. Reference cancel capture returned cleanup_incomplete; Linux runtime gate remains open |
| 7 Sessions/recovery | in_progress | Durable private JSONL, exclusive writer, torn-tail/corruption/version tests, uncertain starts/no replay/acknowledgment/reset grants; normal resume paired through REA. Reference corruption/interrupted-recovery parity unverified |
| 8 Context | in_progress | Conservative accounting, reserves, original summarization, active request/recent groups/unresolved-call preservation, durable summary, failed/ineffective/budget tests pass. Reference compaction unobserved |
| 9 TUI | complete | Bubble Tea over same runtime, multiline input, streaming/activity, scrollable diff approvals, tokens/timing, chooser/resume/history, resize, separate cancel/exit. Five real PTY acceptance cases verify termios restoration; reference text/resize/exit paired, full interactive reference parity unverified |
| 10 Local MCP | complete | Official SDK stdio, explicit server/tool grants, durable startup, namespacing/schema validation/no remote refs, bounded frames/results/timeouts, cancellation/disconnect/group cleanup/no replay. Original fixture and race tests; paired echo/disconnect plus uncertain resume observations |
| 11 Release gates | in_progress | Four stripped CGO-free platform binaries/checksums/build metadata/notices prepared locally; 47 top-level tests, race, vet, fuzz,7 CLI cases,5 PTY cases,14 selected semantic checks, clean vulnerability scan. Linux/other-architecture runtime, live-provider acceptance, full comparison/performance/legal release review open |

Statuses describe this plan's gates, not just the presence of code. Parts 6–8 and 11
remain incomplete; complete rows establish only their stated tested scope.

## Files and tests

Subsystem source and tests: `internal/{core,provider,runtime,tools,
session,tui,mcpclient}`. Executables: `cmd/{harness,scripted-model,
acceptance}`. Docs/build/config: implementation README, Makefile, config example,
release script, VERIFICATION, BENCHMARKS, DEPENDENCIES, LICENSE and notices.
Per-part specifications are under `specs/parts/`; exact evidence IDs and binary
identities are in `research/FEATURE_MAP.md` and retained REA records.

Actually executed, passing:

- `go test -race ./...` and `go vet ./...`; logs in research/evidence.
- Three-second SSE, argument and journal fuzz runs; no failures. Reproduce with
  `make GO=research/tooling/go/bin/go fuzz` from the repository root.
- `go run ./cmd/acceptance`: seven built-executable scenarios, including real
  read→hash-guarded edit→shell assertion with expected file bytes and exit statuses.
- `python3 scripts/test_tui.py`: text, denial, edit approval, active shell
  cancellation and termination; terminal settings restored and first cancel
  leaves the application open.
- `python3 research/compare_semantics.py`: five selected semantic checks.
- `python3 research/compare_extended.py`: nine extended selected checks.
- govulncheck 1.9.0 source/symbol scan: no findings after pinning transitive
  x/text v0.41.0. Direct dependency pins remain those in the plan.
- `make ... release`: darwin/arm64, darwin/amd64, linux/arm64, linux/amd64,
  CGO_ENABLED=0, -trimpath, stripped symbols. No remote publication.

Research comparison settings, unknowns and raw changed/truncated REA results are
retained. Successful local fixtures do not establish universal compatibility or
identical live-model behavior. Reference traces/prompts stay in research and are
not shipped in product release material. All experiment servers were shut down.

## Known limits and next executable tasks

1. Run the deterministic suite on a real Linux runner with Go 1.27.3, including
   race/process/TTY/MCP tests. This host has no Docker/Podman/QEMU executable.
   Repeat architecture runtime checks for intended release platforms; cross-builds
   alone do not satisfy this gate.
2. Reference shell cancellation returned a REA ownership/cleanup limitation on
   macOS (`platform_binary_environment_withheld`). Its experiment child was
   separately cleaned up and documented. Resume that bounded investigation with
   an appropriate runtime provider/host; do not label the partial capture success.
3. Add bounded reference compaction and corrupted/interrupted-journal recovery
   observations using the already pinned artifact/isolated fixture; Go independent
   failure-path tests already pass. Consult `specs/parts/07*` and `08*` first.
4. Live-provider smoke/acceptance requires an explicitly configured endpoint,
   model/context profile, actual credentials and authorization for those calls.
   No credentials were supplied and no live API usage occurred.
5. Complete equivalent workload process-family memory/CPU/cancellation and live
   outcome benchmarking, plus release ownership/branding/reuse review. Local
   timings in BENCHMARKS.md are not a performance superiority claim.

To resume: read this file, docs/VERIFICATION.md and the compatibility
matrix, inspect actual test state, and reuse reference identity/evidence. The
first incomplete part is 6. Do not reacquire/research completed reference slices
or silently change the selected revision.

## Expanded reconstruction checkpoint — continued 5 November 2026

User requested all OpenCode features one by one, with reference to the whole
GitHub repository. The original alpha boundaries no longer bound the future
backlog. The full repository was cloned and checked out detached at the same
verified pin, with 6,627 tracked files across 32 top-level packages. The whole-tree
inventory is research/full-repository-inventory.json; the capability backlog is
research/FEATURE_BACKLOG.md. Source files remain in research/, separately from
original Go code. No upstream code/prompts were translated into the product.

### New working slices

| Feature | State | Contracts/evidence and current scope |
|---|---|---|
| Recursive glob | complete for declared subset | internal/tools/glob.go + tests, actual reference/Go captures, specs/features/glob.md; braces/**/classes, safe bounded traversal, deterministic relative output; exclusions/order/pattern errors deliberately differ |
| Regex grep include patterns | complete for declared subset | internal/tools grep admission/search integration + grep tests; reference-grep/go-grep captures; specs/features/grep.md; file/directory/include scope, max 100 lines, ordered results; ignore/file-path/regex differences explicit |
| Session task list | complete for declared subset | core.Todo + runtime/todo.go/tests, journal replay/context projection, TUI overview and local /todos reader; REA snapshots match, duplicate/custom-string behavior retained, replacement/resume clear verified; no todoread built-in equivalence claim |
| Portable sessions and fork | complete for declared original format | session/portable.go/tests + CLI sessions.go/tests; official reference list/export/import/fork captures; Go capture asserts fresh identities, 29 identical state records, no permissions copied, export 0600; upstream format intentionally unsupported |
| Manual compaction | complete for declared subset | runtime/compact.go/tests; CLI compact ID and TUI /compact; reference local HTTP summarize captured and Go operation captured; durable state, task/recent-group/active-request preservation; automatic reference thresholds/pruning still open |

Trusted current system framing is re-established after replay; imported historical
system summaries are treated as untrusted assistant data. Task updates serialize
with other mutations, sync state before acknowledgment, and require the named
todowrite permission. No permissions are restored by portable import/fork/resume.

Actual local validation: full race suite and vet pass; original executable CLI
acceptance still passes; TUI acceptance extended for task approval/display/local
reader/manual compaction; new semantic comparison checks task snapshots,
invalid-type preservation, resume clearing, headless refusal and manual summary.
Reference and Go glob captures are bounded/truncated; do not claim universal
matching from those captures. Specs/features document exact limits and evidence.

### Exact next implementation work

1. T12 skill discovery/loading: inspect full-repo skill discovery/config + tool
   registry, capture local project SKILL.md loading/permissions/missing skill,
   then implement original bounded Go skill catalog and lazy content loading.
2. T07 patch application with create/edit/delete/move/multiple-file failure
   contracts, preserving the existing permission and stale-file boundaries.
3. R04 build/plan/custom agent and model profiles, then structured questions and
   controlled subagents; keep the runtime/journal/cancellation shared.
4. Continue remaining IDs in FEATURE_BACKLOG across all packages. Full server/API,
   ACP, web/desktop, provider auth, plugins, LSP and authenticated integrations
   remain pending; user inputs are needed only for actual unavailable credentials,
   remote publication or platform access, not routine feature work.

The earlier Linux/live-provider/public-release gaps persist. This checkpoint
records implemented slices and preserves the larger unfinished objective.

Final expanded check: 68 top-level tests pass under race detector, vet is clean, seven CLI/six PTY acceptance cases pass, glob-parser fuzz 67,442 executions pass, and the fresh vulnerability scan has 0 findings. Platform artifacts and SHA256SUMS are rebuilt. New tests also verify session-command help without configuration/identity.


## Remaining-feature batch — skills, patches, profiles, commands and questions

Continued against the full original pinned repository; no revision change or
upstream implementation/prompt copying. Original behavior contracts and exact
reference/Go observations are under specs/features and research/evidence.

| Slice | State for declared subset | Implementation/evidence |
|---|---|---|
| T12 local skills | complete | internal/skills + runtime skill admission, local catalog/metadata/lazy body/resource paths, project+explicit roots, original bounded frontmatter; reference-skills* and go-skills* captures |
| T07 apply_patch | complete | internal/tools/patch.go + actual runtime integration; create/update/delete/move, full-patch previews and prevalidation/staging, exact unique context+mandatory source hashes, no-overwrite destinations, explicit partial effects; 10 paired scenarios pass |
| R04 profiles/build/plan | complete for explicit JSON profiles | core/profiles + CLI selectors, env/CLI precedence, custom instructions/model profiles, runtime strict plan policy and visible mode; actual reference agent captures + go-plan-forced-write |
| R05 text slash commands | complete | internal/commands + runtime/TUI; bounded nonrecursive substitution, no implicit shell/file access, local catalog; reference substitution captures + go-command-quoted/literal |
| T11 structured questions | complete for original CLI/TUI subset | typed shared-runtime request/reply, TUI choice/multiple/custom/dismiss/cancel, no grant transfer; reference HTTP/headless captures + Go needs_input/actual PTY checks |

Important boundaries: Go explicit skill approval, root order and parser subset
are intentional differences. Plan mode is stricter than the reference and cannot
be widened by grants. Commands remain text-only. Patches are individual durable
publications, not multi-file transactions; tool_observation retains partial
outcomes while the unresolved start blocks replay until inspection/acknowledgment.
Headless questions add needs_input/exit 4, retaining the request for inspection;
automatic saved-question continuation is not implemented. Existing incomplete
input relationships are repaired on the next run without inventing answers.

Configuration now rejects unbounded/special-file reads; skill traversal uses
anchored roots, refuses symlinks/FIFOs and verifies content identities at load.
No added runtime dependencies, live provider calls, user-home skill scanning,
remote publication or unrelated global configuration changes were performed.

Validation record: rest-tests-race.txt, rest-vet.txt, rest-cli-acceptance.json,
rest-tui-acceptance.json, rest-semantic-comparisons.json, patch-semantic-comparison.json
and rest-govulncheck.json under research/evidence. Specs state actual captures versus
source-only facts. Release artifacts are regenerated after final verification.

### Next executable slices

1. Controlled subagents (T10): inspect both full-repo task runtimes, capture
   scripted parent/child sessions, permissions and cancellation. Add bounded child
   execution over the shared provider/journal with separate authority; do not
   create an unbounded background fleet or infer grants from parent text.
2. Local webfetch (T13) fixture and implementation, then LSP lifecycle/diagnostics
   with an original controlled language server before external installed servers.
3. Continue the remaining repository-wide FEATURE_BACKLOG: fuller command/skill
   discovery and continuation, native providers/auth, server/API/ACP, plugins,
   formatters/worktrees, web/desktop, integrations and platform runtime gates.

Whole OpenCode parity remains incomplete. Earlier platform/live-provider/release
review gates remain open. These completed slices describe only their documented
contract coverage; unsupported behavior stays explicit in the compatibility matrix.


Final validation for this batch: 112 top-level tests pass with the race detector,
vet is clean, seven compiled-CLI acceptance scenarios and nine actual PTY
scenarios pass. Skill frontmatter fuzz 270,248 and patch grammar fuzz 180,851
executions pass; renewed govulncheck reports 0 findings. Four CGO-free platform
artifacts/checksums are rebuilt. Cross-builds remain distinct from Linux/AMD64
runtime verification. No experiment fixture servers are intentionally left running.

## Foreground subagents, webfetch and explicit LSP — continued 5 November 2026

Three further slices are working over the shared runtime. Full pinned repository
and artifact identities remain unchanged; source assistance, actual REA observations
and deliberate differences are separated in specs/features/{subagents,webfetch,lsp}.md.

| Slice | Declared implemented scope | Verification/limitations |
|---|---|---|
| T10 foreground child tasks | Separate durable child histories, same-ID continuation with parent/type/project ownership, independent child grants, general/explore/custom types, parent plan cap, no recursive/background fleets, bounded tasks/turns/deadlines, nested events and cancellation | Reference child/inheritance/interrupt captures plus go-subagents; same-ID history confirmed; stricter missing-ID/authority/explore behavior explicit; real child mutation/cancellation in PTY tests |
| T13 explicit web fetching | HTTP(S) GET text/Markdown/raw HTML, bounded parsing, same-origin redirects, no provider credentials/cookies/proxy, named approval, result digest/truncation | Eleven local reference/Go cases; cross-origin/binary/bounds intentionally stricter; no browser execution or remote/live services |
| T15/R17 local LSP subset | Explicit configured stdio servers, lazy dual approval, durable startup, definition/references/hover/symbols/implementation/diagnostics, UTF16 handling, sync, bounded protocol/cancellation/process cleanup | Six paired reference/Go fixture operations; independent malformed/out-of-order/timeout/disconnect/special-file/edit-request tests; no auto-install, native-server/advanced-operation parity not claimed |

New fields/packages: Config.LSP, runtime child/web/LSP lifecycle, internal/lspclient,
internal/webfetch and original internal/testutil protocol fixtures (not linked into
harness executable). Child operation grants use child:TOOL; ordinary parent grants
never transfer. LSP has independent lsp and lsp_start:NAME authority. Partial or
uncertain outcomes retain tool_observation and unresolved starts; no replay.
Runtime managers close concurrently before the terminal event. TUI cleanup waits
for the active run and now covers an executing child shell in real PTY acceptance.

New tokenizer dependency pinned to golang.org/x/net v0.62.0; MVS selects x/sync v0.24.0,
x/sys v0.49.0 and x/text v0.43.0. The full 39-module selected graph has exact copied license
texts and digests in licenses/inventory.json; original UI/MCP/schema pins unchanged.
A renewed source/symbol vulnerability scan reports no findings for this graph.

Actual checks: next-tests-race.txt, next-vet.txt, next-cli-acceptance.json,
next-tui-acceptance.json, next-govulncheck.json, lsp-fuzz.txt, webfetch-fuzz.txt and
next-semantic-comparisons.json under research/evidence. Independent semantic
projections are not canonical REA equivalence. Webfetch/LSP rendered captures
are truncated; webfetch also retains ambient-process unknowns despite owned-group
verification. No universal lifecycle/absence-of-effects claim is inferred.

### Next dependency-ready implementation work

1. Formatter discovery/admission and post-edit behavior using a controlled local
   formatter, then Git diff/snapshot/undo protections in disposable repositories.
2. Native provider adapter(s) and complete provider/model profile capability/error
   mapping with local scripted endpoints; live auth/account flows need actual
   user credentials only when their own acceptance gates are reached.
3. Local server/API/events and ACP over the established coordinator, then remaining
   full-repository backlog: plugins, remote MCP/auth, richer LSP/subagents/skills,
   workspace/worktrees, complete TUI, web/desktop, GitHub/other integrations and
   runtime platform coverage. Those are still unfinished rather than silently deferred.

No live model spending, remote publication, implicit language-server install or
user-home skill scanning occurred. The whole OpenCode reconstruction remains
incomplete. Earlier Linux/other-architecture runtime and release-review gates persist.

Final verification for the child/web/LSP batch: 133 top-level tests and their subcases pass with the race detector; go vet is clean. Seven CLI scenarios and eleven real PTY scenarios pass, including child mutation approval and running-child cancellation. All four CGO-free platform binaries were rebuilt and their SHA256 checksums verified. Cross-compilation does not establish runtime coverage on the other platforms.

## Explicit formatters and file checkpoint history — continued 5 November 2026

The next working batch adds post-edit formatter execution and durable per-file
checkpoints with diff/undo/redo. It uses the same pinned full repository and host
artifact identity. Detailed source/actual-runtime evidence and deliberate
compatibility differences are in specs/features/{formatters,snapshots}.md.

Implemented behavior:

- Explicit local formatter configuration and side-effect-free availability,
  optional installed gofmt detection, literal argv expansion, deterministic
  extension selection, separate format:NAME authority, bounded supervised
  processes/artifacts, final edited-file identities and cancellation checkpoints.
  No formatter is enabled or installed implicitly. Failed formatting preserves
  the edit and reports its own outcome; headless ungranted execution is skipped.
- Opt-in before/after checkpoints for built-in file mutations including final
  formatting, using private content-addressed blobs outside the project. Runtime
  snapshot list/diff/undo/redo and the changes CLI support conflict-checked file
  restoration. All hashes/modes/existence/blob integrity are checked before any
  publication, with individual file publications and durable applied progress.
  Uncertain checkpoints block further work until inspection/acknowledgement;
  nothing is replayed. Git metadata/index and conversation history are untouched.
- Fourteen PTY cases include real formatter approval, denial and executing-process
  cancellation. New package/runtime/CLI tests cover stale restore refusal,
  independent authority, final formatted identity, patch moves/creation, binary
  bytes, modes, durable reopen, concurrent reads, private storage, corrupt data,
  links/FIFO, partial cancellation and explicit uncertain recovery.

Reference fixtures observed successful and exit7 formatter mutations, disabling,
selection/env/cwd, Git snapshot track/patch/diff/exclusions and real session
revert/unrevert. Reference revert overwrote an external touched-file change in the
fixture; Go deliberately refuses that conflict. Original Go REA captures verify
matching selected formatter bytes, undo/redo of final formatted creation and
exit1 preserving external conflict contents. Selected projections in
changes-semantic-comparisons.json are not canonical REA equivalence. All Go
captures have truncated overall observations from incomplete process shutdown
sampling; the formatter capture also retains ambient-process unknowns despite
verified owned-group cleanup. Reference limits remain explicit in each dossier.

Validation inventory:153 top-level tests plus subcases under race, clean vet,
seven CLI and fourteen actual PTY cases. Local fuzz runs completed60,219 webfetch
argument cases and26,293 HTML cases without failure; HTML suppression now also
handles self-closing raw-text tokens. Renewed source/symbol govulncheck has0
findings; no dependency change in the exact39-module graph. Four CGO-free
Darwin/Linux AMD64/ARM64 binaries are refreshed; cross-builds are not native
runtime acceptance on all four platforms. Current logs use changes-* and
formatter-tui-acceptance.json under research/evidence.

Remaining scope is explicit: full built-in formatter catalog/project discovery,
whole-tree Git snapshots, message/part-range revert/unrevert/history cleanup, GC
and snapshot portability are unfinished. Next dependency-ready work is native
provider adapter capability/error mapping with controlled endpoints, followed by
local server/API/events and ACP; broader backlog also includes plugins, remote
MCP/auth, workspace/worktrees, richer LSP/subagents, complete TUI, web/desktop,
integrations and platform runtime gates. Whole OpenCode parity remains incomplete.
No live provider spending, remote publication or implicit tool installation occurred.

Final four artifact SHA256 checks pass. research/compare_changes_batch.py also
compares all three saved Go checkpoint after-state blobs to actual reference final
file states; all match. Snapshot scope/conflict differences remain deliberate.

## Native protocol, Harvest integrations and supplemental repositories —6 November2026

Continued implementation now adds five working capabilities over the same
coordinator and durable state:

1. Native Anthropic Messages text/client-tool encoding and bounded streaming,
   distinct x-api-key/version auth, usage/cache accounting, explicit protocol
   selection, typed/redacted HTTP failures and redirect refusal. Incomplete
   streams commit no calls and are never retried after the stream starts.
2. Harvest-inspired explicit compaction/task/explore→profile routing, including
   mixed Chat Completions parent/native child, custom-agent precedence, summary
   model input admission, durable resolved identity and unchanged permission caps.
3. Strict optional task.output_schema with preflight no-fetch/bounds and final
   JSON validation, preserving child IDs/history/effects on invalid output without
   automatic repair/re-execution. The existing JSON Schema dependency is reused.
4. Aider-inspired bounded Go declaration maps and SHA256-validated parse caching,
   built from the standard Go parser; no compiler/syntax-engine dependency.
5. Harvest/Crush-informed session-scoped shell/formatter output retrieval with
   bounded byte pages/tail/base64 over the existing retained1MiB prefix, so an
   agent can inspect needed output without rerunning the command.

The full Harvest checkout is pinned to27f0e9fb87c9d11afa27dc273ec5f85f09b1d111
(8,032 tracked files). Its recent native migration handoff explicitly marks work
uncompiled/untested; source/test presence is not shipped-runtime validation.
Additional full checkouts: Aider5dc9490bb35f9729ef2c95d00a19ccd30c26339c,
Goosed66323246cb21f159994415c1f589368b85b0bb1 and
Crushc7854d54a8fc2eef37f5c31ee4e829318b1ea13c. Relevant implementations/tests,
license hashes, maturity and dispositions are recorded in
research/comparisons/{manifest.json,ASSESSMENT.md} and harvest/ASSESSMENT.md.
Goose's separate compaction-model/structured-summary design informed the fit
assessment; full typed-section compaction is still a follow-on. No upstream
implementation or prompts were copied, no upstream builds/installers were run,
and source-only references are excluded from product releases.

Pinned OpenCode native Messages complete/drop runtime captures were retained in
research/native-provider-reference-*-evidence.json. Their headers/tool-result/text
facts are checked by compare_native_provider.py. The observed reference aborts a
dropped read and continues to final text/exit0; Go deliberately fails once without
executing/replaying incomplete tool input. These are selected behavior checks,
not canonical REA equivalence or full provider parity. Each capture preserves
truncation/process-sampling/ambient-identity limits and its own build identity.

Current validation:184 top-level tests plus subcases pass under race; vet clean.
Seven compiled-CLI acceptance cases and15 actual PTY cases pass, including typed
child approval/results. Seven additional Go native/Harvest REA scenarios cover
mixed protocol, valid/invalid results, compaction routing, native read/text,
partial stream,401 and refused redirect. Controlled Go efficiency CLI evidence
checks declaration maps/cache and output retrieval through the normal runtime.
Structured-output fuzz19,334 and native SSE fuzz67,105 executions pass; renewed
govulncheck has0 findings in the unchanged39-module dependency graph.

Measured original32-file fixture:297,654 source bytes→6,330 declaration JSON bytes
(97.87% smaller by bytes), with32 repeated parse-cache hits; a64-byte output tail
page retrieves the end marker from70,017 retained command-output bytes. Local
100-iteration16-file microbenchmark on Apple M5 Pro/darwin-arm64 reports warm
440,100ns/op/233,865B/op/566allocs versus cold898,184ns/op/523,265B/op/6,231allocs.
These are bounded original fixtures, not tokenizer/live-cost, end-to-end or
cross-agent performance comparisons. Both map paths still read/hash source.
BENCHMARKS.md and research/evidence/{efficiency-report.json,repomap-benchmark.txt}
retain measurement scope. Targeted source reads remain necessary for behavior.

Open work: server/API/events and ACP; richer native content/auth/provider adapters,
whole-tree/history snapshots, complete formatter discovery, workspace/background
jobs, general output quotas/artifact portability/GC, generated-file guards,
structured compaction sections and multilanguage ranked context. Full OpenCode
parity and all earlier live/provider/platform/public-release gates remain open.
No live provider spending, remote publication or implicit tool installation occurred.

Final release verification for this batch: all four rebuilt Darwin/Linux
AMD64/ARM64 SHA256 entries pass; the host arm64 artifact reports0.1.0-dev. Current
README/verification/benchmark/provenance/compatibility/config files accompany the
binaries. Final CLI/PTY/native/efficiency captures pass; native comparison was
rerun against those retained records. Overall Go captures remain truncated by
process sampling, with explicit ambient-process unknowns in the mixed-child case.
Owned fixture listeners were stopped. Cross-builds still do not establish runtime
acceptance on the other platforms or public-release readiness.

## Local HTTP API

The new `harness serve` command exposes project-scoped session creation and
listing, prompt streaming, correlated permission and question replies, and
cancellation. It reuses the in-process coordinator, provider selection and
durable journals. It streams existing runtime events directly as NDJSON with
a write deadline and one active run per server. Challenge IDs are single-use.
A disconnected client cancels its run. Cross-project journal access is
rejected before journal repair.

The design comparison used the pinned OpenCode HTTP session and event routes,
Harvest's RPC command and UI response handling, and Goose's ACP prompt and
cancel dispatch. OpenCode's source separates session prompt/abort from
project-scoped SSE delivery. Harvest correlates UI replies with pending RPC
requests. Goose dispatches prompt and cancellation through separate protocol
messages. The Go API applies these patterns using a request-scoped NDJSON
stream and the existing coordinator. The upstream wire contracts are separate
work. See the usage contract in README.md.

Source references for this slice are the pinned [OpenCode event handler](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/opencode/src/server/routes/instance/httpapi/handlers/event.ts),
[Harvest RPC mode](https://github.com/Sanidhya14321/harvest/blob/27f0e9fb87c9d11afa27dc273ec5f85f09b1d111/packages/coding-agent/src/modes/rpc/rpc-mode.ts),
and [Goose ACP server](https://github.com/aaif-goose/goose/blob/d66323246cb21f159994415c1f589368b85b0bb1/crates/goose/src/acp/server.rs).

The full race suite passed with 187 top-level tests and vet passed. The compiled
Darwin ARM64 server completed an approved shell run, confirmed the server
credential was absent from the shell environment, and shut down on SIGTERM.
All four rebuilt artifact checksums passed. Evidence is retained in
research/evidence/local-api-{race.txt,vet.txt,build.txt,cli.json}.

## ACP stdio adapter

`harness acp` adds ACP v1 initialization, durable session creation and resume,
session close, text prompts, tool and text updates, one-time permission replies,
and cancellation. It runs the existing coordinator in process and translates
its events directly to JSON-RPC. Each session accepts approved client-provided
stdio MCP definitions in addition to configured servers. One prompt is active
per process, and the session working directory must resolve to the configured
project. The adapter advertises resume and close. Media, structured questions
via ACP, client-hosted filesystem and terminals, and session history replay
remain separate scope.

Reference inspection covered the [ACP v1 prompt and cancellation contract](https://agentclientprotocol.com/protocol/v1/prompt-turn),
[OpenCode permission mapping](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/opencode/src/acp/permission.ts),
[Harvest event mapping](https://github.com/Sanidhya14321/harvest/blob/27f0e9fb87c9d11afa27dc273ec5f85f09b1d111/packages/coding-agent/src/modes/acp/acp-event-mapper.ts),
and [Goose ACP dispatch](https://github.com/aaif-goose/goose/blob/d66323246cb21f159994415c1f589368b85b0bb1/crates/goose/src/acp/server.rs).
The compiled binary fixture verifies the Go wire behavior with a local model.
The upstream adapters were inspected through source for this slice.

## Generated-file edit protection

Built-in edits, replacements, and patch updates, deletions, and moves now check
existing source bytes for explicit generated headers during preview and
execution. The detector inspects at most 8 KiB and stops at the first code.
A rejected source blocks the full patch before staging. User configuration can
opt out with `allow_generated_edits`, with ordinary permissions and hashes
still required. The guard covers explicit headers, with filename heuristics
left out to keep handwritten sources with generated-looking names editable.

Single-file execution now validates the live mutation once before staging
and skips regenerating the approval preview during execution. The publication
hash recheck still protects against changes after staging. The comparison drew
on [Harvest's generated-file policy](https://github.com/Sanidhya14321/harvest/blob/27f0e9fb87c9d11afa27dc273ec5f85f09b1d111/crates/pi-edit/src/path_policy.rs),
[Aider's pre-edit path filtering](https://github.com/Aider-AI/aider/blob/5dc9490bb35f9729ef2c95d00a19ccd30c26339c/aider/coders/base_coder.py),
and [Crush's write review path](https://github.com/charmbracelet/crush/blob/c7854d54a8fc2eef37f5c31ee4e829318b1ea13c/internal/agent/tools/write.go).

The same 100-iteration exact-edit benchmark allocated 742,109 bytes per
operation after the change, down from 1,719,649, and allocations dropped
from 545 to 302. The full 190-test race suite and vet passed, and all four
rebuilt binaries passed checksum verification. The benchmark measures
this fixture on Darwin ARM64.

## Verifiable compaction records

Compaction now saves the working view and a versioned receipt in the same
journal record. The receipt checks the view and evidence metadata on resume and
portable import. The trusted system message continues to be rebuilt from the
current configuration. Legacy compaction records remain readable. The shared
record schema also lets portable import accept compaction model metadata that
the runtime already writes.

The runtime carries up to six bounded observations from discarded tool
results, including successful file fingerprints and a recent shell exit code.
It uses final hashes after formatting when available. The observations are
historical task data. The runtime adds them to the prompt only when they fit,
and computes their exact JSON size without re-encoding the retained turns.
The receipt checks record consistency. Current filesystem verification and
repository-wide memory remain separate work.

Design references include [OpenCode's recent turn budget](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/opencode/src/session/compaction.ts),
[Harvest's file-operation evidence](https://github.com/Sanidhya14321/harvest/blob/27f0e9fb87c9d11afa27dc273ec5f85f09b1d111/packages/coding-agent/src/core/compaction/compaction.ts),
and [Goose's structured compaction](https://github.com/aaif-goose/goose/blob/d66323246cb21f159994415c1f589368b85b0bb1/crates/goose-context-management/src/structured.rs).

## Opt-in project memory ledger

The first ledger slice adds operator capture, listing, recall, and deletion
through `harness memory`. Notes live in a private JSON file keyed by canonical
project path. Writes use an exclusive project lock and atomic publication.
The read-only `memory_recall` tool is available when user configuration enables
memory, including in plan mode. It ranks lexical matches locally, checks up to
12 distinct candidate evidence files, and returns up to three fresh notes.
The tool gives notes no permission authority. Automatic extraction and recall
evaluation remain future work.

Source comparison covered [Harvest's project-scoped memory consolidation](https://github.com/Sanidhya14321/harvest/blob/27f0e9fb87c9d11afa27dc273ec5f85f09b1d111/packages/coding-agent/src/memories/storage.ts),
[Goose's directory-scoped memory tools](https://github.com/aaif-goose/goose/blob/d66323246cb21f159994415c1f589368b85b0bb1/crates/goose-mcp/src/memory/mod.rs),
and [Aider's repository-context caching](https://github.com/Aider-AI/aider/blob/5dc9490bb35f9729ef2c95d00a19ccd30c26339c/aider/repomap.py).
The ledger follows the staged design in the local memory research brief.
