# Snapshot and session revert investigation

This records pinned reference observations and source contracts. It does not
claim full OpenCode snapshot equivalence for the Go reconstruction. Reference:
v1.19.5, commit810c0eba1a7348036f8bdf61e6632d0c36c6876c. Artifact checksum,
license and host identity are retained in research/reference-manifest.json.
Source analysis used normal tools; runtime observations used installed
rea-agents6.8.0 after reading the existing REA/runtime/process guides.

## Source-established behavior

Relevant retained source files:

- packages/opencode/src/snapshot/index.ts.
- packages/opencode/src/session/revert.ts.
- packages/opencode/src/cli/cmd/debug/snapshot.ts.
- packages/opencode/src/server/routes/instance/httpapi/groups/session.ts.

Snapshots are enabled only for Git projects when effective `snapshot` is not
false. State uses a separate Git directory/index beneath OpenCode data keyed by
project/worktree. Source object databases may be shared through Git alternates;
the source index may seed initial snapshot indexing. Git configuration disables
filesystem monitoring and autocrlf and supports symlinks. An in-process semaphore
serializes operations for the private Git directory; this is not an observed
cross-process writer-lock guarantee.

Candidate paths are staged from tracked changes and untracked entries in the
selected directory scope. Source-repository ignore rules are checked against
candidate paths. Newly ignored paths are dropped from the private index.
Untracked regular files larger than2MiB are excluded. This size rule is not a
blanket exclusion of every already tracked large file. `track()` writes and
returns a Git tree hash. `patch(hash)` returns changed absolute paths; `diff`
returns a textual Git diff. Debug CLI exposes track, patch and diff, not direct
restore/revert.

`restore(tree)` reads the private index from the tree and forcibly checks out its
entries; source does not explicitly remove every extra path absent from the tree.
`revert(patches)` deduplicates changed paths, restoring each from the corresponding
recorded tree or removing it when absent there. It can batch nonoverlapping paths.
Some checkout/restore failures are logged or fallback rather than propagated as
fatal transactional errors. No global atomic rollback is implied.

Session revert rejects busy sessions, finds message/part boundary and subsequent
recorded patch parts, snapshots the current state, reverts affected paths and
stores revert metadata/diffs. Unrevert restores the saved pre-revert tree and
clears revert metadata. Cleanup after continuing from a reverted position removes
later historical messages/parts according to the saved boundary. Source facts
about part-level selection and cleanup were not all runtime exercised here.

## Actual Git snapshot CLI observations

Disposable research/snapshot-project was initialized with a local fixture-only
Git identity. Initial committed files: tracked.txt, delete.txt, .gitignore.
Initial untracked files: small untracked.txt, ignored.txt, and large.bin of
2MiB+1. No user's repository was changed.

Scenarios are research/snapshot-{track,patch,diff,disabled,nongit}-scenario.json.
Captures and Evidence IDs:

| Capture | Evidence ID | Observed result |
|---|---|---|
| snapshot-track.json | ev_dd0f72a082b025f7a05665c3635681f32bcceb38e7d368fba9fbea3ac6973f0e |40hex tree hash |
| snapshot-patch.json | ev_7bc6121400e0a7f6e6d80482611a3af3c8f82c7460f234ad870a81adc381c885 |changed absolute file paths |
| snapshot-diff.json | ev_c1c075bd1827bdbbc8a9bf7e39a712a569200aa2b963081245d458493796cff7 |textual Git diff |
| snapshot-disabled.json | ev_13bf08cfb6df3c396c979aae8e19b4f6538466a385265ae5aa1e7af38460e7fe |`undefined`, exit0 |
| snapshot-nongit.json | ev_f6bd4c5a5dc1d54e3e743318d5c556b40573d769e686b2cea9466712e4c6b8fb |`undefined`, exit0 |

All files are under research/evidence. After the initial track, the fixture
changed tracked.txt and untracked.txt, deleted delete.txt, created new.txt and
changed ignored.txt/large.bin. Patch and diff included the four expected small
paths, excluding the ignored and oversized untracked paths. The initial hash is
retained in research/snapshot-initial-hash.txt. Disabled behavior used explicit
`snapshot:false`; non-Git behavior used a separate plain directory.

## Actual session revert/unrevert observation

Original research/snapshot-revert-driver.mjs runs an actual pinned OpenCode
server bound to127.0.0.1 with a fixture-only password, plus a loopback scripted
model. It creates a session through the local API, requests two writes, records
messages and then invokes session revert/unrevert. All requests specify the
disposable Git directory through x-opencode-directory. No live provider or
remote server was contacted.

Scenario: research/snapshot-revert-scenario.json.
Configuration: research/snapshot-revert-config.json.
Capture: research/evidence/snapshot-revert.json,
`ev_aab2605341efecc28d2a612f42a1e35261a959927efef5bcc84ed129245ae642`.
The captured executable is the original Node orchestration driver; the actual
pinned OpenCode executable is its launched child. This is not a direct-binary
capture identity assertion. Returned session metadata reports version1.19.5.

Actual states:

| Stage | target.txt | added.txt | unrelated.txt |
|---|---|---|---|
| Before agent |baseline target |absent |baseline unrelated |
| After agent |agent changed |agent created |baseline unrelated |
| External edit before revert |external target change |agent created |external unrelated change |
| After revert |baseline target |absent |external unrelated change |
| After unrevert |external target change |agent created |external unrelated change |

Thus reference revert overwrote an external change to the agent-touched target,
removed the agent-created file and preserved the unrelated external change.
Unrevert restored the pre-revert target/created-file state. This is an observed
behavior, not a recommended safety policy. A Go stale-version refusal should be
an explicit deliberate difference.

Full report/messages/requests are retained in research/evidence/snapshot-revert-
{report,messages}.json and snapshot-revert-requests.jsonl. The report contains
revert metadata and snapshot/diff state; message records contain actual snapshot
and patch parts. `python3 research/snapshot-verify.py` passes CLI/state assertions
and writes snapshot-semantic-report.json.

## Limits and reproducibility

The final session driver capture exited0, overall truncated:false, owned group
verified, temporary root removed. Some sampled Git children have environment
identity withheld in cleanup metadata; inspect the full Evidence. CLI captures
all exited0 with owned groups verified but incomplete shutdown process sampling;
patch capture also lists environment-withheld processes. These limits prevent
claims of complete process equivalence. Filesystem observations are bounded
before/after snapshots, inherited environment is unrecorded, and executable
identity is sampled before launch. Network behavior is supplied by retained
controlled server/provider reports; REA process capture is not a network trace.

Initial session-driver capture had a fixture cleanup wait bug after all state
assertions completed. It ended by REA idle timeout and is retained separately as
snapshot-revert-initial*. The driver was corrected to bound child waiting and
close fixture keepalive connections, then rerun from reset fixture files. Only
the final exited0 capture is used for the completed runtime observation.

To reproduce, reset the disposable files to the contents above, use the saved
scenario environment/config paths, start the formatter model only for formatter
runs and invoke REA capture-process with --format json. The snapshot session
wrapper owns its model/server lifecycle. The debug CLI scenarios use separate
isolated reference data. Preserve output outside observed project roots.
Unknowns include interrupted restore, lock contention, symlink effects, unusual
filenames, worktree/gitdir variants, already tracked large files, concurrent
external mutations, historical cleanup and partial revert failure. No
transactional or crash-recovery guarantee was established by these observations.

All owned formatter/session fixture listeners were stopped, and ports18784–18786
were checked with no listeners after capture completion. No product code was
changed in this investigation.

## Original Go file checkpoints and intentional differences

The original internal/snapshots package implements opt-in per-file tool checkpoints
rather than the reference Git-tree/history mechanism. It works outside Git and
never executes Git, alters the user's index or rewrites conversation history.
Config.snapshots or --snapshots captures before create/write/edit/apply_patch and
seals after any admitted formatting. Shell, MCP/LSP and arbitrary formatter side
effects outside those paths are excluded. There is no automatic GC or checkpoint
blob transfer in session export/import/fork.

Dedicated private storage must be outside the project and its ancestors. A held
exclusive writer lock protects each store. Content-addressed binary-safe blobs
and versioned metadata record hashes, existence and file modes. Limits are64
paths,4MiB per file,16MiB per side and1024 records. Only relative regular-file paths
with existing parents are supported; links, special files, traversal and .git
metadata are refused. Read-only model tools serialize their own manager access;
external simultaneous writers are rejected.

changes list/diff/undo/redo SESSION --id CHECKPOINT and model snapshot_list/diff/
undo/redo expose history. Model restore tools need independent permission; CLI
undo/redo are explicit user restore instructions. Saved text diff is a bounded
full +/- replacement preview, not a minimal Git diff; binary diff is metadata.
All current hashes/modes/existence and saved target blobs are prevalidated before
restore. Later changes cause refusal, intentionally differing from observed
reference revert's external-change overwrite. Each path publishes independently;
external writers can still race final checks, and changes before Seal can be
included in the saved after-state. There is no multi-file atomicity guarantee.

Captured/applying states survive interruption and block later capture/restore.
Inspect actual files and progress, then changes acknowledge explicitly retires
uncertainty without restoring or replaying files. Acknowledged checkpoints cannot
be undone/redone. Runtime's unresolved tool starts separately require its existing
inspection/--acknowledge-uncertain flow. No automatic recovery replays an effect.

Go REA captures, retained under research/evidence:

- go-snapshot-undo.json: ev_3a190dcd2a19b557a1aaf83d30ace726b44851a3824bf5972c13c2a19d99f14c.
- go-snapshot-redo.json: ev_07af4ff33eb38531e17d962241fdfb3024d95c654222b96c454dd412750a4edf.
- go-snapshot-conflict.json: ev_99c0ca2c57a014b68bca14f8d99a91dd4d39999927fde2e74c0184e6c696890d.

Original scripts/capture_changes_batch.py verifies removal/restoration of a created
file including final formatted bytes, and exit1 with preserved external contents
on conflict. Owned groups were verified; process sampling remains incomplete.
Independent tests verify actual Git-index preservation, creation/deletion/modes,
binary bytes, reopening, invalid stores, corrupt metadata/blobs, FIFO/links,
partial cancellation, pending-state admission and explicit acknowledgement.
changes-semantic-comparisons.json is an original selected-fact assertion record,
not canonical REA equivalence. Whole-tree Git snapshots, message/part revert,
unrevert/history cleanup, worktree variants and GC remain unimplemented.

The newer packages/core/src/snapshot.ts exposes Location-scoped capture/files/
diff/preview/restore/checkout over newer Git abstractions. It was inspected as a
separate source surface; the retained runtime experiments exercise the pinned
CLI/server path above and do not establish behavioral equivalence across branches.
