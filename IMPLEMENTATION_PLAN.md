# Independent Go Coding Harness — REA Execution Prompt and Part-by-Part Build Plan

## 0. Copy-and-run implementation prompt

### How to use this file

Give the implementing coding agent this complete Markdown file and the OpenCode GitHub URL:

```text
GITHUB_URL=https://github.com/anomalyco/opencode
```

The GitHub URL is the only required project-identification input. The agent must resolve the repository, verify the reference pin, acquire the appropriate artifact or build it when necessary, prepare REA, investigate the actual harness, and implement the Go reconstruction part by part. Do not require the user to provide a local reference binary, a source-code tour, or manually written behavior specifications. Actual credentials, host access, or approvals that cannot be discovered remain explicit execution inputs; use scripted local models to keep development moving without live API credentials.

Use the following prompt together with the detailed specification in the rest of this file. This is an implementation instruction, not a request for another research report.

### Master prompt

```text
You are implementing an independently designed coding harness in Go.

REFERENCE INPUT
GITHUB_URL=https://github.com/anomalyco/opencode

OBJECTIVE
Use this GitHub URL as the starting point. Acquire and investigate OpenCode,
use the REA reverse-engineer-anything skill extensively to understand shipped
artifacts and actual runtime behavior, and build a working reconstruction in Go.
Build it part by part. Keep extending the same original Go codebase until the
first-release capabilities in this document work and are verified.

Do not stop after producing a plan, a source-code summary, an architecture
report, empty scaffolding, or interfaces without working behavior. Research is
part of implementation. Turn each completed investigation into a small working
Go capability, verify it, checkpoint progress, and continue to the next part.

FIXED PRODUCT DECISIONS
- Original code and original prompts; Apache-2.0 for original project material.
- Go core and built-in tools. External shells, Git, project test runners and
  explicitly configured MCP servers are allowed.
- macOS and Linux first.
- First release includes a headless CLI, TUI, and local stdio MCP client.
- Start with the specified OpenAI-compatible Chat Completions protocol.
- Allow project reads; request approval for writes, shell commands, and MCP
  execution unless an explicit session or run grant exists.
- Follow the detailed runtime, tool, persistence and recovery contracts below.
- Aim for the documented behavior of the selected reference features. Keep a
  parity matrix; do not silently substitute a generic chatbot or claim full
  OpenCode equivalence while features are missing or unverified.

START FROM THE URL
1. Inspect the workspace and existing progress before creating duplicate work.
2. Resolve the canonical repository and inspect its license and release data.
3. Use the pinned reference version and commit in this document unless the user
   explicitly supplies another revision. Verify that the repository and pin
   agree; record artifact identity and version rather than following a moving
   branch during the build.
4. Acquire the matching official artifact for the host, or build the pinned
   revision if no suitable artifact exists. Preserve acquisition/build evidence.
5. Prepare disposable projects, isolated reference configuration, and an
   independently written scripted model endpoint for repeatable experiments.

USE REA THROUGHOUT THE BUILD
Read the REA reverse-engineer-anything skill and applicable guides. Discover
actual connected tools or the installed CLI capabilities before calling them.
Prefer an existing installation; when absent, acquire the pinned REA package
locally within the authorized workspace scope. A skill alone does not supply
an MCP connection. Prefer the CLI when it avoids unnecessary global registration
changes. Report actual missing permissions or prerequisites precisely.

For each runtime-observable feature, collect REA evidence from the reference,
record the observed contract, implement the Go capability, then capture the Go
version and compare the relevant behavior. Keep evidence IDs or saved Evidence
records, artifact digests, invocation settings, coverage and limitations.

Use the available REA operations for process capture, interactions, process
lifetime, filesystem effects, retained evidence, feature tracing, and artifact
inspection as applicable. For supported Bun standalone artifacts, inventory
embedded modules, extract selected stored content, and analyze the shipped
JavaScript when this answers a real feature question. Inspect native boundaries
only when the feature actually depends on them and a suitable provider exists.

Use normal repository tools for ordinary complete-source architecture analysis,
as the REA skill directs. REA must be substantial in runtime investigation and
comparison, not a ceremonial setup step. Do not run irrelevant decompilation,
force every source file through REA, or repeat identical captures to inflate tool
usage. Do not claim REA evidence when only source inspection or shell output
exists. Do not claim Bun bytecode was decompiled when it was only inventoried.

If REA is unavailable for a required observation, diagnose that specific blocker,
continue independent supported tasks, and mark that observation blocked. Do not
silently replace REA evidence with assumptions or mark the affected part complete.

REPEAT THIS LOOP FOR EVERY PART
A. Investigate: ask bounded feature questions and inspect the reference with
   applicable REA artifact/runtime tools, supported by repository inspection.
B. Specify: write observable inputs, outputs, state changes, errors, permission
   behavior, cancellation and recovery rules. Mark facts, inferences and unknowns.
C. Implement: add the smallest working Go capability that satisfies that contract.
D. Verify: run independent fixtures, required failure-path tests, and reference/Go
   comparisons. Explain intentional differences rather than hiding them.
E. Checkpoint: update progress, compatibility, evidence and the next exact task.

IMPLEMENT IN THIS ORDER
Part 0  Workspace, URL resolution, reference pin, Go/REA readiness and provenance.
Part 1  Reference feature map, scripted-model driver and initial REA captures.
Part 2  Go executable, configuration, event stream and a working text interaction.
Part 3  Streaming provider adapter, errors, cancellation and multi-turn runtime.
Part 4  Read-only file tools, validated calls and bounded read scheduling.
Part 5  Permissions, exact-match edits, file creation and stale-file protection.
Part 6  Shell execution, process groups, output limits and cleanup.
Part 7  Durable sessions, resume and uncertain-effect recovery.
Part 8  Context accounting, budgets and basic compaction.
Part 9  TUI connected to the existing runtime.
Part 10 Local stdio MCP tool discovery, execution and lifecycle.
Part 11 Cross-feature comparisons, release verification, notices and binaries.

WORKSPACE AND CHECKPOINTS
Unless an existing project location has already been selected, use
<current-workspace>/go-harness-reconstruction/ with separate research/, specs/
and implementation/ areas. Do not write code into the reference checkout.
Maintain PROGRESS.md at this project's root. Record each part as not_started,
in_progress, blocked or complete; include changed files, tests actually run,
REA evidence, remaining gaps and the next executable task.

On continuation, read PROGRESS.md, inspect the actual code and test state, and
resume the first incomplete part. Reuse existing reference identity and evidence.
If the session ends before completion, leave a precise checkpoint. Do not label
an unfinished reconstruction complete because a conversation or tool budget ends.

PROVENANCE AND HONEST CLAIMS
Keep reference source, extracted code and raw evidence in research/. Implement
from reviewed behavioral contracts and independently authored fixtures. Do not
translate recovered code or copy upstream prompts and call the result original.
Track deliberate reuse and retain its applicable notices. Source separation and
fresh AI contexts are not legal certification. Use your own product identity.
Do not promise identical stochastic model answers, universal compatibility,
legal clearance, or speed improvements that have not been measured.

AUTONOMY AND COMPLETION
Continue through the parts without asking for confirmation after every routine
step. Ask only for genuinely missing inputs, necessary host permissions or
consequential scope changes. Do not publish, create remote repositories, spend
on live model calls, or make unrelated global configuration changes without
appropriate authorization. Keep live-provider tests explicit and run local
scripted tests independently.

After each part, report what was learned through REA, what now works in Go,
what was tested, what remains uncertain and what you will build next. Then
continue while independent authorized work remains.

Start now with Part 0, followed by Part 1 and the first working Go slice. The
intended output is a functioning, tested Go coding harness built incrementally,
with reproducible REA-backed comparisons and the documented compatibility gaps.
```

### Status of this document

This edit prepares the prompt for a future implementation task. It does not start the reconstruction in the current document-editing chat. In that future task, supplying the URL with this prompt authorizes the staged implementation workflow; the implementing agent must not stop after saving or rewriting a plan.

---

## 1. Project decisions and execution boundaries

### Destination

This complete plan is saved in the following folder and Markdown document:

```text
/Users/sanidhyavats/Desktop/Go-Coding-Harness-Plan/
└── IMPLEMENTATION_PLAN.md
```

This file is both a ready-to-use implementation prompt and its detailed execution specification. The current revision is edited in place; future implementation work belongs in the separate reconstruction workspace described in the master prompt.

**Current status — 5 November 2026:** The implementation prompt has been revised to start from the OpenCode GitHub URL, require extensive applicable REA investigation and comparisons, and build the Go harness in ordered working parts. Reference research and planning exist; REA runtime captures, toolchain installation, harness implementation, dependency clearance, and benchmarks have not been performed in this document-editing task.

### Project objective

Build an independently designed coding harness in Go that reproduces selected, useful behaviors of OpenCode and provides a stable foundation for future custom features.

The first public alpha must let a developer:

1. Open a local project.
2. Configure a model endpoint and API credentials.
3. Ask the agent to investigate or change code.
4. Review proposed edits and shell commands.
5. Watch responses and tool activity stream into the terminal.
6. Cancel active work.
7. Resume a saved session.
8. Connect explicitly configured MCP tools.

### Confirmed decisions

| Area | Decision |
|---|---|
| Required reference input | `GITHUB_URL=https://github.com/anomalyco/opencode` |
| Reference product | OpenCode from `anomalyco/opencode` |
| Reference version | `v1.19.5` |
| Reference commit | `810c0eba1a7348036f8bdf61e6632d0c36c6876c` |
| Original project license | Apache-2.0 |
| First release | Headless CLI, terminal UI, and MCP client support |
| Initial provider protocol | OpenAI-compatible Chat Completions with streaming tool calls |
| Default permissions | Automatically allow project reads; request approval for writes and commands |
| Implementation language | Go runtime and Go built-in tools |
| External tools allowed | Shells, Git, project test runners, language servers, and optional MCP servers |
| Initial operating systems | macOS and Linux |
| Initial distribution | Standalone binaries |
| Development executable name | `harness`; final branding is a release task |

“OpenAI-compatible” will mean the specific tested protocol subset documented below. It will not imply compatibility with every endpoint that uses that label.

### Initial release boundaries

Include:

- Text conversations and streamed responses.
- File listing, reading, searching, creating, and editing.
- Shell command execution.
- Explicit permission decisions.
- Session persistence and recovery.
- Context limits and basic compaction.
- A terminal interface.
- Local MCP servers using stdio.
- Machine-readable events and local diagnostics.

Defer:

- Windows support.
- Desktop and browser interfaces.
- Full OpenCode API or configuration compatibility.
- Importing OpenCode sessions.
- JavaScript plugin compatibility.
- Provider subscription OAuth flows.
- Native Anthropic and Responses API adapters.
- Images, audio, browser control, and computer use.
- Subagents, distributed execution, and background agent fleets.
- Remote MCP authentication and hosting.

Maintain these deferred features in a compatibility matrix rather than silently expanding the alpha.

---

## 2. Reference investigation, REA, and independent development

### 2.1 Why OpenCode is the reference

OpenCode provides a capable reference with an MIT-licensed root project, headless JSON output, and server interfaces suitable for observation. The pinned source includes scripted model-server tests, demonstrating a useful boundary for deterministic investigation.

Use the pinned release rather than tracking its changing development branch. The target is selected external behavior; its internal TypeScript architecture does not become the Go architecture.

Sources: [pinned release](https://github.com/anomalyco/opencode/releases/tag/v1.19.5), [pinned license](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/LICENSE), [headless command](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/opencode/src/cli/cmd/run.ts), [scripted model tests](https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/opencode/test/lib/llm-server.ts).

### 2.2 Keep research and implementation material separate

Organize future project work into three areas:

| Area | Contents |
|---|---|
| Research | Reference artifacts, upstream checkouts, REA captures, raw observations, license records |
| Specifications | Reviewed behavioral contracts, compatibility matrix, independently authored scenarios |
| Implementation | Original Go code, original prompts, tests, and project documentation |

The implementation should consume behavioral specifications and independently authored fixtures.

Do not use recovered source, extracted prompts, or detailed source-derived pseudocode as automatic Go-generation inputs. If any upstream material is deliberately incorporated, record it as reuse or adaptation and retain its applicable licensing requirements.

This separation documents provenance. It does not certify legal independence. Upstream source has already been inspected during this research, and a fresh AI conversation would not erase that history.

### 2.3 Establish reference identity

Before running reference experiments:

1. Acquire the selected release from its official source.
2. Record the release, commit, platform, acquisition URL, and artifact SHA-256.
3. Verify publisher checksums or signatures where available.
4. Preserve the relevant license and attribution records.
5. Record the executable’s reported version.
6. Use an isolated configuration and disposable project.
7. Disable unrelated extensions and integrations.
8. Record all configuration that affects the experiment.

A missing publisher checksum must be recorded as a limitation; a locally calculated checksum establishes the identity of the tested bytes, not publisher authenticity.

### 2.4 Use REA for runtime evidence

Apply REA’s [reverse-engineer-anything skill](https://github.com/morluto/rea/blob/71f680d7c1aa7a45ba0325fe87b104744b0c105f/.agents/skills/reverse-engineer-anything/SKILL.md).

Use normal repository tools for complete-source architecture questions. Use REA when the question depends on a shipped artifact or observed execution.

Pin `rea-agents` to **6.8.0** for the initial investigation. Inspect the installed CLI help and capabilities before constructing scenarios. The inspected source revision and published package must be recorded separately; package metadata does not establish that their source identities are identical.

REA is development tooling. The finished Go agent must not require REA, Node, or Bun.

### 2.5 REA investigation procedure

**REA is a recurring workstream across the reconstruction, not a one-time setup task.** For every part with observable reference behavior, produce a reference observation, a Go observation once that slice exists, and a comparison or an explicit explanation of the observation's limits. Reuse applicable existing captures rather than rerunning unchanged experiments.

Use a feature dossier for each part:

| Field | Required content |
|---|---|
| Question | The exact behavior being reconstructed |
| Reference identity | Repository URL, pinned revision, artifact digest and configuration |
| REA operations | Actual tool/CLI calls and supported host/provider capabilities |
| Evidence | Evidence IDs or saved records with scope and coverage |
| Observed contract | Inputs, outputs, state transitions and effects |
| Implementation | Original Go behavior and its relevant files |
| Comparison | Semantic results, filesystem effects, process outcomes and intentional differences |
| Unknowns | Unavailable evidence, truncation, unsupported behavior and remaining questions |
| Exit gate | The tests and evidence required before the part can be complete |

For artifact-level questions, route the actual package or executable through REA's advertised artifact tools. Use `open_binary` and `inspect_artifact` where required by the selected route. For supported Bun packages, inspect embedded modules and use `analyze_javascript_application` or feature tracing on compatible extracted JavaScript when needed. Use `inspect_analysis_view` and evidence export where advertised to retain and revisit relevant findings. These are candidate operations, not a promise that every installed REA release or target supports them: inspect actual schemas and availability first.

Keep basic CLI, process, file-effect, permission and recovery questions grounded in process captures. Do not use static graph relationships as evidence that a call executed. Do not request native decompilation merely because the application has a native executable wrapper.

Start with these scenarios:

- A plain text response.
- Read a file, edit it, and run tests.
- Reject an edit against changed contents.
- Refuse a permission request.
- Cancel a running command.
- Resume after interruption.
- Recover from a dropped model stream.
- Handle an MCP server that disconnects.

Use `capture_process_scenario` and `compare_process_captures`, or their documented CLI equivalents. Capture semantic JSON events separately through the reference driver.

For paired comparisons:

1. Reset the disposable project.
2. Run the reference.
3. Save its capture outside observed directories.
4. Reset the same absolute project and observation paths.
5. Run the Go implementation.
6. Compare the captures and semantic outcomes.

REA comparison contracts include working directories, observation paths, and capture settings. Keep those settings equivalent. Differences in executable and arguments are expected. [REA process-capture documentation](https://github.com/morluto/rea/blob/71f680d7c1aa7a45ba0325fe87b104744b0c105f/docs/process-capture.md)

REA captures are bounded observations and do not establish equivalence beyond their coverage. The capture mechanism is not a sandbox.

### 2.6 Build an independent reference driver

Write a small Go test service that emits scripted Chat Completions responses. It must support:

- Text chunks.
- Fragmented tool arguments.
- Multiple tool calls.
- Delayed responses.
- Malformed payloads.
- HTTP errors.
- Connection termination.
- Missing usage information.

Express fixtures as semantic actions such as “read this path” and “replace this text.” Map those actions to each implementation’s documented tool interface.

The driver must compare:

- Required event order.
- Tool-call/result pairing.
- Final file contents.
- Exit status.
- Command effects.
- Permission behavior.
- Session recovery.

Do not require identical generated prose, internal IDs, or terminal layout unless a particular field is explicitly part of the compatibility contract.

### 2.7 Maintain a compatibility matrix

For every investigated behavior, record:

| Field | Meaning |
|---|---|
| Feature | Behavior being compared |
| Reference identity | Release and artifact checksum |
| Evidence | Capture or source reference |
| Contract | Independently written expected behavior |
| Status | Exact, compatible, intentionally different, deferred, or unverified |
| Acceptance test | Test demonstrating the implemented behavior |
| Limitations | What has not been established |

Initial editing will intentionally use stricter matching than OpenCode’s broader matching strategies. Record that difference explicitly.

### 2.8 Licensing and ownership

Apply Apache-2.0 to original project code and original documentation. Preserve the applicable licenses and notices for dependencies or incorporated material.

Maintain:

- An origin ledger for code, prompts, fixtures, documentation, and assets.
- A dependency inventory.
- Third-party license texts and required notices.
- Contribution terms establishing the rights the project receives.
- An accurate record of AI assistance and human contributions.

Do not replace another author’s notices with the project’s copyright statement. Apache-2.0 for the new project does not relicense upstream material. [Apache-2.0 terms](https://www.apache.org/licenses/LICENSE-2.0)

For India, Section 52(1)(ac) conditionally covers studying program functioning to determine underlying ideas and principles. Section 52(1)(ab) addresses essential interoperability information under specified conditions. These provisions are not general authorization to publish copied software. [Copyright Act, Section 52](https://copyright.gov.in/Copyright_Act_1957/chapter_xi.html)

Before public release, obtain a targeted review of actual reuse, ownership, dependency obligations, branding, and any contractual or protection-measure issues encountered. This plan does not constitute legal clearance.

---

## 3. Implementation specification

### 3.1 Toolchain and dependencies

Use these initial pins:

| Component | Selection |
|---|---|
| Go | `1.27.3` |
| Terminal framework | `charm.land/bubbletea/v2` at `v2.1.1` |
| MCP implementation | `github.com/modelcontextprotocol/go-sdk` at `v1.8.0` |
| JSON Schema validation | `github.com/santhosh-tekuri/jsonschema/v6` at `v6.0.3` |
| HTTP, JSON, logging, testing | Go standard library |
| Persistence | Versioned append-only JSONL journal |
| Built-in content search | Go regular expressions and file operations |

Record dependency versions in `go.mod` and `go.sum`. Do not use floating versions in release builds.

Build release binaries with `CGO_ENABLED=0`. Verify the selected dependency graph supports that setting. Race-detector tests may use a separate build configuration.

Sources: [Go downloads](https://go.dev/dl/), [Bubble Tea release](https://github.com/charmbracelet/bubbletea/releases/tag/v2.1.1), [MCP SDK release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0), [JSON Schema release](https://github.com/santhosh-tekuri/jsonschema/releases/tag/v6.0.3).

### 3.2 Runtime architecture

Use one agent runtime shared by the CLI and TUI.

```text
CLI / TUI
    ↓
Run coordinator
    ├── Session journal and context builder
    ├── Provider adapter
    ├── Tool scheduler
    │      ├── Permission policy
    │      ├── Built-in tools
    │      ├── Process supervisor
    │      └── MCP client
    └── Event stream and diagnostics
```

Use these subsystem boundaries:

| Subsystem | Responsibility |
|---|---|
| Agent | Own session state, control turns, process completed tool calls |
| Providers | Convert messages and stream provider responses |
| Tools | Validate and execute built-in operations |
| Process supervision | Own command lifecycle, output, cancellation, and cleanup |
| Sessions | Journal, replay, recovery, context projection |
| Policy | Evaluate and request permissions |
| TUI | Render events and collect user decisions |
| MCP | Connect explicitly configured external tools |
| Diagnostics | Structured logs, timings, and local troubleshooting |

One goroutine owns mutable session state. Workers return results through channels; they do not modify conversation history directly.

Do not add a general agent framework or public Go SDK to the alpha. Keep implementation interfaces internal while behavior stabilizes.

### 3.3 CLI and configuration

Provide these commands:

```text
harness run --project <directory> --prompt <text> --format text|json
harness chat --project <directory>
harness sessions list
harness sessions show <session-id>
harness resume <session-id> --prompt <text>
harness doctor
```

Defaults:

- `run` executes one user request and exits.
- `chat` opens the TUI.
- Resumption creates a new run within the saved session.
- Only one active writer may own a session.
- JSON output contains events only; logs go to stderr or a log file.
- The alpha does not expose an HTTP server.

Use JSON configuration to avoid an additional configuration parser. Precedence is:

**CLI flags → explicitly supported environment variables → user configuration → defaults.**

Project instructions may influence agent behavior but cannot grant permissions or start executable integrations. Load MCP command definitions from explicit user configuration.

Store API-key environment-variable names in configuration. Never store actual API keys in session records or normal logs.

### 3.4 Event interface

Create a versioned JSON event envelope containing:

- Schema version.
- Session ID.
- Run ID.
- Monotonic sequence number within the run.
- Timestamp.
- Event type.
- Event-specific payload.

Required event families:

- Run started and finished.
- Assistant message started, updated, and completed.
- Tool requested, started, updated, and finished.
- Permission requested and resolved.
- Context compacted.
- Error or recovery required.

Every run emits exactly one terminal event with one of these statuses:

`completed`, `failed`, `cancelled`, `needs_permission`, `budget_exhausted`, or `cleanup_incomplete`.

The TUI consumes this event interface. It must not maintain a second interpretation of agent state.

### 3.5 Provider contract

Implement Chat Completions over `net/http` with configurable endpoint, model, credentials, context window, and output budget.

The adapter must:

- Parse server-sent events across arbitrary network chunk boundaries.
- Assemble tool arguments by tool-call index.
- Preserve tool-call IDs.
- Accept usage-only chunks with empty choices.
- Validate complete arguments before any execution.
- Distinguish normal completion, length limits, provider errors, and interrupted streams.
- Reject unsupported required capabilities explicitly.

Official documentation confirms that streamed arguments can be incomplete or invalid and require application validation. [Streaming events](https://developers.openai.com/api/reference/resources/chat/subresources/completions/streaming-events)

Do not hard-code a model name. Require an explicitly configured model and context window for live operation.

Omit optional generation parameters unless the selected profile enables them. Make output-token parameter naming configurable because compatible endpoints differ.

Use the scripted local service for development. A real endpoint is required only for the live acceptance milestone.

Retry policy:

- At most two automatic retries for retryable connection failures or HTTP errors before response content has been accepted.
- Honor `Retry-After` within a bounded delay.
- Do not transparently restart a partially consumed response.
- Do not execute tool calls from an interrupted assistant message.
- Never retry a side-effecting tool automatically because the model request failed.

A future native Responses adapter remains separate. OpenAI recommends Responses for new OpenAI-specific applications; this initial Chat Completions choice serves the requested compatible-endpoint scope. [API reference](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)

### 3.6 Agent loop and scheduling

Each turn follows this sequence:

1. Commit the user input.
2. Build a bounded model context.
3. Stream an assistant response.
4. Commit the completed assistant message.
5. Validate its tool requests.
6. Obtain required permissions.
7. Record tool-start checkpoints.
8. Execute admitted tools.
9. Commit tool results.
10. Continue the model conversation or finish the run.

Initial limits:

| Limit | Default |
|---|---:|
| Model turns per run | 64 |
| Concurrent read-only tools | 4 |
| Concurrent mutations, shell calls, or MCP calls | 1 |
| Tool-argument payload | 1 MiB |
| Inline tool-result payload | 64 KiB |
| Text-file operation size | 4 MiB |
| Default shell timeout | 120 seconds |
| Maximum requested shell timeout | 600 seconds |

Schedule adjacent independent reads together. Treat mutations, shell operations, and MCP calls as barriers. Return results to the conversation in declared call order even when reads finish in another order.

Malformed tool arguments produce a tool error without executing the operation. Repeated identical invalid calls must hit a bounded loop guard rather than continue indefinitely.

### 3.7 File and search tools

Provide:

| Tool | Required behavior |
|---|---|
| `list_files` | Enumerate project files with bounded results |
| `read_file` | Return selected text, line information, and a full-file SHA-256 |
| `search` | Search text using Go regular expressions |
| `write_file` | Create a file or replace one against an expected version |
| `edit_file` | Apply an exact text replacement against an expected version |
| `shell` | Execute an approved command in the selected project |

File tools operate within the selected project root. Use traversal-resistant Go filesystem APIs and reject unsupported special files or escaping paths. File-tool restrictions do not sandbox approved shell commands.

Editing rules:

- Require the expected SHA-256 for existing files.
- Re-read and check it immediately before publication.
- Require exactly one match unless `replace_all` is explicitly requested.
- Reject ambiguous or absent matches.
- Preserve unaffected bytes and line endings.
- Validate all edits to one file before writing.
- Publish through a temporary sibling file and atomic replacement where supported.
- Preserve existing permission bits.
- Do not promise multi-file transactions or universal atomic compare-and-swap against unrelated programs.

New-file creation must fail if the destination appears before publication.

For Git worktrees, Git may supply the tracked/unignored file inventory. Actual reading and searching remain Go operations. For other directories, use a bounded Go traversal and documented exclusions.

### 3.8 Permission behavior

Default policy:

- Automatically allow file listing, reading, and searching inside the project.
- Ask before each proposed file mutation.
- Ask before each shell command.
- Ask before starting an explicitly configured MCP server.
- Ask before invoking MCP tools.

Show the actual path and diff for file changes. Show the command, working directory, and timeout for shell execution.

Offer explicit grants for the current session. Session grants expire when the session process ends; resumption does not silently restore authority.

Headless operation must not wait indefinitely for unavailable input. Emit `needs_permission` and exit unless the user supplied the relevant run-level grant.

MCP annotations and repository instructions cannot expand permissions.

### 3.9 Process supervision

On macOS and Linux:

- Start commands in an owned process group.
- Drain stdout and stderr concurrently.
- Keep an inline preview and a bounded retained artifact.
- On cancellation, stop new work and signal the process group.
- Allow a two-second graceful termination period, then escalate.
- Bound pipe-draining and final cleanup.
- Surface incomplete cleanup explicitly.

The supported lifecycle contract covers the owned process group. Commands that deliberately detach or escape it are outside that guarantee and must be documented.

A cancelled command may already have changed files or contacted services. Cancellation means execution was interrupted; it does not mean its effects were rolled back.

Go’s default command cancellation is insufficient by itself for the required group lifecycle. [Go process documentation](https://pkg.go.dev/os/exec)

### 3.10 Sessions and recovery

Use one versioned JSONL journal per session with an exclusive writer lock.

Persist:

- Session identity and project root.
- User messages.
- Completed assistant messages.
- Tool-start records.
- Tool results.
- Permission decisions.
- Compaction records.
- Run termination and recovery markers.

Synchronize the tool-start record before beginning a side effect. Synchronize completed messages and tool results before treating them as durable.

Transient streaming deltas need not be crash-durable; the TUI must not present them as committed history.

Recovery rules:

- Recover a valid prefix when the final journal record is incomplete.
- Refuse unsupported journal versions.
- Report corruption inside the journal rather than silently skipping records.
- Mark a started tool without a result as interrupted with an uncertain outcome.
- Never automatically replay that operation.
- Require explicit acknowledgment or resolution of uncertain side effects before continuing.
- Reacquire all permissions after resumption.

Use owner-readable storage permissions for session data and retained output artifacts.

### 3.11 Context management

Require a configured context window and reserve space for the next response and tool results.

Begin compaction near 75% of the configured window. Preserve:

- System instructions.
- The active user request.
- Unresolved tool calls.
- Recent complete tool-call/result groups.
- File identities and important task state.

Use an independently authored summarization prompt. Store the summary as a new journal record while retaining original history.

If summarization fails or does not reduce context enough, stop with a clear context-budget error. Do not silently remove unresolved tool relationships or recent user instructions.

### 3.12 Terminal UI

Use Bubble Tea over the existing runtime event stream.

Required UI capabilities:

- Multiline prompt input.
- Streamed assistant text.
- Tool activity and bounded output previews.
- Diff review.
- Permission dialogs.
- Cancel control.
- Session selection and resume.
- Model/profile display.
- Token and timing information when available.

Escape terminal control sequences from untrusted content. Restore the terminal on normal exit, error, and cancellation. Handle resizing without changing agent state.

The first cancellation interrupts the active run. Exiting the application is a separate action.

### 3.13 MCP client

Use the official Go SDK with stdio transport for the alpha. [SDK documentation](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0)

Support:

- Explicitly configured local server commands.
- Initialization and tool discovery.
- Namespaced tool names.
- Input-schema validation.
- Tool invocation and cancellation.
- Bounded output and timeout handling.
- Disconnect reporting and process cleanup.

Treat MCP tools as potentially side-effecting regardless of annotations. Do not automatically replay calls after disconnects.

Do not fetch remote JSON Schema references during tool validation. Servers requiring unsupported schema behavior must receive an actionable compatibility error.

Defer remote transports, OAuth, sampling, and automatic server installation.

---

## 4. Part-by-part execution roadmap

### Execution rule

Execute one small working slice at a time: **REA investigation → behavioral specification → Go implementation → tests and comparison → checkpoint → next part**. Begin the next part only when its dependencies are verified. Do not postpone all coding until an exhaustive analysis of the entire product is finished, and do not jump to a monolithic rewrite after the first overview.

The following remains an **8–12 working week planning estimate** for one experienced full-time Go engineer with AI assistance. Parts vary in size; this is not a one-part-per-day schedule. Reference runtime availability, host setup and newly discovered behavior can change the estimate.

### Ordered build parts

| Part | Reference and REA investigation | Working Go deliverable | Required exit gate |
|---|---|---|---|
| **0. Resolve and prepare** | Resolve the GitHub URL; verify the pinned repository/revision/license; acquire or build its matching host artifact; record digests; read the REA skill; verify actual CLI/tools and host readiness | Separate workspace, reference manifest, origin ledger, progress file and working Go toolchain | A real reference command and a small REA process-capture smoke scenario run successfully, or their specific blockers are recorded; no fabricated evidence |
| **1. Establish the reference** | Capture text interaction and a bounded read–edit–test reference scenario; inspect the relevant shipped artifact/modules where needed; record CLI events and file effects | Independently authored Go scripted-model service and reference driver; feature dossiers and initial compatibility matrix | Scripted reference flows are repeatable; evidence and unknowns are saved; initial implementation contracts exist |
| **2. First executable slice** | Use the reference text interaction and configuration observations as the behavior contract | Buildable Go executable, explicit configuration, versioned events and one end-to-end streamed text interaction with the scripted service | The executable runs from a clean build; JSON stdout is valid; a Go process capture and semantic comparison exist |
| **3. Provider and agent loop** | Probe streaming boundaries, multiple turns, network failures, finish reasons and cancellation using REA captures plus semantic traces | Chat Completions adapter, complete-message handling, bounded retries, cancellation, turn limits and runtime state transitions | Fragmented streams and failure cases pass; incomplete tool requests cannot execute; run termination is unambiguous |
| **4. Read-only tools** | Observe file listing, reading and search, including tool pairing and output limits | `list_files`, `read_file`, `search`, tool-schema validation and bounded read scheduling | Original fixtures verify bytes/hashes, pairing, traversal restrictions and result ordering; compare reference and Go file effects |
| **5. Permissions and edits** | Capture approval/refusal, exact edit cases, ambiguity, external mutation and new-file outcomes; inspect shipped edit behavior where relevant | Permission requests/grants, file creation, atomic publication where supported, `edit_file` and stale-file checks | No mutation without required authorization; stale and ambiguous edits fail without changing the file; differences from OpenCode's matching are explicit |
| **6. Shell and lifecycle** | Capture successful/failing commands, large output, interrupts, timeouts and process settlement | Approved shell execution, owned process groups, concurrent pipe draining, output artifacts and bounded cleanup | Real lifecycle tests pass on supported hosts; REA observations support the stated cleanup scope; uncertain effects are reported |
| **7. Sessions and recovery** | Observe reference persistence and resume; capture interrupted effects and compare externally visible recovery | Versioned JSONL journal, writer lock, durable checkpoints, session commands, resume and unresolved-effect handling | Truncated-tail and corruption cases are distinguished; pending mutations are never replayed automatically; resumed permissions are reacquired |
| **8. Context and compaction** | Probe long conversations and context-boundary behavior with bounded scripted responses; capture observable continuity and failures | Context accounting, output reserve, basic summarization and preserved tool/message relationships | Compaction preserves the active task and unresolved calls; inadequate or failed summaries produce an actionable budget result |
| **9. Terminal UI** | Use REA terminal captures for interaction, resize, approval, cancellation and exit behavior; compare declared interactions rather than requiring visual duplication | Bubble Tea UI over the same tested runtime, diff review, tool previews, session selection and terminal restoration | CLI/TUI share state semantics; resize/cancel/exit scenarios pass; original UI differences are documented |
| **10. MCP integration** | Observe tool discovery, admission, calls, disconnections and teardown using a controlled MCP server and applicable REA process captures | Local stdio MCP client, namespaced tools, validation, explicit admission and bounded lifecycle | A small independent Go MCP fixture server verifies discovery, calls, invalid input, cancellation, disconnect and cleanup without duplicate effects |
| **11. Whole-agent verification** | Repeat only relevant paired reference/Go scenarios after changes; compare REA evidence and semantic outcomes; document remaining coverage | Reproducible acceptance suite, measured performance, complete provenance/notices, docs and release binaries | Section 5 release gates pass; compatibility gaps and unverified claims are published accurately; no automatic remote publication |

Part 1 may implement the research driver before the product runtime exists. Keep that driver and upstream artifacts in research, with their own origin record. This does not authorize copying the reference's fixtures or implementation into the Go product.

### Required output after every part

Produce a short part report containing:

1. The reference behavior investigated and the actual REA evidence collected.
2. The capability now working in Go and the files changed.
3. Test commands actually run and their results.
4. Reference/Go comparison outcomes and intentional differences.
5. Remaining unknowns or blockers, with affected coverage.
6. The updated progress state and the next exact implementation task.

After reporting, continue to the next ready part within the authorized implementation task. Do not ask whether to proceed after every routine part. A blocked REA observation remains blocked; continue independent work without claiming its gate passed. Never treat an unfinished part as complete just to advance the table.

### Persistent progress and continuation

Maintain these records under the reconstruction workspace:

| Record | Purpose |
|---|---|
| `PROGRESS.md` | Current part, status of every part, completed checks, blockers and next executable action |
| `research/reference-manifest.json` | Source URL, selected revision, artifact identity, acquisition/build and runtime configuration |
| `research/FEATURE_MAP.md` | Investigated behavior and links to actual REA evidence |
| `research/evidence/` | Saved Evidence records, comparisons and invocation context |
| `specs/COMPATIBILITY_MATRIX.md` | Exact/compatible/different/deferred/unverified behavior and acceptance tests |
| `specs/parts/` | Reviewed per-part behavior contracts and acceptance conditions |
| `implementation/` | Independently authored Go code, prompts and product tests |

Before resuming, inspect these records and actual code/test state. Reuse valid reference identity and evidence. Do not recreate the project, repeat completed investigation, or declare a different reference version without recording that change.

### First working sequence

1. Resolve the supplied URL, inspect existing state and establish the reference/REA/Go prerequisites.
2. Acquire and identify the target; complete an actual REA smoke capture.
3. Build the independent scripted model service and reproduce a reference text interaction.
4. Write the first behavior contract and implement the first working Go interaction.
5. Capture and compare that Go slice; move through the remaining parts using the same loop.

These are dependency steps, not promised dates. The first week should leave real reference evidence, a reproducible driver, reviewed initial contracts and the earliest working Go slice that has passed its gates. Do not substitute another large planning document for these outputs.

---

## 5. Verification, release, and future features

### 5.1 Required acceptance scenarios

Build independently authored scenarios covering these groups:

| Group | Required scenarios |
|---|---|
| Streaming | Fragmented UTF-8, split SSE frames, empty usage chunks, early EOF, provider errors |
| Tool calls | Fragmented JSON, malformed arguments, duplicate IDs, unknown tools, correct result pairing |
| Files | Read–edit–test, ambiguous replacements, stale hashes, CRLF, Unicode, binary rejection, path escape refusal |
| Permissions | Approval, denial, session grants, headless refusal, permissions reset on resume |
| Commands | Success, nonzero exit, large output, timeout, cancellation, child-process cleanup |
| Persistence | Truncated final record, internal corruption, conflicting writers, crash before and after tool execution |
| Context | Successful compaction, failed compaction, unresolved tools preserved, budget exhaustion |
| TUI | Resize, cancellation, permission dialogs, terminal restoration |
| MCP | Discovery, schema failure, timeout, disconnect, cancellation, teardown |
| End-to-end | Multi-turn repository change followed by passing project tests |

Acceptance requires the expected outcome and failure behavior. A test that merely repeats implementation logic is insufficient.

### 5.2 Quality checks

Run meaningful checks as the relevant subsystem becomes available:

- Unit tests for state transitions and parsers.
- Integration tests for file publication, subprocesses, and persistence.
- Race-detector tests on exercised concurrent paths.
- Bounded fuzzing for SSE, tool arguments, and journal parsing.
- Runtime tests on both macOS and Linux.
- Cross-build checks for supported release architectures.
- Dependency vulnerability and license review.

Passing cross-compilation does not establish runtime correctness on that architecture. Publish the actual runtime test matrix.

### 5.3 Comparison and performance measurements

Keep correctness and performance results separate.

Use scripted model responses to measure harness overhead. Use repeated paired live-model tasks to assess practical coding outcomes.

Record:

- Cold and warm startup time.
- Peak process-family memory.
- CPU time.
- File-search and edit latency.
- Cancellation and cleanup latency.
- Successful task completion.
- Input/output tokens where available.
- Tool failures and retries.
- Provider and model identity.
- Hardware, operating system, fixture, and run count.

Do not claim improved reliability or speed solely because the implementation is Go. Publish comparisons only after equivalent workloads have run.

### 5.4 Release acceptance

The alpha is ready when:

- All required deterministic scenarios pass.
- No known silent file corruption remains.
- Interrupted effects are represented accurately.
- Required process-group cleanup works on supported hosts.
- CLI and TUI agree on runtime state.
- MCP failures cannot silently trigger duplicate tool execution.
- Session recovery is documented and tested.
- Credentials are excluded from normal logs and journals.
- The dependency and origin inventories are complete.
- Required license texts and notices ship with the binaries.
- Compatibility limitations are published.
- Documentation explains configuration, permissions, recovery, and external-tool requirements.

Release artifacts:

- macOS ARM64 and AMD64 binaries.
- Linux ARM64 and AMD64 binaries.
- SHA-256 checksums.
- Version and build metadata.
- License and third-party notices.
- Installation and troubleshooting documentation.
- Compatibility and benchmark reports.

Do not publish or create a remote repository automatically as part of preparing the plan.

### 5.5 Future feature sequence

Expand through separately tested milestones:

1. A native second provider adapter.
2. A constrained planning mode.
3. LSP-backed navigation and diagnostics.
4. Session branching and migration support.
5. Bounded subagents with separate permissions and cancellation.
6. Remote MCP transports and authentication.
7. Custom workflows and process-based extensions.
8. Project-specific capabilities selected from real usage.

Every feature must use the established runtime, event, permission, and recovery boundaries. Avoid adding a second orchestration loop inside plugins or UI components.

### 5.6 Assumptions and remaining execution facts

The plan fixes engineering defaults; these facts still require execution evidence:

- The selected OpenCode binary’s exact runtime behavior.
- Its artifact checksum.
- Successful REA capture on the host.
- A fully installed and working Go toolchain.
- Runtime compatibility of the locked dependency graph.
- A configured live model endpoint.
- Final authorship, dependency, and branding clearance.
- Measured performance.

Local inspection found Git, Node, Bun, and `xcrun`. Go was not found on PATH or in the common locations checked, and the REA command was not on PATH. Their installation or configuration belongs to the environment milestone.

**Prompt revision complete:** This file now explicitly starts from the OpenCode GitHub URL, requires substantial applicable REA investigation and comparison throughout the build, and directs a future implementing agent to recreate the harness in Go part by part. This document-editing task has not started those implementation parts. When the user supplies this file and the URL as an implementation task, begin Part 0 and continue through the ordered parts instead of stopping at a plan.
