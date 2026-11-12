# GoHarness benchmark

`goh-bench` runs pinned coding tasks through `goh run` and regrades every result in a clean copy
with hidden tests, following the SWE-bench design. The agent never sees the hidden tests, and a task
counts as resolved only when both its fail-to-pass and pass-to-pass checks succeed.

## Build and check the tasks (no model, no network)

```sh
make bench-check        # builds bin/goh-bench and confirms each task fails before and passes with its reference solution
```

## Run

`goh-bench` uses whatever model `goh` is configured for. With the ChatGPT sign-in, that is `gpt-6.1-sol` by default.
**This spends your plan's quota:** about one agent run per task, per arm, per attempt.

```sh
bin/goh-bench --goh bin/goh --attempts 3 \
  --arm "base=" \
  --arm "nocheck=--completion-check=false"
```

Add `-host-grading` to include the tasks that have no image (see **Isolation** below); without it they are
reported as errors and not run.

Each arm is a name plus extra `goh run` flags, so any switch can be A/B tested (for example
`--reasoning-effort xhigh` or `--verify "go test ./..."`). Results go to `bench/results/<time>/`:
- `results.jsonl`: one line per attempt;
- `summary.md`: per-arm resolved rate, false completions, budget stops, average tool calls and time, and tokens (including cached).

## Task format

```
bench/tasks/<id>/
  task.json   {"id", "prompt", "setup": [...], "fail_to_pass": "cmd", "pass_to_pass": "cmd", "max_run_seconds", "max_tool_calls"}
  repo/       starting tree the agent works in
  hidden/     tests copied over the agent's result before grading (never shown to the agent)
  solution/   reference fix, used only by --dry-run to validate the task
```

Real-project tasks replace `repo/` with a `git` source (`{"url", "commit"}`) and usually an `image`, so the
starting tree is cloned at a pinned commit and graded in a container.

**Isolation.** With an `image`, setup, the agent and grading each get their own container, and each mounts only
its own tree. The agent's container (no network, read-only dependency cache) mounts only `agent/` and is stopped
when the run ends. The grader is a fresh container that mounts only `grade/`, the agent's final files copied
with the hidden tests on top. Tasks without an image run the agent's shell through GoHarness's own sandbox, but
grading runs the model-edited code directly on this machine. Those tasks are refused unless you pass
`-host-grading`.

The suite has 22 tasks in Go and Python (11 each), covering:
- single-function bugs (off-by-one, median, Unicode reversal, interval merging, wrapping, pagination);
- multi-file and multi-package changes (config and server, units and report, a pricing API rename across a package, environment-variable overrides);
- concurrency (a data race, context cancellation in a worker pool);
- API behaviour (HTTP validation, error wrapping, semantic versions, an LRU cache, generics);
- data handling (CSV quoting, cent-exact money, duration parsing, slugify, a mutable-default bug).

Run several attempts per task (`--attempts 3`); with 22 tasks, differences of a task or two between arms are noise. Add harder tasks from your own repositories for longer maintenance work.

## Hard set (`hard-*`, 14 tasks)

These tasks come from `bench/tools/gen_hard.py`, built on two correct mini-codebases:
- **Go inventory service:** models, store with atomic batches, event bus, TTL cache, pricing, HTTP API and CSV reports.
- **Python billing package:** money, periods, proration, exchange rates, payments with retries, usage aggregation.

Each task plants one realistic bug and describes only its **symptom**, like a bug report, with no file names. The hidden tests check the full intended behaviour, and the reference solution is the original correct code, so every task is valid by construction. The covered bugs:
- cache invalidation not wired across packages;
- non-atomic batch updates;
- per-line tax rounding;
- cursor paging under deletion;
- an event-bus deadlock;
- CSV quoting and stable order;
- February/leap-year proration;
- an exchange-rate cache keyed without the day;
- double charges on retry;
- UTC day bucketing.

**Compound tasks (`hard-*-combo-*`, 4 tasks):** each combines three of the bugs above, in different files, behind one vague release report that names symptoms only and says the QA suite is unavailable. The run passes only if all three hidden test sets pass, so fixing the obvious bug and stopping is a failure.

## Real-project set (19 tasks)

Mined with `bench/tools/mine_go.py` from real bug-fix commits in eight Go libraries (`spf13/cast`,
`tidwall/gjson`, `tidwall/sjson`, `pelletier/go-toml`, `BurntSushi/toml`, `dustin/go-humanize`,
`gorilla/mux`, `Masterminds/semver`). Results: [`docs/bench/real-rc162.md`](../docs/bench/real-rc162.md).
The miner and these tasks run third-party test code, so run them only inside a container or VM.

Their `hidden/` and `solution/` files are copied from the upstream projects and stay under the upstream
licenses (MIT, BSD-3-Clause); the license texts and a task index are in [`licenses/`](licenses/README.md).
Keep that index current when mining new tasks.
