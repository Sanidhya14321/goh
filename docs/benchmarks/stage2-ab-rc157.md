# Stage 2 A/B on rc.157 (hard set, 1 attempt per arm)

Model `gpt-6.1-sol` over the ChatGPT sign-in, 10 hard tasks, arms `base` (completion check on) and
`nocheck` (`--completion-check=false`). Graded in a clean copy with hidden tests.

| Arm | Runs | Resolved | False completions | Avg tool calls | Avg seconds | Input tokens | Cached | Output tokens |
|---|---|---|---|---|---|---|---|---|
| base | 10 | 9 | 1 | 10.2 | 64 | 308733 | 11008 | 17893 |
| nocheck | 10 | 10 | 0 | 10.3 | 64 | 301128 | 30720 | 18441 |

Notes:
- The completion check fired in **0 of 20 runs**: the model always ran checks itself, so both arms ran
  the same harness behaviour and the difference is run-to-run noise.
- The `base` false completion was again `hard-py-double-charge` (it is now 2 of 3 base attempts across
  stages), a model failure the reminder cannot address: the model's own tests passed.
- The `base` `hard-go-stale-price` run failed on a dropped stream (`interrupted stream: missing finish
  or DONE`); the partial work still passed grading. Not retried by design (no automatic replay).
- Conclusion: for this model the completion check costs nothing and does not trigger; further A/B
  runs of this arm are not worth the quota. Better targets: the false-completion pattern (checking
  changes against the real dependency rather than a self-written fake) and stream-drop recovery.
