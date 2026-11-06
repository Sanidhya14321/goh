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
