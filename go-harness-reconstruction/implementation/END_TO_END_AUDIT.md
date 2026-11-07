# GoHarness end-to-end audit

7 November 2026 · RC17 · macOS ARM64 · Go 1.27.3

## What ran

- Live GoHarness used the account's `gpt-6.1-sol` model on the official OpenAI Responses endpoint. The user authorized GoHarness in the browser. The compiled harness streamed a marker, read a disposable file, applied a permission-gated edit, verified the result, and resumed the same session. A separate live PTY run connected through `/connect`, selected a model and sent a prompt. The [live evidence](../research/evidence/v1-chatgpt-live.json) contains only the check results.
- Local fixture and unit tests exercised PKCE callback state, ID-token signature and nonce checks, credential rotation, partial stream refusal, and the existing Chat Completions, Anthropic, CLI, TUI and ACP surfaces.
- RC16's Linux amd64 cross-build passed and Windows amd64 failed. Native Linux and Windows behavior remain release gates. The RC16 platform and vulnerability reports are still retained alongside the RC17 evidence.

## Recovery by area

| Area | Working recovery or alternate path | Missing or unverified |
| --- | --- | --- |
| Setup and authentication | Chat opens before setup. Provider and profile navigation preserve an unfinished draft. A 401 sends the user back to provider setup with the journal intact. GoHarness supports **Continue with ChatGPT**, private token storage, renewal and browser reauthorization. API key and gateway setup remain selectable. | Multiple saved ChatGPT accounts, a live expired-grant test, and OS keychain storage. A corrupt credential file still needs guided recovery. |
| Provider transport | Bounded retry for connection errors, timeouts, 429s and server failures. An explicit rejection of `stream_options` removes just that optional field for one retry. Complete Chat Completions JSON replies can carry text, tool calls and usage. Responses releases tools only after successful completion. RC18: ordered `fallback_profiles` failover for failures before any response body (connection, 408, 429, 5xx), journaled and shown in chat; fixture and loopback-HTTP tested. | Per-provider cooldowns, a live multi-provider failover test, and a consistent retry countdown in the UI. Partial streams still require manual retry or profile selection by design. |
| Context and model roles | Context accounting reserves output and compaction preserves active turns and tool groups. Model roles can use a separate profile. Compaction failure retains the saved history. RC18: compaction uses the same explicit failover list. | Live failover during compaction. More live coverage for compaction and model switching. Responses reasoning items are retained only for matching request routes. |
| File search, repository map and reads | Direct file reads, glob and grep remain available when the declaration map or LSP lacks coverage. The composer can insert a relative path. | Live `@` refresh, OpenCode-style content attachments, and complete ignore-rule parity. |
| Edits, patches and checkpoints | Hash and content guards reject stale edits. Review can deny changes. With snapshots enabled, `goh changes` inspects and restores recorded checkpoints. An interrupted batch can report partial publication. | Transcript-level rewind and redo in the TUI. Shell and external effects need inspection and independent recovery. |
| Shell and cancellation | Esc or Ctrl+C stops an active run. Shell deadlines and process-group cleanup are bounded. The journal records uncertain effects and retained artifacts. | Repeat cancellation under load and native Linux and Windows cleanup tests. Previously observed child cleanup failures remain a release gate. |
| Sessions and durability | Session listing, preview, resume, branch, Markdown export and private portable JSON offer alternate inspection and recovery paths. Torn journal tails can be recovered. Uncertain effects require inspection before explicit acknowledgment. | Guided in-chat recovery and a user-facing recovery workflow for corrupted journals. |
| Formatter and LSP | Formatter denial or absence preserves the edit. Direct file search and reads provide an alternative to LSP. Integrations require configured commands and separate authority. | Automatic degradation from failing integration startup and live tests with additional servers. |
| MCP and child tasks | Built-in tools and the main agent can work with integrations disabled. Failures and unresolved effects remain visible. Typed child output has validation. | Per-integration health and retry controls. Third-party servers and parent/child cleanup under load need more native coverage. |
| Memory, skills and custom commands | Memory can be disabled. Prompts and direct tools still work without optional skills or custom commands. Task state is journaled. | A recovery panel for failed indexes, skill discovery diagnostics and failed custom expansions. |
| CLI, local API and ACP | Direct CLI use remains available when an ACP client or local API integration fails. ACP fixture tests verify grants, cancellation, resume and EOF cleanup. | Broader real-client interoperability and cross-platform tests. |
| Terminal UI | Wide and compact layouts, modal Escape behavior, keyboard editing, and draft recovery are covered by PTY tests. The OpenCode-style `/connect` picker and account model selector also passed a live turn. | OpenCode parity for configurable keymaps, themes, editor handoff, transcript jumps and visual approval states. Some navigation still starts a new terminal view. |
| Packaging and installs | Local macOS install and Linux build verified. Existing release script builds the Darwin and Linux artifacts. | Windows requires replacements for Unix file locks, no-follow file opens and process groups. Native platform and installer smoke tests are pending. |

Recovery for a mutating operation starts by inspecting its recorded state. Automatic fallback must preserve the original approval scope and avoid repeating a tool whose effects are uncertain.

## ChatGPT account connection

GoHarness uses the [official Sign in with ChatGPT flow](https://developers.openai.com/siwc/token-sharing-open-source/sign-in) and [public Responses endpoint](https://developers.openai.com/siwc/token-sharing-open-source/models-and-inference). Browser authorization, tool results, and resumed history passed on this Mac. The current implementation has one authorized account and a local catalog default. Renewal and redirect restrictions were tested with fixtures. Live revocation, token expiry, account policy failures, real third-party integrations and native tests on other systems remain open.

ChatGPT preview restrictions include a stateless HTTP stream and no request-level output token cap. GoHarness uses local context and turn limits, and presents its approved local tools through a namespace. [Preview limitations](https://developers.openai.com/siwc/token-sharing-open-source/preview-limitations)
