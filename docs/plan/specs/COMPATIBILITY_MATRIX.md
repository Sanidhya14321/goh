# Compatibility matrix

Reference identity: OpenCode v1.19.5, executable SHA256 `feaafead8cd563af0c2dcb386b0757e4355e327ebb6bd1aaad5440fe2f83596c`.

> **Later changes (reviewed at RC168).** Rows record the state when their evidence was captured. Since then:
> exact edits gained whitespace-tolerant matching that must still match exactly one place (`internal/tools/flexible_edit.go`);
> sessions gained `fork --before-turn` boundary forks; native Anthropic gained extended/adaptive thinking;
> whole-project Git snapshots (`internal/projectsnap`), an OS shell sandbox and hooks were added. Upstream-format
> session import, desktop/browser clients and plugins remain out of scope.

This matrix separates reference observations from Go acceptance tests. Go implementation and passing unit tests alone do not establish compatibility. The first comparisons are semantic (tool pairing, effects, outcome), not identical prose, identifiers, timing, or terminal layout.

| Feature | Reference evidence | Go contract | Comparison status | Limits |
|---|---|---|---|---|
| Local compatible provider text | `reference-text.json` | Stream versioned text delta and committed message events | Compatible selected text outcome; intentionally different event schema | No live provider or identical content guarantee |
| Fragmented tool JSON | `reference-edit.json` | Assemble only complete validated calls; preserve IDs | Compatible selected fragmented JSON/tool pairing | Bounded scripted fixture |
| Read/edit/test | `reference-edit.json` | Read hash; exact unique edit; shell test with explicit authority | Compatible selected file and command outcome | One UTF8 LF file |
| Exact edits and stale hashes | `reference-stale-preserved.json` | Require full SHA256 and unique old text | Intentionally stricter; measured external-mutation difference | OpenCode broader matching strategies not reconstructed |
| Denied mutation | `reference-deny.json` | Explicit permission result; no mutation | Intentionally different refusal interface | Reference removes denied tool; forced call becomes invalid |
| Headless ask permission | `reference-ask.json` | Emit needs_permission terminal state | Intentionally different exit/status contract | Reference auto-rejected edit and exited0 after error tool event |
| Interrupted model stream | `reference-drop.json` | Fail incomplete message without tool execution | Intentionally stricter termination | Reference repeated invalid tools until timeout; capture truncated |
| SIGINT shell lifecycle | `reference-cancel.json` partial error | Owned Go process group with escalation and bounded drains | Reference comparison blocked | macOS ownership inspection withheld platform child environment; manual experiment cleanup separately recorded |
| Sessions and uncertain effects | `reference-resume.json`; Go uncertain resume captured | Versioned durable journal; no automatic replay | Compatible normal text resume; recovery internals unverified | Go tests are independent acceptance |
| Context and compaction | Reference unobserved | Preserve task/tool relationships; explicit budget outcome | Unverified | Token estimates must be documented |
| TUI | `reference-tui-ready.json`, `go-tui-selected.json` | Same runtime, approval, cancel, resize, safe text | Compatible selected text/resize/exit; differing session-selection interaction | No visual equivalence claim |
| Local stdio MCP | `reference-mcp.json`, `reference-mcp-disconnect.json` | Explicit admission/execution, schema checks, no replay | Compatible echo; stricter Go disconnect recovery | No remote transports/OAuth/sampling |
| Windows, subagents, native Responses/Anthropic | Implemented (RC17–RC27) | Native CI on Windows; scripted subagent and provider fixtures | Live providers pending user keys | See VERIFICATION.md |
| Desktop/browser, plugins, upstream session import | Outside V1 | Deferred | Deferred | See original plan |

See [research feature map](../../../research/FEATURE_MAP.md) for exact Evidence IDs, scopes and findings. Paired capture results, when available, must be saved with equivalent working directory, environment, observed paths and capture settings. REA residual process/environment/network unknowns preclude universal equivalence claims even when selected observations agree.

## Paired capture results

The actual paired captures are `research/evidence/go-{text,edit,ask,drop,cancel}.json`. Raw REA comparisons are retained as `compare-{text,edit,ask,drop}.json`; they report changed or truncated rather than equivalence. Their comparison includes raw terminal format, filesystem checkpoint timestamps and settlement durations. The observed before/after file inventories are equal between reference and Go text/edit fixtures.

`python3 research/compare_semantics.py` checks the selected behavioral contract against retained observations; it passed all five cases. Its JSON report is `research/evidence/semantic-comparisons.json`. This original projection is explicitly not REA Evidence. It confirms text equality, per-implementation tool/result pairing, the exact file replacement and shell test result, no mutation after unavailable permission, no execution after incomplete provider output, and explicit Go cancellation with verified REA group cleanup. It deliberately accepts the documented differences in permission and EOF termination.

The Go cancellation capture succeeded; the reference cancellation capture returned cleanup_incomplete. That reference comparison remains blocked despite successful Go behavior. The Go cancellation scenario's tool timeout is five seconds while the reference tool timeout was 35 seconds; both received the identical scheduled 3.5-second SIGINT, before either tool timeout. This permits a bounded interrupt observation but does not establish timeout parity.

Extended selected-fact acceptance: `python3 research/compare_extended.py` passed nine cases (ambiguity, missing old text, retained-text external mutation, normal resume, simultaneous reads, MCP echo, MCP disconnect, uncertain Go resume, TUI). Results and exact Evidence IDs are retained in `extended-semantic-comparisons.json`. This does not close the Part8 reference-compaction gap, Linux runtime gap, or reference crash-corruption/interactive-permission gaps. Go TUI includes explicit new-session selection; reference and Go TUI events/visuals need not be identical.

## Expanded repository-wide reconstruction

The user expanded scope beyond the original alpha. Full checkout and package/path
inventory: research/reference/source and research/full-repository-inventory.json
(6,627 tracked files, 32 top-level packages, exact original pin). Broad backlog:
research/FEATURE_BACKLOG.md. Previously deferred features are now pending work,
not removed from the reconstruction objective.

| New feature | Actual reference observation | Go implementation and measured difference |
|---|---|---|
| glob | glob-reference.json | glob-go.json; recursive patterns/braces match, output lexical/relative with stricter exclusions and malformed-pattern rejection; both captures have truncation limits |
| grep include filters | reference-grep.json | go-grep.json; Go regex/include/directory matches verified; restricts named file rather than searching siblings, different ignore/exclusion policy, lexical structured results |
| Session tasks | reference-todo.json plus db/resume | go-todo.json, go-todo-resume.json, go-todo-ask.json; successful arrays match exactly, invalid type preserves state, empty array clears; stricter unknown-field checks and explicit headless grant required |
| Manual compaction | reference-compaction-manual.json | go-compaction-manual.json; manual operations complete and retain journal history; HTTP reference route versus CLI/TUI Go commands, original summary and preservation policy; auto thresholds unverified |
| Session export/import/fork | session-reference-{list,export,import,fork}.json | go-session-portable.json; original versioned portable format, fresh identities, equal completed state, authority excluded; upstream-format import and boundary fork not implemented |

Each contract is under specs/features/. Selected semantic checks in
research/compare_new_features.py and grep-semantic-comparison.json are original
projections, not REA Evidence. Raw captured build identities and unknowns remain
in their canonical REA records. Full UI, providers/auth, subagents, patch engine,
skills, server/ACP, plugins, LSP, integrations and other backlog surfaces remain
incomplete; acquiring the full repository is not a completeness claim.

| Additional reconstructed slice | Reference/Go evidence | Intentional differences / remaining scope |
|---|---|---|
| Local skills | reference-skills*.json / go-skills*.json | Explicit approval, project/explicit roots, deterministic first definition, bounded YAML subset, tool-description metadata; no remote indexes/global implicit scans |
| apply_patch | patch-reference.json / patch-go*.json, ten semantic checks | Required source hashes, unique exact context, no overwrite/auto-parent creation, bounded prevalidation/staging and explicit uncertain partial effects |
| Profiles/build/plan/custom agents | agent-reference*.json / go-plan-forced-write.json | Original JSON profile schema; strict plan forbids all mutation/shell/MCP despite grants; current mode per invocation |
| Custom slash text commands | command-reference*.json / go-command-quoted/literal.json | Pure one-pass text expansion; shell/file substitution stays literal; project Markdown discovery and command-specific agent/model pending |
| Structured questions | reference-questions-*.json / go-question-headless.json and actual PTY checks | Go reply bounds/cardinality stricter; explicit headless needs_input, rerun in chat; no automatic saved-pending-question continuation |

The version1 development event vocabulary now includes question_requested,
question_resolved, command_expanded, skill_diagnostics and needs_input terminal
status (exit 4). Questions never replace permission decisions. Extra event families
and the new status are additive API developments, not full OpenCode API compatibility.

| Further slice | Retained observations | Original implementation / differences |
|---|---|---|
| Foreground subagents | reference-subagents* plus go-subagents.json | Same-runtime private child journals/owned continuation, separate child grants, strict plan/explore caps and bounded tasks; unknown IDs refused, no recursive/background or child integrations |
| webfetch | webfetch-reference.json / go-webfetch.json | Text/Markdown/raw HTML/local redirects/error cases; stricter same-origin, UTF8/binary,4MiB/64KiB limits; no provider headers/cookies/proxy |
| Explicit local LSP | lsp-reference.json / go-lsp.json | Six operations plus independently tested diagnostics/lifecycle; explicit command/dual authority, no install/replay, scalar-column→UTF16 convention differs, advanced/native-server parity open |

See specs/features/{subagents,webfetch,lsp}.md. Selected checks in
research/compare_next_batch.py pass; captures remain bounded and build-specific.
Ambient-process unknowns and rendered truncation are retained, not normalized
away. Whole repository-wide compatibility remains incomplete.

| Continued slice | Reference evidence | Original scope and deliberate differences |
|---|---|---|
| Post-edit formatting | formatter-enabled/disabled.json; go-formatters.json | Explicit local config, separate format:NAME authority, deterministic order, literal all-occurrence substitution, supervised/bounded process output and cancellation; no full built-in catalog/discovery |
| File checkpoints/diff/undo/redo | snapshot-{track,patch,diff,disabled,nongit,revert}.json; go-snapshot-{undo,redo,conflict}.json | Opt-in per-file checkpoints outside Git, hash/mode conflict refusal, CLI/model restores and explicit uncertain recovery; no Git-tree capture or history-range revert/unrevert/GC |

Reference session revert overwrites touched external changes in the observed
fixture; Go refuses such stale restores. Process observations remain bounded with
incomplete shutdown sampling; go-formatters also lists ambient-process unknowns.
See specs/features/{formatters,snapshots}.md and changes-semantic-comparisons.json.

| Continued feature | Evidence | Original scope / open differences |
|---|---|---|
| Native Anthropic Messages | native-provider-reference-{complete,drop}-evidence.json; go-native-* and mixed-role captures | Streamed text/client tools, usage, separate native auth, redacted typed HTTP errors and redirect refusal. Unsupported reasoning/media/cloud auth/continuations rejected; dropped stream fails once in Go rather than observed reference continuation. |
| Harvest-inspired model roles | Pinned source model-roles/model-resolver, original unit/runtime/CLI captures | Explicit compaction/task/explore→profiles, model-budget admission and resolved identity; no catalog/aliases/fuzzy/fallback/cost assumptions. |
| Strict child JSON output | Pinned Harvest structured-subagent/output-validator source; Go valid/invalid captures + typed-child PTY | Optional object JSON Schema, no loading/refs; final strict validation without repeated effects. No JTD/inherited/permissive/incremental/provider-native output modes. |
| Declaration context | Pinned Aider repomap source/tests; Go efficiency capture/benchmark | Go syntax declarations and hash-validated32-entry cache,32KiB result; no language graph/type-checking/full ranked repo map. |
| Output retrieval | Harvest output/artifact and Crush spill/view source/tests; Go efficiency capture | Session-scoped shell/formatter ID byte/tail/base64 pages; existing1MiB prefix only, no full-output retention/general artifacts/GC/portability/background jobs. |

Pinned supplemental source inventory/license digests and feature disposition:
research/comparisons/{manifest.json,ASSESSMENT.md}. Original selected projections
retain process sampling/truncation limits and do not establish canonical REA or
full cross-agent equivalence. All new implementation/prompt material is original.
