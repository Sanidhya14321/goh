# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository layout

- Root — the product: **GoHarness** (`goh`), an original Go coding assistant (headless JSON/text CLI, Bubble Tea TUI, file/shell tools, durable sessions, MCP, ACP, LSP). Go module `github.com/Sanidhya14321/goh`, Go 1.27.4. Code lives in `cmd/` and `internal/`; installers and the release script in `scripts/`, Python terminal test suites in `scripts/e2e/`; dependency licenses in `licenses/`; `CHANGELOG.md` at the root.
- `docs/` — index in `docs/README.md`: `USER_GUIDE.md`, `ARCHITECTURE.md`, `DEPENDENCIES.md`, `ORIGIN_LEDGER.md`; `benchmarks/` (agent results and microbenchmarks); `development/` (`STATUS.md`, `TESTING.md`, `DESIGN_DECISIONS.md`, `VERIFICATION.md`; `history/` keeps the full rc.1–rc.179 changelog and verification log, which are not edited further). When a milestone lands, update `CHANGELOG.md`, add a section to `docs/development/VERIFICATION.md`, and update `STATUS.md` if a gap opened or closed. Keep docs short and current; the verification log is the only append-only history.
- `docs/plan/` — the user's `IMPLEMENTATION_PLAN.md` (must remain unchanged) and `specs/` (behavioral contracts, `COMPATIBILITY_MATRIX.md`).
- `research/` — local reference material and comparison evidence (pinned OpenCode v1.19.5 source under `research/reference`, captured evidence under `research/evidence/`, comparison capture scripts under `research/scripts/`, notes and reports, the local Go toolchain under `research/tooling/go`). Mostly git-ignored. **Never copy code, prompts, or fixtures from research into `cmd/` or `internal/`** — see `docs/ORIGIN_LEDGER.md`. Product code is independently authored Apache-2.0; reference material is only for observing behavior.

## Commands (run from the repository root; local toolchain: `GO=research/tooling/go/bin/go`)

```sh
make build        # bin/goh (+ bin/harness symlink) and bin/scripted-model, CGO disabled
make test         # go test ./...
make race         # go test -race ./...
make vet
make fuzz         # bounded fuzz targets (3s each)
make acceptance   # builds, then go run ./cmd/acceptance (CLI fixture scenarios)
make e2e          # acceptance plus the PTY suites in scripts/e2e (macOS/Linux)
make install      # build and copy bin/goh to ~/.local/bin
make release      # scripts/release.sh -> dist/goh-{darwin,linux,windows}-{arm64,amd64}; pushing a v* tag publishes via .github/workflows/release.yml
make demo         # scripted-model text scenario
make bench-check  # builds bin/goh-bench (cmd/bench) and dry-runs bench/tasks

go test ./internal/runtime -run TestName     # single test
```

Python integration scripts in `scripts/e2e/` drive the compiled binary: `test_tui.py` (PTY scenarios), `test_setup.py`, `test_acp.py`, `test_attach.py` (serve + attach), `test_resume_setup.py` (`chat --session` through provider setup), and `test_openai_live.py --binary bin/goh` / `test_openai_live_extra.py` (live — make real inference requests with the saved ChatGPT auth; don't run without the user's approval). `research/scripts/capture_*.py` and `research/compare_*.py` produce comparison evidence against reference products.

Tests are deterministic: use `cmd/scripted-model` (a scripted local model endpoint) rather than live credentials.

## Architecture

- `cmd/harness` — `goh` entrypoint: CLI subcommands (`chat`, `sessions`, `auth`, `serve`, ACP mode), launcher and pickers. Bare `goh` opens the TUI.
- `internal/runtime` — the agent loop: system/summarization prompts (`runtime.go`), tool dispatch and permission policy, context compaction, subagents, skills, memory, todos, questions, LSP/webfetch integration.
- `internal/provider` — model adapters: Anthropic SSE, OpenAI Chat Completions (with JSON-reply and stream-option fallback), OpenAI Responses (incl. `openai_chatgpt` protocol bound to the official endpoint). Shared error classification in `errors.go`.
- `internal/core` — config, provider profiles/routing, user settings, ChatGPT OAuth (PKCE, private credential store; `credential_store.go` seals secrets with a key from `internal/secretstore`, the OS keychain/Secret Service/DPAPI), shared types.
- `internal/session` — durable conversation journal (the source of truth for resume, fork, export), portable JSON export, Markdown transcripts, compaction records.
- `internal/tools` — exact file tools, patch grammar, glob/grep, shell execution with approval, change tracking, repo map.
- `internal/tui` — Bubble Tea UI: composer, command palette, `/providers` manager, `/connect` `/models` `/fork` `/export` `/timeline` `/status` dialogs. Providers are named profiles with their own route; the active one is mirrored into the top-level settings (`internal/core/providers.go`, flows in `cmd/harness/connect.go`).
- `cmd/bench` — `goh-bench` benchmark runner (hidden tests, clean regrading, tamper detection, A/B arms); tasks in `bench/`, reports in `docs/benchmarks/`. `cmd/acceptance` — CLI fixture scenarios.
- Enforcement packages: `sandbox` (sandbox-exec on macOS, Landlock on Linux), `rewind`/`snapshots`/`projectsnap` (undo, rewind, whole-project snapshots), `ignore`, `platform`.
- Other: `remote` (client for `goh serve`, used by `goh attach`), `memory`, `websearch`, `mcpclient` (stdio MCP), `acp`, `lspclient`, `localapi` (serve), `commands` (custom command expansion), `skills`, `snapshots`, `structured`, `formatters`, `webfetch`.

Credential rule: `OPENAI_API_KEY`/`ANTHROPIC_API_KEY` are only sent to official HTTPS hosts; gateways use a saved key or their own `api_key_env`.

Dependencies are pinned; new dependencies require updating `DEPENDENCIES.md`, `THIRD_PARTY_NOTICES` and `licenses/`. Real-project benchmark tasks (`bench/tasks/*` with a `git` source) contain upstream files; when adding one, add its license text and index row in `bench/licenses/`.
