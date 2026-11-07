# GoHarness end-to-end audit

7 November 2026 · RC16 · macOS ARM64 · Go 1.27.3

## What ran

- The compiled GoHarness reached OpenAI's public Chat Completions endpoint and classified HTTP 401 as authentication failure. It used a disposable project and no key. Authenticated GoHarness inference on OpenAI remains unverified.
- The official Codex CLI was already signed in with ChatGPT. A read-only, ephemeral run in a disposable directory returned the requested marker. This verifies the account through Codex. GoHarness still needs its own connection to the account.
- A local OpenAI-compatible fixture exercised the compiled terminal application, including setup, authentication recovery, profile changes, prompt editing, session resume and fork, export, and the compatibility path from a rejected streaming option to a complete JSON reply. Seven CLI scenarios, 15 terminal scenarios, ACP integration, race tests and vet cover the rest of the local verification surface.
- The current dependency scan reports zero findings. Linux amd64 cross-build passed. Windows amd64 failed to build. Cross-building Linux provides no evidence of native Linux process cleanup. Raw platform diagnostics and the test reports are under `research/evidence/v1-audit-*`.

## Recovery by area

| Area | Working recovery or alternate path | Missing or unverified |
| --- | --- | --- |
| Setup and authentication | Chat opens before setup. Provider and profile navigation preserve an unfinished draft. A 401 sends the user back to provider setup with the journal intact. Official API variables are bound to their HTTPS origins. Selecting a different route clears an inherited gateway variable. | Native **Continue with ChatGPT** and live authenticated acceptance. Gateway credentials need an explicitly named variable or a saved key scoped to the exact URL. |
| Provider transport | Bounded retry for connection errors, timeouts, 429s and server failures. An explicit rejection of `stream_options` removes just that optional field for one retry. Complete JSON replies can carry text, tool calls and usage. | Automatic failover across explicitly configured providers, per-provider cooldowns and consistent retry status in the UI. Partial streams require manual retry or profile selection. |
| Context and model roles | Context accounting reserves output and compaction preserves active turns and tool groups. Model roles can use a separate profile. Compaction failure retains the saved history. | Automatic recovery from a failed summary provider. Native Responses history and reasoning-state compatibility. |
| File search, repository map and reads | Direct file reads, glob and grep remain available when the declaration map or LSP lacks coverage. The composer can insert a relative path. | Live `@` refresh, OpenCode-style content attachments, and complete ignore-rule parity. |
| Edits, patches and checkpoints | Hash and content guards reject stale edits. Review can deny changes. With snapshots enabled, `goh changes` inspects and restores recorded checkpoints. An interrupted batch can report partial publication. | Transcript-level rewind and redo in the TUI. Shell and external effects need inspection and independent recovery. |
| Shell and cancellation | Esc or Ctrl+C stops an active run. Shell deadlines and process-group cleanup are bounded. The journal records uncertain effects and retained artifacts. | Repeat cancellation under load and native Linux and Windows cleanup tests. Previously observed child cleanup failures remain a release gate. |
| Sessions and durability | Session listing, preview, resume, branch, Markdown export and private portable JSON offer alternate inspection and recovery paths. Torn journal tails can be recovered. Uncertain effects require inspection before explicit acknowledgment. | Guided in-chat recovery and a user-facing recovery workflow for corrupted journals. |
| Formatter and LSP | Formatter denial or absence preserves the edit. Direct file search and reads provide an alternative to LSP. Integrations require configured commands and separate authority. | Automatic degradation from failing integration startup and live tests with additional servers. |
| MCP and child tasks | Built-in tools and the main agent can work with integrations disabled. Failures and unresolved effects remain visible. Typed child output has validation. | Per-integration health and retry controls. Third-party servers and parent/child cleanup under load need more native coverage. |
| Memory, skills and custom commands | Memory can be disabled. Prompts and direct tools still work without optional skills or custom commands. Task state is journaled. | A recovery panel for failed indexes, skill discovery diagnostics and failed custom expansions. |
| CLI, local API and ACP | Direct CLI use remains available when an ACP client or local API integration fails. ACP fixture tests verify grants, cancellation, resume and EOF cleanup. | Broader real-client interoperability and cross-platform tests. |
| Terminal UI | Wide and compact layouts, modal Escape behavior, keyboard editing, draft recovery, and provider setup are covered by PTY tests. | OpenCode parity for configurable keymaps, themes, editor handoff, transcript jumps and visual approval states. Some navigation still starts a new terminal view. |
| Packaging and installs | Local macOS install and Linux build verified. Existing release script builds the Darwin and Linux artifacts. | Windows requires replacements for Unix file locks, no-follow file opens and process groups. Native platform and installer smoke tests are pending. |

Recovery for a mutating operation starts by inspecting its recorded state. Automatic fallback must preserve the original approval scope and avoid repeating a tool whose effects are uncertain.

## ChatGPT account connection

OpenAI documents a [Sign in with ChatGPT flow for eligible open-source apps](https://developers.openai.com/siwc/token-sharing-open-source) and [Responses inference for ChatGPT plan usage](https://developers.openai.com/siwc/token-sharing-open-source/models-and-inference). GoHarness needs a native Responses adapter for text, tool calls, tool results and cancellation, plus an app-owned authorization flow with browser consent, PKCE, token validation, private storage and refresh. Account entitlements still require a successful live turn. The existing Codex session demonstrates account availability separately and its stored credentials remain outside GoHarness.

A native acceptance run should cover sign-in, one streamed response, a permission-gated read/edit/verify task in a disposable project, durable resume, interruption, and expired-grant recovery. Keep this distinct from fixture transport and protocol checks.
