# Changelog

GoHarness release candidates in chronological order. Dates are commit dates.
Each entry lists user-visible changes; evidence for each is in
[VERIFICATION.md](VERIFICATION.md). No stable release has been published yet.

## Unreleased

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
