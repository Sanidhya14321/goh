# Built-in tools

Permissions belong to the runtime: call `Prepare`, present its preview, obtain
permission for side-effecting operations, record a durable start, then `Execute`.
Execute repeats validation and stale checks after approval. Neither method grants
permissions. `Prepare` is side-effect free.

File paths are relative to the configured project. Go `os.Root` anchors filesystem
operations, and symlink components are rejected. Text files must be regular UTF-8
without NUL, at most 4 MiB. Input arguments are at most 1 MiB and tool results at
most 64 KiB. For long reads, select lines. Traversal skips `.git`, `node_modules`,
`.harness`, symbolic links and special files, visits at most 20000 entries, and
returns at most 10000 files. Searches return at most 1000 matches. Search explicitly
reports skipped unreadable, binary and oversized files.

`read_file` returns full-file SHA-256 plus the selected text and line range.
`edit_file` requires `expected_sha256`, `old_text`, `new_text`; ambiguous matches
are refused unless `replace_all` is true. `write_file` replaces a file only with
an expected hash, otherwise creates exclusively. `create_file` always creates
exclusively. Parent directories must already exist. Writes stage and synchronize
a sibling temporary file, recheck the current hash, preserve existing permission
bits, atomically rename for replacement or hard-link for exclusive creation,
and synchronize the parent directory. Original line endings remain unchanged.
There is no universal atomic compare-and-swap against unrelated external writers
between the final hash check and rename, nor a multi-file transaction. A failed
parent directory sync is reported as a possible completed publication.

`shell` accepts `command`, optional relative `cwd`, and `timeout_seconds` (default
120, maximum 600). Shell commands are not sandboxed. On macOS/Linux they run in
an owned process group, drain stdout/stderr concurrently, retain an 8 KiB preview
and a private output artifact capped at 1 MiB. Cancellation/timeouts signal the
group with TERM, allow up to two seconds, then KILL. Parent exit also settles
remaining members of the group. Retention truncation never stops pipe draining.
Cleanup guarantees cover the owned group and observed parent/pipe settlement;
deliberately detached processes are outside scope. Cancellation cannot undo
already performed effects. stdout/stderr are combined in observed drain order,
which need not equal source program write order. Artifact storage requires a
trusted private directory supplied by the caller.

`glob` discovers paths using `pattern` and optional relative directory `path`.
Patterns support recursive basenames, `**` directory components, `*`, `?`,
character classes, and bounded comma-brace alternatives. Results are at most100
lexicographically sorted project-relative files with explicit truncation. The
same directory/symlink exclusions apply. Its deterministic ordering and strict
pattern errors intentionally differ from the OpenCode reference. See
`specs/features/GLOB.md` at the project root for evidence and exact bounds.

`apply_patch` accepts `patchText` plus an `expected_sha256` map for all existing
sources. It supports exact Add/Update/Delete/Move patch operations and obtains
one permission decision for the complete inventory. Parents must exist. Every
source/destination and the planned result budget validate before publication;
all writes stage first. Late failure or cancellation reports interrupted effects
and requires runtime recovery acknowledgment. Publications are individual file
operations with no multi-file rollback guarantee. See
`specs/features/apply-patch.md` for grammar, evidence, limits and differences.
