# Design decisions

Why GoHarness works the way it does, component by component. Each section lists the sources that
were weighed (papers, open-source harnesses, articles), what was chosen, and the release that
delivered it. Vendor figures are marked as such and should be treated with caution.

For the code layout see [ARCHITECTURE.md](../ARCHITECTURE.md); for measured results see
[benchmarks](../benchmarks/README.md).

## Tool calling and file editing (RC146)

Sources:
- [Pre-action verification for LLM agents (arXiv:2609.11957)](https://arxiv.org/pdf/2609.11957): fuzzy search/replace applies 97.7% of stressed edits against 17.6% for exact matching, but silently misapplies about 2.3% unless ambiguity is refused.
- [Copy-as-Decode (arXiv:2604.18170)](https://arxiv.org/pdf/2604.18170): search/replace failures come from anchor ambiguity.
- [Edit format study (arXiv:2510.12487)](https://arxiv.org/html/2510.12487v2): search/replace is best for strong models.
- [The harness problem](https://blog.can.ac/2026/02/12/the-harness-problem): patch is the weakest format for most models, and the edit tool alone moved one model from 6.7% to 68.3%.
- [Four CLI agents on one model](https://charlesazam.com/blog/deepdive-benchmark/): Gemini CLI and OpenCode fall back to fuzzy matching; Mistral Vibe uses it only for errors.

Chosen: keep exact matching and hash guards; add a whitespace-tolerant fallback that requires a unique match, plus precise error locations. Line-anchored edits were built as an opt-in (RC159) and A/B tested: gpt-6.1-sol used them once in 13 edits with no change in results, so they stay off.

## Agent loop and control (RC140–RC147)

Sources:
- [ECLoop: evidence-conditioned execution (arXiv:2607.28815)](https://arxiv.org/pdf/2607.28815): gating commitment on task evidence adds 4.8–11.8 points of Pass@1 on SWE-bench Verified and saves tokens.
- [DeepSWE (arXiv:2607.07946)](https://arxiv.org/pdf/2607.07946): weaker models often submit without running any test (18% of runs for one model).
- [Premature completion pattern](https://agentpatterns.ai/patterns/anti-patterns/premature-completion/): harness-only pre-completion checks moved Terminal Bench 2.0 from 52.8% to 66.5%, though they may add cost for strong models.
- [Sonar on loop verification](https://www.sonarsource.com/blog/loop-engineering-without-verification-is-just-automation/): external checks, not model claims, should gate completion.

Chosen:
- run outcomes (RC140);
- the user-configured verify/repair cycle (RC144);
- shared budgets (RC145);
- a one-time evidence reminder when changes are unchecked (RC147), switchable for benchmark A/B tests.

Found and fixed along the way: automatic messages were counted as user turns.

## Context and memory (RC138–RC148)

Sources:
- [The Complexity Trap (arXiv:2508.21433)](https://arxiv.org/abs/2508.21433): observation masking matches summarization's solve rate at about half the cost; the best masking window kept the last 10 observations; a hybrid saves a further 7–11%.
- [Compaction vs summarization](https://www.morphllm.com/compaction-vs-summarization) (vendor blog, treat figures with caution): summarization loses detail and causes re-reading loops; verbatim retention is preferred.
- From the external review: *Lost in Context Compaction* (keep constraints separately) and *Bad Memory* (threat-model automatic memory writes).

Chosen:
- the hybrid already in place: mask first, summarize only when needed;
- typed receipts (RC138);
- verbatim pinned user instructions (RC142);
- revalidated memory recall (RC142);
- a 10-result masking window (RC148).

Memory writes stay user-initiated only, by design.

## Repository understanding (RC149)

Sources:
- [Repository map pattern](https://www.agentpatterns.ai/context-engineering/repository-map-pattern/): Aider's tree-sitter and PageRank map; Claude Code chose agentic search instead.
- [Codebase-Memory (arXiv:2603.27277)](https://arxiv.org/html/2603.27277): structural index over MCP.
- [Structural indexing for Codex CLI](https://codex.danielvaughan.com/2026/07/06/structural-codebase-indexing-coding-agents-codex-cli-mcp-knowledge-graph-navigation/).
- [Monorepo index vs grep](https://dailyaiworld.com/blogs/monorepo-coding-agents-structural-index-grep-verdict) (unverified figures).

The evidence is thin and largely vendor-run. Gains concentrate in multi-file changes; small repositories do fine with grep.

Chosen: keep agentic search (grep, glob, read and LSP) as the primary path, and extend the cheap declaration map to the common benchmark languages. Not chosen: a persistent graph index or PageRank ranking without evidence. Revisit if benchmarks show time spent finding the right file.

## Providers and routing (RC131, RC139, RC143, RC150)

Sources:
- [Responses API reference](https://developers.openai.com/api/reference/python/resources/responses/methods/create): `prompt_cache_key` groups requests for cache hits; `include: reasoning.encrypted_content` keeps stateless reasoning.
- [Reasoning items cookbook](https://learn.chatgpt.com/cookbook/examples/responses_api/reasoning_items): Responses raised cache utilisation from 40% to 80% in OpenAI's test.
- [Encrypted reasoning with store=false](https://community.openai.com/t/how-to-use-reasoning-encrypted-content-with-store-false-stateless/1286934): pass reasoning items back every turn.
- [xAI agent caching guide](https://docs.x.ai/developers/model-capabilities/text/best-practices): one cache key per conversation, append-only.
- The claude-api reference: cache breakpoints, adaptive thinking, effort.

Already in place: stateless Responses with encrypted reasoning passed back. Added: Anthropic breakpoints (RC131), correct usage attribution (RC139), current effort levels and adaptive thinking (RC143), and a per-session `prompt_cache_key` with a safe fallback (RC150).

## Safety and permissions (RC130–RC144)

Sources:
- [Spotlighting (arXiv:2403.14720)](https://arxiv.org/pdf/2403.14720): datamarking beats delimiting against simple attacks, and randomized markers are needed.
- [Spotlighting/datamarking implementation notes](https://github.com/realArcherL/spotlighting-datamarking): does not hold against adaptive attackers.
- [Layered defenses](https://www.deepinspect.ai/blog/spotlighting-prompt-injection-defense) (vendor): keep tool calls and outbound requests behind independent policy checks.
- From the review: CaMeL, AgentDojo, GitInject.

Decision: enforcement independent of the model is the primary defence. That means the environment filter, sandbox, deny rules on every path, no `.git` writes, private-address refusal, scoped grants and fail-closed hooks, all exercised by the adversarial canary (RC144). Datamarking is **not** applied to file content, because it rewrites whitespace in code the model must edit exactly. External web and MCP results already carry explicit untrusted labels.

## Sessions and durability

Reviewed: every journal record is `fsync`ed before returning, damaged journals can be salvaged, and recovery is journal-derived (RC138). No change needed. `/snapshots` in the TUI (RC155) lists project snapshots with exact restore commands.

## Subagents

Decision: keep children sequential and in the foreground, with shared tool-call and time budgets (RC145) and no nested delegation. The public evidence on multi-agent systems shows breadth-first research gains at a large token cost, and parallel editing agents conflict on shared files. Parallel read-only exploration is a candidate for a benchmark A/B test, not a default.

## Integrations (RC135, RC137, RC151)

MCP has reconnect (RC135), duplicate-key and validated-object safety (RC137), completed-but-oversized results (RC140) and resources (RC151). MCP prompts as `/mcp:<server>:<prompt>` (RC160). Still open: OAuth, which needs a real OAuth server.

## Interfaces (RC152)

Headless `goh run` is benchmark-ready: per-run verify and budget flags, `--allow all`, and distinct exit codes.

## Benchmark runner (RC153)

Sources:
- [SWE-bench harness reference](https://swebench.com/SWE-bench/reference/harness): isolated environments, hidden tests, resolved means FAIL_TO_PASS plus PASS_TO_PASS.
- [Claw-SWE-Bench write-up](https://www.besthub.dev/articles/how-harness-design-alters-coding-agent-scores-insights-from-the-first-independent-claw-swe-bench-8db2751ea3e2) (secondary): harness design alone moves Pass@1 by up to 27 points.
- [Position: coding benchmarks are misaligned (arXiv:2606.17799)](https://arxiv.org/html/2606.17799): a single pass rate hides process quality.
- From the review: DeepSWE, Terminal-Bench 2.1 and SWE-Bench Pro on verifier fixes and time budgets.

Built: clean regrading with a hidden-test overlay and tamper detection; false-completion, budget, tool-call, time and token metrics; arms for one-change-at-a-time A/B tests. The suite has since grown to 22 basic, 14 hard and 19 real-project tasks; results are in [benchmarks](../benchmarks/README.md).
