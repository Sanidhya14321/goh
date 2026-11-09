# File-change checkpoints

This original package implements opt-in, per-operation checkpoints for explicitly
named project files. It is a deliberately narrower reconstruction than OpenCode's
Git tree snapshots and message-range revert. It works in a non-Git project,
preserves binary bytes, and never executes Git or changes the user's index,
branches, commits, chat history, or unrelated files.

Open a dedicated directory outside both the project and its ancestors; Close
releases its exclusive writer lock. Capture saves the before state durably, Seal
saves the after state, and Abort is permitted only when current files still match
the before state. A snapshot stores content-addressed private blobs and JSON
metadata. Files are capped at 4MiB, each side at 16MiB, and checkpoints at 64 paths;
a directory holds at most 1024 records. The package does not automatically prune
history or orphaned blobs after failed captures.

Undo and Redo verify all current hashes, existence, permission modes and target
blob integrity before changing any files. Each individual file is staged and
published, with durable application progress. A multi-file operation is not one
filesystem transaction. Cancellation or a publication/checkpoint failure can
leave partial effects; the applying record remains pending and later application
or capture is blocked. Inspect Read, Pending and the actual files, then explicitly
Acknowledge to preserve the evidence and retire that checkpoint. No recovery
operation replays an uncertain write. Acknowledged checkpoints are ineligible for
Undo/Redo.

Only regular files and existing directory parents are supported. Project-relative
paths exclude symlinks, traversal and `.git` metadata. Creation uses a no-overwrite
link; replacement uses a staged rename. External writers can still race after a
final validation; this is not a universal compare-and-swap or protection against
concurrent actors with unrestricted filesystem access.

Diff compares stored before/after bytes rather than the current working tree. Its
text preview uses full replacement lines, not a minimal unified diff, and shares
a 32KiB preview budget. Binary changes expose metadata without a text preview.
Callers must additionally bound serialized tool/CLI envelopes, since path and
entry metadata and JSON escaping add bytes.
