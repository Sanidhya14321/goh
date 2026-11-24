# Benchmarks

Every benchmark result for GoHarness, and how to reproduce it. Agent runs use `gpt-6.1-sol` over the
ChatGPT sign-in unless noted; a task counts as solved only when hidden tests (or, for questions, a hidden
answer check) pass in a clean regrade.

## Results

| Run | Harness | Tasks | Attempts | Solved | False completions | Tool calls | Time |
| --- | --- | --- | ---: | --- | ---: | ---: | ---: |
| Basic | rc.156 | 22 Go and Python bug fixes | 22 | 22 (100%) | 0 | 6.0 | 42 s |
| Hard | rc.156 | 10 symptom-described bugs | 26 | 25 (96%) | 1 | 8.7 | 59 s |
| Compound | rc.159 | 4 tasks, three bugs each | 8 | 8 (100%) | 0 | 20.9 | 139 s |
| Real projects | rc.162 | 19 fixes from 8 Go libraries | 19 | 14 (74%) | 5 | 11.8 | 77 s |
| Real projects, `gpt-6.2-sol` | rc.180+ | same 19, completion check on/off | 26 | 20 (77%) | 6 | 15.1 | 122 s |
| Prompt range | 1.1.0 | 7 tasks from `hello` to a 6,330-line file and an 85 KB log, 4 builds | 56 | 56 (100%) | 0 | 2.6 | 14 s |
| Hard, before/after 1.1.0 | 1.1.0 | 6 hard bug fixes | 18 | 18 (100%) | 0 | 10.3 | 64 s |

- Synthetic bug fixes are saturated for this model; real-project fixes are not. The real-project misses are
  incomplete fixes that pass the project's own tests but not the hidden ones.
- No A/B switch changed results (completion check, line-range edits, the name-filtered-tests reminder), so
  all stay at their defaults.

## 1.1.0: fewer tokens across the prompt range

Same tasks, two attempts each, per attempt:

| | Before | After |
| --- | --- | --- |
| `hello` | 1,350 input tokens, 1 request | 1,222 tokens, 1 request |
| Bug fix in a 6,330-line file | 28.5k tokens, 10 requests, 61 s | 15.2k tokens, 6.5 requests, 43 s |
| All 7 prompt-range tasks | 11.9k tokens, 3.3 requests, 16 s | 9.8k tokens, 2.8 requests, 13 s |
| 6 hard bug fixes | 37.0k tokens, 10.7 tool calls, 66 s | 20.2k tokens, 9.0 tool calls, 57 s |

What made the difference: tool results sent as plain text instead of JSON escaped twice (−14% input per
request on its own); `go test` no longer failing once in the sandbox on Go's build cache; a whole-file read
of a file over 64 KB answered with its length instead of an error; the tools offered in a default setup cut
from 21 to 17. One idea was
measured and dropped: returning the first 32 KB of a large file raised large-file input 5.7×, because those
lines are resent with every later request.

## Experiments (partial)

Each experiment against `base=` on the 8 Go hard bug fixes, two attempts each, per attempt (2026-11-24,
`--host-grading`). The Python hard and spec tasks did not run: the ChatGPT usage limit stopped the run,
and the 14 runs it refused were not scored.

| Arm | Solved | False completions | Input tokens | Requests | Time |
| --- | --- | ---: | ---: | ---: | ---: |
| base | 16/16 | 0 | 61.6k | 9.6 | 106 s |
| `recoverable-compression` | 16/16 | 0 | 73.7k (+20%) | 10.7 | 113 s |
| `submission-evidence` | 16/16 | 0 | 82.0k (+33%) | 12.5 | 113 s |
| `retrieval-expansion` | 16/16 | 0 | 57.7k (−6%) | 9.3 | 104 s |

Every arm solved everything and read every gold file, so on these tasks the experiments can only cost:
evidence and compression add requests and tokens for no gain; retrieval is within noise (−54% on one task,
+33% on another). All stay off.

Next to measure: `project-listing`. 69 of 74 runs that used a tool began with `list_files`, a 2.9 s
round trip of 1.3k tokens before any work; the experiment sends a project of up to 80 files' list up front.

## Other harnesses

What each agent sends a model, recorded against a loopback mock (no inference; o200k_base token counts;
`research/scripts/capture_harness_requests.py`):

| | goh 1.1.0 | OpenCode 1.19.5 | Claude Code 2.1.330 |
| --- | --- | --- | --- |
| `hello` request | 2,278 tokens | 7,212 (+636 title request) | 12,098 |
| System prompt / tools | 57 / 17 | 2,019 / 10 | 1,542 / 12 |
| Reading a 6,330-line file | its length and how to read it | lines 1–2,000 (19,039 tokens) | lines 1–4,020 (35,640 tokens) |

## Harness speed

One Apple M5 Pro, no model involved (`go test -bench` in the named packages):

| Measure | Result |
| --- | --- |
| Start to first prompt (120×36 terminal) | 39 ms, ~23 MB (OpenCode: 1,400 ms, ~820 MB) |
| Search 128 files (2.4 MB) | 7.9 ms |
| Exact edit of a 114 KB file, durable | 8.7 ms |
| Chat frame, 120 KB transcript | 0.07 ms (1.1.0: 0.15 ms) |
| Redraw of a 200-turn session, nothing new / after new text | 30 µs / 279 µs (1.1.0: 136 / 414 µs) |
| Streaming a 2,000-line code block, per update | 2.6 ms (1.1.0: 3.6 ms) |
| Reopened chat's messages, 20,000-record journal | 0.24 ms, 0.5 MB (1.1.0: 7.0 ms, 31 MB) |
| Opening a 20,000-record journal | 40 ms, 79 MB |
| Resume of a settled session (64 calls, 32 KiB results) | 4 µs (1.1.0: 2.9 ms, 10 MB) |
| Headless `hello`, scripted model, `GOH_PHASES` (p50 / p95 of 20) | 15 / 21 ms: setup 7, request 2, after final reply 7 (journal 2) |
| Headless read + reply, same | 28 / 36 ms: tool 6, first text 22 |
| Interactive: submit → first text with a 400 ms provider delay; final text → idle | 416 ms; 0–18 ms |
| goh's own time in a 3-request run | ~80 ms of 11.6 s (journal writes) |

## Running

```sh
make bench-check                      # build bin/goh-bench; each task must fail before and pass with its solution
bin/goh-bench --goh bin/goh --attempts 2 --only spec-chat-hello,spec-bugfix-large \
  --arm "base=" --arm "nocheck=--completion-check=false"
python3 bench/tools/compare.py bench/results/<before> bench/results/<after>
```

This spends model quota. Put Go on `PATH`, and keep `--out` outside the system temporary folder (the
sandbox refuses a writable temporary folder that overlaps the project). Tasks without a container image
need `-host-grading`, which runs model-edited code on this machine.

- **Task sets** in `bench/tasks`: 22 basic and 14 hard (`hard-*`, `bench/tools/gen_hard.py`) Go and Python
  bug fixes, 7 prompt-range tasks (`spec-*`, `bench/tools/gen_spectrum.py`), and 19 real-project fixes
  (`bench/tools/mine_go.py`; upstream licenses in `bench/licenses`; run them in a container).
- **Task format**: `task.json` with `id`, `prompt`, `fail_to_pass`/`pass_to_pass` commands and test
  identities, or `answer` (`patterns`, `forbid`, `read_only`, `example`) for questions; optional `category`,
  `size`, `expect` (efficient requests and tool calls), `image`, `git`. `repo/` is the starting tree,
  `hidden/` the tests the agent never sees, `solution/` the reference used by `--dry-run`.
- **Grading**: a clean copy of the starting tree plus the agent's changes (test files excluded), with the
  hidden tests on top. Each result records solved, false completion and false verification, requests, tool
  calls, input/cached/output tokens from the session journals, time, time to first token, project writes
  and whether the run stayed within `expect`; `summary.md` also breaks results down by category and size.
- **Gold files**: `bench/tools/gold_files.py` fills each task's `gold_files` (what its solution changes, or
  a real task's `source_files`); results then report recall of those files and time to the first one, and
  `compare.py` shows recall per arm. Run it again after regenerating tasks.
- **Task trees** are git repositories with the starting files committed (from 2026-11-25; earlier runs
  had none, and `git diff` failed there).
- **Holdout**: six fresh Go and Python bug fixes generated by `research/holdout/gen_holdout.py`, kept out
  of the repository so no model has seen them (`--tasks research/holdout/tasks`).
- **Experiments** (`--experiment submission-evidence`, `recoverable-compression`, `retrieval-expansion`,
  `project-listing`)
  are off by default ([partial results](#experiments-partial)); compare one at a time against `base=`. A run whose
  provider never answered (a usage limit, an outage) is reported as `no_model_reply` and not scored.
