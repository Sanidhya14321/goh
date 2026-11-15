# Microbenchmarks

Local performance measurements of the harness itself: no model inference is involved. All runs
are on one development Mac (Apple M5 Pro, macOS 26.5, ARM64, Go 1.27.3) with filesystem caches
available, so they show orders of magnitude, not cross-platform guarantees. Raw records are under
`research/evidence/` (local reference material).

## Startup and memory

Process launch to the first interactive prompt in a 120×36 pseudo-terminal, with a configured
loopback provider and isolated home directories. Three sequential trials per product (RC30).

| Product | Median | Trials (ms) | Parent RSS |
| --- | ---: | --- | --- |
| GoHarness 1.0.0-rc.30 | 39 ms | 39.27, 39.16, 50.39 | ~23 MB |
| OpenCode 1.19.5 | 1,400 ms | 1452.88, 1399.61, 1390.43 | ~820 MB |

GoHarness startup has stayed between 36 and 45 ms in every check since RC6. The reference product
varied between about 920 and 1,420 ms across runs on the same host. Thirty sequential
`goh version` invocations take a median of 8.3 ms.

Not measured: inference latency, interaction latency under a live model, or task throughput (see
the [agent benchmarks](README.md) for those).

## Tools

```sh
go test ./internal/tools -run '^$' -bench . -benchmem -count 3
```

| Workload | Time per operation | Allocated |
| --- | --- | --- |
| Search over 128 files (~2.4 MB) | 7.8–7.9 ms | ~8.0 MB |
| Exact edit of a ~114 KB file with durable publication | 8.5–8.9 ms | ~1.7 MB |
| Declaration map, 16 files, warm parse cache | 0.44 ms | 234 KB |
| Declaration map, 16 files, cold | 0.90 ms | 523 KB |

On a 32-file fixture (298 KB of source) the declaration map is 6.3 KB of JSON, 98% smaller in
bytes. It omits bodies and comments, so it guides reading rather than replacing it.

## Terminal rendering

| Workload | Time per call |
| --- | --- |
| Cached transcript view, 120×36, ~120 KB of text | 0.2 ms (0.24 ms with true colour and highlighting) |
| Re-parsing a ~90 KB unfinished code block (worst case while streaming) | 6.1 ms |
| Status dialog view | 0.41 ms |
| Session preview of a 1 MiB journal (64 KiB prefix budget) | 0.06 ms |

## Context compaction

`go test ./internal/runtime -run TestMaskingMeasurement -v`: six 6,000-byte `read_file` results
in a 40,000-token window, sizes from the runtime's conservative byte estimator.

| Mode | Before | After | Summary calls | Summary input |
| --- | ---: | ---: | ---: | ---: |
| Masking | 37,901 | 14,625 | 0 | 0 |
| Summarize only | 37,901 | 13,001 | 1 | 25,530 |

Masking avoids a ~25 KB summary request for a slightly larger context. Answer quality after
compaction is not measured here.
