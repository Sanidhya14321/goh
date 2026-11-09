# GoHarness Gap Analysis

*9 November 2026 · version `1.0.0-rc.129` · compared against the architecture in [ARCHITECTURE.md](ARCHITECTURE.md)*

This note lists every gap, loophole and missed point found when comparing GoHarness with other
open-source coding harnesses. It is a working list for deciding what to build next. Nothing here has
been fixed yet.

## How this was checked

| Harness | Language | How compared |
| --- | --- | --- |
| **OpenCode** v1.19.5 | TypeScript | Full pinned source checkout plus runtime captures (`research/reference`, `research/FEATURE_BACKLOG.md`) |
| **Crush** (Charm) | Go | Pinned source checkout, `research/comparisons/crush` |
| **Goose** (Block / AAIF) | Rust | Pinned source checkout, `research/comparisons/goose` |
| **Aider** | Python | Pinned source checkout, `research/comparisons/aider` |
| **Codex CLI**, **Cline / Roo Code**, **OpenHands**, **Continue** | various | **Not checked out.** Compared from general knowledge of their published features, so verify before acting on those rows. |

GoHarness code claims were checked directly against `cmd/` and `internal/`, with file references.

Severity: 🔴 **High**: a security or data-safety loophole, or a gap most users will hit.
🟠 **Medium**: a real capability gap. 🟢 **Low**: polish or niche.

---

## 1. Security loopholes (fix these first)

| # | Severity | Loophole | Evidence | Who does better |
| --- | --- | --- | --- | --- |
| S1 | ✅ RC130 | **Shell commands inherit every environment variable, including `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` and any cloud tokens.** An approved command, or one written after a prompt injection, can read and exfiltrate them (`env`, `printenv`, `curl -d "$OPENAI_API_KEY"`). | `internal/tools/process.go` and `jobs.go` never set `cmd.Env`, so Go passes the full environment. | Codex CLI filters the environment by default. Goose has an egress inspector. |
| S2 | ✅ RC130 | **MCP and LSP servers also receive the full environment.** A third-party MCP server gets every API key in your shell. | `internal/mcpclient/client.go:466`, `internal/lspclient/process.go:32` copy `os.Environ()`. | Most harnesses pass only an explicit `env` list to MCP servers. |
| S3 | ✅ RC134 (opt-in) | **No OS sandbox.** Once approved, a shell command can touch any file and the network; the tool descriptions say "not sandboxed". Approval is the only barrier. | `internal/tools/tools.go:131`, `jobs.go:61`; no seatbelt or landlock code. | Codex CLI (macOS Seatbelt, Linux Landlock + seccomp, network off by default); OpenHands (Docker). |
| S4 | 🟠 | **Shell rules match text, not intent.** `allow: "go test"` cannot be widened with `&&` (compound commands always ask), but aliases, `env VAR=… cmd` or a script the model wrote earlier can still run under an allowed prefix. | Documented in END_TO_END_AUDIT ("a guardrail"). | Codex sandbox makes rule accuracy less critical. |
| S5 | 🟠 | **No prompt-injection scanning.** Webfetch pages, MCP results and repository files are only *labelled* untrusted in the system prompt. Nothing inspects them for injected instructions or data-exfiltration URLs. | `internal/runtime/runtime.go:30,141` | Goose `security/` (pattern scanner, adversary inspector, egress inspector). |
| S6 | 🟠 | **Session journals are plaintext.** Anything secret the model read or printed (a `.env` file, a token in a log) is stored in clear, permanently, in `~/…/sessions/*.jsonl`. Redaction only applies to exports. | `internal/session/redact.go` is used by export and transcript paths. | None encrypt either, but Codex and Claude Code redact known secret patterns before writing. |
| S7 | 🟠 | **ChatGPT tokens and saved keys sit in 0600 files, not the OS keychain.** | END_TO_END_AUDIT, credentials row | Crush (`pinentry`, OS keyring), Goose (keyring), Codex (keyring option). |
| S8 | 🟢 | **No egress allowlist for `webfetch` / `websearch`.** Each fetch is approved individually, but there is no "only these domains" policy. | `internal/webfetch` | Codex (network off), Goose egress inspector. |
| S9 | 🟢 | **Background jobs survive a hard kill.** After a SIGKILL of `goh`, jobs keep running until the next start offers to stop them (Unix only; no Windows orphan handling). | END_TO_END_AUDIT, shell row | — |

## 2. Agent capabilities

| # | Severity | Gap | Who has it |
| --- | --- | --- | --- |
| A1 | ✅ RC132 | **No hooks** (run a user command before or after a tool, on permission prompts, on finish). Hooks are how teams enforce lint, block paths and log activity. | Crush (`PreToolUse`, `PostToolUse`, `PermissionRequest`…), Goose, Claude Code. OpenCode uses plugins. |
| A2 | ✅ RC133 (opt-in; TUI pending) | **No whole-project snapshots / revert across turns.** Undo covers only files edited through our tools; anything changed by a shell command (`npm install`, `sed -i`, codegen) cannot be undone. | OpenCode (Git tree snapshots), Cline / Roo (shadow-Git checkpoints), Aider (Git auto-commit). |
| A3 | 🟠 | **No Git auto-commit or commit-message mode.** | Aider (commits every change with a generated message; `/undo` reverts the commit). |
| A4 | 🟠 | **No automatic lint/test → fix loop.** The model must decide to run tests itself. Formatters and LSP diagnostics exist, but there is no configured "after edits, run X and feed failures back" step. | Aider (`--auto-lint`, `--auto-test`), Goose recipes. |
| A5 | 🟠 | **No plugin or custom-tool system.** Only MCP can add tools. | OpenCode (JS/TS tools and plugins), Goose (extensions), Crush. |
| A6 | 🟠 | **No recipes / reusable workflows** with parameters that can be shared and run headlessly. Custom commands are text templates only. | Goose recipes (YAML, parameters, sub-recipes, scheduler). |
| A7 | 🟠 | **Subagents run one at a time, in the foreground.** No parallel or background child tasks. | OpenCode (concurrent tasks), Claude Code (parallel subagents). |
| A8 | 🟠 | **Repository map covers Go only.** Other languages get no structural map. | Aider (tree-sitter, many languages, PageRank ranking). |
| A9 | 🟢 | **No architect/editor split** (one strong model plans, a cheaper model writes edits). Model roles exist only for summaries and titles. | Aider architect mode. |
| A10 | 🟢 | **No "watch files for AI comments" mode.** | Aider `--watch-files`. |
| A11 | 🟢 | **No scheduled / cron runs.** | Goose scheduler. |
| A12 | 🟢 | **No `code mode` / scripted tool composition.** | OpenCode (experimental), Goose code-execution extension. |

## 3. Providers and models

| # | Severity | Gap | Who has it |
| --- | --- | --- | --- |
| P1 | ✅ RC131 | **No Anthropic prompt caching** (`cache_control`). Long sessions resend the whole prefix at full price, often 5–10× the cost of cached reads. | OpenCode, Crush, Goose, Aider, Claude Code. |
| P2 | 🟠 | **No native Gemini, Bedrock, Vertex or Azure adapters.** These work only through an OpenAI-compatible gateway, which loses provider-specific features and auth (AWS SigV4, Google ADC). | OpenCode (via the AI SDK), Goose, Crush. |
| P3 | 🟠 | **No built-in model catalog** (context windows, prices, capabilities). Users must set `context_window` and prices by hand. | OpenCode (models.dev), Goose (`model_catalog`), Crush (catwalk). |
| P4 | 🟠 | **Several provider paths have never run against real services:** ChatGPT reasoning summaries, the 401 retry, Anthropic extended thinking, live failover. | VERIFICATION / END_TO_END_AUDIT |
| P5 | 🟢 | **No local-inference bundle.** Ollama works through the Chat adapter, but nothing detects or helps set it up. | Goose (`goose-local-inference`). |
| P6 | 🟢 | **No PDF or audio input.** Images are supported. | Claude Code, OpenCode (PDF). |

## 4. MCP and integrations

| # | Severity | Gap | Who has it |
| --- | --- | --- | --- |
| M1 | 🟠 | **MCP tools only; no resources, prompts or elicitation.** | Goose (elicitation, resources), Claude Code, OpenCode. |
| M2 | 🟠 | **No MCP OAuth.** Remote servers need static header tokens. | OpenCode, Goose (`oauth`), Claude Code. |
| M3 | ✅ RC135 | **No MCP reconnect.** A remote MCP server that drops is stopped for the rest of the run. | OpenCode, Goose. |
| M4 | 🟢 | **No legacy MCP SSE transport.** Only stdio and Streamable HTTP. | OpenCode, Goose. |
| M5 | 🟢 | **No IDE extension.** ACP covers Zed-style editors, but there is no VS Code or JetBrains plugin. | Cline, Continue, Roo (VS Code native), Claude Code. |
| M6 | 🟢 | **No GitHub Action / CI agent mode.** `goh pr` checks a PR out locally only. | OpenCode (`opencode github`), Claude Code Action, Codex. |

## 5. User interface

| # | Severity | Gap | Who has it |
| --- | --- | --- | --- |
| U1 | 🟠 | **No desktop or web UI.** | OpenCode (web + desktop), Goose (desktop app), OpenHands (web). |
| U2 | 🟠 | **The TUI cannot attach to a running `goh serve`** (no remote or headless-server workflow). | OpenCode `attach`, Crush client/server. |
| U3 | 🟢 | **No completion notification** (terminal bell or OS notification when a long run finishes or needs approval). | Claude Code, Crush, Codex. |
| U4 | 🟢 | **No voice input.** | Aider (`/voice`), Goose (dictation). |
| U5 | 🟢 | **Approval dialogs look different from OpenCode, and some navigation starts a fresh terminal view.** | END_TO_END_AUDIT, TUI row |
| U6 | 🟢 | **No accessibility or localisation work** (screen-reader mode, translated strings). | OpenCode (partial). |

## 6. Sessions, data and operations

| # | Severity | Gap | Who has it |
| --- | --- | --- | --- |
| D1 | 🟠 | **No session sharing.** Deliberately not built, because it uploads conversations; it needs your decision. | OpenCode `/share`. |
| D2 | 🟠 | **No observability export** (OpenTelemetry traces of runs, tool latency, errors). `goh stats` is local only. | Goose (`otel`, tracing), Codex. |
| D3 | 🟢 | **Only one ChatGPT account at a time.** | — |
| D4 | 🟢 | **No `goh upgrade`** until a release exists. | OpenCode, Crush (`update`), Goose. |
| D5 | 🟢 | **No project-level configuration file.** Settings are user-level only (a deliberate safety choice, but teams cannot share settings through the repository). | OpenCode (`opencode.json`), Crush (`crush.json`), Claude Code. |

## 7. Quality and release gaps

| # | Severity | Gap |
| --- | --- | --- |
| Q1 | 🔴 | **No release published.** Installers, `goh upgrade` and any outside user depend on one. |
| Q2 | 🟠 | **Windows shell path (`cmd.exe`) and Windows file-privacy ACLs have no dedicated tests**, even though CI runs on Windows. |
| Q3 | 🟠 | **No end-to-end benchmark against real tasks** (e.g. SWE-bench-style). Comparisons so far are behaviour captures, not success rates. |
| Q4 | 🟢 | **Fuzzing is bounded to 3 s per target** in `make fuzz`; no long-running fuzz job in CI. |
| Q5 | 🟢 | **Historical docs still describe the old `implementation/` layout** (old CHANGELOG/VERIFICATION entries, `docs/plan/IMPLEMENTATION_PLAN.md`, which must stay unchanged). |

---

## 8. Where GoHarness is already ahead

These are not gaps, but they matter when you choose what to keep:

- **Hash-guarded edits:** no other inspected harness *requires* the file's current hash before every write.
- **Uncertain effects are never replayed:** interrupted tools are journaled and need explicit acknowledgement.
- **Salvage of damaged sessions** into a new session.
- **Credential discipline:** keys go only to official hosts; failover only through an explicit list.
- **Repository files cannot pick the model, provider or agent.**
- **A single static Go binary** with deterministic, key-free tests, including PTY tests of the TUI.

## 9. Recommended order

1. **S1 + S2: scrub the environment** for shell, jobs, MCP and LSP. Pass a safe base set (PATH, HOME, LANG, TERM…) plus anything the user lists explicitly. This is small and closes the biggest loophole.
2. **P1: Anthropic prompt caching.** Small, with a large cost saving.
3. **A1: hooks** (pre/post tool, permission, finish), user config only.
4. **A2: whole-project Git snapshots**, so shell changes can also be undone.
5. **S3: an optional OS sandbox** for shell (macOS `sandbox-exec`, Linux Landlock), with network off as an option.
6. **M1–M3: MCP resources/prompts, reconnect, OAuth.**
7. **Q1: publish a pre-release.**
