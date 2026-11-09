# Foreground subagents: observed contract

Pinned reference: OpenCode v1.19.5 at810c0eba1a7348036f8bdf61e6632d0c36c6876c. Original fixtures under research/subagents drive the actual shipped executable through a local compatible endpoint. Reference source/prompts remain in research; implementation consumes this independent external contract.

## Task schema and selected agents

Captured advertised input requires description, prompt and subagent_type strings. Optional task_id and command are strings. Word-count guidance on description is descriptive, not an observed enforced constraint. The ordinary profile did not advertise background. Source has optional background only enabled through the experimental-background flag and rejects background=true without it; background execution is not established by these foreground observations.

`reference-subagents.json`, `ev_d5c5a8ec7a4fa05a9cb108ce6c442a9073a8a5a60214ed6fb351b91152b76ec8`, exit0, captured five parent task calls:

1. general creates a separate child session, carrying the parent's session ID in metadata. The result supplies child session ID, selected model and completed text.
2. Passing the returned task_id to general resumes that same child. Its next model request contains the original child user message and previous assistant reply. The task result confirms the fixture saw its earlier history.
3. explore creates a separate child session and returns its final text.
4. Unknown subagent_type produces a tool error naming the invalid type; the scripted parent can continue.
5. A syntactically plausible but nonexistent task_id silently falls back to creating a fresh child session.

Missing task IDs falling back is an observed reference behavior. A Go implementation that rejects missing IDs or checks parent ownership more strictly should document that deliberate safety difference. No attempt was made to resume another user's or unrelated session.

Task output includes child identity and completed state. This is a tool result to the parent, not a second public agent loop in the UI. Child text is not automatically a user approval or permission grant.

## Child tool inventories and permissions

Companion model-request captures observed these advertised inventories with default controlled config:

- general: bash, edit, glob, grep, read, skill, webfetch, write.
- explore: bash, glob, grep, read, webfetch.
- Both omit task and todowrite. Explore omits edit/write/skill; bash is nevertheless present. Therefore a strictly read-only Go explore profile would intentionally differ from the selected reference profile's command capability.

`reference-subagents-probe.json`, `ev_4e71e6fdf6d259503e0c1165c16a3154014fa7cd60821eb5418a78bb9ac5e17d`, captured an explore child forced to request write. The child received invalid-tool failure and later completed its scripted text; no probe.txt file was created. Parent task can complete after the child has recovered from an earlier tool failure; it is not automatically failed by every historical tool error.

`reference-subagents-deny-read.json`, `ev_e4acdf1b52c4e3099fe27cae43b8115f59e7c00f6102ddd34a16eac508a13d78`, runs under an explicit global read-deny config. Child inventory no longer exposes read. This establishes configured restriction application, not alone the distinction between parent-session and parent-agent inheritance.

Source assistance from opencode/src/agent/subagent-permissions.ts identifies the intended inheritance rule: parent session deny rules and external_directory rules propagate; parent agent rules govern that agent while the child's profile controls its own tools. Child task/todo defaults are denied unless child profile explicitly configures them. These fine-grained inheritance distinctions remain source-assisted until an isolated session-vs-agent fixture verifies them.

## Cancellation scope

`reference-subagents-cancel.json`, `ev_3365ac1dfb5600d9f4df5041c4f05fe0a83b4a1aa8c1cd74ea8a79e43c3f0c53`, interrupts the parent with scheduled SIGINT while the child waits on a delayed local provider response. The reference exits signal2, and REA verifies its owned process group removed. No child shell/process was launched in this observation. It does not establish rollback, detached-process cleanup, saved child state after abrupt exit, or delivery of a structured cancelled task result. Source cancellation handlers target the child run, but that source fact must not be mislabeled a full runtime guarantee.

## Source-assisted limits and variants

Pinned task implementation resides in packages/opencode/src/tool/task.ts; search found no equivalent core/src/tool/task.ts in this revision. The separate core/v2 registry exists, but no second task implementation was invented for this contract. Agent source defines general/explore and custom profile merging. Task source checks ancestry depth against config.subagent_depth (default1), selects child-profile model if present or parent's model/variant otherwise, and creates child sessions titled from the description/profile. Only inherited fixture model and same-child continuation were runtime observed here.

Unverified: nested depth errors, custom child model, primary-profile selection, task_id ownership, concurrent children, parent-session-vs-agent deny distinction, child writes under approval, cancellation durability, background launch/update/promotion/notification, terminal task UI and child failure propagation edge cases. Those belong to separate bounded fixtures. Do not claim full subagent parity from this first foreground slice.

Original Go acceptance should cover durable parent-child IDs, child-context isolation, profile/model selection, explicit task authority, all side effects retaining their own authority, continuation under reacquired permissions, missing IDs, depth/budgets, deterministic result pairing, error outcomes and cancellation. Original prompts must be independently authored. If Go deliberately restricts explore shell or parent ownership more tightly, expose the difference in compatibility records.

## Parent agent vs session authority: additional runtime evidence

`reference-subagents-inherit-http.json`, `ev_07bf702ad42c61e9df302c6afe401ccf0de2f997b7d16a774b17e8785b1164ac`, captures an original HTTP client starting the pinned loopback server and creating two disposable parent sessions. The common build-agent config denies read. The first parent has no session deny; its general child advertises read. The second parent adds a session read-deny rule; its general child omits read. GET session children explicitly reports the parentID and inherited read-deny rule for that child. This establishes the fine-grained distinction previously source-assisted above for the observed read case. Other permission categories and external_directory remain unverified.

This capture launches Node as the selected executable and the pinned server as its owned child. Exit0, verified owned group cleanup; rendered output budget truncated, relevant raw response/model inventories retained. It uses no real API service or credentials.

`python3 research/subagents/check_reference.py` checks selected observed facts and passed. It is an independent projection, not a producer of REA Evidence. It verifies child continuation, default tool profiles, missing/unknown IDs, no forced explore write effect, agent-vs-session read permission distinction, parent-child IDs and bounded provider-wait SIGINT cleanup.

## Original Go runtime slice

`task` now runs child instances of the same coordinator/provider/journal. Built-ins
are general (build) and explore (strict plan); user-configured agent profiles may
also be selected. Every task requires approval. Child actions require separate
`child:TOOL` approvals/grants; parent grants are not copied. Parent plan caps the
child to read-only mode, including custom profiles. Child processes do not inherit
MCP/LSP managers, and model recursion is disabled at depth 1. Web fetch still needs
a separate child grant and is unavailable in strict explore/plan mode.

Bounds: 8 tasks per parent run, 16 model turns per child, 180-second child deadline,
64 KiB child prompt and bounded returned summary. Full child history remains in
its own private session journal. Child events are wrapped in subagent_event under
the parent's ordered envelope; they never directly mutate parent conversation.
Normal continuation uses task_id, validated against both parent-owned lineage and
the child's recorded parent/type/project. Unknown or cross-parent IDs are refused,
instead of the reference's missing-ID fallback to a fresh child. The child current
profile is explicit config; no parent conversation, permissions or executable
integrations are silently imported.

Interrupted/uncertain child effects stop the parent task too, with task_id in the
observation; no automatic replay occurs. Inspect/resolve both journals as needed.
Headless missing child authority/input propagates needs_permission/needs_input to
the parent, with the recorded child ID available for explicit continuation. No
background task fleet or nested delegation is implemented in this slice.

Go REA: go-subagents.json,
`ev_f43f71dafead3f6feaf392f511b3d8d899e34fe1bd283763bd7adc0f6bfaef3b`:
create and same-ID resume preserve child history, explore uses a distinct session,
unknown agent and missing task_id fail explicitly, parent completes. Actual PTY
tests also approve a child-only file mutation separately and cancel an executing
child shell (pid fixture confirms command entry) while the app remains open.
Unit/integration tests cover non-inheritance, cross-parent/type ownership,
plan/explore caps, recursion/task limits, cancellation and resumption grants.
These safety policies are intentional differences from observed reference rules.
