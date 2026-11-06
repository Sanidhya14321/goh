# Glob file discovery contract

Reference: OpenCode v1.19.5, commit
810c0eba1a7348036f8bdf61e6632d0c36c6876c, executable identity in
research/reference-manifest.json. Source assistance: retained
research/reference/features/glob.ts and ripgrep.ts. Those files are research
material, not implementation inputs or translated code.

## Observed reference facts

The local scripted model requested eight glob calls through the actual pinned
reference. REA capture: research/evidence/glob-reference.json,
`ev_772121507815977cc31dce69d4adec257f0ff0c5c64b62cb086837a6a1b437a8`.
Requests and actual tool-result messages are retained in
research/evidence/glob-reference-requests.jsonl.

- Arguments are pattern:string and optional path:string directory.
- `*.go` and `**/*.go` both discover matching basenames at arbitrary depths.
- `**/*.{go,ts}` handles alternatives. Directory path scopes traversal.
- Explicit patterns included hidden files, an ignored file, and node_modules.
- No match is `No files found`. A file used as base directory returns an error.
- `[unclosed` produced no files, rather than a malformed-pattern error.
- Output paths are absolute. Observed output was not lexical and not ordered by
  ascending/descending fixture mtime. Source inspection did not show mtime sort;
  no broader ordering guarantee has been inferred.
- The 100-result cap is source-established, not runtime-probed in this capture.

## Original Go behavior

`glob` automatically admits read-only project discovery. Arguments use the
same names, but optional path must be project-relative and an existing directory.
Return JSON contains project-relative `files`, `count`, `matched`, `truncated`,
`path`, `pattern`, and `visited_entries`. Paths sort lexicographically.

Supports `*`, `?`, Unicode, Go-style character ranges/negation, shell `[!x]`
negation, complete-component `**` matching zero or more directories, and nested
comma braces. A slash-free pattern matches basenames recursively. Pattern size
is at most4096 UTF-8 bytes; at most128 components,64 brace alternatives,8 nested
brace expansions. Leading `./` is accepted. Absolute/parent traversal patterns,
negative pattern lists, empty components and malformed grammar fail explicitly.

Read-only traversal uses os.Root, rejects symlink base components and skips
symlink entries and special files. Existing tool exclusions `.git`,
`node_modules`, `.harness` remain. Hidden files and gitignored files otherwise
participate. Traversal visits at most20000 entries/10000 regular files; returns
at most100 results and a bounded64KiB serialized result. Truncation is explicit.
No external command or new dependency is required.

Differences are deliberate: relative structured output, stable lexical order,
strict malformed-pattern errors, root/symlink restriction, additional exclusions,
and accurate truncation when exactly100 matches exhaust the directory. Advanced
ripgrep glob features and whole filesystem compatibility remain unsupported.

## Verification and comparison

Go REA capture research/evidence/glob-go.json:
`ev_6762e209af42e90289ee1b27cf9ead459b20ca1b065bf82238e8cd39cf899699`.
Both reference and Go exited0, owned groups verified, observed project snapshots
unchanged. `python3 research/compare_glob.py` passes8 semantic cases, recording
intentional differences in evidence/glob-semantic-comparison.json.

REA comparison research/evidence/glob-comparison.json:
`ev_0c1c87052c30a0e087734481267285b5d057ebe22e8397b23a0c1850a7b3dec4`.
Its broad result is changed; terminal/process comparison is unknown. Reference
process sampling ended incomplete, Go rendered terminal frames exceeded retained
budget, inherited environment is unrecorded, and process observations are sampled.
Both saved raw terminal streams are retained within budget. The broad comparison
is not a semantic equivalence assertion and does not override the explicit
per-case semantic checks.

Acceptance: internal/tools/glob_test.go tests matching, scope, safety, Unicode
paths, malformed grammar, result ordering/caps and cancellation. Full tools race
suite passed5.822s. FuzzGlobPattern ran3s/82822 executions without failure on
macOS arm64. Source/package identity and all capture limitations remain as in
the reference manifest and complete retained Evidence records.
