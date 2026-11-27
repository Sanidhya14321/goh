# Changelog

GoHarness releases, newest first. Dates are commit dates.
Each entry lists user-visible changes; evidence for each is in
[VERIFICATION.md](docs/development/VERIFICATION.md).

## Unreleased

**Checks that count** (from the benchmark: 14 of 66 runs got a needless reminder)
- A passing check chained after commands that changed files (`gofmt -w a.go && go test ./...`) now verifies
  those changes; before, the run was reminded to check again.
- A Go test run whose packages all pass from the test cache (`ok pkg (cached)`) counts as a passing test run.
- After the completion reminder, the final reply still says what changed instead of only "tests pass".
- A command that fails because `python` (or `pip`) is missing says that `python3` is installed.
- A project of up to 80 files has its file list sent with the session's context, so a task need not spend
  its first request on `list_files`: 18% fewer input tokens and 12% less time on the Python hard and
  prompt-range tasks, 5% and 6% on the holdout; `hello` costs 49 more tokens.
- Each prompt starts 50–140 ms sooner in a mid-sized project: the walk that lets goh notice files changed
  by commands now runs beside the first model request, and a reply that used no tools skips the closing
  walk. `list_files` with a limit stops walking at the limit (159 → 3 ms in a 6,600-file tree).
- Project walks (the change check after each command, `list_files`, `grep`, `repo_map`) open only the
  ignore files a folder has and open each folder from its parent: the change check 138 → 64 ms in a
  6,600-file tree and 53 → 20 ms in this repository, `repo_map` 241 → 167 ms.
- `grep` over a 6,600-file tree takes 0.21 s instead of 1.73 s: files the walk listed skip a second
  per-folder symlink check, each is opened from its open folder, and several are read at once (results,
  their order and the limits are unchanged).
- `goh-bench` task trees are git repositories with the starting files committed, so `git diff` and
  `git status` work as in a real project (they failed 28 times in earlier runs).

**Benchmarks you can show**
- goh runs as a [Harbor](https://github.com/harbor-framework/harbor) agent in Apple containers
  (`bench/harbor_agent/run.sh -d terminal-bench@2.0`), on this machine's ChatGPT sign-in.
- `goh auth export FILE [45m]` writes the sign-in without its refresh token for a disposable machine: it
  can use the access token but never renew or revoke the sign-in here.
- File tools accept absolute paths inside the project (`/app/main.py`), in patches and their
  `expected_sha256` keys too; Terminal-Bench prompts name files that way, and 38 calls were refused. A path
  outside the project is refused with the project named.
- Checks written as scripts count: an inline script whose body is a here-document
  (`python3 - <<'PY' … PY`) and a `set -e` script of checks. The completion reminder fired in 37 of 50
  Terminal-Bench runs because these were not recognized, costing about two requests each.
- `webfetch` decodes single-byte charsets (Windows-125x, ISO-8859-x, KOI8; from the header or an HTML
  `<meta charset>`) instead of refusing the page, and an HTTP 403 says the site may refuse automated requests.
- A shell command stopped at its own timeout now returns to the model (status `timed_out`, with a hint to
  raise `timeout_seconds` or use `job_start`) instead of ending the run; Terminal-Bench ended a run there.
- A shell command that runs a system package manager (`apt-get`, `dpkg`, `yum`, `apk`, …) gets at least
  300 s, even if the model asked for less: Terminal-Bench showed a 20 s limit stopping `dpkg` midway, which
  left the package database broken and failed the task's own tests.
- `goh run --keep-jobs` leaves background jobs running after the run ends, writing to a log file instead
  of a pipe, for tasks that ask for a service left up (Terminal-Bench `pypi-server`: the server the model
  started was gone when the tests ran). A shell command that leaves processes in the background now says
  they were stopped and to use `job_start`.

**Tests on every save**
- `goh watch` reruns the tests whenever files change and settle. A failure can be fixed on the side (press
  Enter, or `--auto`): your working copy is copied into `<repo>-watch`, a headless run fixes it there, the
  fix is checked with the same tests and saved as a patch, and `a` applies it. Your own files are only
  changed when you apply.

**Several tasks at once**
- `goh parallel TASK... [-- RUN FLAGS]` runs each task headless in its own Git worktree and branch,
  up to 4 at a time, then lists each outcome, changed files, reply and session; `--commit` commits each
  branch, `--file` reads tasks from a file, and `--models A,B` runs each task with several models to
  compare them.

**Dollar budgets**
- `max_run_cost` (`--max-run-cost`) stops a run once its estimated spend reaches a USD amount, priced from
  `prices` per model (child tasks included; budget notes report it). `goh stats` prices dated model names
  from their base entry.

**Web search without a search service**
- `"websearch": "provider"` lets `websearch` search through the model provider's hosted search (the
  ChatGPT sign-in, OpenAI Responses, Anthropic), in a request of its own: a short answer and the URLs it
  cites. Before, web search needed a self-hosted SearXNG endpoint.

**Scraping with webfetch**
- `webfetch` takes `selector` (a CSS subset: tag, `#id`, `.class`, `[attr=value]`, descendant, `>`,
  commas) to return only the matching parts of an HTML page, and `links: true` to list its links as
  absolute URLs.

**MCP**
- `goh mcp add NAME -- COMMAND ARGS…` and `goh mcp add NAME --url URL [--header H=ENV] [--sse]` add a
  server to the user settings; `goh mcp remove NAME` removes it.
- A local server that changes its tools mid-run has them listed again, and the next request offers the
  new list (the list is rebuilt only on a change, so the prompt cache is kept).
- A server can ask you for input during a call (form elicitation): its fields appear as chat questions
  and the answers go back to it, typed as its form asks. Headless runs do not offer this.
- Remote servers on the older HTTP+SSE transport work with `"transport": "sse"`; requests stay on the
  server's origin.

**Research round after Terminal-Bench** (list and reasons in [design decisions](docs/development/DESIGN_DECISIONS.md#after-terminal-bench-20-november-2026))
- A run capped by `max_run_seconds`, `max_tool_calls` or `max_run_tokens` tells the model what is left when a
  budget passes 50%, 75% and 90% (one line on the newest tool result; uncapped runs are unchanged). The
  Harbor agent passes each task's time limit, less 30 s.
- `job_output` takes `wait_seconds` (up to 600) and `until` (a regular expression): it returns when the job
  exits or its output matches, instead of the model polling or writing sleep loops (105 `job_output` calls
  for 69 jobs and 39 sleep loops on Terminal-Bench).
- New experiment `task-recitation` (off): before a long run finishes, the task is quoted back once to check
  each explicit requirement. The Harbor agent takes `--ak experiment=NAME[,NAME]`.
- Opening a saved session reads its journal in one buffer and decodes records in the form goh writes
  without a decoder per line (others still go through the strict decoder): a 20,000-record journal opens
  in 9.6 ms and 23 MB instead of 37 ms and 79 MB, for resume, attach and the session pickers.

**Terminal: clearer and faster**
- Your prompts sit in a filled band with a bar, so turns stand apart; every reply starts its own paragraph.
- Tool calls read as what they do (`▸ Read  a.go`, `▸ Shell  go test ./...`). A read is one line
  (`a.go · 120 lines`); command output shows its last lines with a count of the rest (`/details` shows all).
- A finished edit, write or patch shows the lines it changed on green and red washes, live and when resumed.
- Inline code and bold render in color and weight; code blocks get a gutter instead of a frame (2 rows saved).
- The status line (spinner, time, steps) sits just above the input; the session header is gone (2 rows saved).
  The footer shows context use instead of repeating the model.
- Approvals are a panel: "Allow Edit src/a.go?" with the keys highlighted and the diff on washes.
- The default theme asks the terminal for its background and tints panels and diffs from it; nothing waits
  for the answer.
- Rendering is faster: a redraw with nothing new 136 → 30 µs, after new text 414 → 279 µs, a streamed
  2,000-line code block 3.6 → 2.6 ms (ASCII and common symbols skip grapheme segmentation; the project
  folder is no longer resolved on every frame).

**Long sessions**
- Reopening a chat reads only the messages it shows (newest first, within the display budget) instead of
  copying the whole journal: 7.0 ms and 31 MB → 0.24 ms and 0.5 MB at 20,000 records, the same at 1,000.
- A repeated prompt in a reopened chat shows its own time, not the time of an earlier identical prompt.
- Returning from a picker or settings checks the journal by its length instead of copying it.
- Resuming a session whose calls all finished no longer scans the journal (2.9 ms, 10 MB → 4 µs, 11 KB).
- An ordinary export over 100,000 records is refused instead of writing an archive import rejects.

**MCP**
- Remote MCP servers that require OAuth: `goh mcp login NAME` signs in through the browser (discovery,
  dynamic client registration or a configured `oauth_client_id`, authorization code with PKCE); runs use
  and refresh the saved token. `goh mcp logout NAME` forgets it; `goh mcp list` shows who is signed in.
  Tokens are kept per server URL in the credential store and move with `goh auth store`.
- In the chat: `/mcp login NAME` and `/mcp logout NAME`; a server stopped for lack of sign-in points there.

**Measuring where a run's time goes**
- `GOH_PHASES=1` adds one `run_phases` event before `run_finished`: setup, request assembly and size,
  compaction, each provider attempt (start, first text, end, retry wait, route), time per tool and their
  union, approval and question waits, and the finish steps (verification, cleanup, title, journal, snapshot,
  hooks). Text mode prints it as one line. Off by default; nothing is journaled.
- `goh-bench`: a run the model never answered (a provider usage limit, say) is `no_model_reply` and not
  scored; a host-graded run whose test command (`go`, `python3`) is not on PATH stops before any model request.

## 1.1.0 — 2026-11-23

Fewer tokens and faster runs for every kind of prompt, from `hello` to debugging a 6,000-line file,
measured on a new prompt-range benchmark: 18% fewer input tokens and 19% less time per task, hard bug
fixes 37k → 20k tokens, nothing solved before left unsolved ([results](docs/BENCHMARKS.md)).

**Faster and cheaper**
- Tool results reach the model as plain text instead of JSON escaped twice (14% fewer input tokens per request).
- Fewer tools are offered with every request (21 → 17 in a default setup) and their descriptions are shorter.
  `search`, `create_file` and the snapshot tools stay callable but are not offered; `skill` only when skills exist.
- `go test`, npm and pip work in the Auto-access sandbox: their caches go to a private temporary folder, so the
  first test run no longer fails and gets retried.
- Reading a whole file over 64 KB returns its length and how to read it instead of an error; a line range that
  is too large returns the lines that fit. `total_lines` no longer counts a final newline as an extra line.

**Terminal**
- Each reply ends with a dim line: model, time, tokens in and out. The sidebar splits tokens in and out under a
  context bar.
- The running timer counts in tenths of a second, shown once.
- Unverified changes show in amber; approval key hints never split a key from its label; an approval review
  restarts from the top after a resize.
- A fetched page is one line in chat; `/details` cycles through output, page previews, or one line per result.
- Replies render cleanly: links, emphasis, word wrapping, and code blocks closed with their message.
- The chat screen is kept while `/models`, `/agents`, `/settings` or `/providers` is open; model lists are
  cached for five minutes; HTTP 429 shows as "Rate limited".
- The provider you selected stays active when two saved providers share a route.

**Correctness**
- Compaction summaries no longer carry project-note text, and a failed compaction reports its real cause.
- A fallback route no longer changes the primary's window or token estimate; usage from discarded attempts
  counts toward `max_run_tokens` before any retry.
- Older images are not resent when they would push a request over the 16 MiB limit.
- A check whose failure the shell swallowed (`|| true`, `; echo done`, `| tail`) no longer counts as verified.
- `webfetch` converts a page's main content and keeps an excerpted page whole as an artifact.
- Markdown export masks multiline secrets; names starting with two dots (`..config`) are inside their folder;
  text is never cut mid-character; the ChatGPT sign-in wait can be cancelled; hook feedback is kept.

**Benchmarks**
- `goh-bench` grades answers as well as tests and reports results by prompt category and size; seven
  prompt-range tasks and `bench/tools/compare.py` are added. All results are in `docs/BENCHMARKS.md`.

## 1.0.0 — 2026-11-21

First stable release: everything since rc.180 below.

- **All 55 benchmark tasks are graded by test identity:** the 19 container tasks pass the stricter dry run (with Docker) and carry recorded fail-to-pass and pass-to-pass tests. The dry run no longer rejects a task whose failing Go test panics or crashes its test binary on the starting tree; tests that never ran there are left out of the pass-to-pass identities and stay covered by the aggregate check.
- **Follow-up review fixes (reviewed at `1d0e29b`):**
  - A command that only *mentions* a test runner (`echo "go test"`) is no longer a check, and a test run that reports zero tests does not verify anything. The configured verifier is now a journaled effect: if it is interrupted or its cleanup is uncertain, the run stops and resuming needs `--acknowledge-uncertain`. A verifier that rewrites files is run once more; if it keeps changing them, the outcome is `unverified`.
  - Automatic shell access asks when the boundary cannot be enforced: with `sandbox.container` set (below Full access), or when the project is inside a writable temporary folder.
  - `/find` no longer crashes on Unicode history whose case changes byte length, and keeps late matches in long lines.
  - Snapshot lock waits are cancellable and a live holder is never taken over; opening a session settles an interrupted trash or restore first; `goh resume ID --session OTHER` is refused; the token allowance is checked before automatic compaction (`run_finished` adds `usage_missing_calls`).
  - A subagent's retry removes only its own text; held prompts survive leaving the chat for a picker or settings.
  - `/providers` Test connection is cancellable; `--endpoint`/`--base-url`/`--protocol` chats keep `/models` choices to themselves; `goh attach --continue` picks the latest activity (new `GET /v1/sessions/latest`); an answer the attached server rejects is asked again.
  - An MCP restart with uncertain cleanup stays pending (the run stops) and the call is revalidated against the restarted server's tools.
  - `goh-bench`: zero tests, skips, expected failures and ambiguous identities are not passes; `--dry-run --record-identities` stores test identities (all 36 native tasks now graded by identity); grading starts from the baseline plus the candidate's non-test changes; host mode runs the agent with `--access auto` and no shell grant; retrieval credits only successful reads; the reproducer audit scores only a newly added test; token totals include subagents and auxiliary calls.
  - Research: `submit_evidence` runs serially (one record per revision); `recall_observation` ids are bound to the exact recorded result and pages are lossless UTF-8.
  - CI: a race-instrumented attach job and an offline benchmark dry-run job. README lists exit code `6`.
- **Verification results tell the truth:** only a recognized check (a test runner, a build, lint or type check, an explicit assertion such as `test`/`diff`/`grep -q`, or running the program) counts as verifying a change; `pwd`, `ls` or `git status` after an edit used to report `verified`. `run_finished` adds `verification_detail` (change tracking `known`/`unknown`, and the evidence: source, command, kind, scope, exit code, revision) and `run_id`. When the project cannot be fingerprinted, the configured `verify` still runs after commands that may write. The verifier journals `verification_started` before it runs.
- **A failed `verify` is visible and fails automation:** `goh run` exits **6** when the configured check still fails after its repair attempts, and text mode prints `Verification failed: COMMAND (exit N)`. Chat shows the failure persistently (in red and in words) and **holds queued prompts** in order instead of sending them; Enter on an empty composer sends the next, ↑ takes the newest back. The same happens after a cancel, failure or a needed approval. Reopened chats and the session picker show the last run's result.
- **Each operation reports only itself:** a typed `!command` reports its own outcome (a write is `unverified`, not `no_changes`), never inherits an earlier failure, and never triggers auto-title. Titles read only what you typed, not `@file` contents. `run_finished` hooks get the outcome and run before the final event.
- **Access is enforced, not assumed:** commands allowed because they only read run in a sandbox where nothing in the project is writable (they ask where there is no sandbox). Under Auto, subfolders saved with a stricter level stay read-only to commands; on Linux, where Landlock cannot restrict a folder inside a writable one, such commands ask. A session-only level now belongs to its session: a new session or workspace starts from the saved levels.
- **Attach:** the screen no longer races the background reload (found by the race detector); a cancelled run is shown as cancelled even when a late approval reply is refused; approvals offer only the once/deny the server accepts; the access indicator says the server's levels apply; a failed `/new` keeps the session and draft; the picker's new-session row works.
- **Sessions and workspaces:** `goh resume ID --agent X` and `goh compact ID` use the session's own project (as `--session` did); a conflicting `--project` is refused. Choosing a session from another project opens it there; a missing, locked, damaged or uncertain session gets its own recovery screen instead of "add a provider".
- **Runtime fixes:** a first request under the hard limit is no longer refused because of the soft compaction trigger; repeating `/compact` on a short chat is a no-op; failed fallback routes cool down like the primary; a failed MCP start whose process cannot be confirmed stopped stays an uncertain effect; `replace_all` checks the result size before allocating; snapshot capture and restore use private Git indexes and serialized restores; LSP keeps Windows drive and UNC paths and follows incremental sync; salvage never reads an oversized corrupt line whole; interrupted trash/restore moves are reconciled; `repo_map` honours `.gitignore` and reads query-named files first.
- **Chat and composer:** Backspace and ←/→ move by grapheme (👍🏽, accents, flags); ↑/↓ follow wrapped rows before history; free-text questions take typing at once; a subagent's retry removes only its own abandoned text; `/find` also lists earlier matching turns from the journal; long shell output names its retained artifact; the header and `/status` name the model that actually served a turn after a fallback; the compact layout leads with workspace, local/attached scope and access; key hints follow your bindings; model lists load behind a screen where Esc cancels; `/models` in a `--config` chat stays with that chat; tiny pickers show the highlighted option; `goh chat --help` prints usage first; standalone `goh settings` says "Exit settings". ACP tells the editor which question needs an answer and when verification failed.
- **Benchmark integrity (`goh-bench`):** grades by observed test results (`go test -json`, `unittest -v`, `pytest -rA`) with distinct `test_failure`, `build_failure`, `no_tests`, `timeout` and `tool_error`; the dry run requires a test failure (not a build error) on the starting tree and checks pass-to-pass tests on both revisions, which exposed two hard tasks whose starting tree did not compile (fixed). Grading restores the pre-run test files and compares protected files with the real baseline; arms are parsed without shell evaluation and interleaved; every run gets a fresh folder and a manifest; the summary separates resolution, reliability and false verification and counts auxiliary and child tokens.
- **Research prototypes (opt-in):** `--experiment submission-evidence`, `recoverable-compression` and `retrieval-expansion`, plus benchmark-only retrieval telemetry and a reproducer audit. See [experiments](docs/BENCHMARKS.md); no live comparison has been run yet.
- **Access levels:** Ask (default), Auto-edit, Auto and Full access say how much goh does without asking. Shift+Tab cycles the level for the session (also mid-run); `/access` saves one for the project or for a single folder inside it, so one folder can have full access while the rest still asks; `--access` sets it for `goh run`/`goh chat`. Auto-edit makes edits in the folder and runs commands that only read inside the project; Auto also runs every command from the folder inside the OS sandbox, writing only there and in temporary folders (where there is no sandbox, as on Windows, commands still ask); Full access asks for nothing. `a` in an approval allows it and raises the level so similar requests stop asking. Ask/deny rules and the loop guard always win. Shift+Tab used to repeat Tab (previous agent).
- **Live status line:** instead of "Working", a run shows what it is doing: waiting for the model, thinking, "Writing web/index.html · 1.8k tokens (42/s)" while a tool call streams in (all three provider formats), "Running npm test · 12s", plus the step and tool counts.
- **Faster turns in large projects:** goh fingerprints the project at the start of each run and after every edit and command, to notice changes. It ignored the project root's `.gitignore`, so in a repository with large ignored folders each fingerprint walked them (1.07 s in goh's own tree, 74k files) and then gave up at its 20,000-file limit, which also left change detection off. The root rules now apply (63 ms, and detection works), and a project that is really too large is walked once per run instead of after every step. In this repository a "hi" run went from 7.0 s to 3.1 s end to end.
- **Prompt cache hits:** Responses requests carry the session in a `session_id` header as well as `prompt_cache_key`. Over the ChatGPT sign-in, follow-up turns hit the cache in 5 of 9 cases instead of 1 of 9 (up to 21.8k of 22.2k prompt tokens cached), which is cheaper and faster on long sessions.
- **Sandbox:** a session folder reached through a symlink (`/var` → `/private/var`) could hide the project's own parent from sandboxed commands; paths are now compared resolved.
- **Overloaded providers are retried:** OpenAI's in-stream `server_is_overloaded` (and `server_error`, Anthropic's `overloaded_error` and `api_error`) now resends the turn after 5, 10 and 20 s, shown as "Provider busy (overloaded); retrying…". It used to end the run; this cut off 5 of 19 runs in a live benchmark. Usage limits and request errors still end the run at once. The Anthropic stream error now names its type.
- **Connecting from a `--config` chat works:** a chat started with `--config FILE` reads only that file, so a provider added with `/connect` (or switched to in `/providers`) was saved but never reached it, and the chat stayed on "waiting for setup". The launcher now passes the new provider's route to that chat; its key is found by route as before. Found by the live ChatGPT terminal check.
- **`goh auth refresh`** renews the ChatGPT sign-in at once and says plainly if it was revoked; `goh auth status` shows when the access token expires. `status` used to report "saved locally" for a sign-in that could no longer be renewed.
- goh names itself `goh` with its real version to language servers, MCP servers and editors (ACP), and in the web-fetch User-Agent. It used to send an old development name with version 0.1.0. Saved settings and session files keep their existing names.
- **Completion check covers name-filtered tests:** when the only tests that passed after a change were picked by name (`go test -run`, `pytest -k`/`::`, `jest -t`, `cargo test NAME` and similar), the run is asked once to run the full tests of what it changed. In the rc.162 real-project benchmark every miss was reported verified: the model's own checks passed while the fix was incomplete or broke an existing test in the same package. Measured on the same 19 tasks with `gpt-6.2-sol`: no change, because that model already runs the full tests after a filtered run ([report](docs/BENCHMARKS.md)).
- **Package managers:** Homebrew (`brew tap sanidhya14321/goh https://github.com/Sanidhya14321/goh`) and Scoop (`scoop bucket add goh https://github.com/Sanidhya14321/goh`) install stable releases; the formula and manifest are generated from each release's checksums. `goh upgrade` on such a copy names `brew upgrade goh` or `scoop update goh` instead of replacing it. Releases also carry a CycloneDX SBOM and GitHub build provenance. Binaries stay unsigned; the README says what that means for browser downloads.
- **LSP:** a language server that exits before starting now says why. goh adds the server's last two error lines (for example rustup's "Unknown binary 'rust-analyzer'") instead of only "connection closed: EOF". gopls, pyright and typescript-language-server were verified live.
- **One terminal program per session:** opening providers, connect, models, agents, settings, the project or session picker, starting a new chat, forking, undo/redo/revert, plan approval and Tab agent switching no longer leave and re-enter the screen and restart the chat program. Every screen is shown inside one program, so there is no flash between screens, keys typed meanwhile go to the next screen, and dialogs are drawn over the live chat, which reflows when the terminal is resized (it used to be a frozen snapshot). `goh setup`, `goh settings` and `goh providers` work the same way. A damaged session's recovery question still appears on the normal terminal.
- **Saved keys stay outside the project, also through symlinks:** the check that refuses a project folder holding goh's saved keys compared a symlink-resolved project with an unresolved key folder, so it was skipped when the key folder did not exist yet or the home path ran through a symlink (macOS `/var` → `/private/var`). It now resolves the nearest existing parent.
- **Windows:** "Ollama is not running at …" (and other local providers) is shown when nothing is listening; Winsock's refused-connection error was not recognised and a raw `connectex` error appeared instead.
- **CI:** a `vulncheck` job runs `govulncheck` (also `make vulncheck`); the Linux terminal suite keeps its private home outside the test project; a Windows test isolates `USERPROFILE`.
- **End-to-end audit fixes** (real-world prompts against live Groq, real websites, clangd and an MCP server):
  - **Rate limits no longer end long runs:** a 429 now waits as long as `Retry-After` asks (decimals accepted, up to 30 s), or 2 s then 4 s without one. It used to wait at most 5 s (100 ms without the header), so a Groq run failed partway through once its per-minute token budget ran out.
  - **`webfetch` reads JSON APIs:** its Accept header now allows JSON and any other type, so `api.github.com` and similar APIs no longer answer 415. Errors name the status (`HTTP 404 Not Found`), and a refused cross-origin redirect names its destination (query stripped) so the model can fetch it as a separately approved request.
  - **`lsp`:** `documentSymbol`, `workspaceSymbol` and `diagnostics` no longer require a line and column; the position operations say which of them is missing.
  - **Patch approvals show diffs:** `apply_patch` previews each file as a unified diff instead of both whole versions, where a change deep in a long file could fall past the excerpt and never be shown.
  - **Shell output keeps its end:** a long command's inline preview has its first and last 4 KiB with a marker naming the omitted bytes and the artifact to read, instead of only the start (where test summaries and compiler errors are not).
  - **Terminal UI:** tool output is shown as written, not as Markdown (a Python `# comment` was drawn as a heading and `**kwargs` lost its asterisks); grep, glob and file listings show as lines instead of JSON; webfetch, LSP, skill, patch and todo calls get readable one-line labels; subagent activity names the tool it ran; the reply after tool output starts its own paragraph.
  - **`goh run` text output:** one `▸ tool  args` line per call (including calls refused before they start) and an `[error]` line per failure on stderr. "Approval required" with the full preview is no longer printed for tools already granted with `--allow`; when a run stops for permission, it shows the request and the `--allow` flag that grants it.
  - **Configured MCP servers no longer block `goh run`:** a headless run used to end with `needs_permission` before its first request when any configured server lacked `--allow mcp_start:NAME`, even if the task never used it. Such servers are now left out with a notice naming the flag. Denying a start when asked (chat, ACP) still ends the turn.
  - **Rejected tool calls are retried:** when a provider refuses a reply mid-stream because the model produced an invalid tool call (Groq's `tool_use_failed`), the request is sent again up to twice (nothing ran), and the notice says so instead of "Stream broke". Other in-stream errors now show their code and type (never the message).
  - **gpt-oss can patch files:** `apply_patch` accepts `patch` as another name for `patchText`. gpt-oss models use that name, and Groq checks calls against the schema, so every patch they wrote was refused before goh saw it.
  - A connection that drops or times out mid-reply is reported as an interrupted stream (re-sent with `--retry-interrupted`), not as a raw `read tcp …: operation timed out` socket error.
  - **Small per-request caps work:** a provider that refuses a request as too large and states its cap (Groq's free tier: "Limit 8000, Requested 30042"; HTTP 413, or a 429 whose request exceeds the limit) no longer ends the run. goh takes the cap as the window, shrinks the output reserve and the context to fit, and resends the turn, with a notice. Only the two numbers are read from the body. A 429 that resets in more than two minutes (a daily quota) fails at once with the reset time instead of three short retries.
  - **Token estimates follow the provider's counts:** goh counted one token per byte, about 4–5 times too many for its own instructions and tools (9.7 KB, which Groq counts as about 1,960 tokens). The smallest request the provider has counted now anchors that fixed part, and later content counts at the ratio the provider reported for it. Summary requests carry no tools, so they don't use the anchor.
  - **Large fetched pages stay reachable:** a page over 16 KiB is also saved as an artifact. `read_artifact` gains `pattern` (matching lines with byte offsets), so a model on a small window can search the page instead of seeing only its navigation. Oversized results in the newest turns are cut to an excerpt in every compaction path, older excerpts become receipts, and `grep`/`search`/`read_file` on an artifact ID say to use `read_artifact`.
  - An expired or revoked ChatGPT sign-in says so and names `goh auth login` instead of "HTTP 400; reconnect with /connect".
  - `goh run --endpoint …/chat/completions` (or `/messages`, `/responses`) with ChatGPT as the active provider uses the protocol the URL implies instead of failing with "ChatGPT authorization requires the official OpenAI Responses URL". `make acceptance` runs with a private home folder and passes on machines where ChatGPT is active.
  - `goh doctor` no longer calls a ChatGPT setup invalid for lacking a context window (runs read it from the service).
  - `goh export --help` prints usage instead of exporting; `goh export --project DIR` picks that project's latest session.
  - `repo_map` reports the languages it read (`"languages": ["python"]`) instead of always `"language": "go"`.

- **Project browser at start:** `goh` shows where it was started, with that folder, its parent, recent projects and its subfolders (tagged git, Go, Node, Python…). Enter opens a folder, → looks inside, ← goes up, and typing filters or completes a path like `~/code/ap`. The first time a folder is opened goh asks to trust it (it and its subfolders then open directly; trust grants no tool permissions). The home folder is explained and skipped instead of blocking chat. `/workspace` and `/settings` → Workspace use the same browser instead of a text field. `goh workspaces` lists trusted folders; `goh workspaces forget PATH` removes one. `--project`, `--session` and `--continue` skip the browser.
- Selected rows in pickers use dark text on the accent color in every theme.
- **Starting in the home folder no longer looks like a broken provider:** goh refuses the home folder as a project (its saved keys live there), but chat said "No model connected" and Enter opened provider setup, so adding a key and picking a model led back to the same screen. Chat now says "Chat can't open this folder", and Enter opens the project-folder picker, which accepts `~/…` paths and saves on Enter. The CLI error says the same.
- **`/models` saves the choice:** picking a model stores it on the provider, so it survives the next chat and restart (it used to apply to one chat only, with "set model in config to keep it"). The list shows the provider's chat models and the other saved providers to switch to; the confusing "Default profile" and raw profile rows are gone. Groq's speech and safety-classifier models (Whisper, Orpheus, Prompt Guard) are hidden there and in `goh models`.
- **Terminal colors by default:** the new default `terminal` theme draws on the terminal's own background and text color instead of painting near-black cells, which left the window margins, the first frame and the gap between screens in a different color. `dark`, `light` and `high-contrast` remain. Screens send about half the bytes.
- Single-field forms (the workspace folder) save on Enter; in longer forms Enter on the last field saves. `goh --version` and `goh -v` print the version.
- Terminal test suites run with a private home folder, so they never read or change the user's settings.
- `make install` replaces the installed binary instead of overwriting it in place, which made macOS kill the new `goh` on launch.
- **Fast typing survives a change of screen:** keys typed while one screen closes and the next opens (Tab to switch agent, Enter on a picker, a slash command) used to be dropped; they are now replayed into the next screen in order, so `/agents⏎build⏎hi` typed at once works.
- While chat waits for setup, Enter on an empty composer opens the provider picker. The setup text is shorter and wraps at word boundaries on narrow terminals, and narrow hint rows keep Ctrl+D exit instead of wrapping mid-key.
- **Docs:** the changelog and verification log keep current releases only; rc.1 – rc.179 detail moved to `docs/development/history/` with a short milestone summary here. The origin ledger is a summary table, with the early per-release notes archived. `docs/DEPENDENCIES.md` lists all seven direct dependencies.
- **Clear CLI errors:** with no provider, `goh run`, `resume`, `models` and `doctor` say "no model provider is connected yet" and how to connect one, instead of an internal URL error. A missing `--project` folder, an unknown session ID and `goh run` with no message each get a one-line explanation. `goh chat` without a terminal (cron, CI) says to use `goh run`. `goh worktree list` outside a Git repository says so instead of printing git's failure.
- `goh resume` accepts flags before the session ID, and the message as words (`goh resume ID add a test`) like `goh run`.
- `goh --help` groups commands under Work, Sessions and changes, Providers and settings, Integrations and Maintenance.
- `/settings` matches the other pickers: aligned names, descriptions and shortcut keys, with "Back to chat" last. The "Run a GoHarness command" item (CLI arguments typed as a JSON array) was removed; run commands from the shell.
- **Repository layout:** `CHANGELOG.md` moved to the root; documentation is indexed in `docs/README.md`, with benchmark reports in `docs/benchmarks/` and development records (status, testing, design decisions, verification log) in `docs/development/`. The progress log, end-to-end audit, component plan and review tracker were folded into a short `STATUS.md` and removed. Terminal test suites moved to `scripts/e2e/` (`make e2e` runs them); OpenCode comparison capture scripts moved out of the product tree. `make install` and `make clean` were added.
- Removed dead code: an unused no-follow open helper, snapshot `Read` and `Abort`, and `projectsnap.Store.Project`.
- **First run:** a fresh install opens a welcome picker of providers instead of a blocked chat with an internal configuration error. "Not now" leads to chat, where typing a message and pressing Enter opens the picker and keeps the message as the draft. Esc on a provider's key form or model list returns to the picker, and a failed step (a rejected key, Ollama not running) is explained there so it can be retried. When the key in use is rejected, a message opens `/providers` instead of doing nothing.
- **Approvals keep the conversation visible:** the approval preview and questions dock above the composer, with the conversation above them, instead of replacing the transcript.
- **Edit previews are real diffs:** an approval shows the changed lines with three lines of context and `@@` line numbers, not the whole old and new file. Long lines are cut individually, removals and additions have separate budgets, and a file's final newline is no longer shown as an empty changed line.
- Pickers size to their content so descriptions are not cut mid-word; key-hint rows drop whole hints on narrow terminals; truncated text ends in "…"; empty sessions preview as "No messages yet."
- **Security:** a "for this session" shell approval of a command that runs code given as an argument (`bash -c`, `node -e`, `python3 script.py`, `find -exec`/`-delete`, `go test -exec`, `git -c`, quoted or `$`-expanded text) now covers only that exact command. On macOS the sandbox keeps `.git/hooks` and `.git/config` read-only, so a sandboxed command cannot plant a hook that git later runs unsandboxed. `shell_rules` deny rules also match `/bin/rm`, `command rm`, `FOO=1 rm`, `sudo`/`env` prefixes and `sh -c '…'` scripts. On Windows, `api_key_env` names compare without case, so `openai_api_key` cannot send the official key to a gateway.
- **Runtime:** an MCP server restarted during a call no longer leaves the session needing `--acknowledge-uncertain`; a malformed tool call batched behind a read-only call is refused instead of run with `{}`; an empty model reply is no longer saved to history (it broke later Anthropic requests); the verify repair prompt shows the end of a long check output, where failures are; prompts that start with an absolute path (`/var/log/app.log shows…`) are sent instead of rejected as unknown commands.
- **Providers:** Anthropic requests accept tool-call ids from other providers (repeated across turns or with other characters) by remapping them; streamed tool calls from servers that repeat the id and name in every chunk are no longer doubled. `goh auth logout` clears the local sign-in even when OpenAI cannot be reached. Two goh processes renewing ChatGPT at once now wait for each other instead of failing.
- Provider manager: editing a custom provider's URL with the key left blank keeps (moves) its saved key; a provider whose key comes from an unset variable can be saved with a typed model; a keychain error is reported instead of being read as "needs an API key"; settings are written before the key so a failed write leaves no orphan secret; editing an inactive provider no longer resets the chat's model choice; renaming the provider named by `HARNESS_PROFILE` warns.
- Linux sandbox: `/`, `/home` and `$HOME` can be listed (the protected folders' contents stay unreadable). Windows: `glob` and `grep` `include` work under nested folders. `job_start` says it is sandboxed when it is. Stale MCP resource and prompt sessions are dropped when a server restarts.
- **Multiple providers, and a provider manager:** every connected provider, ChatGPT included, is saved under a name with its own key. Before, each setup replaced the last, and a ChatGPT choice lasted only for that chat. `/providers` (also `p` on the status screen, `goh setup` and `goh providers`) lists them with model, host and key state. Enter switches new chats to a provider; `m` changes its model from the provider's list, `k` replaces the key after checking it (or reauthorizes or switches ChatGPT), `t` tests the connection, `e` edits a custom provider, and `d` twice removes one with its saved key. `goh providers list|use|remove` does the same without the UI. A provider configured by hand at the top level of the settings appears as `current` and gets a name the first time it is changed.
- **Custom providers:** one form with name, URL, protocol (a ←/→ selector with an explanation of the selected protocol), key, model, context window and key variable. The old "URL type" field is gone: a URL ending in `/chat/completions`, `/responses` or `/messages` is used exactly. Ctrl+S checks the URL and key against the model list before saving. A blank model is picked from that list, and a provider without one can still be saved with a typed model.
- **`/connect`** goes straight to adding a provider; `/providers` manages them. Error hints now point to `/providers` (k to replace a rejected key, t to test).
- Settings forms gained choice fields and per-field help, wrapped at word boundaries.
- Provider manager fixes: renaming a custom provider in `e` now keeps its other profile settings (`output_budget`, `output_parameter`) and renames it in agents, `model_roles`, `fallback_profiles` and `profile`; before, the settings were dropped and a referenced provider was silently left behind as a duplicate. Removing a provider no longer deletes a saved key that another provider still uses through the same endpoint spelled differently (trailing slash, or full `…/chat/completions` URL). The custom form rejects an invalid name before checking the connection instead of failing after the model is picked. A provider that refuses a request sent without any key now says it needs one, not that "this key" was rejected. `e` on a hand-configured ChatGPT entry no longer opens the custom form.
- **Use a different ChatGPT account** (`/connect`, or `goh auth switch`): signs in to another ChatGPT account or workspace with a new registration and host identity. Before this, reauthorizing reused the first registration, which OpenAI limits to the workspace it was made in, so another account failed with `3p_login_workspace_scope_denied`. The old sign-in is replaced, and its renewal token revoked, only after the new sign-in succeeds.
- ChatGPT sign-in says which account a reauthorization needs, tells you to press Ctrl+C if OpenAI shows an error page, and when cancelled names `goh auth switch` as the way out.
- **`/connect` provider catalog:** Anthropic (Claude API), OpenAI API, OpenRouter, Google Gemini, Groq and Ollama join Continue with ChatGPT. Choosing one opens its API key page, takes the key, and checks it with the provider before saving. Models are then listed for that key, without non-chat models, and context windows are taken from the list when reported. The manual form remains for other URLs.
- **Key checks:** 401 and 403 responses, Google's 400 "Please pass a valid API key", and OpenRouter's `/key` endpoint (its model list is public) all report "did not accept this key" instead of a raw HTTP status. A key is sent only to its provider's own host, with no redirects.
- `GOH_NO_BROWSER=1`, or an SSH session, shows sign-in and key links instead of opening a browser. On Windows the browser is now opened too.
- Claude Pro and Max subscriptions are not offered: Anthropic does not permit subscription sign-in in third-party tools.
- **Docs:** the README is now a short overview (features, install, quick start, interfaces, benchmarks, build); the full reference moved to [USER_GUIDE.md](docs/USER_GUIDE.md), organized by topic.

## 1.0.0-rc.180 — 2026-11-12

First published release.

- **`goh upgrade`:** installs the latest release over the running `goh`. It checks the binary's single entry in `SHA256SUMS` and runs the new binary before replacing anything; any failure leaves the installation unchanged. `--check` only looks, and `--version TAG` picks a release. Downloads use HTTPS, or plain HTTP only to a loopback address. On Windows the previous executable is kept as `goh.exe.old`.
- **Release workflow:** pushing a `v*` tag runs vet and the tests, builds all six platforms with `make release`, and publishes the release. The binaries, `SHA256SUMS`, both installers, the notices and `licenses.tar.gz` are attached. The notes come from this changelog.
- The repository history no longer contains the 19 MB `harness` build output committed in rc.170 and removed in rc.174.

## Before the first published release (rc.1 – rc.179, 5–12 November 2026)

None of these were published; `v1.0.0-rc.180` is the first release. In outline:

| Releases | Milestones |
| --- | --- |
| rc.1 – rc.30 | Agent loop, durable journal, file and shell tools, approvals, terminal UI, ChatGPT sign-in, provider failover, undo and redo, compaction masking, damaged-session recovery, native CI and installers for macOS, Linux and Windows |
| rc.31 – rc.100 | Session management and export, queued prompts, background jobs, shell and file rules, loop guard, image input, reasoning effort, remote MCP, LSP, web search, plan handoff, usage statistics, live ChatGPT verification |
| rc.101 – rc.135 | Guided salvage, cost estimates, file mentions, shell completion, `goh pr`/`worktree`/`config`/`uninstall`, hooks, whole-project snapshots, shell sandbox, MCP reconnect, Anthropic prompt caching |
| rc.136 – rc.162 | External review fixes, run outcomes, scoped grants, verify/repair, shared budgets, completion check, repository map, benchmark runner and the live benchmark runs |
| rc.163 – rc.179 | TUI audits, two more external reviews, `goh attach`, Go 1.27.4, the OS credential store |

Every entry in full: [changelog for rc.1 – rc.179](https://github.com/Sanidhya14321/goh/blob/v1.0.0/docs/development/history/CHANGELOG-rc1-rc179.md).
