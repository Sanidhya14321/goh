# Persistent memory and evaluation for local coding agents

## What established memory designs are applicable to a local harness?

### Takeaway

Keep working context, durable session evidence, and reusable cross-session knowledge distinct. Published designs support selective retrieval from an external store and structured experience notes, with substantial variation in their evidence and costs.

### Cited Findings

- Packer et al.'s MemGPT (2023, v2 February 2024) treats the model window as a fast tier and recall/archival storage as a slower tier navigable with model-issued functions. It evaluated document analysis and extended conversations on then-current models. Its architecture suggests durable context outside a model window, while its task results address domains other than repository editing. [MemGPT paper](https://arxiv.org/html/2310.08560v2)
- Park et al.'s Generative Agents (UIST 2023) keeps a natural-language experience stream and retrieves with recency, importance and relevance, then synthesizes higher-level reflections used for planning. The study uses simulated social behavior, a useful precedent for linking derived notes back to observations. [Generative Agents paper](https://arxiv.org/abs/2304.03442)
- Xu et al.'s A-Mem (NeurIPS 2025) uses notes with descriptions, keywords and tags, creates links to earlier records, and can revise records when new ones arrive. Its published QA comparisons are conversational; the later VibeMemBench evaluation of A-MEM reports negative point estimates on all three tested coding solvers. [A-Mem paper](https://proceedings.neurips.cc/paper_files/paper/2025/hash/19909c36f51abc4856b4560aff3d36d6-Abstract-Conference.html) [VibeMemBench Table 2](https://arxiv.org/html/2609.23570v1)
- Chhikara et al.'s Mem0 (April 2025 preprint) extracts facts and selects add/update/delete/no-op. It reports a 26% **relative** LLM-judge gain against the paper's OpenAI memory baseline on LoCoMo, Table 2 reports J scores of 66.88% for base Mem0, 68.44% for graph Mem0, 52.90% for OpenAI, and 72.90% for full-context. The respective end-to-end p95 times were 1.440, 2.590, 0.889 and 17.117 seconds in that experiment. This is a vendor-authored comparison with a specific historical setup and judge. The study's figures should stay separate from coding success claims. [Mem0 §§2, 4](https://arxiv.org/html/2504.19413)
- Letta's February 2026 Context Repositories is a product implementation precedent for editable local files, explicit agent navigation and Git versioning. Its July 2026 Context-Bench V2 description uses anonymized production memory profiles and cleaned variants, but the benchmark itself is private. Treat these sources as architecture descriptions and vendor observations. [Context Repositories](https://www.letta.com/blog/context-repositories/) [Context-Bench V2](https://letta.one/blog/evaluating-memory-in-production-agents/)
- LongMemEval-V2 (May 2026 preprint) evaluates 451 curated queries about specialized web environments and constructs histories up to 500 trajectories/115M tokens. On tested small/medium tiers, AgentRunbook-C lets a coding agent navigate trajectory files plus manifests and verification helpers. Accuracy averages 72.5% versus 69.3% for off-the-shelf Codex, 57.8% for AgentRunbook-R, and 48.5% for the strongest simple RAG baseline with slices plus notes. Table 2 gives C 74.9/70.1%, Codex 69.9/68.7%; small-tier overall improvement is marked significant, medium-tier overall is not. C query time 108.3/139.9 s versus Codex 177.2/185.8 s, R 26.9/25.8 s and simple slice-plus-notes 0.2/0.3 s. The downstream reader was fixed Qwen3.5-9B, while coding controller was GPT-5.4-mini at high effort. This establishes file-based evidence navigation as an effective option in that environment, at significant latency; it does not establish fully offline inference or coding task gains. [LongMemEval-V2 Table 2 and §5](https://arxiv.org/html/2605.12493v1)
- NeurIPS 2025 MINJA demonstrates that ordinary queries can cause agents to store malicious memory records without direct store access. The demonstrated threat motivates separating stored text from control authority even when the store is local. The measured attack rates depend on tested agents and attacker assumptions. [MINJA paper](https://proceedings.neurips.cc/paper_files/paper/2025/hash/42a97bbd9844d2bf68596730af80bcdf-Abstract-Conference.html)

### Inferences

- Begin with local readable records and a small, deterministic metadata/lexical index. Use typed notes about procedures, verified repository facts, user preferences and failed approaches, with an evidence pointer, repository/branch or commit scope, status, and date. Keep a queryable link to the raw record while injecting only compact hints. This combines the evidence retention and selective navigation patterns above.
- Require explicit promotion or trusted run evidence for reusable notes. Present recovered notes as scoped claims, and check live files, permissions and task constraints independently before acting. Preserve old versions or supersession relationships so uncertainty and conflicts remain inspectable.
- Distinguish local persistence from local computation. A local store with a hosted model still exposes any retrieved context included in the outgoing prompt. Benchmarking the storage strategy should account for retrieval cost separately from model execution.

### Gaps

- The above sources do not establish SQLite versus files as a universal winner or imply that a vector database is necessary.
- Private production benchmarks cannot validate claims about the target harness. The papers do not resolve retention/deletion policy, trusted promotion thresholds, or safe cross-worktree identity for this project.

## Does persistent memory improve execution on real coding tasks?

### Takeaway

Verified prior experience can save search effort and sometimes rescue difficult tasks. The strongest recent direct tests also find regressions and retrieval failures, so evaluate harmful reuse as a first-class outcome.

### Cited Findings

- MemDev-Bench (Chen et al., arXiv October 20, 2026) pairs 555 target tasks with 4,869 executable historical tasks, selected from older PRs and labeled for relevance. Evaluation uses only the Verified subset, 344 target tasks and 2,987 historical tasks across 12 Python repositories, Claude Code plus DeepSeek-V4-Flash, and no automatic compaction. Blocking public-solution sources lowered nominal no-memory resolve from 93.31% to 73.16% on all 344, illustrating answer leakage in evaluation. Main treatment cohorts are 133 baseline-hard Improvement tasks and a 50-task Preservation sample from baseline-easier tasks. [MemDev-Bench §3](https://arxiv.org/html/2610.27524v1)
- On MemDev Improvement tasks, no-memory resolves 35.3%, FullTraj 39.6%, and the best structured system 42.6% (Claude-Mem under Repo history). Structured systems show 9.9–13.2% Rescue and 4.8–9.9% Harm. On Preservation tasks no-memory resolves 94.0%, versus averages 93.5% for Noisy and 93.3% for Repo history across five memory systems. FullTraj analysis links lower agent step counts to process information missing from compact memories. Repo history took 113.1 steps versus 185.1 for NoMem, with reported input tokens increasing from 6.45M to 32.86M. These are author-run results on one agent/model. [MemDev-Bench Tables 2–3, §§4–6](https://arxiv.org/html/2610.27524v1)
- VibeMemBench (Fan et al., September 20, 2026 preprint) has 111 target tasks from 90 SWE-rebench V2 repos and 3,634 historical trajectories. Frozen verified experience yields gains of 1.1–4.5 pp on four of five solvers, with no change on the fifth. Every paired bootstrap interval includes zero. Eleven of twelve tested retrieval framework/solver combinations are at or below matched no-memory, and GLM-5+Mem0 has a wholly negative 95% interval (−10.59 to −0.45 pp). An irrelevant-memory injection control is 0.7–3.4 pp below no-memory on its three solvers. The selected tasks had a positive uplift for the reference solver by design and the transfer pool omits Claude and GPT. Separate useful information availability from autonomous retrieval success. [VibeMemBench Tables 1–2](https://arxiv.org/html/2609.23570v1)
- Stratum (Sehgal et al., October 26, 2026 preprint) puts repository and experience layers behind paged tools. Across eight real bugs in requests and Flask, root-cause-file re-diagnosis is 21/24 with notes from a different model versus 6/24 with retrieval only, median 43.3k versus 196.6k input tokens. Entire-repo single-call context gives 14/16 at 164.4k tokens per call, and a pooled store gives 8/8 in one trial per bug. The successful writer cost median 83k once. Eight already-diagnosed bugs and two small repos limit transfer to independent tasks and patch resolution. The paper reports a trial-level Fisher test, while trials share bugs. [Stratum Table I and §V](https://arxiv.org/html/2610.36153v1)
- SWE-ContextBench (Zhu et al., 2026 preprint v3) compares oracle retrieval and autonomous retrieval with full trajectories and summaries. In its 99-task Lite comparison, no-context resolves 26.26%, Oracle Summary resolves 34.34%, Free Summary 22.22%, Mem0 24.24%, OpenViking 29.20% and Supermemory 30.30%. Free and oracle conditions answer different questions about retrieval quality and should be kept separate. [SWE-ContextBench v3](https://arxiv.org/html/2602.08316v3)
- SWE-Bench-CL (Joshi et al., 2025 preprint) organizes issues chronologically by repository and proposes forward/backward transfer, forgetting and efficiency measures. Its abstract describes a preliminary analysis and evaluation protocol; use it as a design precedent rather than a quantified effectiveness result. [SWE-Bench-CL](https://arxiv.org/abs/2507.00014)

### Inferences

- Use temporally valid history with both relevant and irrelevant distractor tasks. Test what happens after a migration, branch switch or failed approach, and require current evidence before reuse.
- Preserve process details such as the command that actually ran, its exit/result, where search started, a failed hypothesis and the condition that invalidated it. Add small references to transcripts rather than indiscriminately injecting them.
- Keep paired no-memory, raw-history, oracle-evidence and automatically retrieved arms. Compare both rescue and harm per task under identical model/tools/budgets. Freeze historical artifacts before scoring and isolate target fixes from public retrieval.

### Gaps

- No result establishes a universal cross-model or cross-repository gain. The cited large benchmarks are recent preprints and this pass did not verify independent replications.
- A root-cause-file hit is a diagnostic endpoint, whereas a passing patch is a task outcome. Neither alone demonstrates durable, secure, low-cost local operation.

## How should persistent memory be evaluated?

### Takeaway

Pair end-to-end execution with tests that locate the failure layer. Report effectiveness, regressions, costs and change handling under the same model and retrieval budget.

### Cited Findings

- LongMemEval (Wu et al., ICLR 2025) has 500 manually created questions covering information extraction, multi-session reasoning, temporal reasoning, knowledge updates and abstention. Its S and M histories use about 115k and 1.5M tokens per problem. It explicitly decomposes system design into indexing, retrieval and reading. [LongMemEval](https://arxiv.org/html/2410.10813)
- LoCoMo (Maharana et al., ACL 2024) uses multi-session synthetic persona conversations grounded in event graphs and human editing, plus QA, event summarization and dialogue generation tasks. Its conversational recall score should not be used as a proxy for code resolution. Later implementations differ in whether adversarial items are scored and which judge is used. EvalMem's stated LoCoMo run uses all 1,986 QA items with gpt-4o-mini as its default backbone. [LoCoMo](https://aclanthology.org/2024.acl-long.747/) [EvalMem Table 2](https://arxiv.org/html/2609.22231v1)
- MemoryAgentBench (Hu et al., July 2025 preprint) incrementally supplies content and tests accurate retrieval, test-time learning, long-range understanding and conflict resolution. FactConsolidation controls contradictory updates and their ordering. Main comparison multi-hop conflict accuracy reaches at most 6% in the reported setup, with stronger reasoning variants discussed separately. This supports a conflict/update stress test as its own category. [MemoryAgentBench §§3–4](https://arxiv.org/html/2507.05257v1)
- Mem2ActBench (ACL 2026) measures whether memory enables correct tool selection and parameter grounding. It reverse-generates 400 tool tasks from evolving memory chains, with 91.3% judged strongly memory-dependent in human validation. Its task construction is distinct from real repository issues. [Mem2ActBench](https://aclanthology.org/2026.acl-long.370/)
- EvalMem (September 2026 preprint) separates whether a fact was stored, whether native retrieval exposed usable evidence, and whether generation answers given oracle evidence. Across seven systems in default LoCoMo its mean retrieval, encoding and generation defect incidences are 22.1%, 7.7% and 6.5%; LongMemEval-S figures are 18.3%, 8.5% and 5.4%. These are multi-label diagnostic rates in that setup and should not be treated as mutually exclusive or independent failures. [EvalMem §§3–4](https://arxiv.org/html/2609.22231v1)
- ContextWeave (August 2026 preprint) reconstructs sequential office workflows and measures workspace outcome, preferences and misleading recall under a fixed model. Best tested component improves Workspace Score 68.08→78.20 and Preference Score 41.50→70.60. These are benchmark-specific scores on downstream work, not percentages of coding tasks resolved. [ContextWeave](https://arxiv.org/html/2608.04830v1)

### Inferences

- In a deterministic harness suite, record capture precision, evidence retention, retrieval Recall@k/MRR, relevance under a byte/token cap, stale/contradiction outcomes, and whether given oracle evidence the model executes the right command or edits the right file. Treat persisted provenance and authority boundaries as testable properties.
- For downstream testing, measure paired patch pass rate, rescue, harm, time, model tokens, tool steps, write-time processing, storage growth and p50/p95 retrieval. Report CIs clustered at task/repository level or clearly disclose repeated seed dependence. Report model, retrieval recipe, histories, splits, judge or execution rubric and fixtures, including invalid/no-memory cases.
- Add targeted cases for restart/reopen, previous failures, changed test commands, branch-scoped conflicting facts, unrelated projects, and hostile material embedded in retrieved text. Verify that raw histories, model summaries and confirmed notes remain distinguishable after compaction.

### Gaps

- Conversational and web-environment papers provide useful diagnostics but do not directly establish thresholds for the Go harness.
- No checked paper gives a universal safe memory admission threshold or proves that a particular local database architecture prevents memory poisoning. Production acceptance needs a project-specific threat model and fault injection.
