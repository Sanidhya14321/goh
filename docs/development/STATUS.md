# Project status

*Last reviewed: 15 November 2026 · latest tag `v1.0.0-rc.180` plus unreleased changes in the
[changelog](../../CHANGELOG.md).*

GoHarness is feature-complete for a 1.0 release candidate. Every milestone passes the gates in
[TESTING.md](TESTING.md); per-release evidence is in [VERIFICATION.md](VERIFICATION.md). This page
lists only what is **not** done or **not** verified, so it stays short.

## Verified live

Over the ChatGPT sign-in (`gpt-6.1-sol`): streaming, tool calls, resume, image input, reasoning
effort, plan handoff and end-of-request task state, plus 113 benchmark attempts that exercised the
full tool loop ([benchmarks](../benchmarks/README.md)). LSP diagnostics and server restart were
verified against real `clangd`, `gopls`, `pyright` and `typescript-language-server`. Over Groq (`chat_completions`): rate-limit waits, the GitHub REST
API through `webfetch`, and a stdio MCP tool, in real-world prompts (see the end-to-end audit in
[VERIFICATION.md](VERIFICATION.md)). Failover from a failed route to Groq, for a turn and for compaction, and the DeepWiki remote MCP server were checked live.

## Verified only with fixtures

These paths have deterministic tests but no live run yet:

- ChatGPT token expiry and the 401-renewal retry;
- Responses reasoning summaries and prompt-cache savings from task state;
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
| Shell rules | Rules match command text, not intent | A guardrail, not a security boundary; the sandbox is the boundary |
| Background jobs | A hard kill of `goh` cannot stop its jobs; the next start finds and offers to stop them (Unix) | No supervisor process |
| Budgets | Runs can be capped by tokens, tool calls and time, not by dollars | Prices vary by plan; `goh stats` shows cost estimates |
| MCP | No OAuth for remote MCP servers | Needs a real OAuth MCP server to build against |
| Providers | A provider's per-request cap is learned from its first refusal (one wasted request per run in `goh run`; a chat session keeps it). Providers whose refusal does not state the cap still fail with HTTP 413 | Groq states it; others vary |
| Attach | `goh attach` works only with a loopback `goh serve` | Remote servers are out of scope |

Not attempted by decision: custom tool plugins (they would run repository code), session sharing
(it would upload conversations).

## Next

1. **Live checks** for the fixture-only paths above, starting with provider failover and token
   expiry.
2. **First stable release** once the user approves publishing.
