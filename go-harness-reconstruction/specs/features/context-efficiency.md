# Bounded declaration context and output retrieval

Original tools inspired by source-backed comparison with Harvest/Aider/Crush;
research/comparisons/ASSESSMENT.md records exact pins/contracts/tests and the
Goose compaction-related follow-ons. No code/prompts were reused. These additions
are narrower than any upstream's complete feature set.

## repo_map

A read-only tool for Go files or directories, using go/parser and go/ast from the
standard library. Returns top-level function/method signatures and type/constant/
variable names, file lines and SHA256 hashes; bodies, literal values and comments
are omitted. It provides optional substring query filtering and exact-name-first,
then path/line ordering, not dependency/PageRank semantic ranking. A map is an
orientation aid, never proof of function behavior or authority to act.

Path is project-relative, checked through os.Root with symlink/special-file
refusal. Descendant .git/node_modules/.harness/vendor trees are skipped; .gitignore
is not interpreted. Source limits:256KiB/file,256files/8MiB,20,000 visited entries,
2,000 matched symbols,500 returned symbols (default200),512-byte signatures and
32KiB serialized results including escaped metadata. Parse errors/skips/truncation
are explicit. Existing plan mode permits the tool without process execution.
A32-entry FIFO parse cache stores bounded declaration data, guarded against
concurrent tool reads. Source SHA256 is checked each time, so same-size/same-mtime
external edits invalidate cached syntax. A warm hit still reads/hashes the file.
No disk cache, compiler, type-checker, tree-sitter or additional module is used.

## read_artifact

Shell and formatter results now include artifact_id, a basename in the current
session's existing private artifact directory. read_artifact accepts only those
shell-<digits>.log identifiers, no arbitrary path. It opens regular non-symlink
files<=1MiB and returns a requested byte page<=8192 (default4096), next_offset,
eof, retained length and SHA256. Tail mode selects the end of retained data and
cannot combine with a nonzero offset. Text output repairs split/invalid UTF8 and
flags repair; base64 allows byte-exact reconstruction across arbitrary page
boundaries. JSON result escaping is bounded below the shared64KiB envelope.

This retrieves only the existing capped output prefix; it does not retain the
actual end of a command whose discarded bytes exceed1MiB. Other sessions/child
artifact directories are separate. Model-guessed paths, links, FIFOs and oversized
files are rejected. Filesystem actors with unrestricted local access can still
modify session artifacts; digests report the bytes actually read rather than
establish immutable provenance. No general MCP/LSP result spilling, disk quota,
artifact GC or export/import/fork portability is implemented. The tool is
read-only and permitted in plan mode; it never reruns the producing command.

## Actual validation

Independent Go CLI/REA fixture scripts/capture_efficiency_batch.py maps32 generated
files twice, then emits70,017 bytes from an explicitly approved local command and
reads a64-byte tail page to retrieve a marker absent from the inline preview.
297,654 source bytes produce6,330 map JSON bytes, a97.87% byte reduction for that
fixture; the repeated scan reports32 parse-cache hits. This is not tokenizer/cost
or cross-agent benchmarking, and omitted bodies still require targeted reads.
The original local cold/warm16-file benchmark is retained in repomap-benchmark.txt
and BENCHMARKS.md. Both paths read/hash current source; results are host-specific.
Exact evidence ID/build identity are in research/evidence/efficiency-report.json
and go-efficiency.json. Captures remain bounded with sampled-process unknowns;
selected assertions are not canonical REA equivalence.

Tests cover declarations/generics/receivers, body/value exclusion, cache invalidation,
parse errors, output limits/concurrency/eviction/cancellation; artifacts cover
byte-exact base64 pagination, tail, empty/end offsets, invalid identifiers/arguments,
link/FIFO/size refusal, session ownership, escaped binary data and real shell IDs.
A runtime tool cycle verifies shell→artifact retrieval and adjacent map calls
through existing journal/permission/coordinator logic without pending effects.
