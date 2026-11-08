# Changelog

GoHarness release candidates in chronological order. Dates are commit dates.
Each entry lists user-visible changes; evidence for each is in
[VERIFICATION.md](VERIFICATION.md). No stable release has been published yet.

## Unreleased

## 1.0.0-rc.93 — 2026-11-08

- Opt-in `"auto_title": true` asks the model for a short session title after
  the first completed run, as OpenCode does. The request sends only the first
  prompt (up to 2,000 characters), with no tools or files. It uses the new
  `title` model role when configured (for example a small, cheap profile),
  otherwise the main model. It never replaces a title you set, and failures
  keep the default title. Off by default because it is an extra request.

## 1.0.0-rc.92 — 2026-11-08

- `goh acp`, `goh serve` and `goh models` follow the same provider rule as
  chat and run: with no provider configured, a saved ChatGPT sign-in is
  used. `goh models` then lists the ChatGPT models (as `goh auth models`
  does). An explicit `--config` or `--profile` always wins.

## 1.0.0-rc.91 — 2026-11-08

- Fixed: `goh run` ignored a saved ChatGPT sign-in when no provider was
  configured, so `goh run fix the tests` failed where `goh chat` worked. Both
  now share one rule: the sign-in is used only when no provider, endpoint or
  model is configured at all, so explicit providers are never replaced.
- `goh doctor` reports whether a ChatGPT sign-in is saved and checks the
  provider that chat and run would actually use.
- Note: binaries installed for RC73–RC90 reported version `0.1.0-dev`
  because a later build step rebuilt them without the version stamp. The code
  was as described; from RC91 the installed binary carries its version.

## 1.0.0-rc.90 — 2026-11-08

- Typing `@` in the TUI now offers subagents (`explore`, `general` and
  custom ones) whose names match, ahead of project files, so agent mentions
  can be completed like file mentions.

## 1.0.0-rc.89 — 2026-11-08

- `@general`, `@explore` or `@<custom subagent>` in a prompt asks the model
  to delegate that work through the task tool, as OpenCode's agent mentions
  do. The mention adds a short bracketed note to the saved prompt and grants
  nothing: the task tool keeps its normal approval, and child agents never
  receive the note. File mentions such as `@shot.png` are unaffected.

## 1.0.0-rc.88 — 2026-11-08

- `goh sessions list --format table` prints sessions newest first with ID,
  last change, title and project (as OpenCode's `session list` does). JSON
  stays the default for scripts.

## 1.0.0-rc.87 — 2026-11-08

- `goh run` and `goh chat` accept `--reasoning-effort
  minimal|low|medium|high|off`, overriding the configured level for that
  invocation (the CLI counterpart of `/effort` and OpenCode's `--variant`).
  Invalid levels are refused before any request.

## 1.0.0-rc.86 — 2026-11-08

- `/effort minimal|low|medium|high|off` changes reasoning effort for the
  current session from the TUI, as OpenCode's model variants do; `/effort`
  alone shows the current level. The saved configuration is unchanged, a
  change waits for the current run to finish, and Anthropic profiles keep
  using `thinking_budget`.

## 1.0.0-rc.85 — 2026-11-08

- Top-level `goh export [ID]` and `goh import FILE`, as in OpenCode. Without
  an ID, export uses this directory's most recent session. Flags are the
  same as `goh sessions export` (`--format markdown`, `--output`, ...).

## 1.0.0-rc.84 — 2026-11-08

- `goh mcp list` shows configured MCP servers (local command or remote URL)
  and the environment variables they use, flagging header variables that are
  not set (like `opencode mcp list`). It starts nothing and never prints
  secret values, URL credentials or query strings.

## 1.0.0-rc.83 — 2026-11-08

- `goh agents` lists the built-in, configured and project agents with
  their mode, description and model profile, plus any project agent files
  that were skipped and why (like `opencode agent list`).
- Test hardening: the Windows cancellation-under-load test now allows slow
  shell start-up and reports early exits (CI had timed out once).

## 1.0.0-rc.82 — 2026-11-08

- Built-in `/review` command, as in OpenCode: a read-only review of
  uncommitted changes by default, or of a commit, branch or path given as
  the argument (`/review main`, `goh run --command review HEAD~1`). It only
  expands to a prompt, so the usual tool approvals apply, and a command file
  named `review` replaces it.

## 1.0.0-rc.81 — 2026-11-08

- Editors connected over ACP receive the model's reasoning as
  `agent_thought_chunk` updates, so Zed and other ACP clients can show it in
  their thinking view. `goh run --format json` streams it as
  `reasoning_updated` events; text output is unchanged.

## 1.0.0-rc.80 — 2026-11-08

- The TUI shows the model's reasoning as dimmed `┊` lines before the answer
  or tool call it led to, and `/thinking` hides or shows it (as OpenCode's
  thinking toggle does). Sources: Anthropic extended thinking, and OpenAI
  Responses reasoning summaries, which are requested (`summary: "auto"`)
  only when `reasoning_effort` is set. Reasoning is display-only: it is
  never sent back as text.
- Not yet verified live: whether the ChatGPT and API endpoints accept the
  summary request. If one refuses, unset `reasoning_effort`.

## 1.0.0-rc.79 — 2026-11-08

- `goh --help` lists every command with a one-line description; `models`
  and `stats` were missing from the old one-line usage. A test keeps the list
  in step with the commands GoHarness actually accepts.
- Test fix: `scripts/test_setup.py` expected the `/models` dialog title from
  before RC59 and now passes again.

## 1.0.0-rc.78 — 2026-11-08

- `goh run --command NAME [words]` runs a custom or built-in command
  headlessly, as OpenCode's `run --command` does; the message words become
  its arguments (for example `goh run --command review src/api.go`).

## 1.0.0-rc.77 — 2026-11-08

- `goh run --title TEXT` names the session, as OpenCode's `run --title`
  does; the title appears in `/sessions` and session lists.

## 1.0.0-rc.76 — 2026-11-08

- `goh run --file PATH` (repeatable) attaches project files to the message,
  as OpenCode's `run --file` does. Text files are included inline (256 KiB
  each, 1 MiB in total); images become `@path` attachments. Paths outside the
  project, binary files and symlink escapes are refused.

## 1.0.0-rc.75 — 2026-11-08

- Markdown transcripts (`/export`) mask well-known credential formats (OpenAI,
  Anthropic, GitHub, Slack, AWS and Google keys, Bearer tokens, PEM private
  keys), keeping the prefix so you can see what was removed. This is a
  safety net only: secrets without a recognisable shape pass through. The
  portable JSON export stays exact so it can be imported.

## 1.0.0-rc.74 — 2026-11-08

- `/exit` and `/quit` leave GoHarness, as in OpenCode. During a run they
  cancel first, like Ctrl+D, rather than leaving mid-effect.

## 1.0.0-rc.73 — 2026-11-08

- `/rename title` renames the open session from the chat (as in OpenCode);
  `/rename` with no title restores the default from the first prompt. The
  title is saved beside the journal, which stays unchanged.

## 1.0.0-rc.72 — 2026-11-08

- Fixed (found in live testing): resuming a session with a small context
  window (32k) failed with "context compaction did not reduce context
  enough". The compaction safety margin now scales with the window.
- `goh auth login` always prints the sign-in link, so you can open it in a
  private window or another browser to choose the account.
- Live checks on a ChatGPT account passed: streaming, approved tool use,
  resume, TUI connect, image input, `reasoning_effort`, end-of-request task
  state and the plan handoff (`scripts/test_openai_live_extra.py`).

## 1.0.0-rc.71 — 2026-11-08

- `websearch` tool for a SearXNG-compatible search service you configure
  with `websearch_url` (no third-party account needed). Each search needs
  approval; results are titles, links and snippets for `webfetch` to read.

## 1.0.0-rc.70 — 2026-11-08

- Anthropic extended thinking: set `thinking_budget` (at least 1024, below
  `output_budget`). Thinking blocks are kept exactly as received and sent
  back with their turn, as the API requires during tool use; they never
  reach other providers. `/status` shows the budget. Not yet checked
  against the live API.

## 1.0.0-rc.69 — 2026-11-08

- Fuzzy matching in the command palette and every picker (`/models`,
  `/sessions`, `/copy`, `/jobs`, `/changes`, themes): type letters in order,
  such as `sts` for `/status`. Exact substrings still rank first.

## 1.0.0-rc.68 — 2026-11-08

- The `lsp` tool adds `goToTypeDefinition`, `prepareCallHierarchy`,
  `incomingCalls` and `outgoingCalls`, matching OpenCode's language-server
  operations. Each is used only when the server advertises it.

## 1.0.0-rc.67 — 2026-11-08

- Plan handoff: in plan mode the agent can propose switching to build with
  its finished plan (`plan_exit`, as in OpenCode). You see the plan in the
  approval; approving continues the same session in Build with "Carry out
  the approved plan." ready to send. Declining keeps you in plan mode.

## 1.0.0-rc.66 — 2026-11-08

- Background jobs left running after GoHarness was force-killed are found
  on the next start: `/jobs` and `/status` list them as orphaned, and
  Ctrl+K stops one after checking the process still runs the recorded
  command (so a reused process ID is never stopped). On Windows they are
  listed but must be stopped manually.

## 1.0.0-rc.65 — 2026-11-08

- `compaction_threshold` (50–95, default 75) sets how full the context gets
  before automatic compaction; `/status` shows it. Compaction still runs
  whenever the next request would not otherwise fit.

## 1.0.0-rc.64 — 2026-11-08

- `/details` shows or hides full tool output; hidden, each result is one
  line with a count of the remaining lines, as with OpenCode's tool-details
  toggle. `/find` searches what is on screen.

## 1.0.0-rc.63 — 2026-11-08

- `file_rules`: allow, deny or always ask edits by path pattern (for
  example never edit `*.env*`, edit `*_test.go` without asking). Applies to
  every edit tool and subagents; a patch is denied if any file it touches
  is denied.

## 1.0.0-rc.62 — 2026-11-08

- Context meter: the sidebar shows how full the context window was at the
  latest reply (for example `Context: 50% of 200k`), and `/status` shows the
  token count, as OpenCode does. It appears once the provider reports usage.

## 1.0.0-rc.61 — 2026-11-08

- ACP: editors can pass remote (HTTP) MCP servers as well as local ones;
  the adapter advertises `mcpCapabilities.http`. Header values from the
  editor are kept in memory only, never shown in approvals, and the same
  HTTPS/no-redirect rules apply.

## 1.0.0-rc.60 — 2026-11-08

- `goh stats` shows sessions, prompts and token usage per model from your
  local sessions, like `opencode stats` (`--days N`, `--project DIR`,
  `--json`). Token usage is saved per reply from this release on, when the
  provider reports it.

## 1.0.0-rc.59 — 2026-11-08

- `/models` also lists the models your provider offers, below your
  profiles, so you can switch model for the current chat without editing
  config. If the list can't be fetched, the picker says why.

## 1.0.0-rc.58 — 2026-11-08

- `goh models` lists the models your configured provider offers (one ID per
  line), like `opencode models`; `--profile` asks another profile's
  provider. The key goes only to the configured host and redirects are
  refused. Sign in with ChatGPT keeps using the account catalog in `/models`.
- Windows CI is green again after a test cleanup fix (`5fc2e3c`).

## 1.0.0-rc.57 — 2026-11-08

- Remote MCP servers: set `url` instead of `command` to use an MCP server
  over Streamable HTTP, with tokens supplied through `headers_env`
  environment variables. HTTPS only (HTTP for localhost), no redirects,
  header values never shown in approvals.

## 1.0.0-rc.56 — 2026-11-08

- File checkpoints are on by default (best effort), as OpenCode snapshots
  are, so `/undo`, timeline revert and `/changes` restore and show edits
  without setup. If a checkpoint cannot be saved, the edit still happens and
  is reported as not restorable. `"snapshots": true` keeps the previous
  strict behavior; `"snapshots": false` turns them off. `/status` shows the
  mode.
- Fixed: an edit whose checkpoint was skipped is no longer reported as
  interrupted.

## 1.0.0-rc.55 — 2026-11-08

- `/changes` lists the files the agent changed in this session and shows
  each checkpoint's diff. It needs file checkpoints (`"snapshots": true` or
  `--snapshots`) and explains how to enable them otherwise.

## 1.0.0-rc.54 — 2026-11-08

- `--continue` (for `goh run` and `goh chat`) resumes this project's most
  recent session, and `goh run --session ID` continues a specific one, as
  with `opencode run --continue`. With no previous session, `--continue`
  says so instead of starting a new one.

## 1.0.0-rc.53 — 2026-11-08

- `goh run fix the failing tests` takes the message as plain words, like
  `opencode run`, with flags before or after. `-` inserts standard input:
  `git diff | goh run - review this change`. Standard input is read only
  when `-` is given.

## 1.0.0-rc.52 — 2026-11-08

- Project agent files also load for `goh serve` (from `--project`) and the
  ACP adapter (from its working directory), and `/status` lists agent files
  that were skipped and why.

## 1.0.0-rc.51 — 2026-11-08

- Project agent files: `NAME.md` in `.opencode/agent`, `.opencode/agents`,
  `.claude/agents` or `.harness/agents` defines an agent (frontmatter
  `description`, `mode: plan` for read-only; the body is its instructions).
  They appear in the agent picker, Tab cycling and as subagent types.
  Configured agents win, and a file cannot pick the model profile.

## 1.0.0-rc.50 — 2026-11-08

- Image attachments: mention a PNG, JPEG, GIF or WebP file with `@` (for
  example `what's wrong in @screenshot.png?`) and it is sent to the model
  as an image, on all three provider protocols. Up to 4 images and 8 MiB per
  prompt, 5 MiB each, from inside the project only; contents must match the
  extension. Not yet checked against a live vision model.

## 1.0.0-rc.49 — 2026-11-08

- `reasoning_effort` (`minimal`, `low`, `medium`, `high`) for OpenAI
  reasoning models over Chat Completions, Responses and Sign in with
  ChatGPT; shown on `/status`. Refused for Anthropic until extended thinking
  is supported. Not yet checked against a live provider.

## 1.0.0-rc.48 — 2026-11-08

- The sidebar shows how many background jobs are running, and `/status`
  lists every job with its state and command.

## 1.0.0-rc.47 — 2026-11-08

- `/init` asks the model to study the project and write or improve
  `AGENTS.md`, as in OpenCode. Text after `/init` adds focus. A command file
  or configured command named `init` replaces it.

## 1.0.0-rc.46 — 2026-11-08

- `shell_rules`: allow, deny or always-ask shell commands by pattern (also
  for background jobs and subagents). Allow covers only simple commands; a
  deny matches any part of a compound command. Documented as a guardrail,
  not a sandbox.

## 1.0.0-rc.45 — 2026-11-08

- Loop guard: when the model makes the same tool call with the same
  arguments a third time in a row, GoHarness asks before running it, even
  for approved tools (OpenCode's doom-loop check). Declining tells the model
  to try something else; "allow for session" stops asking. Headless runs
  stop for review instead. Polling `job_output` is exempt.

## 1.0.0-rc.44 — 2026-11-08

- Project instructions fall back to `CLAUDE.md` when there is no
  `AGENTS.md`, as in OpenCode. A personal `AGENTS.md` beside your config
  file applies to every project. `/status` lists the instruction files in
  use. Neither can change tools, permissions or mode.
- Background jobs (RC42) passed native CI on Windows.

## 1.0.0-rc.43 — 2026-11-08

- The config file accepts comments (`//`, `/* */`) and trailing commas, as
  OpenCode's does. Unknown settings are still rejected. Provider setup
  refuses to rewrite a commented config rather than erase the comments.

## 1.0.0-rc.42 — 2026-11-08

- Background jobs: the agent can start long-running commands (dev servers,
  watchers) with `job_start` (approval required), read their output with
  `job_output` and stop them with `job_stop`. `/jobs` lists them, shows
  output and stops one with Ctrl+K. At most 8 run at once; each stop ends
  the whole process tree. Jobs end when GoHarness exits or the chat
  switches session, model or agent, and are never restarted on resume.
  Subagents and plan mode cannot start jobs.

## 1.0.0-rc.41 — 2026-11-08

- `!command` in the composer runs a shell command directly, as in OpenCode,
  without a model request. Its output enters the conversation as tool
  output, so the next prompt can refer to it. Typing it is the approval
  (journaled as `user_typed`); it never grants shell to the model, and plan
  mode refuses it.

## 1.0.0-rc.40 — 2026-11-08

- `/find` highlights matching lines (with a `▸` marker that also shows
  under `NO_COLOR`).
- Project command files: `NAME.md` in `.opencode/command`,
  `.opencode/commands`, `.claude/commands` or `.harness/commands` defines
  `/NAME`, with an optional `description` in frontmatter. Commands in your
  config take precedence; `agent` and `model` frontmatter are ignored.
  Skipped files are listed on `/status`.

## 1.0.0-rc.39 — 2026-11-08

- `/find text` searches the whole conversation, including turns scrolled
  far out of view; ↑/↓ or Enter move between matches, Esc returns.
- CLI: `goh changes revert-turn SESSION --turn N` reverts a turn and all
  later turns.

## 1.0.0-rc.38 — 2026-11-08

- Revert to an earlier prompt, as in OpenCode: in the timeline (Ctrl+T),
  preview a prompt and press Ctrl+U. That turn and every later one are
  undone, checkpointed files restored, and the prompt returns to the
  composer for editing. `/redo` brings everything back until you send.

## 1.0.0-rc.37 — 2026-11-08

- The session task list is sent at the end of each request instead of
  inside the system message, so todo updates no longer invalidate provider
  prompt caches. Not yet measured against a live provider.

## 1.0.0-rc.36 — 2026-11-08

- Session journals record when each entry was written (journal version 2).
  Older sessions still open and keep working; sessions written by this
  release cannot be opened by earlier release candidates, which report an
  unsupported journal version rather than damage.
- `/timestamps` shows or hides when each prompt was sent, including in
  resumed sessions. Fork and salvage keep the original times.

## 1.0.0-rc.35 — 2026-11-08

- `/copy` picks any recent user or assistant message (newest first,
  filterable) and copies it as written, Markdown included.

## 1.0.0-rc.34 — 2026-11-08

- `/status` explains why the configuration needs attention, shows the
  theme and explicit failover profiles (or that failover is off), and marks
  MCP/LSP servers that failed to start and were stopped in this session.
- Windows: deleting a session from the browser or CLI no longer fails
  (`29bf905`).

## 1.0.0-rc.33 — 2026-11-07

- Type while the agent works: the composer stays editable during a run and
  Enter queues prompts, sent in order after successful runs. Failed or
  cancelled runs return queued prompts to the composer instead.

## 1.0.0-rc.32 — 2026-11-07

- Rename (Ctrl+R) and delete (Ctrl+D twice) in the session browser, as in
  OpenCode; deleted sessions go to a trash folder and
  `goh sessions restore ID` brings them back.
- CLI: `goh sessions rename`, `delete` and `restore`.

## 1.0.0-rc.31 — 2026-11-07

- On-screen shortcut hints (footer and sidebar) show the configured keys
  and hide hints for unbound actions.

## 1.0.0-rc.30 — 2026-11-07

- Credential errors name the file and the recovery command
  (`goh auth login` or `/connect`); signing in again replaces a corrupt or
  wrongly permissioned credential file.
- Load test for cancelling parent and child tasks.

## 1.0.0-rc.29 — 2026-11-07

- One damaged session journal no longer hides every session from
  `/sessions` and `goh sessions list`.
- `goh sessions salvage ID` recovers the readable conversation of a damaged
  session into a new one; opening a damaged session explains how.

## 1.0.0-rc.28 — 2026-11-07

- Color themes (`dark`, `light`, `high-contrast`) with a `/theme` picker,
  `<leader>t`, and a validated `theme` setting.

## 1.0.0-rc.27 — 2026-11-07

- Transcript navigation: first/last, previous/next user turn, half-page
  scrolling, copy the last reply (OSC 52) and leader shortcuts for undo/redo,
  all remappable through `keybinds` (`91712ba`).
- `CHANGELOG.md` is included in release packages.
- Windows: process trees are force-stopped on timeout or cancellation, so
  shell timeouts no longer report `cleanup_incomplete` (`fa7c660`).
- Installers run the downloaded binary before replacing an existing
  installation, and keep the old one if it fails (`14f7f54`).

## 1.0.0-rc.26 — 2026-11-07

- `keybinds` configuration remaps the leader, command palette, sidebar,
  timeline, status and editor shortcuts; bad bindings are rejected at load.
- Windows CI is required (`af4d792`).

## 1.0.0-rc.25 — 2026-11-07

- Successful edits include error diagnostics from an already-running
  language server as `lsp_errors` (`3552ee1`).

## 1.0.0-rc.24 — 2026-11-07

- `@` file search honors `.gitignore` and refreshes after runs.
- One-command installers `install.sh` and `install.ps1` with checksum
  verification (`346e181`, `aa72fe6`).
- Windows path, lock and LSP shutdown fixes (`f5ebe0b`).

## 1.0.0-rc.23 — 2026-11-07

- Ctrl+X E and `/editor` edit the draft in `$VISUAL`/`$EDITOR` (`c0e707b`).

## 1.0.0-rc.22 — 2026-11-07

- Windows builds, six-target release with verified `SHA256SUMS`, and CI on
  Ubuntu, macOS and Windows (`962a084`).

## 1.0.0-rc.21 — 2026-11-07

- Automatic compaction masks old tool output before calling a summary
  model (`06d4daa`).

## 1.0.0-rc.20 — 2026-11-07

- `/undo` and `/redo` for the last turn, with checkpointed file restore
  (`1acba9e`).

## 1.0.0-rc.19 — 2026-11-07

- A failing MCP or language server is stopped and skipped; chat continues
  (`33efb72`).

## 1.0.0-rc.18 — 2026-11-07

- `fallback_profiles`: explicit provider failover for failures before any
  response (`fe00d08`).

## 1.0.0-rc.17 — 2026-11-07

- Sign in with ChatGPT, native OpenAI Responses streaming, `/connect` and the
  account model catalog (`c180294`).

## 1.0.0-rc.16 — 2026-11-07

- Provider recovery: drafts survive provider changes, Esc cancels a run,
  compatible-gateway fallbacks (`a29c816`).

## 1.0.0-rc.9 to rc.15 — 2026-11-06 to 2026-11-07

- rc.9 model and agent selectors (`2f0749a`); rc.10 session browser
  (`94998fe`); rc.11 `/status` (`274205d`); rc.12 prompt timeline
  (`8575c79`); rc.13 `@` file completion (`4781697`); rc.14 `/fork`
  (`3ed4869`); rc.15 Markdown export (`b3e60a7`).

## 1.0.0-rc.8 — 2026-11-06

- Baseline import of the existing implementation (`938df64`).
