# Local measurements

Host: Apple M5 Pro, macOS 26.5.1, ARM64, Go 1.27.3. These are local measurements,
with tool and startup workloads recorded separately.

- Search fixture: 128 files totaling approximately 2.36MB; three 200 ms benchmark
  runs measured 7.80–7.93ms/operation and approximately 8.0MB allocated/operation.
- Exact edit fixture: approximately 114KB with durable publication; three 200 ms
  runs measured 8.53–8.90ms/operation and approximately 1.70MB allocated/operation.
- Thirty sequential `harness version` subprocess invocations measured a median
  of 8.31 ms (range 7.74–9.50 ms). This includes parent-side process launch/wait;
  filesystem caches were uncontrolled, so no cold/warm distinction is claimed.

Raw data: ../research/evidence/go-benchmarks.txt and startup-measurement.json.
Rerun with `go test ./internal/tools -run '^$' -bench . -benchmem -count 3`.
Numbers do not establish peak process-family memory, CPU usage, cancellation
latency distributions, provider token efficiency or live task success. Those
measurements remain open. File reads run at most four at a time; side effects
are serialized; streams, tool payloads and retained command output are bounded.

##6 November2026: Go declaration map and paged output

Original local fixture only, Apple M5 Pro darwin/arm64, Go1.27.3. In
research/evidence/repomap-benchmark.txt,100 iterations of the same16-file workload
report warm440,100ns/op,233,865B/op,566allocs/op and cold898,184ns/op,523,265B/op,
6,231allocs/op. Cold resets only the parse cache; both paths read/hash source.
This microbenchmark does not establish cross-platform or end-to-end speedup,
LLM quality, live-provider cost or superiority to upstream agents.

An independent32-file CLI/REA fixture reads297,654 source bytes and emits6,330
bytes of declaration-map JSON (97.87% smaller in bytes, not tokenizer-measured
model tokens). A repeated scan reports32 parse-cache hits. A70,017-byte command
output's ending marker is retrieved with a64-byte tail page although absent from
the inline preview. Limits and exact evidence/build identities are retained in
research/evidence/efficiency-report.json and go-efficiency.json. The map omits
bodies/comments and is not a semantic replacement for reading implementation.

## Terminal startup comparison

On 6 November 2026, three local macOS ARM64 trials measured process launch to
the first usable home prompt in a 120×36 PTY. GoHarness RC6 had a median of
40.30 ms and OpenCode v1.19.5 had a median of 1,380.30 ms. Both used isolated
home directories, true color, configured loopback providers, and disabled
OpenCode model discovery/update checks. The startup interval included no model
request. Runs were sequential with host caches available.

The measured parent-process RSS after the prompt was 22,992–23,072 KiB for
GoHarness and 852,096–889,360 KiB for OpenCode. This metric excludes child
processes. These observations establish local startup behavior for the pinned
binaries and workload. Model response time and coding-task throughput need
separate measurements.

The cached transcript microbenchmark measured about 0.2 ms per View call for a
120×36 viewport backed by approximately 120 KiB of text, with NO_COLOR enabled.
See `research/evidence/ui-render-benchmark-rc6.txt` and
`research/evidence/ui-startup-comparison-rc6.json` for the RC6 measurements.
To reproduce captures, run `scripts/capture_ui.py` in a verification-only
Python environment with pyte and Pillow. Its PNGs render captured terminal
cells with a fixed Menlo font.

RC7 repeats the startup comparison and cached viewport benchmark after adding
composer editing, history, command completion, and Markdown code panels. The
latest records are `research/evidence/ui-startup-comparison.json` and
`research/evidence/ui-render-benchmark.txt`. The latter paints a viewport backed
by about 120 KiB of transcript text and reports about 0.2 ms per cached View call
with NO_COLOR enabled. Full token generation and Markdown table/syntax rendering
are outside this benchmark.

RC8 adds table layout and lexical code highlighting. The true-color cached
viewport benchmark reports 0.240 ms per View call. Re-parsing an approximately
90 KiB unfinished Go code block reports 6.13 ms per call. The latter invalidates
the render cache on every iteration to represent incoming stream deltas.
Raw records are `research/evidence/ui-rich-color-benchmark.txt` and
`research/evidence/ui-rich-render-benchmark.txt`. The latest startup comparison
uses the same local method and is retained in `ui-startup-comparison.json`.

RC10 adds a bounded journal-preview benchmark. A 1 MiB fixture journal is
previewed with a 64 KiB prefix budget in approximately 0.061 ms per operation,
with 148,106 bytes and 32 allocations per operation on the recorded host.
This is a local repeated-read workload with filesystem caches available.
See `research/evidence/ui-session-preview-benchmark.txt`. It measures one
preview read and parse rather than an entire multi-session browser load.

RC9 exercises model/agent dialogs after the first-prompt timing interval. The three-trial startup medians are 45.35 ms for GoHarness and 1386.50 ms for OpenCode. Both executables use isolated homes and a configured loopback fixture. The latest PNG captures include both applications’ model and agent dialogs.

RC10 startup medians were 44.36 ms for GoHarness and 1363.11 ms for OpenCode across three trials. The session-browser captures occur after the first-prompt timing interval.

RC11 status rendering measured about 0.411 ms per View call for a cached local snapshot in a 120×36 viewport with NO_COLOR enabled. Credential checks and formatter discovery happen when opening or refreshing the dialog and are outside this rendering interval. See research/evidence/ui-status-benchmark.txt.

RC11 first-prompt startup medians were 40.17 ms for GoHarness and 1375.68 ms for OpenCode over three local trials. No inference occurs in that timing interval.
