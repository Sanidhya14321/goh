# Structured questions: observed contract

Pinned reference: OpenCode v1.19.5, commit `810c0eba1a7348036f8bdf61e6632d0c36c6876c`. All fixtures are original. Reference source remains in research/reference/source; no upstream prompts or implementation were copied into the product.

## Evidence and isolation

- `research/evidence/reference-questions-headless.json`, `ev_e1630fae50c77f58f3a82d993bf7a15c306a357acebf59b007d368757d1e160d`: real pinned headless run, forced question call handled as unavailable/invalid despite enabling question flag, model continued, exit0. Rendered terminal budget truncated; raw listed result retained.
- `research/evidence/reference-questions-http.json`, `ev_9b59228b3e69c853cffc8ddfc981b1b013863a3a7157b9587eec8438ae7894ca`: REA captured the original Node HTTP driver launching the pinned server on loopback28785 and exercising five cases. Exit0 with owned group cleanup verified; rendered terminal budget truncated. The captured executable identity is Node, and pinned server is an owned child (not mislabeled as the directly launched executable).

The controlled server used isolated HOME/XDG/config and an independently written loopback model at28784. No remote hosting, accounts or model credentials were used. HTTP replies belong only to newly created disposable fixture sessions. Driver logs record HTTP status/body and terminal tool messages; they are actual outputs inside the REA capture. They do not prove SSE event arrival order, which was not separately recorded.

## Tool input

The advertised tool accepts:

```json
{"questions":[{"question":"Complete prompt","header":"Short heading","options":[{"label":"Choice","description":"Explanation"}],"multiple":false}]}
```

Required: questions array; each question/header/options; each option label/description. All listed text fields are strings; multiple is an optional boolean. Header/label length guidance is descriptive, not an observed schema constraint. Empty options worked for a free-text fixture question. The tool Prompt schema does not advertise custom; the broader question Request Info type has optional custom with default true. Source and actual advertised schema must be distinguished.

A missing question header produced a tool schema error before any pending question appeared. The same ID continued to pair with the error result. Other constraints must not be invented as reference facts.

## Pending request and answer behavior

In the local server flow the model's complete question call creates a pending request with `que_` ID, session ID, ordered question list, and originating message/call ID. GET /question exposes pending requests; successful reply or reject resolves/removes the pending request. The POST prompt_async call returns204; the question tool remains pending until resolved.

Reply input is `{answers:string[][]}` with rows in question order. Observed behaviors:

- Single choice returned its chosen label.
- A multiple-choice row accepted two labels.
- A question with empty options accepted arbitrary free text.
- Labels outside advertised options were accepted.
- A row accepted two values despite multiple being absent on that question.
- Extra answer rows were preserved in metadata, even when there were fewer questions.
- An empty answer array was accepted; absent/empty rows render Unanswered.

Therefore the observed API does not enforce option membership, per-question cardinality or matching answer-row count. A stricter Go UI/API can improve validation, but must label that behavior as intentionally stricter rather than exact parity.

Tool completion records metadata.answers with the supplied values. Human-readable output maps each question to its row joined with commas, ignoring extra rows in this textual projection. The title reports the number of questions. Successful resolution returns200/true. A second reply to the resolved ID returns404 with a question-not-found error; it does not replay or mutate the earlier answer.

## Dismissal, headless behavior and authority

Rejecting an outstanding request returns200/true, yields a tool error saying the question was dismissed, and allows the scripted agent to continue to its next assistant response. Repeated access after resolution returns404. This is distinct from a tool permission approval: answering a question does not authorize file writes, commands, MCP or external effects.

The ordinary headless run injects a question-deny rule. Actual capture confirmed that question was absent and a forced call was routed to invalid-tool behavior, with no indefinite input wait. Setting OPENCODE_ENABLE_QUESTION_TOOL alone did not override that headless rule. For the HTTP server fixture, explicit app-client/enable flags exposed the question tool and the local HTTP client supplied the interaction.

Original Go design may expose headless `needs_input`/questions events and terminate clearly or hide question until an input channel exists. It must avoid indefinite unavailable-input waits, preserve question/tool relationships and cancellation, and never interpret answers as permission grants.

## Source assistance and unknowns

Pinned files read using normal source tools: opencode/src/tool/question.ts, question/index.ts, schema/src/v1/question.ts, server question groups/handlers, cli/cmd/run.ts and runtime-flags.ts. Source identifies asked/replied/rejected events and removes pending state on cancellation/teardown. That event/cancellation lifecycle remains source-assisted until an explicit controlled observation tests it.

Still unverified: rich TUI single/multiple/free-text editor interactions, keyboard dismissal, pending question cancellation and process crash recovery, remote/authenticated clients, event ordering, concurrent responders and maximum payload bounds. Question limits, terminal controls, multiline free text and response capture must be tested independently in Go. Original implementation/paired capture is pending.

## Additional typed/empty/custom edge observation

`reference-questions-http-edge.json`, `ev_d5a45073bc4bdd680d1d8e1f920293fa892246fd09312ad1a94bd3d354581c5f`, captured three additional loopback cases, exit0; rendered output truncated while listed raw responses were retained. Numeric reply element `answers:[[7]]` returned400 and did not consume the pending request; a subsequent string reply succeeded200. `questions:[]` created an empty pending request and completed successfully after `answers:[]`. The model supplied `custom:false` in a tool question: the pending request omitted that unknown Prompt property and accepted an arbitrary custom reply. The raw tool input still retained the model-supplied custom property in recorded input; this is distinct from the service's parsed request.

## Original Go runtime/TUI slice

The `question` tool is now available on the shared runtime. Requests are a
serialized interactive barrier, not a permission request; replies cannot change
grants. TUI supports single choice, multiple selections, custom text, dismissal
and active-run cancellation. Questions and resolutions are journaled without
creating an uncertain external-effect start. If input is cancelled, the next run
repairs the missing tool-result relationship without replay or fabricated answer.

Headless Go runs emit question_requested and terminate `needs_input`/exit 4 rather
than blocking. The saved question_pending record is inspectable; rerun the request
in chat to answer. Automatic continuation of a saved pending question is not
implemented. `needs_input` is an additive development event-status extension.

Go intentionally validates 1–8 questions, unique labels/cardinality, bounded UTF-8
payloads/replies, required fields and no unknown tool keys. This is stricter than
the observed permissive reference server. Tool input supports question/header/
options/multiple only; custom text is always available in the original TUI.
Reference `custom` field is not claimed as part of its tool schema.

`go-question-headless.json`, `ev_a59abf3eef294cadebbb4ab6ca52e3839b4c15de20ede10c50e345d4909b5ba2`,
captures the new explicit no-input outcome. Actual PTY tests cover multi-selection,
single selection, custom text, dismissal, cancellation and terminal restoration
in rest-tui-acceptance.json. Runtime tests cover typed validation, no authority
transfer, headless completion/pairing, cancellation and safe continuation. Rich
reference TUI interaction parity remains unverified.
