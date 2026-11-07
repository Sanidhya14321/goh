# Origin ledger

Original project identity: Independent Harness. Original source, prompts, fixtures,
and documentation are Apache-2.0. This implementation was AI-assisted in Codex,
with the user's plan providing requirements. No human authorship or legal
clearance is inferred from this ledger.

| Material | Origin | Handling |
|---|---|---|
| implementation/cmd and internal Go source/tests | Independently authored from the plan and reviewed behavioral contracts during this task | Apache-2.0; no upstream code translation |
| Runtime system and summarization prompts | Original text in internal/runtime/runtime.go | Apache-2.0; reference prompts not copied |
| Scripted-model endpoint and acceptance driver | Original Go fixtures in implementation/cmd | Apache-2.0; deterministic, no live credentials |
| Research scripted-model.mjs and comparison driver | Independently authored investigation fixtures | Apache-2.0; behavioral actions mapped to each product |
| OpenCode binary and root license | Official v1.19.5 release; pinned commit in reference-manifest.json | Retained only under research/reference; not linked or shipped as original product |
| Upstream request/prompt traces | Actual reference model requests saved as research evidence | Research only; never used as Go product prompts or redistributed in release package |
| REA skill/guides and package | morluto/rea skill commit 71f680d7c1aa7a45ba0325fe87b104744b0c105f; rea-agents6.8.0 npm package | Development tooling; source revision and package identity separate; not runtime dependencies |
| Go distribution | Official go1.27.3 archive, verified publisher SHA256 | Local development toolchain; bundled notices retained |
| Dependencies | Exact selected Go graph in go.mod/go.sum | Inventory, original notices and license hashes under implementation/licenses |
| Project plan | User-provided IMPLEMENTATION_PLAN.md | Remains unchanged outside reconstruction workspace |

Implementation did not copy OpenCode implementation files, recovered functions,
or fixtures. Reference configuration/source was inspected to drive observed
behavior. Source separation is a provenance practice, not legal certification.
No remote publication, branding clearance, or ownership certification occurred.

Contributions: by submitting a contribution for inclusion, contributors must have
the right to license it under Apache-2.0 and preserve applicable third-party
notices. No external contributions have been collected during this task.


Expanded work also acquires the complete official repository checkout at the
same pinned commit under research/reference/source (all 6,627 tracked files).
It remains reference material, not product source, and keeps upstream notices.
New glob/grep, task state, portable sessions and manual compaction implementations,
fixtures and tests are original Apache-2.0 material. Reviewed contracts state
source assistance versus actual runtime observations and intentional differences.

Further original source/fixtures: internal/skills, tools/patch, core/profiles,
commands, runtime skill/policy/question integration and TUI question controls.
Development observations use the full unchanged pinned repository and separately
identified shipped artifact. Local SKILL.md and question/patch/command fixtures
are independently authored; reference built-in skills and prompt templates are
not incorporated. Existing Go dependencies only; Apache-2.0 and third-party
notices remain unchanged. No remote publication or legal clearance is claimed.

Further original material: runtime foreground child coordinator, internal/webfetch,
internal/lspclient, protocol test fixtures in internal/testutil, and their contracts,
tests/local capture drivers. The HTML tokenizer deliberately reuses pinned
Go x/net/html with BSD notices retained; this is recorded dependency reuse, not
an originality claim over that dependency. No reference task prompts or source
functions were translated into product implementation. Exact updated module/text
digests are under implementation/licenses; no legal certification is implied.

5 November continued batch: internal/formatters, internal/snapshots, runtime
formatting/change-tools integration, tool mutation-path/digest/literal-process
adapters and changes CLI are original Go implementations. Controlled formatter,
Git/session reference fixtures, capture driver, tests, contracts and comparison
projections were independently authored. Pinned upstream formatter/snapshot/revert
source was inspected as behavior evidence; no source code or prompts were copied.
This batch adds no third-party modules or source vendoring. Existing exact
39-module license inventory remains applicable.

6 November2026 continued batch: native Anthropic encoding/parsing/transport,
explicit role profile overlays, strict child-result validation, standard-library
Go declaration context/cache, session-scoped output retrieval and their tests/
fixtures are independently authored original Go/source material. OpenCode pinned
protocol source/runtime, official Anthropic streaming docs, Harvest role/result
source and Aider/Goose/Crush context/output source were consulted as behavior and
architecture evidence. No upstream implementation or prompts were copied.
Supplemental commits, root-license digests and source-only inspection scope are
recorded in research/comparisons/manifest.json. Harvest migration maturity and
package-license variations are retained in its assessment; Crush's inspected
FSL source is not linked/vendored into the product. No dependency was added or
upgraded; exact39-module license inventory remains applicable.

6 November 2026 UI redesign. OpenCode v1.19.5 at
`810c0eba1a7348036f8bdf61e6632d0c36c6876c` was inspected through its home,
prompt, session sidebar, logo component, theme, and official screenshot assets.
The pinned executable was also run in an isolated PTY with a loopback provider
configuration. OpenCode's public GitHub repository and web screenshot search
were consulted. Layout and interaction observations informed original Go code
for the welcome screen, prompt panel, sidebar, palette, settings styling, and
bounded transcript rendering. GoHarness uses its own wordmark. Upstream source
and artwork were not incorporated into the executable. No Go dependency was
added. The pyte/Pillow verification environment is separate from the product.

RC7 UI continuation. OpenCode's pinned prompt history bindings and session
Markdown renderer were inspected alongside public screenshots and repository
documentation. The Go composer, history navigation, slash completion, palette
integration, and bounded Markdown presentation are original implementations.
Captured sessions and benchmarks use the existing isolated PTY driver. No
upstream source, prompt, or artwork was added to the executable.

RC8 UI continuation. The pinned OpenCode session renderer's grid-table and
syntax-style integration was inspected. Original Go table parsing, alignment,
wrapping, narrow-pane fallback, and lexical highlighting use the standard-library
scanner and the already pinned ANSI module. Public image search and an isolated
OpenCode session with a local fixture provided visual reference. No upstream
rendering code or additional runtime dependency was incorporated.

RC9 UI continuation. The pinned OpenCode model/agent dialogs and command
bindings, public screenshots, and the isolated executable were consulted.
The Go option picker, configured-profile selection, agent cycling, and new-chat
navigation are original implementations using the existing Bubble Tea and
configuration/runtime layers. The Git baseline captures existing RC8 work;
new UI milestones use actual commit dates.

RC10 UI continuation. OpenCode's pinned session list, timeline, and generic
selection components were inspected alongside public screenshots and the
isolated executable. Session filtering, asynchronous preview batches, bounded
read-only journal excerpts, and browser resume navigation are original Go
implementations. The existing journal format, runtime resolver, and ANSI
module are reused. No runtime dependency was added.

RC11 UI continuation, 7 November 2026. The pinned OpenCode status dialog,
public screenshots, and the isolated executable were consulted. The Go local
snapshot, configuration shortcuts, credential-presence checks, formatter
discovery integration, and shared modal composition are original code using
existing runtime, configuration, and rendering layers.

RC12 UI continuation, 7 November 2026. OpenCode's pinned dialog-timeline.tsx and
dialog-message.tsx were inspected at reference commit
810c0eba1a7348036f8bdf61e6632d0c36c6876c, alongside the official GitHub source
and isolated executable. The searchable recent-prompt list, explicit composer
recall, cached preview, and draft-preserving modal handling are original Go code.
The existing option picker, prompt history, and safe ANSI composition are reused.
Upstream revert, fork, and file undo are not part of this milestone.
Reference: https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/tui/src/routes/session/dialog-timeline.tsx

RC13 UI continuation, 7 November 2026. OpenCode's pinned prompt autocomplete
component, the official TUI documentation, and the isolated v1.19.5 executable
informed the placement and keyboard interaction of the file menu. GoHarness
implements its own batched Go directory scan, fuzzy ranking, and text path
insertion using the existing composer. The source reference is
https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/tui/src/component/prompt/autocomplete.tsx

RC14 UI continuation, 7 November 2026. OpenCode's pinned
dialog-fork-from-timeline.tsx and the isolated OpenCode executable informed
the full-session and before-turn choices. The Go implementation uses its
existing option picker and durable portable journal publication, with a
one-based journal turn boundary and a separate editable draft.
Reference:
https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/tui/src/routes/session/dialog-fork-from-timeline.tsx

RC15 UI continuation, 7 November 2026. OpenCode's pinned export dialog and
transcript formatter informed the filename field and the tool-details option.
The Go implementation uses the existing settings editor, journal, and
private file publication. The OpenCode executable provided a PTY reference.
Reference:
https://github.com/anomalyco/opencode/blob/810c0eba1a7348036f8bdf61e6632d0c36c6876c/packages/tui/src/ui/dialog-export-options.tsx
