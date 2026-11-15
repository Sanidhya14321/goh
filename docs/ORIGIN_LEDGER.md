# Origin ledger

Original project identity: GoHarness (`goh`; earlier working name "Independent Harness").
Original source, prompts, fixtures, and documentation are Apache-2.0. This implementation
was AI-assisted (OpenAI Codex for the early milestones, then Claude Code, as recorded in
commit trailers), with the user's plan providing requirements. No human authorship or legal
clearance is inferred from this ledger.

| Material | Origin | Handling |
|---|---|---|
| cmd/ and internal/ Go source/tests | Independently authored from the plan and reviewed behavioral contracts during this task | Apache-2.0; no upstream code translation |
| Runtime system and summarization prompts | Original text in internal/runtime/runtime.go | Apache-2.0; reference prompts not copied |
| Scripted-model endpoint and acceptance driver | Original Go fixtures in cmd/ | Apache-2.0; deterministic, no live credentials |
| Research scripted-model.mjs and comparison driver | Independently authored investigation fixtures | Apache-2.0; behavioral actions mapped to each product |
| OpenCode binary and root license | Official v1.19.5 release; pinned commit in reference-manifest.json | Retained only under research/reference; not linked or shipped as original product |
| Upstream request/prompt traces | Actual reference model requests saved as research evidence | Research only; never used as Go product prompts or redistributed in release package |
| REA skill/guides and package | morluto/rea skill commit 71f680d7c1aa7a45ba0325fe87b104744b0c105f; rea-agents6.8.0 npm package | Development tooling; source revision and package identity separate; not runtime dependencies |
| Go distribution | Official go1.27.4 archive (go1.27.3 before RC174), verified publisher SHA256 | Local development toolchain; bundled notices retained |
| Dependencies | Exact selected Go graph in go.mod/go.sum | Inventory, original notices and license hashes under licenses/ |
| Real-project benchmark tasks | `hidden/` and `solution/` files under `bench/tasks/` whose `task.json` names an upstream `git` source | Third-party (MIT / BSD-3-Clause); exact upstream license texts and copyright notices in `bench/licenses/`; test material only, not shipped in binaries |
| OpenCode UI screenshots | `research/evidence/opencode-ui-*.png`, captured from the pinned MIT-licensed release for side-by-side comparison | Research evidence only; not product assets |
| Project plan | User-provided IMPLEMENTATION_PLAN.md | Kept unchanged at docs/plan/IMPLEMENTATION_PLAN.md |

Implementation did not copy OpenCode implementation files, recovered functions,
or fixtures. Reference configuration/source was inspected to drive observed
behavior. Source separation is a provenance practice, not legal certification.
No remote publication, branding clearance, or ownership certification occurred.

Contributions: by submitting a contribution for inclusion, contributors must have
the right to license it under Apache-2.0 and preserve applicable third-party
notices. No external contributions have been collected during this task.

## Reference material consulted

Behaviour, never code, was taken from these sources. Each feature's notes from the early release
candidates are in [history/ORIGIN-NOTES-rc1-rc26.md](development/history/ORIGIN-NOTES-rc1-rc26.md).

| Source | Used for |
| --- | --- |
| OpenCode v1.19.5 at `810c0eba1a7348036f8bdf61e6632d0c36c6876c` (pinned source and executable, run in an isolated terminal) | Terminal UI layout and interactions (home, prompt, sidebar, palette, dialogs, timeline, fork, export, file menu), revert workflow, editor handoff, keybindings, post-edit diagnostics |
| Official OpenAI and Anthropic documentation | Sign in with ChatGPT, Responses and Messages streaming, prompt caching |
| Aider, Goose, Crush and Harvest (source inspected, not linked or vendored) | Context, output and role-routing architecture; the installer's run-before-replace check |
| gitignore(5) | Ignore-file matching |
| Papers and articles listed in [design decisions](development/DESIGN_DECISIONS.md) | Agent-loop, editing, context and safety choices |

`golang.org/x/net/html` is reused as a dependency (BSD notices retained), not claimed as original.
