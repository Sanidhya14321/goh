# GoHarness User Guide

The complete reference for `goh`. For installation and a quick start, see the
[README](../README.md).

- [Providers](#providers)
- [Terminal UI](#terminal-ui)
- [Headless CLI](#headless-cli)
- [Configuration](#configuration)
- [Permissions and rules](#permissions-and-rules)
- [Access levels](#access-levels)
- [Shell sandbox](#shell-sandbox)
- [Hooks](#hooks)
- [Verification and budgets](#verification-and-budgets)
- [Built-in tools](#built-in-tools)
- [Sessions and recovery](#sessions-and-recovery)
- [Context and compaction](#context-and-compaction)
- [Undo, checkpoints and snapshots](#undo-checkpoints-and-snapshots)
- [Agents, profiles and model roles](#agents-profiles-and-model-roles)
- [Skills and custom commands](#skills-and-custom-commands)
- [Subagents](#subagents)
- [MCP](#mcp)
- [LSP](#lsp)
- [Formatters](#formatters)
- [Web fetch and search](#web-fetch-and-search)
- [Project memory](#project-memory)
- [HTTP API](#http-api)
- [Attaching the UI to a server](#attaching-the-ui-to-a-server)
- [ACP editor adapter](#acp-editor-adapter)

---

## Providers

### Connecting

The first time `goh` opens with no provider, it shows this list straight away. Esc skips it;
pressing Enter in chat reopens it, keeping any message you typed as a draft. Later, `/connect` adds a provider:

- **Continue with ChatGPT** — signs in with a ChatGPT Plus, Pro or Team plan in the browser.
- **Anthropic, OpenAI, OpenRouter, Google Gemini, Groq** — opens the provider's API key page.
  Paste the key and press Ctrl+S. The key is checked with the provider before anything is saved,
  then the models it can use are listed. Context windows are taken from the provider when reported.
- **Ollama (local)** — lists models from a local Ollama server; no key needed.
- **Custom provider** — name, URL, protocol (`chat_completions`, `openai_responses` or
  `anthropic_messages`), key, model, context window and an optional key environment variable.
  Ctrl+S checks the URL and key against the model list. Leave the model blank to pick from that
  list. A URL ending in `/chat/completions`, `/responses` or `/messages` is used as-is.

Over SSH, or with `GOH_NO_BROWSER=1`, links are printed instead of opened.

Claude Pro and Max subscriptions cannot be connected: Anthropic does not permit subscription
sign-in in third-party tools. Use a Console API key.

### Managing providers

Each provider is saved under a name with its own key. The one added or picked last is used for new
chats. `/providers` (also `p` on the status screen, `goh providers` or `goh setup`) manages them:

```
  › ● anthropic  ·  Anthropic (Claude API)
        claude-… · api.anthropic.com · saved key
      chatgpt  ·  ChatGPT
        gpt-… · api.openai.com · signed in
    + Add a provider

    Enter use  ·  m model  ·  k key  ·  t test
    a add  ·  e edit  ·  d remove  ·  Esc close
```

| Key | Action |
| --- | --- |
| Enter | Use for new chats |
| `m` | Pick another model |
| `k` | Replace the key (checked first; for ChatGPT, reauthorize or switch account) |
| `t` | Test the connection |
| `e` | Edit a custom provider |
| `d` (twice) | Remove the provider and its saved key |

CLI equivalents: `goh providers list`, `goh providers use NAME`, `goh providers remove NAME`.
`/models` lists the current provider's chat models (speech, embedding and safety-classifier
models are hidden), then the other saved providers to switch to. The choice is saved on the
provider, so new chats keep it. A chat started with `--endpoint`, `--base-url` or `--config`
is not a saved provider: there `/models` changes that chat only. `goh models` prints the same
chat-model list.

### ChatGPT

```sh
goh auth login        # browser sign-in
goh auth status       # account, default model and when the access token expires
goh auth refresh      # renew the sign-in now (shows at once if it was revoked)
goh auth models
goh auth switch       # use a different account or workspace
goh auth logout       # revoke and clear local credentials
goh chat --protocol openai_chatgpt
```

GoHarness keeps one ChatGPT authorization per host. OpenAI scopes each sign-in's registration to
the workspace it was made in, so reauthorizing while the browser is signed in to another workspace
fails with `3p_login_workspace_scope_denied`; use **Use a different ChatGPT account** in
`/connect` (or `goh auth switch`) instead. The old sign-in is replaced only after the new one
succeeds. If OpenAI shows an error page, press Ctrl+C in the terminal.

`goh auth export FILE [45m]` writes the sign-in without its refresh token, for a disposable machine
such as a benchmark container: put it at `~/.config/independent-harness/credentials/chatgpt-account.json`
(mode `0600`) there. goh can use its access token, which lasts about an hour, but can never renew it, so the
copy cannot rotate or revoke the sign-in on this machine; replace the file to extend it.

The `openai_chatgpt` protocol is bound to OpenAI's official Responses endpoint. For a Responses API
key, use `openai_responses` instead.

### Credentials

- `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` are sent only to their official HTTPS hosts and take
  precedence there. For a gateway, save a key in `/providers` or name its own variable with
  `api_key_env`.
- Saved keys are scoped to the exact request URL and protocol and stored in `0600` files.
  HTTPS and loopback URLs support saved keys.
- `goh auth store keychain` seals saved keys and the ChatGPT authorization with AES-256-GCM using
  a key held only by the macOS Keychain, Linux Secret Service (`secret-tool`) or Windows DPAPI.
  `goh auth store file` reverts; `goh auth store` shows the current choice.
- Keep your workspace separate from the configuration directory so the agent's project scope
  never includes credentials.
- Credential values are never written to logs or journals. Prompts and tool output may still
  contain sensitive project data.

### Protocol details

- **Chat Completions** sends the key as a bearer token. It accepts a non-streaming JSON reply and
  retries once without `stream_options` if the endpoint rejects that field.
- **Anthropic Messages** sends `x-api-key`; `max_tokens` comes from `output_budget`. Prompt-cache
  breakpoints are set automatically; set `"disable_prompt_cache": true` if a gateway rejects
  `cache_control`.
- Redirects are refused. HTTP failures are classified and redacted. Retries happen only before a
  stream starts; partial streams never replay tool calls.

### Reasoning effort and thinking

`"reasoning_effort"` (or `/effort` for one session):

- OpenAI protocols: `none`, `minimal`, `low`, `medium`, `high`, `xhigh`
- Anthropic: `low`, `medium`, `high`, `xhigh`, `max` (sent as `output_config.effort`)

For current Claude models set `"anthropic_thinking": "adaptive"`; older models use
`"thinking_budget"`. Thinking blocks are passed back unchanged during tool use.

### Failover

`fallback_profiles` lists profiles to try, in order, when the active route fails **before
returning any response** (connection failure, 408, 429 or 5xx after three attempts):

```json
{
  "profiles": {
    "backup": {"endpoint": "https://gateway.example/v1/chat/completions",
               "model": "YOUR_BACKUP_MODEL", "api_key_env": "BACKUP_API_KEY"}
  },
  "fallback_profiles": ["backup"]
}
```

Authentication errors, bad requests, cancellation and any failure after streaming began are never
failed over. A fallback whose context window cannot fit the conversation is skipped. The journal
records `provider_failover` and `model_route` events.

---

## Terminal UI

### Choosing a project

`goh` starts in the folder you ran it from when that folder is trusted. Otherwise it opens the
project browser there: the folder itself ("Open …", with what it looks like: git, Go, Node…),
its parent, recent projects and its subfolders.

| Key | Action |
| --- | --- |
| Enter | Open the selected folder (the first time, goh asks you to trust it) |
| → or Tab | Look inside the selected folder |
| ← or Backspace | Go to the parent folder |
| Typing | Filter the list; a path such as `~/code/ap` completes against real folders |
| Esc | Clear the filter, or quit |

Trusting a folder covers its subfolders and means goh opens there directly next time. It grants no
tool permissions: edits and commands still ask as configured. Your home folder itself cannot be a
project, because goh keeps its saved keys there. `--project`, `--session` and `--continue` skip
the browser. `/workspace` (and `w` in `/settings`) opens it from chat. `goh workspaces` lists
trusted folders and `goh workspaces forget PATH` removes one.

### First run

On first launch GoHarness opens a provider picker. Choosing "not now" opens chat; typing a message
and pressing Enter returns to the picker and keeps the message as your draft. Esc on a key form or
model list goes back to the picker, and a failed step is explained there.

### Composer

- **Enter** sends; **Alt+Enter** or **Ctrl+J** adds a line. Multi-line paste is supported.
- During a run, Enter queues up to five prompts, sent in order after the run completes. If the run
  fails, is cancelled or stops for input, queued prompts return to the composer.
- Left/Right, Home/End, Ctrl+A/Ctrl+E, Ctrl+W (delete word), Ctrl+U (clear).
- Up/Down move through lines, then through prompt history.
- Tab completes slash commands, including configured custom commands.
- **Shift+Tab** cycles the access level (Ask → Auto-edit → Auto → Full access), also during a run;
  the footer shows the current one.
- While a run is active, the status line says what it is doing: "Waiting for gpt-6.2-sol · 4s",
  "Thinking", "Writing web/index.html · 1.8k tokens (42/s)", "Running npm test · 12s", with the
  run's total time and its step and tool counts.
- **Ctrl+X E** or `/editor` edits the draft in `$VISUAL`/`$EDITOR`. No shell is invoked; empty
  saves or editor errors keep the original draft.

### File references (`@`)

Type `@` to search project files by name or path, including fuzzy abbreviations. ↑/↓ select, Tab
or Enter insert, Esc dismisses. Paths with spaces are quoted. The index honours nested
`.gitignore` files, skips dot-entries, symlinks, `node_modules` and `vendor`, and scans up to
12,000 entries and 24 levels.

### Sessions

`/sessions` lists conversations newest first. Search matches titles, IDs and workspace paths.
Ctrl+W filters to the current workspace; Tab toggles a full-width preview. Ctrl+R renames; Ctrl+D
(twice) moves a session to trash.

`/fork` branches the conversation, either in full or immediately before a chosen prompt. The
branch shares the workspace on disk and carries no permission grants.

`/timeline` (Ctrl+T) lists up to 200 recent prompts. Enter previews; Ctrl+R loads one into the
composer.

`/export` writes a Markdown transcript to a new private file; tool details are optional.

### Other commands

| Command | Action |
| --- | --- |
| `/status` (Ctrl+G) | Provider, credentials, budgets, usage, grants, MCP/LSP/formatter config |
| `/models`, `/agents` | Choose the model (saved on the provider) or agent. Tab cycles agents on an empty composer |
| `/access` | How much goh does without asking, for this session, the project or one folder in it ([Access levels](#access-levels)). Shift+Tab cycles the level |
| `/new` | New conversation with the current model and agent |
| `/settings` | Providers (`p`), advanced settings as JSON (`s`) and workspace (`w`) |
| `/workspace` | Open the project browser and switch the next chat to another folder |
| `/theme` (Ctrl+X T) | `terminal` (default), `dark`, `light`, `high-contrast`. Set `"theme"` to persist |
| `/todos`, `/skills`, `/commands`, `/mcp` | List tasks, skills, custom commands, MCP prompts |

### Display

Markdown headings, lists, quotes, links (label followed by the address), emphasis, code panels
and tables are rendered; prose wraps between words at the terminal's width. Go, JSON, JavaScript
and TypeScript are syntax highlighted. Tool output is shown literally. A fetched page shows as one
line (address, HTTP status, size, truncation, artifact); `/details` cycles through tool output
shown (the default), the same plus a short preview of each fetched page, and one line per result. Each reply ends with a dim line: the model, the run's time, and its tokens in and out (in is mostly the instructions and tool definitions sent with every request). The default `terminal` theme uses the terminal's own
background and text color, so the screen matches the window edge to edge; `dark`, `light` and
`high-contrast` paint their own background. A session sidebar appears on terminals ≥112 columns. The mouse
wheel scrolls; set `GOH_NO_MOUSE=1` for native text selection. `NO_COLOR` disables color. Ctrl+L
redraws.

### Keyboard shortcuts

Override chat shortcuts with `keybinds`. Values are comma-separated keys; `<leader>x` means leader
then `x`; `none` removes a binding.

```json
{"keybinds": {"leader": "ctrl+o", "command_palette": "ctrl+p,f1", "sidebar_toggle": "none"}}
```

| Action | Default |
| --- | --- |
| `leader` | `ctrl+x` |
| `command_palette` | `ctrl+p,ctrl+k` |
| `sidebar_toggle` | `ctrl+b` |
| `timeline` | `ctrl+t` |
| `status` | `ctrl+g` |
| `editor_open` | `<leader>e` |
| `theme_list` | `<leader>t` |
| `messages_first` / `messages_last` | `<leader>g` / `ctrl+alt+g` |
| `messages_previous` / `messages_next` | `<leader>k` / `<leader>j` |
| `messages_half_page_up` / `_down` | `ctrl+alt+u` / `ctrl+alt+d` |
| `messages_copy` | `<leader>y` (last reply as Markdown, via OSC 52) |
| `messages_undo` / `messages_redo` | `<leader>u` / `<leader>r` |
| `suspend` | `ctrl+z` |

Unknown actions, duplicate bindings and keys reserved by the composer and approval prompts are
rejected at load.

---

## Headless CLI

```text
goh run MESSAGE… [--project DIR] [--format text|json] [--allow TOOLS] [--config FILE]
goh run --prompt TEXT | --command NAME | --file PATH …   # or pipe text in: cat log | goh run explain
goh resume ID MESSAGE… [--acknowledge-uncertain]        # or --prompt TEXT
goh compact ID
goh sessions list|show|preview|rename|delete|restore|salvage|export|import|fork
goh doctor
```

- JSON output contains only versioned events; diagnostics go to stderr. Text output prints the
  reply on stdout and one `▸ tool  arguments` line per tool call (plus `[error]` lines) on stderr.
- A rate-limited provider (HTTP 429) is retried twice, waiting as long as its `Retry-After` asks
  (up to 30 s). A limit that resets later than that (a daily quota) fails at once and says when.
- Headless runs never wait for input. A permission request emits `needs_permission` and exits 3;
  a question emits `question_requested` and exits 4.
- `--allow` grants tools for this process only, e.g. `--allow edit_file,shell`. MCP grants are
  `mcp_start:SERVER` and `mcp__SERVER__TOOL`. Grants are never restored on resume.
- `--access ask|edits|auto|full` sets the [access level](#access-levels) for the project for this
  run; levels saved for folders apply to `goh run` too.
- `--profile` and `--agent` select a profile and agent per invocation.
- Background jobs (`job_start`) stop when the run ends. `--keep-jobs` leaves them running, for a task
  that asks for a server or service left up; their output goes to a log in the session's artifacts, and
  the next goh start in that session storage offers to stop them. A shell command's own background
  processes (`server &`) are always stopped when it finishes, and the result says so.
- `GOH_PHASES=1` reports where the run's time went: setup, each model request (first text, retry
  wait), tools, approval and question waits, and the finishing steps. Text mode prints one `phases:` line
  on stderr; JSON adds a `run_phases` event before `run_finished`. Durations and sizes only.

Exit codes: `0` completed, `1` error, `3` needs permission, `4` needs input, `5` budget exhausted,
`6` the configured `verify.command` still failed after the allowed repair attempts (the run
completed; text mode prints `Verification failed: …`, JSON has `outcome: failed_verification`),
`130` cancelled.

`--experiment NAME` (repeatable, or `"experiments": [...]` in the config) turns on a research
prototype: `submission-evidence`, `recoverable-compression` or `retrieval-expansion`. All are off by
default; see [experiments](BENCHMARKS.md).

`run_finished` carries `outcome` (`verified`, `unverified`, `no_changes`, `failed_verification`)
and, additively, `verification_detail`: `change_tracking` (`known`, or `unknown` when the project
is too large or unreadable to fingerprint; the configured verifier then runs after any command
that may write) and `evidence` (the check behind the outcome: `source` `shell` or `configured`,
`command`, `kind` `test`/`build`/`assert`/`run`/`custom`, `scope`, `passed`, `exit_code`,
`revision`). Only a recognized check counts as verification: a test runner, build, lint or type
check, an explicit assertion, or running the program, judged by the program each part of the
command runs (so `echo "go test"` is not a test), and a test run whose output says it ran no tests
does not count. `pwd`, `ls` or `git status` do not. The configured verifier runs as a journaled
effect: if it is interrupted or its cleanup is uncertain, the run stops and the next one needs
`--acknowledge-uncertain`. A verifier that changes files is run once more on its own result; if it
changes files again, the outcome is `unverified`.

---

## Configuration

The user configuration file is `independent-harness/config.json` under the OS config directory
(`~/Library/Application Support` on macOS, `$XDG_CONFIG_HOME` or `~/.config` on Linux). Providers
added through `/connect` are written there for you; hand-edit it (or `/settings` → `s`) for
everything else. `config.example.json` in the repository points at the local scripted model used
by the tests. Configuration files must be regular files ≤1 MiB; unknown fields fail.

| Field | Meaning |
| --- | --- |
| `protocol` | `chat_completions`, `openai_responses`, `anthropic_messages`, `openai_chatgpt` |
| `base_url` | API base including any prefix such as `/v1`; the protocol path is appended |
| `endpoint` | Full request URL (use instead of `base_url` for custom routes or query strings) |
| `model`, `context_window` | Required |
| `output_budget`, `output_parameter` | `max_tokens`, `max_completion_tokens`, or empty to omit |
| `api_key_env` | Variable holding the key (defaults: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) |

**Precedence:** CLI flags (`--base-url`, `--endpoint`, `--api-key-env`, `--protocol`, …) →
environment (`HARNESS_BASE_URL`, `HARNESS_ENDPOINT`, `HARNESS_MODEL`, `HARNESS_PROTOCOL`,
`HARNESS_API_KEY_ENV`, `HARNESS_CONTEXT_WINDOW`, `HARNESS_OUTPUT_BUDGET`, `HARNESS_SESSION_DIR`,
`HARNESS_PROFILE`, `HARNESS_AGENT`) → selected profile → config file → defaults.

Repository guidance (`AGENTS.md` or `CLAUDE.md`) is sent as a separate, labelled untrusted message
and cannot grant permissions. A guidance file beside your user config is part of the system prompt.

---

## Permissions and rules

Listing, reading and searching are automatic. File changes, shell commands and MCP server starts and
calls need approval. Approval previews show real paths, a unified diff of the change (three lines
of context), or the command, working directory and timeout. Large diffs mark truncation.

"For this session" (`s`) covers similar actions: edits anywhere in the project, URLs on the same
origin, and shell commands with the same first two words (`go test`). A command that runs code
given as an argument (`bash -c`, `node -e`, `python3 script.py`, `find -exec`, quoted or
`$`-expanded text) or that is compound (`;`, `|`, `&&`, redirects) is covered only exactly.

In the approval prompt, `a` allows the request and raises this session's access level so similar
requests stop asking: Auto-edit for an edit, Auto for a command (where commands can be sandboxed).

## Access levels

An access level says how much goh does in a folder without asking. Choose it with Shift+Tab (this
session), `/access` (this session, the project, or one folder inside it), or `--access`.

| Level | Runs without asking | Still asks |
| --- | --- | --- |
| **Ask** (default) | Reading and searching | Edits, commands, network, MCP |
| **Auto-edit** | Edits inside the folder; commands that only read inside the project (`ls`, `cat`, `grep`, `git status/diff/log`, `command -v` …, no redirection into files, no paths outside the project), run **in a read-only sandbox** (nothing in the project is writable) | Other commands, network, MCP; read-only commands where no sandbox exists (Windows) |
| **Auto** | The above, plus every command run from inside the folder, **in the sandbox**: it can write only inside that folder and temporary folders, not inside a subfolder saved with a stricter level, and cannot read goh's settings, keys or sessions. Network stays on unless `sandbox.network` is set to false | Network fetches, MCP; commands where no sandbox exists (Windows); on Linux, commands in a folder that contains a stricter subfolder or goh's own state (Landlock cannot restrict a folder inside a writable one); commands when the project is inside a temporary folder (always writable in the sandbox); every command when `sandbox.container` is set (goh cannot see what the container's mounts allow) |
| **Full access** | Everything, with no sandbox | Nothing (except your ask/deny rules) |

- A level saved for a folder covers it and its subfolders; the deepest saved folder wins. Give one
  folder full access (`/access` → Full access → Only in a folder inside it…) and the rest of the
  project keeps its own level. Saved levels live beside the trusted folders in `workspaces.json`.
- A session level (Shift+Tab, `--access`) replaces the project folder's saved level for this session
  only; deeper saved folders keep theirs.
- `file_rules` and `shell_rules` with `deny` or `ask` win over every level, and the repeated-call
  guard always asks.
- Each decision is journaled with `"source": "access"` and the level, and the folder a command
  was confined to.

### Environment filtering

Shell commands, background jobs, formatters and MCP/LSP servers receive only a safe environment
(PATH, HOME, user, shell, terminal, locale, `XDG_*`, temp folders, toolchain variables such as
`GOPATH`, `JAVA_HOME`, `VIRTUAL_ENV`, and Windows system variables). API keys, tokens, cloud
credentials and proxy settings are withheld. Pass more explicitly:

```json
"env_passthrough": ["DATABASE_URL", "MYAPP_*", "HTTPS_PROXY"]
```

`"*"` passes everything. MCP/LSP server `env` settings are always added.

### Shell rules

```jsonc
"shell_rules": [
  {"match": "git status*", "action": "allow"},
  {"match": "go test *",   "action": "allow"},
  {"match": "*rm -rf*",    "action": "deny"},
  {"match": "git push*",   "action": "ask"}   // ask even if shell is approved for the session
]
```

- The last matching rule wins. Rules apply to `shell` and `job_start`, including subagents, but not
  to `!` commands you type.
- `allow` never covers compound commands (`;`, `&`, `|`, backticks, `$(`, redirects, newlines).
- `deny` is checked against the whole command and each part of a compound command.
- `deny` also checks each part the way it runs: `/bin/rm`, `command rm`, `FOO=1 rm`, `sudo rm`,
  `env … rm` and `sh -c 'rm …'` all match `rm *`. Patterns still match text, not intent
  (`find . -delete` is not `rm`). Treat deny rules as a guardrail; use the sandbox for containment.

### File rules

```jsonc
"file_rules": [
  {"match": "*.env*",    "action": "deny"},   // hidden from reads and never edited
  {"match": "*_test.go", "action": "allow"},
  {"match": "go.mod",    "action": "ask"}
]
```

Patterns match project-relative paths. A `deny` rule also hides files from reads, searches,
listings and `@` attachments. A patch is denied if any file it touches is denied, and auto-allowed
only if every file is allowed.

### Generated files

`edit_file`, `write_file` and `apply_patch` refuse to modify files with an `@generated` marker or
`Code generated … DO NOT EDIT.` header in the first 8 KiB. Set `"allow_generated_edits": true` to
override.

---

## Shell sandbox

Approval decides whether a command runs; the sandbox limits what it can do. It applies to shell
commands, `!` commands, background jobs and formatters, not to MCP/LSP servers or hooks.

```json
"sandbox": {"enabled": true, "network": false, "writable": ["~/.cargo/registry"]}
```

- Writes are allowed only in the project, temp folders and `writable` paths.
- Build and package caches that live elsewhere point at a private per-user folder in the temporary
  area (`GOCACHE`, `npm_config_cache`, `PIP_CACHE_DIR`, `XDG_CACHE_HOME`), unless your environment
  sets them: `go test` works in the sandbox without a retry, later builds reuse that cache, and your
  own caches cannot be changed by a sandboxed command. To use your own cache instead, set the variable
  and list its folder in `writable`.
- GoHarness's own config, credential and session folders are unreadable.
- `"network": false` (default) blocks network access.
- macOS uses `sandbox-exec`. Linux uses Landlock (5.13+; network blocking needs 6.7+). On macOS,
  `file_rules` denies also apply to commands, and the project's `.git/hooks` and `.git/config` are
  read-only (git runs them outside the sandbox); on Linux neither applies. On Linux the folders
  above a protected one can be listed, so names inside it are visible but not its contents.
- Where no sandbox is available (Windows, Linux without Landlock), commands are **refused**, not
  run unconfined.

Without a sandbox, approved commands run as `/bin/sh -c` in the project. Output is captured to a
1 MiB private artifact. Cancellation terminates the process group and escalates after two seconds.

---

## Hooks

Hooks run your own commands at points in the agent loop. They are read only from user
configuration, never from the repository.

```json
"hooks": [
  {"event": "pre_tool", "tools": ["shell"], "command": "/Users/me/bin/guard-shell"},
  {"event": "post_tool", "tools": ["edit_file", "apply_patch"], "command": "/Users/me/bin/lint", "timeout_seconds": 30},
  {"event": "run_finished", "command": "/Users/me/bin/notify"}
]
```

| Event | Behavior |
| --- | --- |
| `pre_tool` | Runs after approval, before the tool. Exit 0 continues; any other exit or timeout refuses (exit 2 shows stderr to the model). Hooks cannot grant approval. |
| `post_tool` | Exit 2 attaches output to the result as `hook_feedback`. |
| `run_finished` | Receives the final status; cannot change it. |

Hooks run without a shell, in the project folder, with the filtered environment plus `GOH_EVENT`,
`GOH_TOOL`, `GOH_SESSION_ID` and `GOH_PROJECT`. A JSON payload arrives on stdin. `tools` accepts
names or patterns such as `mcp_*`. Default timeout 10 s, max 120 s.

---

## Verification and budgets

```json
"verify": {"command": "go test ./...", "repair_attempts": 1, "timeout_seconds": 300}
```

After any run that changed files, the command runs (inside the sandbox if enabled, without an
approval prompt). On failure, its output goes back to the model, up to `repair_attempts` times
(default 1, max 3). The outcome is then `verified` or `failed_verification`. `--verify` sets this
per run.

Without `verify`, a run that changed files is reminded once before it finishes when: no check
passed after the last change; the passing checks used a fake, mock or stub the model wrote; or the
only passing tests were picked by name (`go test -run`, `pytest -k` or `::`, `jest -t`,
`cargo test NAME` and similar), in which case it is asked to run the full tests of what it changed.
`"completion_check": false` turns the reminder off.

Run budgets (child tasks count against the parent):

| Field | Limit |
| --- | --- |
| `max_tool_calls` | Tool calls per run |
| `max_run_seconds` | Wall time per run |
| `max_run_tokens` | Input + output tokens as reported by the provider |

Hitting a budget ends the run as `budget_exhausted` (exit 5).

---

## Built-in tools

| Tool | Purpose |
| --- | --- |
| `list_files`, `read_file` | Read the project |
| `glob`, `grep` | Recursive globs (≤100 files) and Go-regex line search (≤100 lines) |
| `write_file`, `edit_file` | Hash-guarded file changes (`write_file` creates a file or replaces one) |
| `apply_patch` | Multi-file Add/Update/Delete/Move patches |
| `shell`, `job_start` | Commands and background jobs |
| `todowrite` | Session task list (≤1,000 tasks) |
| `question` | Structured questions to the user |
| `task` | Child agent |
| `skill` | Load a skill (offered only when skills are installed) |
| `webfetch`, `websearch` | Web access |
| `lsp` | Language server queries |
| `repo_map` | Go signatures and declarations without bodies |
| `read_artifact` | Read or search (`pattern`) retained shell/formatter output and fetched pages |
| `snapshot_list`, `snapshot_diff`, `snapshot_undo`, `snapshot_redo` | Checkpoint inspection and restore (callable, not offered; you use `/undo`, `/redo`, `/changes`, `/snapshots`) |
| `memory_recall` | Project notes (opt-in) |

`search` (the same as `grep` with a limit), `create_file` (covered by `write_file`) and the snapshot
tools still work when a model or an older session calls them, but are not offered: every offered
definition is sent with every request, and models almost never used them (one snapshot call in 453
benchmark requests).

Tool results reach the model as plain text: the main output verbatim (a file's text, a command's
output, `path:line: text` matches, one path per line), after one line of the fields that carry
information (`sha256`, `exit_code`, `artifact_id`, `truncated`, …), and `Error: ` before a failure.
Sessions, exports and the UI keep the full structured result.

### File editing

- Paths must be inside the project; absolute paths, escapes, symlinks, special files and non-UTF-8
  content are rejected. Nothing may write inside `.git`.
- `read_file` returns a SHA-256. Writes and edits to existing files require `expected_sha256`.
- A file too large for one result (64 KB): reading the whole file returns its line count and size
  and suggests a line range or `grep`, rather than its first screens, which would be resent with
  every later request; an explicit line range that is too large returns the whole lines that fit and
  `next_start_line`.
- `edit_file` must match `old_text` exactly once unless `replace_all` is set. CRLF, file mode and
  unaffected bytes are preserved.
- New files never overwrite an existing path.

### `apply_patch`

```json
{"patchText":"*** Begin Patch\n*** Update File: parser.go\n@@\n-old\n+new\n*** End Patch","expected_sha256":{"parser.go":"SHA256_FROM_READ_FILE"}}
```

Every existing source needs its hash. New and moved destinations must not exist; parent directories
must. All files are validated and staged before publication, but publication is per file, not a
transaction: a late failure stops the run with uncertain effects for you to inspect. Limits: 100
operations, 1 MiB arguments, 16 MiB total data.

### `question`

Up to 8 questions with up to 16 choices each. In the UI: arrows choose, Space toggles, Enter
submits, `c` enters a custom answer, Esc dismisses. Answers are data, never grants.

---

## Sessions and recovery

Sessions are stored in `~/.local/state/independent-harness/sessions` (override with
`session_dir`, `HARNESS_SESSION_DIR` or `--session-dir`) as private, versioned JSONL journals with an
exclusive writer lock. Tool starts are fsynced before execution.

```sh
goh sessions list | show ID | preview ID
goh sessions rename ID TITLE
goh sessions delete ID          # moves to trash
goh sessions restore ID
goh sessions salvage ID         # copy the readable part of a damaged journal into a new session
goh sessions fork ID [--before-turn N]
goh sessions export ID [--format markdown --include-tools] [--output FILE]
goh sessions import FILE --project DIR
```

- A journal cut off by a crash is repaired automatically. Other damage is refused; use `salvage`.
- An interrupted tool can leave uncertain effects. Inspect the project, then continue with
  `goh resume ID --acknowledge-uncertain --prompt TEXT`. Nothing is replayed automatically.
  `--retry-interrupted` re-sends only a dropped model request.
- Export, import and fork create new sessions and never carry permission grants. Exports are
  unredacted and never overwrite existing files. Checkpoint blobs are not exported.
- A session open in any running GoHarness cannot be deleted.

---

## Context and compaction

Context is estimated conservatively: one token per UTF-8 byte (tool schemas included) until the
provider reports real token counts. After that, the smallest counted request (mostly goh's
instructions and tools) is the baseline, and bytes beyond it count at the ratio the provider
reported for them, with a 15% margin. When the context fills, compaction runs:

1. **Masking.** Tool results ≥512 bytes older than the recent turns are replaced with a placeholder
   pointing to the journal. If the result fits in half the context window, no summary is needed.
   Disable with `"compaction_masking": false`.
2. **Summary.** Otherwise the model summarises the masked view, preserving the active request,
   the two most recent tool groups, unresolved calls and the task list.

A tool result larger than the window allows is cut to its beginning plus a note; once it is older,
it becomes a plain receipt. A fetched page or shell log stays readable through `read_artifact`.

**Per-request caps.** Some providers accept less per request than the model's context window. For
example, Groq's free tier allows about 8k tokens, and that includes the requested output. When a
provider refuses a request with a stated cap (HTTP 413, or a 429 whose stated request size exceeds
its limit), goh uses that cap as the window for the rest of the session. It shrinks the output
reserve to a quarter of the cap and the context to fit, then resends the turn; nothing was generated
or run. This happens at most twice per run, and the notice reads "The provider accepts at most N
tokens per request…". A 429 that says to wait longer than two minutes (a daily quota) fails at once
and says when the quota resets.

The full history always stays in the journal. `/compact` or `goh compact ID` summarises manually.
If the essential state cannot fit, the run stops with `budget_exhausted` rather than dropping the
request.

---

## Undo, checkpoints and snapshots

### Turn undo

`/undo` restores files changed by the last turn's built-in file tools, then continues on a branch
before that turn with its prompt back in the composer. `/redo` reverses it until you send a new
prompt. Shell and MCP effects are not reverted; the notice lists them. If a file was changed
outside GoHarness, undo refuses and rolls back anything it had already restored.

CLI: `goh changes undo-turn SESSION`, `goh changes redo-turn SESSION`.

### File checkpoints

Checkpoints of built-in file operations are on by default (best effort). `"snapshots": true` or
`--snapshots` makes them strict (an edit that cannot be checkpointed is refused); `false` disables
them. Limits: 64 paths and 4 MiB per file per operation, 1,024 checkpoints.

```sh
goh changes list SESSION
goh changes diff SESSION --id CHECKPOINT
goh changes undo SESSION --id CHECKPOINT
goh changes redo SESSION --id CHECKPOINT
goh changes acknowledge SESSION --id CHECKPOINT   # retire an interrupted checkpoint
```

Restores check every file's hash, existence and mode first. Multi-file restores are not atomic.

### Whole-project snapshots

To also undo what shell commands change, enable:

```json
"project_snapshots": true
```

The project is recorded at the start and end of every run in a private shadow Git repository
(requires `git`). `.gitignore` is respected and your own repository is never touched.

```sh
goh snapshots list SESSION
goh snapshots diff SESSION 2
goh snapshots restore SESSION 2 [--yes]
goh snapshots undo SESSION --yes
```

---

## Agents, profiles and model roles

```json
{
  "profile": "local",
  "profiles": {"local": {"endpoint": "http://127.0.0.1:18766/v1/chat/completions", "model": "scripted", "context_window": 131072}},
  "agents": {"reviewer": {"description": "Read-only review", "mode": "plan", "profile": "local", "instructions": "Explain findings and suggest tests."}},
  "commands": {"review": {"description": "Review chosen files", "template": "Review $1. Additional context: $2"}}
}
```

- Built-in agents are `build` and `plan`. Custom agents declare one of those modes.
- **Plan mode** exposes only reads, skills and questions. It refuses edits, shell and MCP even with
  explicit grants.
- Select with `--profile` / `--agent`, `/models` / `/agents`, or set defaults in the config.

`model_roles` route specific work to other profiles:

```json
"model_roles": {"compaction": "fast-summary", "task": "worker", "explore": "fast-summary"}
```

Unset roles inherit; `explore` falls back to `task`. Roles never change permissions.

---

## Skills and custom commands

### Skills

Discovered in `.agents/skills`, `.claude/skills`, `.opencode/skills`, `.opencode/skill` and
`.harness/skills` under the project, plus any `skill_paths`. Each skill is a `SKILL.md` with `name`
and optional `description` in YAML frontmatter. Loading a skill needs approval (`--allow skill`)
and returns its body and up to ten resource paths; scripts are never executed. Limits: 256 skills,
40 KiB per `SKILL.md`.

### Custom commands

Defined under `commands` (see above) and usable as `/review parser.go "edge cases"` in the UI or
in `--prompt`. `$ARGUMENTS` is the raw argument text; `$1`, `$2`, … are positional, with the
highest placeholder taking the remainder. Templates without placeholders get the arguments
appended. Expansion is text-only: shell syntax and `@file` stay literal.

---

## Subagents

`task` runs a child agent with its own journal. Inputs: `description`, `prompt`, `subagent_type`
(`general`, `explore`, or a custom agent), optional `task_id` to continue a child, and optional
`output_schema` (JSON Schema ≤16 KiB) for structured output.

- `explore` is read-only. A plan-mode parent limits children to plan mode.
- The parent needs `--allow task`; child tools need separate grants, e.g.
  `--allow task,child:create_file,child:shell`. Parent grants are not inherited.
- Limits: 8 children per run, 16 model turns and 180 s per child.

---

## MCP

### Local servers

```json
{"mcp": {"local": {"command": "/absolute/path/to/server", "args": [], "env": {}}}}
```

Servers come only from user configuration and are never installed automatically. Every start and
call needs approval. Tools are namespaced `mcp__SERVER__TOOL`; schemas with `$ref` are rejected.
Limits: 128 tools per server, 1 MiB frames, 64 KiB arguments and results, 15 s initialisation,
60 s calls. A cancelled or disconnected call is reported as uncertain and never retried.

### Remote servers

```jsonc
"mcp": {
  "docs": {"url": "https://mcp.example.com/mcp", "headers_env": {"Authorization": "DOCS_MCP_TOKEN"}}
}
```

Streamable HTTP over HTTPS only (HTTP for localhost); redirects are refused. Approval prompts show
header names, never values.

**Signing in (OAuth).** A remote server without an `Authorization` header that asks for sign-in is
signed in to once, in the browser:

```sh
goh mcp login docs     # discovers the server's authorization server, registers, opens the browser
goh mcp list           # shows "signed in" under each remote server that is
goh mcp logout docs    # forgets the sign-in
```

In the chat, `/mcp login docs` and `/mcp logout docs` do the same (Esc cancels a sign-in waiting
for the browser); a run that stopped a server for lack of sign-in says so.

The token is saved like the ChatGPT sign-in (a private file, or sealed with the OS credential store
after `goh auth store keychain`), kept for that exact server URL, and refreshed as it expires. Runs
never open a browser: a server that still refuses fails to start with "requires sign-in: run
`goh mcp login NAME`". For an authorization server without dynamic client registration, set the
client registered with it: `"oauth_client_id": "…"` (a public client; PKCE is always used).

### Resources and prompts

Servers offering resources add `mcp_resources` and `mcp_read_resource`. Prompt templates are listed
with `/mcp` and run as `/mcp:<server>:<prompt> [name=value ...]`.

---

## LSP

```json
{
  "lsp": {
    "go": {"command": "/absolute/path/to/gopls", "args": [], "extensions": [".go"], "language_id": "go"}
  }
}
```

The `lsp` tool supports `hover`, `goToDefinition`, `findReferences`, `goToImplementation`,
`documentSymbol`, `workspaceSymbol` and `diagnostics`. Positions are 1-based and needed only by
the first four; the others work on the whole file (or, for `workspaceSymbol`, a query). The call
and server start are approved separately: `--allow lsp,lsp_start:go`. Server requests to edit files or run commands are refused.

After a built-in edit, if that file's language server is already running, up to 20 error
diagnostics are attached to the result as `lsp_errors`. Edits never start a server.

---

## Formatters

```json
"formatters": {
  "gofmt": {},
  "custom": {"command": "/absolute/path/to/formatter", "args": ["--write", "$FILE"], "extensions": [".example"], "timeout_seconds": 30}
}
```

An empty entry uses the executable of that name on PATH. After an approved edit, each matching
formatter needs its own `format:NAME` approval (`child:format:NAME` in subagents). A failed
formatter keeps the edit. Arguments are never evaluated by a shell. Plan mode cannot run
formatters.

---

## Web fetch and search

`webfetch` takes `url`, `format` (`text`, `markdown`, `html`; default markdown) and `timeout`
(default 30 s, max 120 s). Each request needs approval. It sends no API headers or cookies, follows
up to five same-origin redirects, accepts UTF-8 text up to 4 MiB, never executes scripts, and
returns at most 64 KiB. HTML is converted to Markdown or text from the page's main content when the
markup identifies it (a rendered README, a single `<main>` or `<article>`), otherwise from the whole
body; navigation, banners, footers, hidden elements, empty links and list items are left out, and
blank lines are collapsed outside preformatted text. `format: "html"` returns the page unchanged.
A page over 16 KiB, or one the result only excerpts, is also saved as a session artifact (`artifact_id` in the
result), which `read_artifact` can search (`pattern`) or read in parts. A small context window can
then show only the start of a page and still reach the rest. JSON APIs work (the Accept header ends
with `*/*`). An HTTP error reports
its status, not the body. A cross-origin redirect is refused and names its destination, which
the model can fetch as a separately approved request. Private, loopback and link-local addresses
(including cloud metadata) are refused unless `"webfetch_allow_private": true`.

Set `websearch_url` to a SearXNG-compatible JSON endpoint to enable `websearch`. Each search needs
approval and returns titles, links and snippets.

---

## Project memory

Opt-in notes tied to file fingerprints. Enable with `"memory": true`.

```sh
echo 'The parser fixture depends on the legacy grammar.' | goh memory add --project . --path parser.go
goh memory list --project .
goh memory recall --project . --query 'parser grammar'
goh memory delete --project . --id NOTE_ID
```

Notes reach the model only through `memory_recall`, which returns up to three keyword matches and
skips any note whose linked files have changed or disappeared.

---

## HTTP API

```sh
export HARNESS_SERVER_TOKEN="$(openssl rand -hex 32)"   # at least 32 bytes
goh serve --project /path/to/project --config /path/to/config.json
```

Loopback addresses only; port 0 picks a free port. Send `Authorization: Bearer <token>` on every
request.

| Request | Result |
| --- | --- |
| `GET /v1/info` | Project, session directory, model, agent, protocol |
| `POST /v1/sessions` | Create a session (201) |
| `GET /v1/sessions?limit=100&after=...` | List sessions; use `next_after` as the cursor |
| `POST /v1/sessions/{id}/run` `{"prompt": "..."}` | Stream NDJSON; returns `X-Harness-Run-ID` |
| `POST /v1/runs/{run}/inputs/{input}` | Answer a pending permission or question (204) |
| `POST /v1/runs/{run}/cancel` | Request cancellation (202) |

Each NDJSON line has an `event` (same schema as `goh run --format json`) or an `input` challenge.
Answer permissions with `{"allow": true|false}` and questions with `{"answers": [["..."]]}` or
`{"dismiss": true}` on a separate request while still reading the stream. Wait for `run_finished`.

One run at a time per server, 15-minute run deadline, 2-minute challenge deadline. Disconnecting
cancels the run. Sessions with uncertain effects return 409 until a run is sent with
`"acknowledge_uncertain": true`.

---

## Attaching the UI to a server

```sh
export HARNESS_SERVER_TOKEN=...
goh attach 127.0.0.1:4096        # new session
goh attach --continue            # latest session
goh attach --session ID
```

Prompts run on the server with its model, tools, grants and sandbox; approvals appear in the
attached UI. Local-write operations (`!command`, `/compact`, undo/redo, fork, model or provider
changes) are refused while attached. `--token-env NAME` reads the token from another variable.

---

## ACP editor adapter

Configure your editor to launch:

```sh
goh acp --project /path/to/project --config /path/to/config.json
```

ACP v1 JSON-RPC over stdio. Supported: `initialize`, `session/new`, `session/resume`,
`session/close`, text `session/prompt`, `session/cancel`, and `session/request_permission` for each
approval. Clients may supply stdio MCP servers at session creation; they start only after approval.
One prompt runs at a time per process. Media, `session/load` and client-hosted terminals or
filesystems are not supported.
