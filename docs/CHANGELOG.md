# Changelog

GoHarness release candidates in chronological order. Dates are commit dates.
Each entry lists user-visible changes; evidence for each is in
[VERIFICATION.md](VERIFICATION.md). No stable release has been published yet.

## Unreleased

## 1.0.0-rc.162 — 2026-11-10

- **Container sandbox:** `sandbox.container` or `goh run --sandbox-container
  NAME` runs every shell command inside a running Docker container
  (`docker exec`), with the project mounted at the same path. The command
  gets the container's environment, never the host's.
- **Benchmark isolation:** a task with `image` runs setup in a container with
  network, and the agent's shell commands and grading in separate containers
  with no network, mounting only the attempt folder and a dependency cache.
  The miner (`bench/tools/mine_go.py`) runs every Go command in Docker too.
- **19 real-project tasks** mined from bug-fix commits in gjson, cast,
  BurntSushi/toml, gorilla/mux, go-humanize, go-toml v2, sjson and
  Masterminds/semver, each validated in containers and given a symptom-only
  prompt (55 tasks in total).

## 1.0.0-rc.161 — 2026-11-10

- **Token budget:** `max_run_tokens` and `goh run --max-run-tokens N` stop a
  run (status `budget_exhausted`, exit code 5) before the next model turn once
  N tokens (input plus output, as the provider reports them) are used, child
  tasks included. A provider-neutral spending cap; set it in tokens because
  prices differ by provider and plan.

## 1.0.0-rc.160 — 2026-11-10

- **MCP prompts:** servers that offer prompt templates are listed with `/mcp`
  in the TUI and run as `/mcp:<server>:<prompt> [name=value ...]` (free text
  goes to the first argument). The rendered text is labelled as provided by
  that server and grants no permissions; it is journalled like a custom
  command. The model never fetches prompts by itself.
- Refreshed README screenshots (command palette with current commands).

## 1.0.0-rc.159 — 2026-11-10

- **Line-range edits (opt-in):** `line_edits: true` or `goh run --line-edits`
  lets `edit_file` replace lines `start_line..end_line`, numbered as
  `read_file` shows them, instead of matching `old_text`. The whole-file
  `expected_sha256` guard anchors the numbers to the content the model read;
  `old_text`, if also given, must equal those lines. Off by default; the
  default tool schema is unchanged. Added for an A/B test against exact-text
  edits (component plan C1).

## 1.0.0-rc.158 — 2026-11-10

- **Test-double check:** if a run's checks pass but the model wrote a fake,
  mock or stub in a test file during the run, the model is reminded once to
  confirm its calls against the real definitions and run the existing tests.
  Targets the one false completion pattern seen in the benchmark.
- **Opt-in stream retry:** `retry_interrupted_streams` (0-2) and
  `goh run --retry-interrupted N` re-send a model request whose stream broke.
  Off by default; the partial reply is discarded, its tool calls never run,
  and the TUI and CLI say so.
- README front page: overview, screenshots and verified benchmark results.
- Benchmark fix: `hard-py-double-charge` now documents the gateway's
  `lookup()` method its hidden test relies on. Its two false completions came
  from that missing interface; after the fix it was solved 3 of 3.
- The test-double check also recognises stand-ins named after an external
  role (`IdempotentGateway`, `recordingClient`) and mocking libraries.

## 1.0.0-rc.157 — 2026-11-09

- Errors reported inside a Responses/ChatGPT stream now show the provider's
  error code and a short message (for example
  `subscription_sharing_usage_limit_exceeded`), instead of a generic "request
  failed", so a usage limit is distinguishable from an access problem.

## 1.0.0-rc.156 — 2026-11-09

- **Real context window for ChatGPT models:** the window is read from the
  account's model catalog (gpt-6.1-sol: 500,000 tokens) and cached privately
  for a day, instead of a fixed 32,768. Found by the first live benchmark,
  where 4 of 22 short tasks ended in `budget_exhausted` with "compaction did
  not reduce context enough".
- Benchmark: `py-mutable-default`'s prompt now states the requirement its
  hidden test checks (a passed-in list is not modified).

## 1.0.0-rc.155 — 2026-11-09

- **`/snapshots` in the TUI (C10):** lists each run's whole-project changes
  and shows the exact `goh snapshots restore` / `undo` commands (restoring
  rewrites files, so it stays an explicit CLI step).

## 1.0.0-rc.154 — 2026-11-09

- Benchmark suite expanded to **22 validated tasks** (11 Go, 11 Python),
  adding multi-package, concurrency, HTTP, refactoring and parsing tasks.
  Tests the agent can see are also kept in `hidden/`, so grading always uses
  the originals and edits to them are flagged.

## 1.0.0-rc.153 — 2026-11-09

- **Benchmark runner (C11):** `cmd/bench` (`make bench-check`) runs pinned
  tasks through `goh run` and regrades each result in a clean copy with
  hidden tests (SWE-bench style: fail-to-pass and pass-to-pass). It reports
  resolved rate, false completions, budget stops, tool calls, time and
  tokens (cached shown separately) per arm, so any `goh run` flag can be A/B
  tested. Eight validated starter tasks in Go and Python. See
  `bench/README.md`.

## 1.0.0-rc.152 — 2026-11-09

- **Headless runs (C10):** `goh run` gains `--verify CMD`,
  `--max-tool-calls N`, `--max-run-seconds N` and `--completion-check=false`
  (overriding config for one run), `--allow all` for every built-in local
  tool (web, LSP and MCP stay explicit), and exit code 5 for
  `budget_exhausted`. `--reasoning-effort` help lists current levels.

## 1.0.0-rc.151 — 2026-11-09

- **MCP resources (C9):** when a connected MCP server offers resources, the
  model gets `mcp_resources` (list) and `mcp_read_resource` (read text up to
  64 KiB; binary by size). Reading changes nothing and the server was
  approved at start, so these need no per-call approval. Gap analysis M1
  (resources; prompts remain).

## 1.0.0-rc.150 — 2026-11-09

- **Prompt cache key (C5):** Responses and ChatGPT requests carry one
  `prompt_cache_key` per session, so a conversation's append-only requests
  share a cache (as Codex does). If an endpoint names the field as
  unsupported, the request is resent once without it.

## 1.0.0-rc.149 — 2026-11-09

- **Multi-language repository map (C4):** `repo_map` now lists functions,
  classes, types and methods in Python, JavaScript/TypeScript, Rust,
  Java/Kotlin/C#/Scala, C/C++ and Ruby (line patterns, no new dependency),
  alongside the exact Go parser. Same limits, hash-validated cache and
  per-symbol line numbers.

## 1.0.0-rc.148 — 2026-11-09

- **Masking window (C3):** context masking now keeps the 10 most recent tool
  results verbatim whenever that still fits, and only masks older ones;
  previously everything before the last two turns was masked, which could
  make the model re-read files it had just seen.

## 1.0.0-rc.147 — 2026-11-09

- **Completion check (C2):** when a run is about to finish after changing
  files and nothing has checked them since, the model is reminded once to
  run the relevant test or build, or say why none applies. On by default;
  `"completion_check": false` turns it off; skipped when `verify` is set.
- Messages GoHarness writes itself (checks, repair requests, pinned
  instructions) are marked automatic: they no longer count as your turns in
  undo, revert, fork, the timeline or statistics, and are labelled
  "GoHarness (automatic)" in the TUI and exports. This also fixes the
  RC144 verify/repair messages being counted as turns.

## 1.0.0-rc.146 — 2026-11-09

- **Tolerant edits (C1):** when `edit_file`'s `old_text` is not found
  exactly, a match that ignores leading/trailing whitespace on each line is
  accepted **only if it is unique**; the new text is re-indented to the
  file. Ambiguous matches are refused with their line numbers, and a
  not-found error says where the first line of `old_text` does appear.

## 1.0.0-rc.145 — 2026-11-09

- `"max_run_seconds"` bounds a run's wall time; child tasks run inside the
  same deadline, and reaching it ends the run as `budget_exhausted`.
- Child tasks share the parent's `max_tool_calls` budget.

## 1.0.0-rc.144 — 2026-11-09

External review roadmap (R21, partial):
- **Verify and repair:** `"verify": {"command": …}` runs your check after a
  run that changed files; failures go back to the model for a bounded number
  of repair attempts; the outcome is `verified` or `failed_verification`.
- `"max_tool_calls"` caps tool calls per run.
- An adversarial canary test found that a sandboxed shell could still read a
  file denied by `file_rules`. On macOS the sandbox now blocks denied files
  too; on Linux this is a documented Landlock limit.

## 1.0.0-rc.143 — 2026-11-09

External review fixes (R19, R20):
- `reasoning_effort` accepts each protocol's current levels: OpenAI
  `none`…`xhigh`, Anthropic `low`…`max` (sent as `output_config.effort`).
- `"anthropic_thinking": "adaptive"` for current Claude models, with
  summarized reasoning; `thinking_budget` remains for older models. Stored
  thinking blocks are always replayed unchanged during tool use.
- Repository guidance (`AGENTS.md`/`CLAUDE.md`) moved out of the system
  prompt into its own labelled, untrusted user-role message.

## 1.0.0-rc.142 — 2026-11-09

External review fixes (R17, R18):
- Earlier user instructions survive context compaction **verbatim** in a
  separate message (oldest first; later ones take precedence), carried
  forward through repeated compactions, bounded to 30 entries / 16 KB.
- Recalled project-memory notes are rechecked every time context is sent to
  the model; a note that was deleted, whose evidence files changed, or whose
  memory was turned off is withdrawn (the journal keeps the original).

## 1.0.0-rc.141 — 2026-11-09

External review fix (R15):
- "Allow for this session" now covers only the reviewed kind of action:
  shell commands and background jobs with the same first two words (a
  compound command only matches itself exactly), and `webfetch` requests to
  the same origin. Edits and individual MCP tools stay tool-wide. The TUI
  approval line shows what `s` will cover. `--allow` on the command line
  still grants the whole tool.

## 1.0.0-rc.140 — 2026-11-09

External review fixes (R11, R16):
- Completed runs record an `outcome`: `verified` (a shell command succeeded
  after the last file change), `unverified` (files changed, nothing checked
  them afterwards) or `no_changes`. `goh run` prints a note for unverified
  changes. This is a heuristic: any successful command counts as a check.
- An empty model reply (no text, no tool calls) now fails the run instead of
  counting as success.
- An MCP call that completed but returned oversized output is reported as
  completed with a bounded excerpt and a do-not-repeat note, not as an error
  the model might retry.

## 1.0.0-rc.139 — 2026-11-09

External review fix (R14):
- Usage is recorded for **every** model call (turns, summaries, titles) with
  its purpose, and charged to the model that actually served it, so a
  fallback route no longer bills the primary model.
- Cache reads and writes are recorded separately (Anthropic, OpenAI Chat and
  Responses). `goh stats` shows them, and prices accept optional
  `cache_read` and `cache_write` rates (unset means ordinary input price).

## 1.0.0-rc.138 — 2026-11-09

External review fixes (R10, R12, R13):
- Resuming after a crash reuses a tool result that was already journaled
  instead of reporting it as uncertain; a call that never started is
  reported as not executed.
- Masked older tool output keeps a typed receipt (status, exit code, path,
  SHA-256, artifact id). For actions with effects it says not to run them
  again and points to `read_artifact`; only read-only tools are suggested
  for re-running.
- A single result too large for the context window no longer ends the run:
  it is cut to an excerpt with a receipt (and paging advice for files).

## 1.0.0-rc.137 — 2026-11-09

External review fixes (R6–R9):
- A `file_rules` **deny** now also blocks reading, searching (`grep`),
  listing (`glob`, `list_files`), the repository map and `@path`
  attachments, not only edits.
- No tool may create or change anything inside a `.git` folder, at any
  depth, whatever the grants (a planted hook would run on the next commit).
- `webfetch` refuses loopback, private-network, link-local (including cloud
  metadata) and shared addresses, checked on the exact address it connects
  to, for every redirect. `"webfetch_allow_private": true` allows them.
- MCP arguments with duplicate JSON keys are refused, and the server receives
  exactly the object that was validated and approved.

## 1.0.0-rc.136 — 2026-11-09

External review fixes (see docs/REVIEW_FINDINGS.md R1–R5):
- Events from parallel read workers and hooks are numbered and emitted one
  at a time (fixes a data race with `post_tool` hooks).
- A tool's raw result is journaled before `post_tool` hooks run.
- Hooks run under the shell sandbox when it is enabled.
- Once a run decides to stop, no later call in the same batch runs.
- The repeat guard compares canonical arguments and also checks identical
  calls inside a parallel read batch.

## 1.0.0-rc.135 — 2026-11-09

- **MCP reconnect:** when an MCP server exits or its connection drops, the
  next call to it restarts or reconnects the server (through the normal
  approval path) instead of failing for the rest of the run. The call that
  failed is never repeated. At most 3 restarts per server per run. Gap
  analysis M3.

## 1.0.0-rc.134 — 2026-11-09

- **Shell sandbox** (opt-in, `"sandbox": {"enabled": true}`): shell
  commands, background jobs and formatters can write only in the project,
  temporary folders and listed paths; cannot read GoHarness's config,
  credentials or sessions; and have no network unless `"network": true`.
  macOS uses `sandbox-exec`, Linux uses Landlock; elsewhere commands are
  refused rather than run unconfined. Gap analysis S3.

## 1.0.0-rc.133 — 2026-11-09

- **Whole-project snapshots** (opt-in, `"project_snapshots": true`): the
  project is recorded at the start and end of every run in a private shadow
  Git repository, so changes made by shell commands can be listed, diffed
  and reverted with `goh snapshots list|diff|restore|undo`. `.gitignore`d
  files are skipped and the user's own repository is never touched. Restores
  list first and need `--yes`; every restore can be undone. Gap analysis A2.

## 1.0.0-rc.132 — 2026-11-09

- **Hooks:** `pre_tool`, `post_tool` and `run_finished` hooks from user
  configuration. A pre-tool hook can refuse an approved tool (exit 2, or any
  failure: hooks fail closed) but can never approve one; a post-tool hook can
  attach feedback for the model. Hooks get the filtered environment and a
  JSON payload on stdin. Gap analysis item A1.

## 1.0.0-rc.131 — 2026-11-09

- **Anthropic prompt caching:** requests now mark cache breakpoints on the
  last tool, the last system block and the newest message, so each turn
  reuses the previous turn's prefix at cache-read prices instead of paying
  full input price again. `"disable_prompt_cache": true` turns it off for
  gateways that reject `cache_control`. Gap analysis item P1.

## 1.0.0-rc.130 — 2026-11-09

- **Security:** shell commands, background jobs, formatters and MCP/LSP
  servers no longer inherit the full environment. They get a fixed safe set
  (PATH, HOME, locale, terminal, toolchain locations, Windows system
  variables); API keys, tokens and cloud credentials are withheld. New
  `env_passthrough` setting lists extra names (`NAME`, `PREFIX_*`, or `*` for
  everything). Gap analysis items S1 and S2.

## 1.0.0-rc.129 — 2026-11-09

- Repository restructured and renamed to `Sanidhya14321/goh`: the Go project
  (`cmd/`, `internal/`, `scripts/`, `go.mod`, `Makefile`) now sits at the
  repository root; documentation moved to `docs/`, the plan and specs to
  `docs/plan/`, and research material to `research/`.
- The Go module path is now `github.com/Sanidhya14321/goh`.
- Installers download from `github.com/Sanidhya14321/goh` releases.
  No behaviour change.

## 1.0.0-rc.128 — 2026-11-09

- `goh worktree NAME` opens the UI in a separate Git worktree (sibling folder
  `<repo>-NAME` on branch `goh/NAME`), reusing it if it already exists; the
  main checkout is never touched and nothing is removed. `goh worktree list`
  prints `git worktree list`.

## 1.0.0-rc.127 — 2026-11-09

- `goh sessions list --project DIR` (or `--project .`) shows only that
  project's sessions, in every format including `ids`.

## 1.0.0-rc.126 — 2026-11-09

- Shell completion now covers subcommands (`sessions`, `auth`, `mcp`,
  `memory`, `completion`) and session IDs for `resume`, `compact` and the
  `sessions` commands that take one. IDs come from the new
  `goh sessions list --format ids`, which reads only local journals.
  Re-run `eval "$(goh completion zsh)"` (or bash/fish) to pick this up.

## 1.0.0-rc.125 — 2026-11-09

- `goh sessions prune --days N` lists sessions not changed for N days;
  add `--yes` to move them to the trash. Each stays restorable with
  `goh sessions restore ID`, and a session open elsewhere is skipped.

## 1.0.0-rc.124 — 2026-11-09

- `goh stats` adds a tool table (calls and failures per tool, most used
  first, top 15; `--json` has all of them under `tools`), counted from the
  session journals.

## 1.0.0-rc.123 — 2026-11-09

- `goh run` reads piped input without needing `-`:
  `cat build.log | goh run explain this failure`. Only a pipe or a
  redirected file is read (up to 1 MiB); a terminal or `/dev/null` never is,
  and `--prompt` or an explicit `-` keeps the old behaviour.

## 1.0.0-rc.122 — 2026-11-09

- `goh config` prints the effective configuration as JSON: defaults, your
  config file, the ChatGPT default and auto-detected servers and formatters.
  Environment and header values are replaced with `[redacted]`, URL
  credentials and queries are dropped, and secret-looking text is masked.

## 1.0.0-rc.121 — 2026-11-09

- A custom command in your own config can name a subagent:
  `"commands": {"scan": {"template": "Scan $ARGUMENTS", "agent": "explore"}}`.
  The expanded prompt is handed to that subagent like an `@explore`
  mention; the task tool keeps its usual approval. Unknown names are refused
  before anything is sent, and command files in a repository still cannot
  choose an agent.

## 1.0.0-rc.120 — 2026-11-09

- `goh uninstall` lists what it would remove: the running `goh` program
  (and a `harness` link to it), the GoHarness config folder (including saved
  keys and the ChatGPT sign-in) and the state folder (sessions). Nothing is
  removed without `--yes`; `--keep-data` removes only the program. Shell
  configuration and other folders are never touched. On Windows the running
  program is reported for manual deletion.

## 1.0.0-rc.119 — 2026-11-09

- `goh auth list` shows, for the default provider, each profile and
  ChatGPT, where the credential would come from: an environment variable
  (by name), a saved key, the ChatGPT sign-in, or none. It follows the same
  rules as real requests (an `OPENAI_API_KEY` is listed only for the
  official host) and never prints a key or token.

## 1.0.0-rc.118 — 2026-11-09

- Ctrl+Z suspends the terminal UI like other terminal programs; `fg`
  resumes it and redraws. It is the rebindable `suspend` action (set it to
  `"none"` to turn it off) and does nothing on Windows.

## 1.0.0-rc.117 — 2026-11-09

- `goh pr NUMBER` (or `#N`, or a github.com pull request URL) checks the
  pull request out with your own GitHub CLI sign-in and opens the terminal
  UI on that branch. GoHarness never handles a GitHub token, and the UI is
  not opened if the checkout fails.

## 1.0.0-rc.116 — 2026-11-09

- `goh completion bash|zsh|fish` prints a completion script for the
  top-level commands (`eval "$(goh completion zsh)"`, or
  `goh completion fish | source`). Nothing is installed automatically.

## 1.0.0-rc.115 — 2026-11-09

- Opt-in `"formatter_auto": true` adds well-known formatters found on PATH
  (gofmt, rustfmt, ruff, prettier, clang-format, shfmt) for their file
  types. Configured formatters win by name and extension, a disabled one
  still claims its extensions, and nothing from the repository is run.
- Test fix: the ChatGPT-default test isolates Windows home folders, which
  failed RC114's Windows CI on a runner with no sign-in path override.

## 1.0.0-rc.114 — 2026-11-09

- The TUI `/export` dialog writes the RC113 HTML page when the filename ends
  in `.html` or `.htm`, and Markdown otherwise (the default name is still
  `.md`).

## 1.0.0-rc.113 — 2026-11-09

- `goh export --format html` (and `goh sessions export`) writes a
  self-contained HTML page of a conversation, a local, upload-free
  alternative to OpenCode's share links. Text is HTML-escaped,
  recognisable credentials are masked as in Markdown exports, and a
  Content-Security-Policy blocks scripts and remote loads. Light and dark
  styles; `--include-tools` adds tool calls and results.

## 1.0.0-rc.112 — 2026-11-09

- `goh run --session ID --fork` (or `--continue --fork`, also for `goh
  chat`) continues a copy of the session and leaves the original unchanged,
  as OpenCode's `run --fork` does. The fork copies the conversation only:
  no tools are replayed and no approvals carry over.

## 1.0.0-rc.111 — 2026-11-09

- The TUI sidebar shows the estimated cost of this view's replies
  ("~$0.0123 estimated") when the current model has a configured price
  (see RC110). Nothing is shown for unpriced models.

## 1.0.0-rc.110 — 2026-11-09

- Cost estimates in `goh stats`, as OpenCode shows cost: add `"prices":
  {"<model>": {"input": 1.5, "output": 6}}` (USD per million tokens) to the
  config and stats prints an estimate per model and in total. `--json`
  carries `estimated_cost_usd`. GoHarness ships no price table, because
  prices change and differ by account, so unpriced models show tokens only
  with a hint.

## 1.0.0-rc.109 — 2026-11-09

- The grep tool respects `.gitignore` and `.ignore` files (root, nested,
  and those above a scoped path), matching the observed OpenCode grep, which
  leaves gitignored files out. Searching an ignored directory by naming it
  as `path` still works. Glob is unchanged: the reference glob was observed
  to include gitignored files, so GoHarness keeps that behaviour.

## 1.0.0-rc.108 — 2026-11-09

- `@path` mentions of text files now attach the file's contents to the
  prompt, as in OpenCode, so the model can use them without a read tool
  call. Limits: up to 5 files, 64 KiB each and 256 KiB in total, regular
  UTF-8 files inside the project only (read through os.Root, so symlink
  escapes are refused). Images keep their RC50 attachment path. Missing
  paths, directories, binaries, oversized files and agent names such as
  `@general` stay plain mentions.

## 1.0.0-rc.107 — 2026-11-09

- An unknown slash command now suggests the nearest defined one ("did you
  mean /review?"), points to `/commands`, and names the file that would
  define it (`.opencode/commands/<name>.md`).

## 1.0.0-rc.106 — 2026-11-09

- Skill files that cannot be loaded are listed in `/status` with their
  path and reason, beside skipped agent and command files. The transcript
  shows one readable notice per session instead of raw JSON on every run.

## 1.0.0-rc.105 — 2026-11-09

- Provider cooldowns: when a route fails before any response (rate limit,
  timeout, server error or connection failure) and fallback profiles are
  configured, it is skipped for one minute. Later turns go straight to
  the fallbacks instead of waiting on it again. If every fallback fails, the
  cooling route is still tried last. Without fallback profiles nothing
  changes. Each skip is journaled as `provider_failover` with reason
  `cooldown`.

## 1.0.0-rc.104 — 2026-11-08

- Provider retries are now visible. When a request is retried after a
  rate limit, timeout or server error, the TUI status line shows the reason,
  the wait and the attempt ("retrying in 1.5s, attempt 2 of 3"). `goh run`
  prints the same on stderr, and JSON output carries `provider_retry`
  events. Retry behaviour itself is unchanged, and a partially streamed
  reply is still never retried.

## 1.0.0-rc.103 — 2026-11-08

- The session picker (`/sessions`, and `goh chat` with saved sessions) now
  lists damaged sessions as "⚠ damaged … · Enter to recover" instead of
  hiding them. Choosing one leads to the RC102 recovery offer, which asks
  first and keeps the damaged file unchanged.

## 1.0.0-rc.102 — 2026-11-08

- Guided recovery: resuming a damaged session in a terminal (`goh resume
  ID`, `goh chat --session ID`) now offers to recover its readable
  conversation into a new session and opens it. The damaged file is never
  changed. Declining, or any non-interactive use such as `goh run` or piped
  input, keeps the previous error and the `goh sessions salvage` hint.

## 1.0.0-rc.101 — 2026-11-08

- Fixed (found by Windows CI with a real clangd): a language server that
  was slow to publish diagnostics, which is common on cold start, was
  treated as disconnected and stopped, so post-edit error reports and later
  LSP requests failed for the rest of the run. A slow or timed-out
  diagnostics wait now reports "not published yet" and keeps the server
  running. A server whose connection actually closed is still stopped. The
  push-diagnostics wait is now 5 seconds (was 2).

## 1.0.0-rc.100 — 2026-11-08

- Recovery: a language server that crashes or disconnects during a run can
  be started again. The next LSP request asks for a fresh `lsp_start`
  approval (the preview says it is a restart), at most twice per server per
  run. The request that failed is never replayed. Previously a disconnected
  server stayed unusable until a new run.

## 1.0.0-rc.99 — 2026-11-08

- Verification: GoHarness now runs against a real language server in its
  test suite. With clangd installed (as on macOS runners), the test starts it
  behind one approval, receives a real error diagnostic, reports it through
  the post-edit error path the model sees, and resolves a go-to-definition.
  Where clangd is missing, the test is skipped.

## 1.0.0-rc.98 — 2026-11-08

- Opt-in `"lsp_auto": true` adds well-known language servers found on PATH,
  like OpenCode's built-in list: gopls, typescript-language-server, pyright,
  rust-analyzer and clangd. Servers you configure win by name and by file
  extension. Nothing starts until first use, and each server still needs the
  usual `lsp_start` approval.

## 1.0.0-rc.97 — 2026-11-08

- `goh compact ID --focus "..."` steers the summary from the command line,
  like `/compact <focus>` in the TUI.

## 1.0.0-rc.96 — 2026-11-08

- `/compact <focus>` steers the summary, for example `/compact keep the
  migration plan and failing test names`. The focus (up to 500 characters)
  applies to that compaction only; plain `/compact` is unchanged.

## 1.0.0-rc.95 — 2026-11-08

- Recovery: when ChatGPT rejects an access token with HTTP 401 before its
  recorded expiry (for example after a server-side revocation), GoHarness
  renews it once under the shared credential lock and resends the request.
  A 401 means nothing was generated, so no partial answer or tool effect is
  replayed. If another GoHarness process already renewed, its token is used
  without a second renewal. A failed renewal still asks you to reconnect.

## 1.0.0-rc.94 — 2026-11-08

- The TUI sidebar shows the session title above its ID. It is loaded when
  a saved session opens and updates right away after `/rename` or an
  automatic title.

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
