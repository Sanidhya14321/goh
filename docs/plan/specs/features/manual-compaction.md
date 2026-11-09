# Explicit session compaction

Reference: pinned OpenCode v1.19.5, full repository under research/reference/source.
Source assistance: packages/opencode/src/server/routes/instance/httpapi/groups/session.ts
identified the manual summarize route; session/compaction.ts identified summary state
for the bounded probe. No source prompts or implementation were copied.

Actual reference REA: research/evidence/reference-compaction-manual.json,
`ev_356b45e09b1df9afb14b171f790e2d67273951f32aca63ed5eef9fa0012e9c93`.
The captured executable is the original Node HTTP/fixture driver, which owns the
pinned OpenCode serve subprocess (binary identity in reference-manifest). This is
an indirect process observation, not a direct CLI summarize command capture.
The loopback model and server both terminate at the end. Source-model requests
and reference prompts remain only in research/compaction/requests.json.

Observed: create a disposable session, submit a text request through the local
scripted provider, then POST /session/ID/summarize with providerID/modelID and
`auto:false`. Response true; history grew from two to four messages, including a
user compaction part and an assistant summary marked summary:true. Three model
requests were observed (including auxiliary reference work). Original stored
messages remained available. This does not measure automatic token overflow,
prune thresholds, cancellation, or arbitrary provider behavior.

Original Go interface: `harness compact ID` with normal config/provider/session
flags, and TUI `/compact`. It creates a versioned run/event envelope, summarizes
older state with the original Go prompt, journals context_compacted, preserves
active user request, recent complete groups, unresolved calls and task state,
and leaves historical journal records intact. It starts no MCP server or tool.
Uncertain effects prevent compaction until resolved. Failure/insufficient space
has an explicit budget outcome; cancellation is cancelled. Current trusted
system framing is restored on resume/import, while saved summary instructions
are treated as untrusted assistant data.

Go REA: research/evidence/go-compaction-manual.json,
`ev_621e8d7fb88f051e42aeb2cb2d83614ccae238bba0315eb173ffaa4cd6a24d46`.
The manual operation completed, emitted a single durable context_compacted event
and one terminal completed event. A paired selected-fact check is retained in
new-feature-semantic-comparisons.json; HTTP versus CLI differences are deliberate.

Independent tests cover durable manual operation, no integration startup,
empty session refusal, pending-effect refusal, active request/recent group
preservation, task projection, untrusted summary replay, and portable fork of
manual compaction. Automatic reference thresholds/pruning remain unverified.
