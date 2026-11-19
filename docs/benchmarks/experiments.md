# Research prototypes

Three opt-in runtime experiments and two benchmark-only measurements, built after the
verification and grading fixes so their effect can be measured honestly. Every experiment is off
by default and leaves the baseline request unchanged (`TestExperimentsOffLeaveBaselineUnchanged`).
The paper figures cited by the review are the authors' own results. None of these methods has a
GoHarness result yet: the live comparisons below need model quota and have not been run.

| Switch | What it does | Local evidence |
| --- | --- | --- |
| `--experiment submission-evidence` | When changed work is about to finish, the model is asked once for `submit_evidence`: intended behaviour, sources inspected, test scope, checks run with their results, and unresolved gaps. The record is journaled (`submission_evidence`) against the current revision and checked against the commands this run actually executed. It replaces the one-time completion reminder, not the configured `verify`. | `TestSubmissionEvidenceGate`: asked exactly once; a claimed `go test ./...` that never ran is named back to the model; the record carries the current fingerprint |
| `--experiment recoverable-compression` | Masked tool results carry a `recall_id`; `recall_observation` returns the exact original output in 8 KiB pages from this session's journal. Only results recorded in this session, under the same access and retention as the journal. | `TestRecallMaskedObservation`: receipt ids, exact pages, unknown ids refused |
| `--experiment retrieval-expansion` | `repo_map` with a query adds up to 10 files that reference the first five matching symbols, with provenance (`references X`, defined in Y), from a lexical index cached by content digest. Ignore and deny rules apply as for `repo_map`. | `TestRepoMapRelatedExpansion` (references found, stale entries dropped after an edit, off by default). `BenchmarkRepoMapQuery`, 256 files: 8.41 ms base, 8.63 ms with expansion warm (+2.7%), 9.03 ms cold (+7.4%) |
| `gold_files` in `task.json` (benchmark only) | Retrieval recall, seconds to the first gold file, files touched and repeat reads, computed from the session journals. Never shown to the agent. | `TestRetrievalTelemetry` |
| `goh-bench -reproducer` | Test files the agent added run on an isolated copy of the pre-run baseline and on the graded candidate: `fail_to_pass` (evidence), `no_signal`, `fails_on_candidate` or `unknown`. Recorded only; it never changes the grade. | `TestReproducerAudit` |

Before `retrieval-expansion`, `repo_map` itself was fixed: ignored files no longer use up its scan
budget, and files whose path matches the query are read first (`TestRepoMapCoverage`; the
existing `BenchmarkRepoMapWarm`/`Cold` stayed within noise: 463 → 465 µs and 939 → 939 µs).

## Running the comparison

Build once, then compare one change at a time against the baseline with identical budgets,
permissions and tasks. Arms are interleaved per task and attempt. Use the real maintenance tasks
(the `git` tasks with images) for the measurement and the synthetic suite as a smoke test.

```sh
make build && go build -o bin/goh-bench ./cmd/bench
bin/goh-bench --goh bin/goh --attempts 3 \
  --arm "base=" \
  --arm "evidence=--experiment submission-evidence"
bin/goh-bench --goh bin/goh --attempts 3 --arm "base=" --arm "recall=--experiment recoverable-compression"
bin/goh-bench --goh bin/goh --attempts 3 --arm "base=" --arm "expand=--experiment retrieval-expansion"
```

Report per arm from `summary.md` and `results.jsonl`: resolution over scored attempts,
reliability over all scheduled, **false verification**, cached and uncached input, auxiliary and
child tokens, tool calls, p50/p95 time; for retrieval, `retrieval.recall` and
`seconds_to_first_gold` on tasks with `gold_files`. Treat the task (or repository) as the unit
when estimating uncertainty: with 19 real tasks, a difference of one or two tasks is noise.

## Not done yet

- **Live runs.** All comparisons above; they spend model quota and wait on the user's go-ahead.
- **Container tasks.** The 19 Docker-backed tasks could not be dry-run: the Docker daemon was not
  running. The 36 native tasks pass the stricter dry run.
- **Gold files and private holdout.** No task lists `gold_files` yet, and the original Go/Python
  holdout with reviewed alternate solutions and partial fixes is not written. Both are needed
  before the retrieval and evidence arms can be judged.
