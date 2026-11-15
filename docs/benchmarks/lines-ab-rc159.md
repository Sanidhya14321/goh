# Line-range edit A/B on rc.159 (hard set, 1 attempt per arm)

`gpt-6.1-sol`, ChatGPT sign-in, 10 single-bug hard tasks; `lines` adds `--line-edits`.

| Arm | Runs | Resolved | False completions | Avg tool calls | Avg seconds | Input tokens | Cached | Output tokens |
|---|---|---|---|---|---|---|---|---|
| base | 10 | 10 | 0 | 10.7 | 69 | 350421 | 46720 | 18629 |
| lines | 10 | 10 | 0 | 11.2 | 76 | 372550 | 7552 | 20374 |

- With line ranges available, the model used one in 13 `edit_file` calls; it kept exact-text edits
  and `apply_patch`. Both arms solved everything, and no edit failed in either.
- Decision: keep line edits off by default (C1 closed). They add schema text and give this model
  nothing; the exact-text edit with the whitespace-tolerant fallback is sufficient.
- This also confirms the fixed `hard-py-double-charge` (solved in both arms).
