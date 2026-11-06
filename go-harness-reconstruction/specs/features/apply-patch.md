# T07: exact multi-file patch tool

## Reference identity and investigation

OpenCode v1.19.5 at810c0eba1a7348036f8bdf61e6632d0c36c6876c; host,
artifact checksum and license are in research/reference-manifest.json. The
existing REA skill/runtime/process guides were read and installed
rea-agents6.8.0 CLI capabilities inspected. Source assistance was limited to
registry/tool interfaces, permission/publication boundaries and test scenarios
in research/reference/source/packages/opencode. The Go parser and publisher
were independently authored from the contract below; upstream code/prompts
were not translated or incorporated.

Registry source establishes that model IDs containing `gpt-`, excluding `oss`
and `gpt-4`, expose apply_patch in the pinned reference. A local scripted
`fixture/gpt-fixture` model exercised the real tool, and saved requests confirm
it was advertised. Arguments use `patchText`.

Reference capture: research/evidence/patch-reference.json,
`ev_3409d4c5b6339f02c4738acfd4a9e4f84d72fb11a4c471598f30b88a28634cf2`.
Original scripted model: research/patch-model.mjs. Configuration and scenario:
research/patch-opencode.json, research/scenarios/patch-reference.json. Actual
provider requests/results: research/evidence/patch-reference-requests.jsonl.
Fixture starts from research/patch-fixture-initial.json in a disposable project.

Observed facts across10 calls:

- Add creates text; update exact context, delete and update+move succeeded.
- A patch with an initial create and a later missing source failed before the
  initial creation. `should-not-exist.txt` remained absent.
- Repeated matching text selected a match; whitespace-different context matched.
- Add File overwrote an existing destination in this fixture.
- A file externally extended before its update was still edited; external
  appended text survived. No version-hash parameter was supplied or required.
- Empty patch failed.

Source establishes automatic missing-parent creation, formatting and LSP
notifications. These were not inferred as observed effects in this fixture.
No automatic formatting/LSP is implemented by the Go patch tool.

## Original Go contract

Tool name `apply_patch`; arguments:

```json
{
  "patchText": "*** Begin Patch\n*** Update File: target.txt\n@@\n-old\n+new\n*** End Patch",
  "expected_sha256": {"target.txt": "full SHA-256 from read_file"}
}
```

`expected_sha256` may be omitted for add-only patches. It is required for every
existing update/delete/move source, keyed by canonical project-relative path.
Unused keys fail. This deliberately preserves the harness's stricter stale-file
contract. Permission is requested once for the entire patch as `apply_patch`;
headless operation requires an explicit run grant. Prepare returns a complete
path/operation inventory and bounded BEFORE/AFTER excerpts for every target.
Prepare performs no mutation. Execute repeats all validation after approval.

Grammar is the Begin/End Patch envelope, Add/Update/Delete File headers, optional
Move to after Update File, and update hunks starting `@@` or `@@ exact anchor`.
Hunk lines start space(context),minus(remove),plus(add). End of File anchors a
hunk at the source end. Exact anchor and context matches must be unique in the
remaining source; hunks cannot overlap. Insertion-only updates require an empty
file or an explicit End of File anchor. Move-only updates are supported. This
is a declared subset; fuzzy whitespace/Unicode matching, automatic context
repair and alternate envelope formats are unsupported.

File operations remain anchored with os.Root. Absolute/escaping paths, symlink
components, special files, non-UTF-8/NUL source files, duplicate participating
paths and missing parents are rejected. New/move destinations must not exist;
published creations use exclusive hard-link publication. Existing replacements
stage and synchronize sibling files, recheck source SHA immediately before
publication, preserve permission bits and rename atomically. Deletes check
source version then remove; moves publish new content exclusively then remove
the checked source. Parent directories are synchronized. Existing CRLF and
unaffected bytes are preserved; replacement lines follow the original newline
style, including an existing missing final newline at EOF.

All sources/destinations validate, the exact success result is budget-checked,
and all write files stage before the first visible publication. Sources are
rechecked collectively after staging and individually before each publication.
There is no multi-file transaction or universal compare-and-swap with unrelated
programs. A late failure/cancellation can leave earlier publications completed.
Such outcomes return structured `status: interrupted`, a bounded published-path
list, total published count, truncation flag and explicit uncertain-effects text.
The shared runtime retains the durable start and requires recovery acknowledgment;
it never automatically retries the patch. Move destination publication can
succeed while a newly changed source remains; this also reports interruption.
No rollback is claimed.

Limits:1MiB argument JSON,100 operations,4MiB per text file,16MiB aggregate
before/after data,65536 patch lines,20million exact-context comparisons. Approval
and result payloads remain at most64KiB. Excessive planned result/path inventories
fail before mutation; errors are bounded and partial-result trimming preserves
uncertainty markers. Paths and operation details remain in the durable start
arguments when a partial-result path list is truncated.

## Verification and comparison

Go capture: research/evidence/patch-go.json,
`ev_58bd019ebb5dd9bf85fcaf88062a59f9d31f673bee41f7acdaa8f5984a07f051`.
Go scenario uses an isolated `bin/harness-patch` build, identical disposable
absolute project/observation paths and matching capture settings. The final
bounded-diagnostic adjustment has unit/race coverage; its error cases were not
part of this scripted capture. Earlier initial Go capture is retained separately.

Both captures exited0 with owned process groups verified. Semantic assertion
replay `python3 research/compare_patch.py` passes10 cases and verifies exact
expected final files, including preserved refusals and no premature creation.
Report: research/evidence/patch-semantic-comparison.json. This includes final
reference/Go file snapshots and all actual tool-result messages.

REA comparison: research/evidence/patch-comparison.json,
`ev_692c7acc056d1d6c914e996b61edd5fd3c3af99db3e91308b6cd40dd29d039f5`.
Broad process comparison is changed, as expected from deliberate refusal and
output differences; terminal/process dimensions are unknown. Both captures have
incomplete shutdown process samples and cumulative rendered-terminal omissions.
Raw terminal coverage and filesystem snapshot/hash coverage are retained within
budget. Inherited environment remains unrecorded; observed filesystem snapshots
are not syscall traces; executable prelaunch identity is not atomic OS image
binding. No whole-agent equivalence is claimed.

Meaningful tests in internal/tools/patch_test.go verify add/update/delete/move,
multiple hunks, exact uniqueness/anchors, stale changes between approval and
execution and during staging, create/move collisions, symlink/escape restrictions,
malformed grammar, CRLF/mode/no-newline preservation, no early publication on
validation failure, cancellation and late collision with partial effects,
complete bounded two-sided permission previews, planned-result rejection before
mutation and bounded uncertain-result envelopes. FuzzPatchGrammar passed196252
executions over3seconds before the final diagnostic-bound regression; a final
fuzz replay also passed. Full tools race/vet checks pass on macOS arm64; Linux
runtime has not been established by this feature work.

## Shared-runtime acceptance

Original `internal/runtime/patch_integration_test.go` exercises the actual tools,
including read→hash→multi-file patch with approval and CRLF/mode verification;
headless refusal, denial, run/session grants and reset, strict plan-mode refusal,
an external edit made during approval, and partial real-patch cancellation.
Partial outcomes persist as tool_observation and emit tool_updated without resolving
the durable start. Resumption blocks until explicit inspection/acknowledgment,
repairs transcript pairing, and does not replay the patch. This is meaningful
integration coverage beyond isolated parser/publication tests.
