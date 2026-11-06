# Regex search with file include patterns

Pinned reference source: packages/opencode/src/tool/grep.ts and the alternate
core/src/tool/grep.ts. The shipped opencode runtime and alternate core differ;
we use actual executable behavior as the reference contract.

REA reference: research/evidence/reference-grep.json,
`ev_a02a087e5af206b630403d5988becd3d5b9ed5e969cfa970224070b4ac0a14ef`.
Independent local fixture tests default regex, include *.go, case-insensitive
regex scoped to src, a file path as base, invalid regex, no match, brace include.
Observed results: default5 matching lines including hidden and node_modules,
Gitignored and binary fixture files absent; *.go gave1, scoped case-insensitive3,
brace include2, no match0. Invalid regex produces an actionable tool error.
The shipped reference treats a file path as its parent directory and returned2
matches across sibling files; alternate core source narrows to the named file.
This discrepancy is recorded, not silently collapsed into one behavior.

Original Go tool `grep`: pattern string, optional relative file/directory path,
optional include glob using the independently implemented glob matcher. Go regular
expressions, UTF-8 regular files, automatic project-read policy, stable lexical
file order and ascending line order, max100 matching lines and bounded output.
A file path searches only that file (intentional narrower behavior); unsafe and
symlink paths, malformed regex/include patterns and empty patterns fail. Existing
.git/node_modules/.harness exclusions and safe traversal apply. Gitignore rules
are not interpreted. Binary/special/oversized files are skipped explicitly in
skipped_files. Results use the existing Go structured search envelope.

Independent tests cover include/brace filtering, directory/file scope, case-insensitive
patterns, line numbers/order, no matches, binary skipping, schema/regex/path
errors, match limit, and cancellation. This does not promise every ripgrep regex
extension; lookaround and backreferences remain unsupported by Go's regex engine.
