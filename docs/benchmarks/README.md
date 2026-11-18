# Benchmarks

Agent benchmarks run real coding tasks through `goh run` with
[`goh-bench`](../../bench/README.md). Every attempt is regraded in a clean copy against hidden
tests the agent never saw, so a run counts as resolved only when the real tests pass. A *false
completion* is a run that reported success but failed grading.

All runs below use `gpt-6.1-sol` over the ChatGPT sign-in, except the last, which uses `gpt-6.2-sol`. Runs cut off by plan usage limits are
excluded.

| Run | Harness | Tasks | Attempts | Resolved | False completions | Avg tool calls | Avg time | Report |
| --- | --- | --- | ---: | --- | ---: | ---: | ---: | --- |
| Stage 1 | rc.156 | 22 basic (Go, Python) | 22 | **22 (100%)** | 0 | 6.0 | 42 s | [stage1-rc156](stage1-rc156.md) |
| Stage 2 | rc.156 | 10 hard, symptom-described | 26 | **25 (96%)** | 1 | 8.7 | 59 s | [stage2-partial-rc156](stage2-partial-rc156.md) |
| Stage 2 A/B | rc.157 | same 10 hard, completion check on/off | 20 | **19 (95%)** | 1 | 10.2 | 64 s | [stage2-ab-rc157](stage2-ab-rc157.md) |
| Line-edit A/B | rc.159 | 10 hard, line edits on/off | 20 | **20 (100%)** | 0 | 11.0 | 73 s | [lines-ab-rc159](lines-ab-rc159.md) |
| Compound | rc.159 | 4 tasks, three bugs each | 8 | **8 (100%)** | 0 | 20.9 | 139 s | [combo-rc159](combo-rc159.md) |
| Real projects | rc.162 | 19 bug fixes from 8 Go libraries | 19 | **14 (74%)** | 5 | 11.8 | 77 s | [real-rc162](real-rc162.md) |
| Real-project A/B | after rc.180 | same 19, completion check on/off; 13 finished in both arms | 26 | **20 (77%)** | 6 | 15.1 | 122 s | [real-ab-unreleased](real-ab-unreleased.md) |

What the numbers say:

- The synthetic suite is saturated for this model; real-project fixes are not.
- Both synthetic false completions trace to one under-specified task, later corrected.
- No A/B switch (completion check, line-range edits, name-filtered-tests reminder) changed results, so all stay
  at their defaults. The real-project misses are incomplete fixes that pass the project's own tests.

[Microbenchmarks](MICROBENCHMARKS.md) cover the harness's own speed: startup, tools, rendering
and compaction.

To run your own, see [`bench/README.md`](../../bench/README.md).
