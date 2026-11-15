# Compound hard tasks on rc.159 (2 attempts each)

`gpt-6.1-sol`, ChatGPT sign-in. Each task hides three bugs in different files behind one vague
release report; a run passes only if all three hidden test sets pass.

| Arm | Runs | Resolved | False completions | Budget stops | Avg tool calls | Avg seconds | Input tokens | Cached | Output tokens |
|---|---|---|---|---|---|---|---|---|---|
| base | 8 | 8 (100%) | 0 | 0 | 20.9 | 139 | 753601 | 53248 | 33285 |

All four tasks solved twice, with about twice the tool calls and time of the single-bug tasks. The
self-authored suite is now saturated for this model: further differences need larger, real
codebases (mined commits, which require a container) rather than more synthetic bugs.
