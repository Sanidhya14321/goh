# Project status

*Last reviewed: 21 November 2026 · latest release `v1.0.0` ([changelog](../../CHANGELOG.md)).*

GoHarness 1.0.0 is the first stable release. Every milestone passes the gates in
[TESTING.md](TESTING.md); per-release evidence is in [VERIFICATION.md](VERIFICATION.md). This page
lists only what is **not** done or **not** verified, so it stays short.

## Verified live

Over the ChatGPT sign-in (`gpt-6.1-sol`, then `gpt-6.2-sol` on 18 Nov): streaming, tool calls, resume, image input, reasoning
effort and summaries, prompt-cache reads, token expiry and the 401-renewal retry, `goh auth refresh`, plan handoff and end-of-request task state, plus 113 benchmark attempts that exercised the
full tool loop ([benchmarks](../benchmarks/README.md)). LSP diagnostics and server restart were
verified against real `clangd`, `gopls`, `pyright` and `typescript-language-server`. Over Groq (`chat_completions`): rate-limit waits, the GitHub REST
API through `webfetch`, and a stdio MCP tool, in real-world prompts (see the end-to-end audit in
[VERIFICATION.md](VERIFICATION.md)). Failover from a failed route to Groq, for a turn and for compaction, and the DeepWiki remote MCP server were checked live.

## Verified only with fixtures

These paths have deterministic tests but no live run yet:

- Anthropic extended thinking and API-key model lists (no Anthropic key by decision; Claude subscriptions cannot be used);
- editor clients other than the ACP fixture.

## Platform coverage

macOS is the development host and runs every gate, including the pseudo-terminal suites. Native CI
runs unit tests on Linux, macOS and Windows, terminal suites on Linux and macOS, and installer
checks on all three. Changes that only CI can verify are flagged in the changelog entry that made
them. No manual Windows session has been run.

## Known gaps

| Area | Gap | Why |
| --- | --- | --- |
| Release | Binaries are not code-signed or notarized | Needs signing identities |
| Credentials | One ChatGPT account at a time; Claude Pro/Max sign-in not offered | Anthropic does not permit subscription sign-in for third-party tools; use a Console API key |
| Credentials | Linux keychain sealing needs `secret-tool` installed (verified with gnome-keyring) | Uses the desktop Secret Service |
| Sandbox | Landlock cannot hide files denied by `file_rules` from a sandboxed shell (macOS can) | Landlock grants are allow-lists of whole folders |
| Sandbox | On Linux, Auto asks instead of running commands in a folder that contains a stricter subfolder or goh's own state; the read-only `.git/hooks` and `.git/config` rule is not enforced there | Landlock cannot restrict a folder inside a writable one; unit tests pass natively on Linux CI; not yet exercised in an interactive Linux session |
| Benchmark | No task lists `gold_files`; no private holdout exists; the research experiments have no live result | Holdout and gold annotations still to write; live arms need model quota ([experiments](../benchmarks/experiments.md)) |
| Shell rules | Rules match command text, not intent | A guardrail, not a security boundary; the sandbox is the boundary |
| Background jobs | A hard kill of `goh` cannot stop its jobs; the next start finds and offers to stop them (Unix) | No supervisor process |
| Budgets | Runs can be capped by tokens, tool calls and time, not by dollars | Prices vary by plan; `goh stats` shows cost estimates |
| MCP | No OAuth for remote MCP servers | Needs a real OAuth MCP server to build against |
| Providers | A provider's per-request cap is learned from its first refusal (one wasted request per run in `goh run`; a chat session keeps it). Providers whose refusal does not state the cap still fail with HTTP 413 | Groq states it; others vary |
| Providers | A fallback's stated request cap is only used to skip that fallback; its request is not shrunk to fit it. `max_run_tokens` is checked between requests, so one request can finish above it | The primary keeps cap learning; per-route compaction would be a larger change |
| Verification | A check inside a pipeline, after `\|\|`, followed by another command, or using a redirect such as `2>&1` does not count as verification even when it passed | Its exit status cannot be tied to the check from the command text; run the check on its own or use `verify` |
| Attach | `goh attach` works only with a loopback `goh serve` | Remote servers are out of scope |
| CI | On Windows runners, `TestRealClangd` and `TestParentChildCancellationUnderLoad` failed together in 3 of 8 runs on 17 Nov and again on 19 Nov (run 44553177409): clangd took over 15 s to initialize and six PowerShell children took over 30 s to start while other packages ran | Runner load. Both tests now allow a slower start-up (45 s LSP operations in the clangd test, 120 s child start on Windows) without loosening the cancellation and approval checks they measure; watch whether they recur |

Not attempted by decision: custom tool plugins (they would run repository code), session sharing
(it would upload conversations).

## Next

1. **Manual checks:** Zed over ACP, and one interactive Windows session (LSP drive/UNC URIs and the attach/TUI changes are cross-vetted only).
2. **Benchmark:** add `gold_files` and a private Go/Python holdout, then run the experiment arms ([experiments](../benchmarks/experiments.md)).
