# Architecture

GoHarness (`goh`) is one statically linked Go binary (CGO disabled) for macOS, Linux and Windows.
All of its interfaces drive the same agent runtime, and every conversation is an append-only
journal that is the single source of truth for resume, fork, export and recovery.

For what each feature does, see the [user guide](USER_GUIDE.md). For why it was built this way,
see [design decisions](development/DESIGN_DECISIONS.md).

## Repository layout

| Path | Contents |
| --- | --- |
| `cmd/harness` | The `goh` binary: CLI subcommands, launcher, provider setup flows |
| `cmd/scripted-model` | Local model endpoint that replays scripted replies; used by every test |
| `cmd/acceptance` | CLI acceptance scenarios against the compiled binary |
| `cmd/bench` | `goh-bench`, the benchmark runner (tasks in `bench/`) |
| `internal/` | All product packages (below) |
| `scripts/` | Installers, release script, and `e2e/` terminal test suites |
| `docs/` | User guide, architecture, benchmarks, development records; `plan/` holds the original plan and behaviour specs |
| `licenses/`, `THIRD_PARTY_NOTICES` | Exact third-party license texts |
| `research/` | Local reference material, mostly not tracked; never copied into product code ([origin ledger](ORIGIN_LEDGER.md)) |

## Packages

| Group | Package | Role |
| --- | --- | --- |
| Front ends | `tui` | Bubble Tea terminal UI; renders the runtime's event stream and owns no agent state |
| | `acp` | Agent Client Protocol v1 over stdio, for editors |
| | `localapi`, `remote` | `goh serve` loopback HTTP API, and the client `goh attach` uses |
| Engine | `runtime` | The agent loop: prompts, tool dispatch, permissions, hooks, compaction, subagents, verification |
| | `provider` | Chat Completions, Responses (including the ChatGPT sign-in) and Anthropic Messages adapters |
| | `core` | Configuration, providers and routing, ChatGPT OAuth, credential storage, shared types |
| | `session` | Crash-recoverable JSONL journal, export and import, salvage, statistics |
| Tools | `tools` | File reads and exact edits, patches, glob and grep, shell, background jobs, repository map |
| | `snapshots`, `projectsnap`, `rewind` | Per-file checkpoints, whole-project Git snapshots, turn-level undo and redo |
| | `sandbox` | Shell confinement: sandbox-exec on macOS, Landlock on Linux |
| | `mcpclient`, `lspclient` | MCP servers (stdio and Streamable HTTP), language servers |
| | `webfetch`, `websearch` | Bounded URL fetching with private-address refusal; SearXNG-compatible search |
| | `skills`, `commands`, `memory`, `formatters` | Markdown skills, custom slash commands, project notes, formatters |
| Support | `ignore`, `structured`, `platform`, `secretstore`, `testutil` | `.gitignore` matching, child-result validation, OS differences, OS keychain, test fixtures |

```mermaid
flowchart TB
    H["cmd/harness"] --> TUI[tui] & ACP[acp] & API[localapi]
    TUI & ACP & API --> RT[runtime]
    RT --> PR[provider] --> CORE[core]
    RT --> CORE
    RT --> SES[(session)]
    RT --> TL[tools] & SB[sandbox] & SN["snapshots · projectsnap · rewind"]
    RT --> INT["mcpclient · lspclient · webfetch · websearch"]
    RT --> EXT["skills · commands · memory · formatters"]
```

## One prompt, end to end

```mermaid
sequenceDiagram
    autonumber
    participant U as User (TUI / CLI / ACP)
    participant R as runtime.Agent
    participant J as session journal
    participant P as provider adapter
    participant M as Model API
    participant T as tools
    U->>R: prompt (+ @files, /command expansion)
    R->>J: append user_turn
    R->>R: build context (system prompt, skills, memory, masking / compaction if near the limit)
    R->>P: request (tools, history)
    P->>M: HTTPS stream
    M-->>P: text deltas / tool calls / usage
    P-->>R: events (streamed to UI)
    R->>J: append assistant message + usage
    alt tool calls returned
        R->>R: policy check (rules, approvals, loop guard)
        R-->>U: approval prompt (if needed)
        U-->>R: allow / deny
        R->>J: tool_start (effect intent)
        R->>T: execute (hash-guarded, with timeouts)
        T-->>R: result (large output saved as an artifact)
        R->>J: tool_result
        R->>P: continue with tool results
    else final answer
        R->>J: run_finished
        R-->>U: done
    end
```

Large tool output is kept as an artifact and only a bounded preview enters the context; the model
can page through the rest with `read_artifact`.

## Tool calls and safety

Enforcement happens outside the model. Every tool call passes the same pipeline whether it came
from the TUI, the CLI, the API or an editor.

```mermaid
flowchart TD
    TC[Tool call from model] --> L{Same call a 3rd time?}
    L -->|yes| ASK
    L -->|no| K{Tool kind}
    K -->|read-only: read, grep, glob| RUN
    K -->|file write| FR{file_rules}
    K -->|shell| SR{shell_rules}
    K -->|MCP / LSP start| ASK
    FR -->|deny| DENY[Refuse, tell the model]
    FR -->|allow| HASH
    FR -->|ask| ASK[Ask user]
    SR -->|compound command| ASK
    SR -->|allow / deny / ask| ASK
    ASK -->|allow| HASH{Hash still matches?}
    ASK -->|deny| DENY
    HASH -->|no| STALE[Reject: stale view]
    HASH -->|yes| SNAP[Checkpoint before state]
    SNAP --> RUN[Execute]
    RUN --> FMT[Formatter, optional] --> LSPD[LSP diagnostics, optional] --> RES[Result to journal]
```

- **Hash-guarded edits.** `edit_file`, `write_file` and `apply_patch` need the file's current
  SHA-256, so a stale view never overwrites a newer change.
- **Scoped approvals.** "Allow for this session" covers a command prefix, a web origin or a tool,
  never a compound shell command or one that runs code given as an argument.
- **Filtered environment.** Shell commands get a scrubbed environment without provider secrets.
- **Sandbox (opt-in, `sandbox.enabled`).** Shell commands may write only inside the project and
  temporary folders, never `.git/hooks` or `.git/config`, and have no network unless allowed.
- **Loop guard.** A third identical call needs fresh approval.
- **Repository files** (custom commands, skills, agents) cannot choose the provider, model or agent.

## Providers

```mermaid
flowchart TD
    A[Request] --> B{Active profile protocol}
    B -->|openai_chatgpt| C[Responses @ official endpoint<br/>OAuth token, auto-renew]
    B -->|openai_responses| D[Responses API]
    B -->|chat_completions| E[Chat Completions gateway]
    B -->|anthropic_messages| F[Anthropic SSE]
    C & D & E & F --> G{Result}
    G -->|success| OK[Stream to runtime]
    G -->|429 / 5xx / timeout,<br/>before any body| H[Bounded retry]
    H -->|still failing| I{fallback_profiles<br/>configured?}
    I -->|yes| J[Next profile, journaled and shown<br/>provider cooled down]
    I -->|no| ERR[Clear error, session kept]
    G -->|partial stream failed| ERR2[Stop: never auto-replay<br/>user retries explicitly]
    G -->|401| AUTH[Send user to provider setup<br/>draft and journal intact]
    J --> B
```

`OPENAI_API_KEY` and `ANTHROPIC_API_KEY` are sent only to the official HTTPS hosts. Failover to
another provider happens only through an explicit `fallback_profiles` list, and a partially
streamed reply is never replayed automatically.

## Sessions and recovery

```mermaid
flowchart LR
    subgraph Journal["~/…/sessions/ID.jsonl (append-only, private)"]
        r1[user_turn] --> r2[assistant] --> r3[tool_start] --> r4[tool_result] --> r5[compaction] --> r6[run_finished]
    end
    Journal --> RES[resume]
    Journal --> FORK["fork --before-turn N"]
    Journal --> EXP["export JSON / MD / HTML"]
    Journal --> STATS[stats]
    Journal --> RW[rewind / redo turns]
    Journal -->|torn tail / damage| SALV[salvage → new session]
    Journal -->|tool_start without result| UNC[uncertain effect:<br/>inspect and acknowledge, never replay]
    Journal --> TRASH[delete → trash → restore / prune]
```

Every record is fsynced before the runtime continues. A tool call that started but has no result
is an *uncertain effect*: it is shown to the user and never re-run automatically.

## Context management

```mermaid
flowchart TD
    A[Before each request] --> B{Tokens + output reserve<br/>over the window?}
    B -->|no| SEND[Send]
    B -->|yes| C[Mask old tool outputs<br/>keep active turns and tool groups]
    C --> D{Enough?}
    D -->|yes| SEND
    D -->|no| E[Summarise older turns with the<br/>summary role model, same failover list]
    E -->|ok| F[Write compaction record] --> SEND
    E -->|fails| G[Keep full history, report error]
```

## Configuration and credentials

```mermaid
flowchart LR
    UC["User config (JSON)<br/>profiles · roles · rules · MCP · LSP · prices"] --> CFG[core.Config]
    ENV["Env: OPENAI_API_KEY / ANTHROPIC_API_KEY<br/>(official HTTPS hosts only)"] --> CFG
    GW["Gateway api_key_env"] --> CFG
    CRED["Private credential files (0600), or sealed<br/>with a key in the OS store<br/>ChatGPT OAuth tokens, saved keys"] --> CFG
    REPO["Repository files<br/>.harness commands / skills / agents"] -->|"cannot set model,<br/>provider or agent"| RT[runtime]
    CFG --> RT
    CFG --> SHOW["goh config / goh auth list<br/>(redacted)"]
```

## Terminal UI structure

An interactive session (`goh`, `goh chat`, `goh attach`, `goh setup`, `goh settings`,
`goh providers`) runs one Bubble Tea program, the host (`internal/tui/host.go`). The command
palette, theme, changes, copy, jobs, timeline and approval views are overlays inside the chat.
Everything else (provider manager, connect, model and agent pickers, settings, project and
session pickers, and the chat itself after a navigation) is a screen the host shows in turn:
the launcher keeps its blocking style, `runProgram` hands each screen to the host and waits
until it quits. So the terminal never leaves the alternate screen between screens, keys typed
during a change of screen go to the next one, and a dialog over the chat is drawn over the live
chat, which reflows on resize. Messages from a screen's timers and requests are tagged with that
screen and dropped once it closes. A prompt that must read standard input directly (recovering a
damaged session) runs through `tui.Suspended`, which hands the terminal back meanwhile. Outside a
host, `runProgram` still runs a standalone program that closes only once typing pauses and
replays keys typed meanwhile into the next screen.
