# Parts 1–6: initial observable contracts

Facts below are supported by `research/FEATURE_MAP.md`; proposed Go behavior is an original design.

## Text and streams

The local endpoint receives Chat Completions JSON with messages and tools. Complete SSE events contain choices/deltas and finish reasons; tool argument JSON can arrive in several chunks. The reference combines text into a completed headless text event and preserves tool IDs. The Go implementation intentionally publishes versioned delta events as well as committed message events. It must not execute a tool before the full assistant response has finished successfully.

## Read/edit/test

Starting file `message.txt` contains bytes `alpha\n`. A read exposes these contents. A single exact edit changes the file to `beta\n`. A shell test checks the final bytes and reports success. Tool request/result pairs retain the original ID, and each tool result is available in the next provider request. A failed argument validation returns an explicit tool error without file changes.

Go edit preconditions are intentionally stricter than the reference: require the expected SHA256 of the entire file and exactly one old-text match. Reject stale, missing, empty, or ambiguous matches before publication. Keep unaffected bytes, original line endings and permission bits. New file creation must not overwrite a concurrently appearing destination.

## Permission behavior

Reference explicit edit denial removes the edit tool and forced execution is routed to its invalid-tool handler; this bounded observation is not an interactive permission experiment. The original Go policy automatically permits project reads and requires per-operation or explicit run/session authority for mutations, shell commands, MCP admission and invocation. Headless unavailable authority returns needs_permission. Resumption does not restore authority.

## Shell lifetime

The Go contract owns a Unix process group, drains both pipes, bounds retained output, sends termination on cancellation, waits at most two seconds before escalation, and reports incomplete cleanup. It promises no rollback or cleanup of deliberately detached children. The reference SIGINT shell experiment produced partial evidence and a REA cleanup error; it cannot establish successful reference descendant cleanup. This gap must remain visible.

## Not established

Reference stale-file behavior, permission-dialog actions, resume recovery, compaction, TUI resize/restore, and MCP disconnect require their own controlled observations. Unit tests of Go behavior do not establish reference parity.
