# Explicit local LSP client

Pinned source assisted map: packages/opencode/src/lsp/* and tool/lsp.ts, plus
core's alternate filesystem/tool services. Actual reference experimental LSP
capture: lsp-reference.json,
`ev_4b489939c8d168717906327a50297f8040c3b26cfd14904e966523cf175280ce`.
The independently authored research/lsp-fixture/server.go answers real framed
stdio JSON-RPC. No installed production language server or auto-download needed.
Reference observed hover/definition/references/documentSymbol/workspaceSymbol/
implementation; request and server-message logs remain in research. The
reference input character 4 was transmitted as raw zero-based 3 for text a😀bc.
Do not infer unsupported operations or universal LSP compatibility from that.

Original Go single `lsp` tool lazily starts only an explicitly configured local
command selected by the longest matching extension suffix. Tool invocation and
server start need separate approvals (`lsp` and `lsp_start:NAME`). A start record
is synchronized before spawn; initialized handshake resolves the start record.
Approved initialization failure leaves uncertainty; cancellation/disconnect is
never automatically replayed. Process groups and protocol shutdown/TERM/KILL are
owned, concurrent manager cleanup happens before the terminal run event.
Plan mode and child agents currently disable all executable integrations.

Input: operation, filePath, positive 1-based line/character, optional query.
Operations: hover, goToDefinition, findReferences, documentSymbol, workspaceSymbol,
goToImplementation, diagnostics. Go columns count Unicode scalars and are
converted to UTF-16 units; this deliberately differs from the observed reference
raw UTF-16 offset convention. Non-UTF16 servers are rejected. File reads remain
anchored to the project, regular UTF-8 only, no symlinks/special files, 512 KiB cap.
Full-text didOpen/didChange notifications keep versions per run. Pull diagnostics
or bounded push diagnostics are supported; no automatic formatter/edit behavior.

Protocol limits: 1 MiB frames, 8 KiB headers read incrementally, 15-second operations,
64 KiB results (including envelope), 64 cached documents/diagnostic URIs, 8 concurrent
server-to-client requests, 64 pending client requests. The client supports bounded
configuration responses and rejects server-initiated edits/commands. Explicit
commands run with local process privileges; approval is not process sandboxing.
No server installation, automatic restart, workspace edits or remote transports.

Go REA: go-lsp.json,
`ev_335e58ecb62596742a1b927fdc65d6b671380ead1724a0cd34232becfb7d12fa`:
all six reference fixture operations complete with initialized/query result
pairing, owned process cleanup verified. Rendered output is truncated. Go hover
position is 4 at scalar column 4, recorded as a deliberate input convention
difference. Diagnostics and refusal/failure behaviors are independently tested,
not falsely labeled paired reference coverage.

Original Go helper tests independently frame protocol messages, verify UTF16,
full-text sync, push diagnostics, unsupported server-edit rejection, missing grants,
invalid paths/FIFO, capability/encoding/start errors, disconnect/cancel, result
bounds, out-of-order response pairing and blocked writes. Runtime tests verify
both durable startup/query pairs and uncertain recovery/no replay. Frame fuzzing
uses local strings only. Rich call hierarchy/type navigation, automatic language
server discovery/download and native-server compatibility remain future work.

Configured servers may set language_id to override the built-in extension-to-language mapping; suffix selection prefers the longest configured match.
