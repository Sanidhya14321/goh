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
verified against a real `clangd`.

## Verified only with fixtures

These paths have deterministic tests but no live run yet:

- provider failover across real providers, including failover during compaction;
- ChatGPT token expiry and the 401-renewal retry;
- Responses reasoning summaries and prompt-cache savings from task state;
- Anthropic extended thinking and API-key model lists;
- third-party remote MCP services, and language servers other than clangd;
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
| Credentials | Linux keychain sealing needs `secret-tool` installed | Uses the desktop Secret Service |
| Sandbox | Landlock cannot hide files denied by `file_rules` from a sandboxed shell (macOS can) | Landlock grants are allow-lists of whole folders |
| Shell rules | Rules match command text, not intent | A guardrail, not a security boundary; the sandbox is the boundary |
| Background jobs | A hard kill of `goh` cannot stop its jobs; the next start finds and offers to stop them (Unix) | No supervisor process |
| Budgets | Runs can be capped by tokens, tool calls and time, not by dollars | Prices vary by plan; `goh stats` shows cost estimates |
| MCP | No OAuth for remote MCP servers | Needs a real OAuth MCP server to build against |
| Attach | `goh attach` works only with a loopback `goh serve` | Remote servers are out of scope |
| Terminal UI | Provider, connect, model, settings and session-picker dialogs run as separate full-screen programs, drawn over a snapshot of the chat that does not reflow on resize; chat restarts after them (keys typed meanwhile are carried over) | See *Next* below |

Not attempted by decision: custom tool plugins (they would run repository code), session sharing
(it would upload conversations).

## Next

1. **In-chat dialogs.** Move the provider, connect, model and settings flows into the chat
   program as overlays, like the theme, changes, jobs and timeline pickers already are. This
   removes the frozen backdrop and the chat restart between dialogs.
2. **Live checks** for the fixture-only paths above, starting with provider failover and token
   expiry.
3. **First stable release** once the user approves publishing.
