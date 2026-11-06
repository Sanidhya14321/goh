# Session task list: observed contract

Reference: OpenCode v1.19.5, commit810c0eba1a7348036f8bdf61e6632d0c36c6876c; executable identity in research/reference-manifest.json. This is an independently written behavioral contract, not translated upstream code or copied prompts.

## Runtime observations

REA record `research/evidence/reference-todo.json`, `ev_34d78c9ff6aaa33faabcbe2aa080ed42273ae0ad23156e2be53a44885be2241b`, ran the original local `research/todo-model.mjs` fixture. Exit0, untruncated. Complete fragmented tool calls produced these outcomes:

1. `todowrite` accepts `{todos:[{content:string,status:string,priority:string}]}` and returns the full supplied list as JSON plus `metadata.todos`.
2. The tool accepts two simultaneous `in_progress` tasks. It does not enforce one active task or a task transition state machine.
3. Each write replaces the whole ordered list. A later list with one completed task reports only that task; the earlier tasks are not merged.
4. `status` and `priority` are plain strings. Suggested status values are pending/in_progress/completed/cancelled; suggested priorities are high/medium/low. The reference accepted `paused_by_fixture` and `urgent`.
5. Duplicate content is accepted; no task ID field is required. A two-item list with duplicate content and cancelled/completed statuses succeeds.
6. A numeric status is rejected before a successful update. The tool's validation error identifies `todos[0].status` and requires a string.
7. Output title is the count of entries whose status is not exactly `completed`; cancelled and arbitrary strings therefore count. Completed entries remain in the list.
8. `todoread` was absent from the model's advertised tool inventory. A forced call was handled as the reference's invalid-tool result. A Go task-list reader would be an original extension, not pinned built-in parity.

## Permission and durability boundaries

The successful experiment used an explicit isolated reference permission-allow config. It does not establish the default ask behavior. Source-only assistance from pinned tool/session/schema files identifies the permission action as todowrite and storage as session-scoped, ordered whole-list replacement. Persistence and resume observations are recorded separately below when acquired; source assistance alone is not REA runtime evidence.

Original Go implementation should retain session identity, a durable task-list record/event, schema type checks, replacement semantics and ordered rendering. It should not invent stricter status enums or one-in-progress restrictions while claiming exact reference behavior. If stricter policies are deliberately selected, document them as differences. These task-list updates do not authorize project writes, shell, MCP or external actions.

## Acceptance fixtures

Independently authored cases: pending→in_progress→completed snapshots; two in-progress entries; duplicate content; arbitrary strings; numeric status rejection preserving previous list; shorter replacement; empty-list clearing; normal-session resume; no cross-session sharing; event/result ID pairing; permission denial with no state change; journal recovery without replay.

Go implementation/paired capture: pending. Default reference authority and reference invalid-input state preservation still require their own observations.

## Persistence and resume evidence

- `reference-todo-db.json` (`ev_33df6fe5b6497928713b640dd4451e21605847a9d1786abd5777eb5fb28679e8`) captured the pinned official `opencode db --format json` SELECT for this session. After numeric-status validation failure and forced missing-tool call, the database still contained only the two duplicate-content entries in positions0/1. This corroborates whole-list replacement, order, persistence, and invalid-input preservation for the observed case.
- `reference-todo-resume.json` (`ev_13509b24c46d44c4a77809a25c2bb27f6c78332ee219cc1fa0e1daac2e73aad5`) resumed the same session. todoread remained unavailable. A successful `todowrite {todos:[]}` returned empty output/metadata and title0 todos.
- `reference-todo-db-cleared.json` records the same bounded SELECT after resume; it returned[] with exit0, confirming durable clearing rather than leaving earlier rows. These queries inspect reference-owned isolated session storage; no user database was accessed.

The database inspection is a captured official CLI query, not an inference from JSON output alone. It does not establish crash atomicity or locking guarantees beyond this completed run.

## Original Go implementation and comparison

`internal/runtime/todo.go` exposes `todowrite` through the shared runtime. It
serializes updates with other mutations, uses the existing named permission
boundary, synchronizes `todos_updated` before committing state, emits that event,
and returns the ordered supplied array. Plain strings and duplicates are accepted;
invalid types preserve previous state. Explicit limits:1,000 entries and60KiB
serialized task state. Unknown fields are rejected (a deliberate stricter schema
than the observed reference), and permissions do not authorize project changes.
Task state is projected into model context as untrusted data, preserved across
compaction, and recovered on resume/fork/portable import. It grants no authority.

TUI shows a bounded task overview and `/todos` or `/tasks` prints all tasks locally
without a model call. These local reader commands are original extensions;
no `todoread` built-in parity is claimed.

REA Go captures: go-todo.json (`ev_de9c787e3f0ff01ab351c1a14df02a300bcb7d1855320acac2dfe7cc63cad15a`),
go-todo-resume.json and go-todo-ask.json. `research/compare_new_features.py` checks
all five successful snapshots exactly against the reference, including custom
strings, multiple in-progress entries, duplicate content, invalid-type
preservation, resumed clearing and headless refusal without a grant. It passed.
Runtime/TUI tests cover cancellation, state isolation, durability and task display.

## Edge types and default authority

`reference-todo-edge.json` (`ev_b1fc4d9e10fa6f0016f128fb7a380607650895df3d3dd632f831c03f99bcacda`) accepted empty content/status/priority strings and stripped unknown item/root properties from its output. It rejected null todos and missing priority; final empty-list clearing succeeded. The rendered-terminal budget is truncated; retained raw events cover these listed outcomes.

`reference-todo-default.json` (`ev_ffc9d40e095b0f72f5bf56d89607ed591343ec5338c17466d81cdd52bc468e9f`) used a separate isolated config root with no permission key. todowrite updates succeeded by default. `reference-todo-ask.json` (`ev_d782dd4ca43723fbc477de0c9737d3d3a6bd26cbdd0d126b7231c87adab9b042`) explicitly set todowrite ask; headless execution auto-rejected the first update and exited0. A Go policy asking by default would intentionally differ from this observed reference default.
