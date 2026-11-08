# GoHarness end-to-end audit

8 November 2026 · RC64 (earlier sections dated by RC) · macOS ARM64 · Go 1.27.3

## What ran

- Live GoHarness used the account's `gpt-6.1-sol` model on the official OpenAI Responses endpoint. The user authorized GoHarness in the browser. The compiled harness streamed a marker, read a disposable file, applied a permission-gated edit, verified the result, and resumed the same session. A separate live PTY run connected through `/connect`, selected a model and sent a prompt. The [live evidence](../research/evidence/v1-chatgpt-live.json) contains only the check results.
- Local fixture and unit tests exercised PKCE callback state, ID-token signature and nonce checks, credential rotation, partial stream refusal, and the existing Chat Completions, Anthropic, CLI, TUI and ACP surfaces.
- RC25: native CI passes on Ubuntu, macOS and Windows, including installers. Windows still lacks dedicated tests for its cmd.exe shell path and ACL privacy. RC22: Windows amd64/arm64 cross-compile and vet. RC16's Linux amd64 cross-build passed and Windows amd64 failed. Native Linux and Windows behavior remain release gates. The RC16 platform and vulnerability reports are still retained alongside the RC17 evidence.

## Recovery by area

| Area | Working recovery or alternate path | Missing or unverified |
| --- | --- | --- |
| Setup and authentication | Chat opens before setup. Provider and profile navigation preserve an unfinished draft. A 401 sends the user back to provider setup with the journal intact. GoHarness supports **Continue with ChatGPT**, private token storage, renewal and browser reauthorization. API key and gateway setup remain selectable. | Multiple saved ChatGPT accounts, a live expired-grant test, and OS keychain storage. RC30: corrupt or wrongly permissioned credentials name the file and the sign-in command, which replaces them. |
| Provider transport | Bounded retry for connection errors, timeouts, 429s and server failures. An explicit rejection of `stream_options` removes just that optional field for one retry. Complete Chat Completions JSON replies can carry text, tool calls and usage. Responses releases tools only after successful completion. RC18: ordered `fallback_profiles` failover for failures before any response body (connection, 408, 429, 5xx), journaled and shown in chat; fixture and loopback-HTTP tested. | Per-provider cooldowns, a live multi-provider failover test, and a consistent retry countdown in the UI. Partial streams still require manual retry or profile selection by design. |
| Context and model roles | Context accounting reserves output and compaction preserves active turns and tool groups. Model roles can use a separate profile. Compaction failure retains the saved history. RC18: compaction uses the same explicit failover list. RC21: automatic compaction masks old tool output first and skips the summary model when that is enough, so a summary-provider outage no longer blocks compaction in that case. | Live failover during compaction. More live coverage for compaction and model switching. Responses reasoning items are retained only for matching request routes. |
| File search, repository map and reads | Direct file reads, glob and grep remain available when the declaration map or LSP lacks coverage. The composer can insert a relative path. | Live `@` refresh, OpenCode-style content attachments, and complete ignore-rule parity. |
| Edits, patches and checkpoints | Hash and content guards reject stale edits. Review can deny changes. With snapshots enabled, `goh changes` inspects and restores recorded checkpoints. An interrupted batch can report partial publication. RC56: checkpoints are on by default in best-effort mode (a failed capture never blocks an edit and is reported); explicit `snapshots: true` is strict. RC55: `/changes` lists and diffs checkpoints. RC63: `file_rules` deny, allow or always ask edits by path, checked on every path a patch touches. | Transcript-level rewind and redo in the TUI. Shell and external effects need inspection and independent recovery. |
| Shell and cancellation | Esc or Ctrl+C stops an active run. Shell deadlines and process-group cleanup are bounded. The journal records uncertain effects and retained artifacts. RC30: 90 concurrent parent/child cancellations under `-race` left no child process and no automatic replay; native Linux and Windows CI pass. RC41: `!command` runs through the same journaled path. RC42: background jobs (`job_start`/`job_output`/`job_stop`, `/jobs`) are stopped with their process tree when GoHarness exits; a PTY test confirms no survivor. RC45: a third identical tool call needs approval (loop guard). RC46: `shell_rules`; allow never covers compound commands. | A hard kill (SIGKILL) of GoHarness cannot stop background jobs itself; since RC66 the next start lists them as orphaned and stops them on request after verifying the command (Unix). Shell rules match text, not intent (documented as a guardrail). |
| Sessions and durability | Session listing, preview, resume, branch, Markdown export and private portable JSON offer alternate inspection and recovery paths. Torn journal tails can be recovered. Uncertain effects require inspection before explicit acknowledgment. | Guided in-chat recovery and a user-facing recovery workflow for corrupted journals. |
| Formatter and LSP | Formatter denial or absence preserves the edit. Direct file search and reads provide an alternative to LSP. Integrations require configured commands and separate authority. RC19: an approved language server that cannot spawn or initialize is stopped, settled as failed, journaled as `integration_degraded`, and the run continues; it no longer blocks the next run as an uncertain effect. RC99: a real clangd run (start after approval, error diagnostic, post-edit error report, go-to-definition) is part of the test suite. | Live tests with additional servers; restart of a disconnected server within a run. |
| MCP and child tasks | Built-in tools and the main agent can work with integrations disabled. Failures and unresolved effects remain visible. Typed child output has validation. RC19: one failing MCP server (bad command, failed initialization, invalid tools) is stopped and skipped while other servers and built-in tools stay available; permission denial still stops the run for approval. RC57/RC61: remote MCP servers over Streamable HTTP (config `url` with `headers_env`, or ACP client `http` servers); HTTPS only except localhost, redirects refused, header values never shown. RC48: `/status` lists stopped integrations. | Third-party remote MCP services, MCP OAuth, and parent/child cleanup under load need live or native coverage. |
| Memory, skills and custom commands | Memory can be disabled. Prompts and direct tools still work without optional skills or custom commands. Task state is journaled. | A recovery panel for failed indexes, skill discovery diagnostics and failed custom expansions. |
| CLI, local API and ACP | Direct CLI use remains available when an ACP client or local API integration fails. ACP fixture tests verify grants, cancellation, resume and EOF cleanup. | Broader real-client interoperability and cross-platform tests. |
| Terminal UI | Wide and compact layouts, modal Escape behavior, keyboard editing, and draft recovery are covered by PTY tests. The OpenCode-style `/connect` picker and account model selector also passed a live turn. RC34–RC64 added `/find` with highlighting, `/copy`, `/timestamps`, `/details`, `/changes`, `/jobs`, timeline revert, queued prompts, a context meter and provider models in `/models`; 24 PTY scenarios run the real binary. | OpenCode parity for configurable keymaps, themes, editor handoff, transcript jumps and visual approval states. Some navigation still starts a new terminal view. |
| Packaging and installs | Local macOS install and Linux build verified. Existing release script builds the Darwin and Linux artifacts. Native CI runs tests and installer checks on Ubuntu, macOS and Windows (RC22–RC27); one-command installers verify checksums and smoke-run the binary. | No release has been published (awaiting the user's go-ahead); release automation is deferred until features are complete locally. |

Recovery for a mutating operation starts by inspecting its recorded state. Automatic fallback must preserve the original approval scope and avoid repeating a tool whose effects are uncertain.

## ChatGPT account connection

GoHarness uses the [official Sign in with ChatGPT flow](https://developers.openai.com/siwc/token-sharing-open-source/sign-in) and [public Responses endpoint](https://developers.openai.com/siwc/token-sharing-open-source/models-and-inference). Browser authorization, tool results, and resumed history passed on this Mac. The current implementation has one authorized account and a local catalog default. Renewal and redirect restrictions were tested with fixtures. Live revocation, token expiry, account policy failures, real third-party integrations and native tests on other systems remain open.

ChatGPT preview restrictions include a stateless HTTP stream and no request-level output token cap. GoHarness uses local context and turn limits, and presents its approved local tools through a namespace. [Preview limitations](https://developers.openai.com/siwc/token-sharing-open-source/preview-limitations)

## RC20 turn undo and redo

| Workflow | Implemented | Remaining |
| --- | --- | --- |
| Undo/redo | `/undo`, `/redo`, palette entries and `goh changes undo-turn|redo-turn|revert-turn`. RC38: revert any earlier turn from the timeline (matched by prompt content). File checkpoints from the turn are verified and reverted newest first; a refusal rolls back partial work. Conversation continues in a branch; the source journal is untouched. Unrestorable tools are named. Unit tests and a real PTY scenario. | A diff preview before confirming a revert (diffs are available separately in `/changes`). |

## RC29 damaged sessions

| Failure | Behavior | State preserved | User sees | Evidence |
| --- | --- | --- | --- | --- |
| Torn tail after a crash | Repaired on open; `recovery` record appended | All complete records | Session opens normally | Existing session tests |
| Damaged header | Skipped in listings | Other sessions; damaged file unchanged | stderr line with salvage command | `TestListSurvivesOneDamagedJournal` |
| Damage mid-journal | Open refused with `ErrDamaged` | Damaged file unchanged | Error naming `goh sessions salvage ID` | `TestSessionsSalvageCLI` |
| Salvage | New session from readable prefix; incomplete tool group dropped | Turns before the damage | JSON report (kept records, damage offset) | `TestSalvage*` |

## RC34–RC64 recovery additions

| Situation | What happens | Evidence |
| --- | --- | --- |
| Model repeats the same call | Third identical call needs approval; headless runs stop with `needs_permission` | `TestRepeatedCall*` |
| Checkpoint storage unusable | Default mode edits anyway and reports `checkpoint_skipped`; strict mode refuses with the file unchanged | `TestDefaultCheckpointsNeverBlockEdits` |
| Background job left running | Stopped with its process tree when the chat, request or prompt ends | `background_job` PTY scenario, `TestJobStopKillsProcessTree` |
| Remote MCP redirect or missing token | Server is refused or skipped; chat continues | `TestRemoteMCPRefusals` |
| Older binary opens a newer journal | Reported as an unsupported journal version, not damage | `TestInconsistentVersionAndTimeRejected` |
| Interrupted typed `!command` | Recorded like a model shell call, so resume reports an uncertain effect | `TestUserShell*` |

Verified live on ChatGPT (RC72): image input, reasoning effort, end-of-request task state, plan handoff, streaming, tools and resume. Still unverified live: Responses reasoning summaries (RC80), provider failover, ChatGPT token expiry and the RC95 401-renewal retry, task-state cache savings, Anthropic thinking and API-key provider model lists.
