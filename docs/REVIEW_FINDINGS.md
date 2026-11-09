# External Review Findings Tracker

This tracks every finding from the external review (9 November 2026) and its status. A finding is
**validated** only when a test reproduces it on the old code. It is **fixed** only when that test passes and
the full gates pass.

| ID | Pri | Finding | Status |
| --- | --- | --- | --- |
| R1 | P0 | Hooks race with parallel reads (shared event sequence) | ✅ Validated (race detector) and fixed RC136: serialized event emission |
| R2 | P0 | Post-tool hook output merged before the tool result is durable | ✅ Fixed RC136: raw result journaled first, then hooks run |
| R3 | P0 | Hook subprocesses ignore the sandbox | ✅ Fixed RC136: hooks wrapped by the same sandbox policy |
| R4 | P0 | A later call in a batch runs after the run decided to halt | ✅ Validated (marker created) and fixed RC136 |
| R5 | P0 | Identical batched reads bypass the repeat guard; key order disguises repeats | ✅ Validated and fixed RC136: canonical call keys, siblings re-checked |
| R6 | P0 | `file_rules` deny applies to edits only; reads, search and attachments ignore it | ✅ Fixed RC137 (validated by code reading) |
| R7 | P0 | Best-effort snapshot rejects a `.git` path, then the write proceeds | ✅ Validated (git hook planted) and fixed RC137 |
| R8 | P0 | Granted `webfetch` can reach loopback and private services | ✅ Fixed RC137: connect-time address check, opt-in for private |
| R9 | P0 | MCP validates a decoded object but forwards the raw bytes (duplicate keys) | ✅ Fixed RC137 |
| R10 | P1 | Resume replaces a durably recorded tool result with an uncertainty placeholder | ✅ Validated and fixed RC138 |
| R11 | P1 | A tool-free reply always counts as `completed` (no verified/unverified outcome) | ✅ Fixed RC140: outcome field; empty reply fails |
| R12 | P1 | One large early result ends the run with `budget_exhausted` before masking | ✅ Validated and fixed RC138: excerpt plus receipt |
| R13 | P1 | The masking placeholder tells the model to rerun a tool (risky for effects) | ✅ Fixed RC138: typed receipts |
| R14 | P1 | Fallback usage is charged to the primary model; cache reads and writes are collapsed | ✅ Fixed RC139: per-call usage with purpose and cache split |
| R15 | P1 | Session grants are keyed by tool name, not by the reviewed action | ✅ Fixed RC141: command prefix / web origin scopes |
| R16 | P1 | Oversized MCP result after a successful action invites a repeat under the grant | ✅ Fixed RC140 |
| R17 | P2 | Standing user constraints can be lost through compaction | ✅ Validated and fixed RC142: verbatim pinned instructions |
| R18 | P2 | Recalled memory notes are not revalidated on replay | ✅ Fixed RC142 |
| R19 | P2 | Provider capability gaps: OpenAI effort levels beyond `high`; Anthropic adaptive thinking | ✅ Fixed RC143 (fixture-verified; live check pending) |
| R20 | P2 | Repository guidance is appended inside the system message | ✅ Fixed RC143: separate untrusted user message |
| R21 | P3 | Verify/repair workflow, shared run budgets, adversarial fixtures, benchmark suite | ◐ RC144: verify/repair, max_tool_calls and adversarial canary done; wall-time/cost budgets across children and the live benchmark suite remain (benchmark needs live keys) |
| R22 | — | (Found by the canary) Sandboxed shell could read files denied by `file_rules` | ✅ Fixed on macOS RC144; Linux Landlock limit documented |
