# Coding harness and repository memory research

## What helps a coding agent reuse earlier repository work?

### Takeaway

The strongest small-scale proof of concept is inherited investigation notes. Larger, newer evaluations show that retrieval and memory construction often fail to deliver the useful information in repository history. A local feature should prioritize relevance and applicability checks and measure how often memory hurts a previously solvable task. [Stratum Table I](https://arxiv.org/html/2610.36153v1) · [VibeMemBench](https://arxiv.org/abs/2609.23570v1) · [MemDev-Bench](https://arxiv.org/abs/2610.27524v1)

### Cited Findings

- **Stratum** (Sehgal et al., Oct 26 2026, preprint v1) proposes repository-local tiered memory and bounded pages. On eight real bugs and 24 trials per paired condition, inherited notes written by a different model produced 21/24 correct root-cause file localizations versus 6/24 with retrieval alone. Median input tokens were 43.3K vs 196.6K, roughly 4.5× lower. The pooled-store arm had eight trials, all successful. This endpoint is locating a file across sessions, and eight underlying bugs limit generalization to new bugs or complete repairs. [Paper and Table I](https://arxiv.org/html/2610.36153v1)
- **MemDev-Bench** (Chen et al., Oct 20 2026, preprint v1) supplies 4,869 earlier executable tasks for 555 base tasks, with related, noisy and repository-level histories. On the 133-task improvement slice, NoMem averaged 35.3% resolved and Claude-Mem averaged 41.6% with relevant history, with 12.2% rescues and 5.9% harms as reported by the paper. Those rates reveal opposite directions hidden by the net gain. The authors report that direct raw history reduces steps relative to their evaluated memory systems and suggest retaining process details and reassessing applicability. The improvement slice and paper-specific model and harness conditions bound the result. [Paper, §4.1 Tables 2–3](https://arxiv.org/html/2610.27524v1)
- **VibeMemBench** (Fan et al., Sep 20 2026, preprint v1) pairs 111 targets from 90 repositories with 3,634 history trajectories. Verified direct experience injection raised executable resolution by 1.1–4.5 percentage points for four of five held-out solvers and reduced steps for all five. Eleven of 12 ordinary memory-system/solver pairings failed to beat their matched memory-off baseline. Selection requires each target to benefit from an injected experience in a reference setting, so the direct-injection arm is an estimate on a memory-susceptible cohort. [Paper, §§3–4 and Tables 1–2](https://arxiv.org/html/2609.23570v1)
- **Evaluating AGENTS.md** (Gloaguen et al., Feb 2026, research paper) reports a tendency toward lower resolution with repository context files and an inference cost increase above 20% across its evaluated coding agents and conditions. This is evidence about upfront repository context files, so it motivates an explicit no-memory baseline for a different retrieval design. [Paper](https://arxiv.org/abs/2602.11988)

### Inferences

- Candidate Go feature: an opt-in repository evidence ledger. Save typed notes for checked facts, test invocations and outcomes, failed approaches, decisions, and unresolved effects. Store source journal entry IDs, repository/worktree identity, observed revision, content hashes or validation predicates, and optional artifact pointers. Retrieve narrowly by path/symbol/task terms under a byte budget with paginated evidence. Preserve user-authored instructions separately and treat recalled notes as data. Provide correction, expiry, and deletion controls. These design choices adapt Stratum's bounded recall and MemDev-Bench's applicability findings. [Stratum](https://arxiv.org/html/2610.36153v1) · [MemDev-Bench](https://arxiv.org/html/2610.27524v1)
- Benchmark paired runs at fixed model, settings, tool permissions, revision, and task. Compare memory off, lexical retrieval, and known-useful injected evidence. Include irrelevant, stale, adversarial and contradictory notes, isolated worktrees, and histories that precede the test revision. Track executable success, rescue/harm, first relevant file, repeated failed commands, total input/output tokens including indexing, and end-to-end latency. Retain the no-memory path whenever nothing passes the confidence and provenance checks. [VibeMemBench](https://arxiv.org/html/2609.23570v1) · [MemDev-Bench](https://arxiv.org/html/2610.27524v1)

### Gaps

- These results do not establish the best store, ranking algorithm, or automatic acceptance threshold for this Go harness. Recent papers are prioritized for relevance. Comparative citation counts and independent replications were not verified.

## Which context and harness improvements are supported by evidence?

### Takeaway

Measure what survives compaction by continuing from the same checkpoint. Put reproducible state into deterministic records and use a model summary for useful narrative context. The paper results differ by workload, model, and budget. [Compact-Bench](https://arxiv.org/html/2610.31710v2) · [Same Model, Different Harness](https://arxiv.org/html/2608.26218)

### Cited Findings

- **Compact-Bench** (Sun et al., Oct 28 2026 v2, preprint) pairs full-history and compacted continuations from identical workspace state. Across 14 summarizer models, confirmed damage ranges from 5% to 64% among 22 full-history successes. With Kimi-K3 as summarizer, the Kimi Code prompt retained 19/22 successful continuations. The detail-retention cohort is separate: the best LLM summary retained 16.0% of recurring details versus 79.1% for mechanical extraction. Neither detail-retention rate nor the selected 22 successful continuations represents unconditional pass rate. [Paper, abstract and §3](https://arxiv.org/html/2610.31710v2)
- **Slipstream** (Chen et al., May 9 2026 v1, preprint) builds candidate summaries while the agent continues using full history, then asks a judge to compare against the independently produced continuation. Across its SWE-bench Verified and BrowseComp workloads, the authors report gains up to 8.8 points and latency reductions up to 39.7%. These are maxima across experiments under their scheduling and judge assumptions. [Paper](https://arxiv.org/abs/2605.08580v1)
- **AutoCompact** (Zhang et al., Oct 1 2026, preprint) reports +9.2 and +5.0 percentage points over its base model on SWE-bench Verified and SWE-PolyBench Verified respectively. The method trains the model to decide when and how to compact with supervised corrections and task-success reinforcement learning. These figures cannot be assigned to a runtime heuristic that changes only thresholds. [Paper](https://arxiv.org/abs/2610.02163v1)
- **Same Model, Different Harness** (Aug 2026, preprint) tests a package of working-view selection, detectors, interventions and command safeguards with the same model and tasks in each pair. With Qwen3.6 on its 169-task Verified cohort at 20,480 tokens, 43 control tasks vs 72 treated tasks were solved, a +17.2 percentage-point difference with paired bootstrap 95% interval +10.7 to +23.7. The matched wide-window comparison yielded 102 vs 101, with an interval spanning zero. The design estimates the bundle's effect on selected cohorts, not independent effects of individual rules or a universal context-window curve. [Paper, Tables 6–7](https://arxiv.org/html/2608.26218)

### Inferences

- Prototype **checkpoint plus compaction receipt**: retain the active user request, pending approvals and tool call/result pairings, touched paths with hashes, last failed check and re-run command, and task constraints in a structured view derived from the durable journal. Store the prose summary separately and attach pointers to retained events/artifacts. Allow rollback of the model-facing summary while preserving the underlying journal. Check deterministic invariants first. Consider Slipstream-like asynchronous validation only after paired checkpoint fixtures demonstrate the simpler feature’s limits. [Compact-Bench](https://arxiv.org/html/2610.31710v2) · [Slipstream](https://arxiv.org/abs/2605.08580v1)
- Budget experiments should compare full history, summary only, and structured receipt plus summary on the same checkpoint. Keep unresolved side-effect states first-class, measure downstream success and quality, and record full-system costs. [Compact-Bench](https://arxiv.org/html/2610.31710v2)

### Gaps

- Published compaction savings do not predict this harness’s savings. Test the selected provider’s token accounting and cache behavior and include a repeated-run estimate of variability.

## Which benchmarks fit the Go harness and how should it validate a memory feature?

### Takeaway

Separate deterministic harness correctness from model-dependent task success. Repository repair, terminal execution, cross-session memory, and compaction answer distinct evaluation questions. [Doc2Harness](https://arxiv.org/abs/2610.20344v1) · [SWE-bench](https://proceedings.iclr.cc/paper_files/paper/2024/hash/edac78c3e300629acfe6cbe9ca88fb84-Abstract-Conference.html) · [Terminal-Bench](https://arxiv.org/abs/2601.11868)

### Cited Findings

- **SWE-bench** (Jimenez et al., ICLR 2024) established test-based execution on GitHub repository issues. **SWE-agent** (Yang et al., NeurIPS 2024) shows the historical impact of a model-oriented editing/navigation/execution interface: GPT-4 Turbo with that harness solved 12.47% of the original 2,294 tasks. The historical score is useful for provenance, separate from contemporary Verified or Pro scores. [SWE-bench proceedings](https://proceedings.iclr.cc/paper_files/paper/2024/hash/edac78c3e300629acfe6cbe9ca88fb84-Abstract-Conference.html) · [SWE-agent proceedings](https://papers.neurips.cc/paper_files/paper/2024/file/5a7c947568c1b1328ccc5230172e1e7c-Paper-Conference.pdf)
- **Terminal-Bench 2.0** (ICLR 2026) has 89 terminal tasks, each with a distinct environment, human-written solution and verification tests. Use it for terminal workload coverage with a version pin. Temporal memory should use paired tests with controlled histories, which this terminal task setup does not itself provide. [Conference paper](https://openreview.net/pdf?id=a7Qa4CcHak)
- **Doc2Harness** (Ge et al., Oct 14 2026, preprint) isolates build and evaluation environments and runs frozen harnesses against a scripted model. It scores model requests, disk state, process exit, and command output, with positive controls. Its study uses 328 cases and three independent harness builds for each of 19 models. The authors’ model-level harness-construction percentages are unrelated to this project’s readiness estimate. [Paper](https://arxiv.org/abs/2610.20344v1)

### Inferences

- Extend existing scripted local provider fixtures with deterministic checks for cross-repository denial, invalidation after a file change, provenance visibility, budget/pagination, deletion and approval non-inheritance. This can run in CI and remain independent of live model quality. Add a separate opt-in paired model evaluation for usefulness on temporally valid coding histories. [Doc2Harness](https://arxiv.org/abs/2610.20344v1) · [MemDev-Bench](https://arxiv.org/abs/2610.27524v1)

### Gaps

- No papers were reproduced in this review. Keep current paper-reported metrics and planned local acceptance targets separate.

## Which cheap context and interface choices have evidence? (added 7 Nov 2026)

### Takeaway

Before calling a model to summarize, drop stale tool output. Old observations dominate agent context, and masking them has been reported to match summarization at roughly half the cost of an unmanaged agent. Tool and interface design (search, bounded views, lint-gated edits) changes solve rates more than many model-side tricks. [The Complexity Trap](https://arxiv.org/abs/2508.21433) · [SWE-agent](https://arxiv.org/abs/2405.15793)

### Cited Findings

- **The Complexity Trap** (Lindenbauer et al., Aug 2025 preprint) compares no context management, LLM summarization and observation masking (replacing older tool outputs with a placeholder) for SWE-agent on SWE-bench Verified across five model configurations. Masking roughly halved cost against the unmanaged agent and matched or slightly beat summarization on solve rate. Summaries the paper and Hugging Face page report include Qwen3-Coder 480B moving from 53.8% to 54.8%. A PREreview reports that summarization lengthened trajectories by about 15%. These per-configuration figures were read from the abstract page and the review, not re-derived from the tables. [Paper](https://arxiv.org/abs/2508.21433) · [HF page](https://huggingface.co/papers/2508.21433) · [PREreview](https://zenodo.org/records/23176605)
- **JetBrains Research follow-up** (Dec 2025, blog) reports that masking matched summarization only after tuning the masking window per scaffold. A hybrid approach (masking first, summarizing when still over budget) cut cost a further 7% vs masking and 11% vs summarization with Qwen3-Coder 480B on SWE-bench Verified. Vendor blog, not peer reviewed. [Blog](https://blog.jetbrains.com/research/2025/12/efficient-context-management/)
- **SWE-agent** (Yang et al., NeurIPS 2024) argues agents need purpose-built agent-computer interfaces. Reported 12.5% pass@1 on full SWE-bench and 87.7% on HumanEvalFix with GPT-4 Turbo. Its ablations attribute the largest drop to removing the lint-gated edit command. Results are on 2024 models; the absolute numbers are superseded, the ablation direction is the useful part. [Paper](https://arxiv.org/abs/2405.15793)
- **Agentless** (Xia et al., 2024 preprint) uses a fixed localize, repair, validate pipeline without model-chosen actions. v1 reported 27.33% on SWE-bench Lite at $0.34 per issue; v2 reported 32.00% at $0.70. The figures differ by version. It shows structured pipelines can be strong and cheap baselines. [Paper](https://arxiv.org/abs/2407.01489)
- **Lost in the Middle** (Liu et al., TACL 2024) finds a U-shaped curve: models use information at the start and end of long inputs better than the middle, and performance drops as inputs lengthen even for long-context models. Measured on multi-document QA and key-value retrieval, not coding. [Paper](https://arxiv.org/abs/2307.03172)

### GoHarness experiments

1. **Observation masking before summarization (being implemented as RC21).** When compaction triggers, first replace tool results older than the latest N turns with a short placeholder that keeps the call ID, tool name, status and byte count. Call the summarizer only if the masked history still exceeds the budget. Measure with the existing compaction fixtures: summarizer calls avoided, tokens sent, and whether follow-up turns keep the needed facts. Keep the full results in the journal, so masking changes only the model view.
2. **Lint-gated edits.** After an edit, run the configured formatter or LSP diagnostics and return problems with the result. SWE-agent's ablation suggests this is the highest-value interface element. GoHarness already runs formatters; measure diagnostics-on vs off on a small fixed task set.
3. **Placement.** Put the active task, pending approvals and the latest failing check at the end of the model view, and stable instructions at the start, following the U-shaped finding. Test as a paired change on compaction continuations.

### Gaps

Not verified: the exact per-model tables in The Complexity Trap, and whether its window size transfers to GoHarness's tool output sizes. Experiment 1 must tune and report its own window.

### Status update (RC33, 7 Nov 2026)

- Experiment 1 (masking) shipped in RC21; fixture measurement in BENCHMARKS.md.
- Experiment 2 (lint-gated edits) shipped in RC25 as `lsp_errors` from an
  already-running language server; task-success effect unmeasured.
- Experiment 3 (placement) is blocked on live checks. Finding: task state is
  appended to the system message, so every todo update changes the start of
  the request and defeats provider prompt-prefix caching for the whole
  conversation. Moving it to a trailing position helps both caching and the
  recency side of the U-shaped curve, but the encoding differs across Chat
  Completions, Responses and Anthropic Messages and must be verified live.
- Memory: the existing ledger already pins notes to file hashes and rejects
  stale evidence, matching the applicability findings above. The paired
  memory-on/off experiment needs a live model.

### Status update (RC37, 8 Nov 2026)

- Experiment 3 implemented locally. Task state is now a trailing user
  message marked "[Session state, not a new request]", added only to the
  outgoing request (never journaled) and never between a tool call and its
  result. A unit test confirms every earlier message is byte-identical
  across todo updates; encoder tests confirm Anthropic Messages and Responses
  accept a state message after tool results (Chat Completions passes
  messages through). Still unmeasured: real cache-hit rates
  (`cache_read` usage) and whether models mistake the state for a request.
  Both need the live run.
- Journal records carry times (RC36), so later experiments can measure
  per-turn latency from the journal alone.
