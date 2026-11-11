# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository layout

- Root — the product: **GoHarness** (`goh`), an original Go coding assistant (headless JSON/text CLI, Bubble Tea TUI, file/shell tools, durable sessions, stdio MCP, ACP, LSP). Go module `github.com/Sanidhya14321/goh`, Go 1.27.4. Code lives in `cmd/` and `internal/`; Python integration scripts in `scripts/`; dependency licenses in `licenses/`.
- `docs/` — `ARCHITECTURE.md`, `CHANGELOG.md`, `VERIFICATION.md`, `PROGRESS.md`, `END_TO_END_AUDIT.md`, `BENCHMARKS.md`, `DEPENDENCIES.md`, `ORIGIN_LEDGER.md`. Update CHANGELOG, VERIFICATION and PROGRESS (and END_TO_END_AUDIT when relevant) when a milestone lands.
- `docs/plan/` — the user's `IMPLEMENTATION_PLAN.md` (must remain unchanged) and `specs/` (behavioral contracts, `COMPATIBILITY_MATRIX.md`).
- `research/` — local reference material and comparison evidence (pinned OpenCode v1.19.5 source under `research/reference`, captured evidence under `research/evidence/`, notes and reports, the local Go toolchain under `research/tooling/go`). Mostly git-ignored. **Never copy code, prompts, or fixtures from research into `cmd/` or `internal/`** — see `docs/ORIGIN_LEDGER.md`. Product code is independently authored Apache-2.0; reference material is only for observing behavior.

## Commands (run from the repository root; local toolchain: `GO=research/tooling/go/bin/go`)

```sh
make build        # bin/goh (+ bin/harness symlink) and bin/scripted-model, CGO disabled
make test         # go test ./...
make race         # go test -race ./...
make vet
make fuzz         # bounded fuzz targets (3s each)
make acceptance   # builds, then go run ./cmd/acceptance (CLI fixture scenarios)
make release      # scripts/release.sh -> dist/goh-{darwin,linux,windows}-{arm64,amd64}
make demo         # scripted-model text scenario
make bench-check  # builds bin/goh-bench (cmd/bench) and dry-runs bench/tasks

go test ./internal/runtime -run TestName     # single test
```

Python integration scripts in `scripts/` drive the compiled binary: `test_tui.py` (PTY scenarios), `test_setup.py`, `test_acp.py`, `test_attach.py` (serve + attach), and `test_openai_live.py --binary bin/goh` (live — makes real inference requests with the saved ChatGPT auth; don't run without the user's approval). `scripts/capture_*.py` and `research/compare_*.py` produce comparison evidence against reference products.

Tests are deterministic: use `cmd/scripted-model` (a scripted local model endpoint) rather than live credentials.

## Architecture

- `cmd/harness` — `goh` entrypoint: CLI subcommands (`chat`, `sessions`, `auth`, `serve`, ACP mode), launcher and pickers. Bare `goh` opens the TUI.
- `internal/runtime` — the agent loop: system/summarization prompts (`runtime.go`), tool dispatch and permission policy, context compaction, subagents, skills, memory, todos, questions, LSP/webfetch integration.
- `internal/provider` — model adapters: Anthropic SSE, OpenAI Chat Completions (with JSON-reply and stream-option fallback), OpenAI Responses (incl. `openai_chatgpt` protocol bound to the official endpoint). Shared error classification in `errors.go`.
- `internal/core` — config, provider profiles/routing, user settings, ChatGPT OAuth (PKCE, private credential store), shared types.
- `internal/session` — durable conversation journal (the source of truth for resume, fork, export), portable JSON export, Markdown transcripts, compaction records.
- `internal/tools` — exact file tools, patch grammar, glob/grep, shell execution with approval, change tracking, repo map.
- `internal/tui` — Bubble Tea UI: composer, command palette, `/provider` `/connect` `/models` `/fork` `/export` `/timeline` `/status` dialogs.
- `cmd/bench` — `goh-bench` benchmark runner (hidden tests, clean regrading, tamper detection, A/B arms); tasks in `bench/`, reports in `docs/bench/`. `cmd/acceptance` — CLI fixture scenarios.
- Enforcement packages: `sandbox` (sandbox-exec on macOS, Landlock on Linux), `rewind`/`snapshots`/`projectsnap` (undo, rewind, whole-project snapshots), `ignore`, `platform`.
- Other: `remote` (client for `goh serve`, used by `goh attach`), `memory`, `websearch`, `mcpclient` (stdio MCP), `acp`, `lspclient`, `localapi` (serve), `commands` (custom command expansion), `skills`, `snapshots`, `structured`, `formatters`, `webfetch`.

Credential rule: `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` are only sent to official HTTPS hosts; gateways use a saved key or their own `api_key_env`.

Dependencies are pinned; new dependencies require updating `DEPENDENCIES.md`, `THIRD_PARTY_NOTICES` and `licenses/`. Real-project benchmark tasks (`bench/tasks/*` with a `git` source) contain upstream files; when adding one, add its license text and index row in `bench/licenses/`.
