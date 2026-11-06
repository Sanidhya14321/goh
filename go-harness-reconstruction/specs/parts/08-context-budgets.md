# Part 8: Context budgets

## Question and provenance

Can basic context bounds/compaction preserve the active request and complete tool groups?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations



## Original Go contract

Configured context/output budget with conservative byte-based accounting. Compact older complete groups through an original summary prompt; preserve system instructions, active user request, unresolved calls and recent groups. Journal summaries without deleting original history; fail clearly on insufficient reduction or provider error.

## Comparison

No controlled reference compaction observation was collected. Existing scripted responses do not supply realistic tokenizer/usage pressure, so ordinary text captures cannot substantiate context parity. Go implementation tests cover its original contract independently.

## Exit coverage and unknowns

Reference thresholds, token accounting and compaction parity remain unverified. Conservative Go byte accounting is deliberately not a model tokenizer; it may stop early. Failed/ineffective summaries must produce budget_exhausted. This part has an explicit reference-evidence gap.


## Follow-up manual observation

The manual reference observation gap was partly closed during expanded feature
work. See specs/features/manual-compaction.md and canonical
research/evidence/reference-compaction-manual.json. Reference summarize returns
true and retains a summary-marked assistant message/history. Original Go compact
ID and TUI /compact have tested durable behavior and a paired selected-fact
comparison. Automatic thresholds, pruning and precise token accounting remain
unverified; the original byte-estimate difference remains explicit.
