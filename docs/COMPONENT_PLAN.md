# Component-by-Component Completion Plan

The goal is a harness that is complete and ready to benchmark with `gpt-6.1-sol` over the ChatGPT sign-in.
Each component is reviewed against research papers, open-source harnesses and articles. Findings become
tested changes, one release at a time. Status: ⬜ not started · 🔄 in progress · ✅ done.

| # | Component | Code | Status |
| --- | --- | --- | --- |
| C1 | Tool calling and file editing (edit format, matching, errors) | `internal/tools` | ✅ RC146 |
| C2 | Agent loop and control (planning, stopping, error recovery, verification) | `internal/runtime` | ✅ RC140–147 |
| C3 | Context and memory (compaction, masking, pinned state, project memory) | `internal/runtime`, `internal/memory` | ✅ RC138–148 |
| C4 | Repository understanding (repo map, search, LSP) | `internal/tools/repomap.go`, `internal/lspclient` | ✅ RC149 |
| C5 | Providers and model routing (Responses/ChatGPT, retries, caching) | `internal/provider`, `internal/core` | ✅ RC139–150 (live check pending) |
| C6 | Safety and permissions (approvals, sandbox, hooks, injection) | `internal/runtime`, `internal/sandbox` | ⬜ |
| C7 | Sessions and durability (journal, resume, snapshots) | `internal/session`, `internal/projectsnap` | ⬜ |
| C8 | Subagents and orchestration | `internal/runtime/subagents.go` | ⬜ |
| C9 | Integrations (MCP, web, formatters) | `internal/mcpclient`, `internal/webfetch` | ⬜ |
| C10 | Interfaces (CLI headless mode, TUI, ACP, API) | `cmd/harness`, `internal/tui` | ⬜ |
| C11 | Benchmark runner (pinned tasks, clean regrading, metrics) | `cmd/bench` (new) | ⬜ |

## Findings log

Each component section records the sources consulted, the improvements chosen, and the release that
delivered them.

### C1 Tool calling and file editing (RC146)

Sources:
- [Pre-action verification for LLM agents (arXiv:2609.11957)](https://arxiv.org/pdf/2609.11957): fuzzy search/replace applies 97.7% of stressed edits against 17.6% for exact matching, but silently misapplies about 2.3% unless ambiguity is refused.
- [Copy-as-Decode (arXiv:2604.18170)](https://arxiv.org/pdf/2604.18170): search/replace failures come from anchor ambiguity.
- [Edit format study (arXiv:2510.12487)](https://arxiv.org/html/2510.12487v2): search/replace is best for strong models.
- [The harness problem](https://blog.can.ac/2026/02/12/the-harness-problem): patch is the weakest format for most models, and the edit tool alone moved one model from 6.7% to 68.3%.
- [Four CLI agents on one model](https://charlesazam.com/blog/deepdive-benchmark/): Gemini CLI and OpenCode fall back to fuzzy matching; Mistral Vibe uses it only for errors.

Chosen: keep exact matching and hash guards; add a whitespace-tolerant fallback that requires a unique match, plus precise error locations. Not chosen yet: a line-anchored ("hashline") edit tool. It is worth an A/B test in the benchmark (C11) rather than adopting blind, since reports disagree by language.

### C2 Agent loop and control (RC140–RC147)

Sources:
- [ECLoop: evidence-conditioned execution (arXiv:2607.28815)](https://arxiv.org/pdf/2607.28815): gating commitment on task evidence adds 4.8–11.8 points of Pass@1 on SWE-bench Verified and saves tokens.
- [DeepSWE (arXiv:2607.07946)](https://arxiv.org/pdf/2607.07946): weaker models often submit without running any test (18% of runs for one model).
- [Premature completion pattern](https://agentpatterns.ai/patterns/anti-patterns/premature-completion/): harness-only pre-completion checks moved Terminal Bench 2.0 from 52.8% to 66.5%, though they may add cost for strong models.
- [Sonar on loop verification](https://www.sonarsource.com/blog/loop-engineering-without-verification-is-just-automation/): external checks, not model claims, should gate completion.

Chosen:
- run outcomes (RC140);
- the user-configured verify/repair cycle (RC144);
- shared budgets (RC145);
- a one-time evidence reminder when changes are unchecked (RC147), switchable for A/B tests in C11.

Found and fixed along the way: automatic messages were counted as user turns.

### C3 Context and memory (RC138–RC148)

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

### C4 Repository understanding (RC149)

Sources:
- [Repository map pattern](https://www.agentpatterns.ai/context-engineering/repository-map-pattern/): Aider's tree-sitter and PageRank map; Claude Code chose agentic search instead.
- [Codebase-Memory (arXiv:2603.27277)](https://arxiv.org/html/2603.27277): structural index over MCP.
- [Structural indexing for Codex CLI](https://codex.danielvaughan.com/2026/07/06/structural-codebase-indexing-coding-agents-codex-cli-mcp-knowledge-graph-navigation/).
- [Monorepo index vs grep](https://dailyaiworld.com/blogs/monorepo-coding-agents-structural-index-grep-verdict) (unverified figures).

The evidence is thin and largely vendor-run. Gains concentrate in multi-file changes; small repositories do fine with grep.

Chosen: keep agentic search (grep, glob, read and LSP) as the primary path, and extend the cheap declaration map to the common benchmark languages. Not chosen: a persistent graph index or PageRank ranking without evidence. Revisit after C11 measures time-to-right-file.

### C5 Providers and routing (RC131, RC139, RC143, RC150)

Sources:
- [Responses API reference](https://developers.openai.com/api/reference/python/resources/responses/methods/create): `prompt_cache_key` groups requests for cache hits; `include: reasoning.encrypted_content` keeps stateless reasoning.
- [Reasoning items cookbook](https://learn.chatgpt.com/cookbook/examples/responses_api/reasoning_items): Responses raised cache utilisation from 40% to 80% in OpenAI's test.
- [Encrypted reasoning with store=false](https://community.openai.com/t/how-to-use-reasoning-encrypted-content-with-store-false-stateless/1286934): pass reasoning items back every turn.
- [xAI agent caching guide](https://docs.x.ai/developers/model-capabilities/text/best-practices): one cache key per conversation, append-only.
- The claude-api reference: cache breakpoints, adaptive thinking, effort.

Already in place: stateless Responses with encrypted reasoning passed back. Added: Anthropic breakpoints (RC131), correct usage attribution (RC139), current effort levels and adaptive thinking (RC143), and a per-session `prompt_cache_key` with a safe fallback (RC150).
