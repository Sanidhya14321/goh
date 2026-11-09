# Part 3: Provider and loop

## Question and provenance

How are fragmented streams, complete tools, EOF and multiple turns represented?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `cmd/` and `internal/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-text.json` — `ev_35c087c15feb1082b7a7a6df1761de679a2c6e19be5a80ea019ebce3e4bf1db3`
- `go-text.json` — `ev_0737231b1841c995b0b6387599a0024692063b793f74ebad043e57023b7d195b`
- `reference-drop.json` — `ev_02d3780f10b042a608283eaabe11f4d43803fd2a9ccfcc8bc8f6a9fa8c7938b1`
- `go-drop.json` — `ev_378dc4fb1a85dca38da6c8f716dde1a42dfe089de527f7b340076b79c82d3b6e`
- `reference-edit.json` — `ev_7dfd2590059d9606a94fd7a843caec96b881d3693def21b4b1717c92838475de`
- `go-edit.json` — `ev_83dc064570a4507894a4cfe9c0c61a694c6fdf93447f191914db1809dfc3e1d0`

## Original Go contract

Assemble SSE records and tool arguments; retain call IDs; execute only after successful completed assistant messages; stop partial streams without replay. Publish one terminal run event. Configured retry policy applies only before accepted content.

## Comparison

Local text and complete tool flows have matching selected semantic outcomes. EOF intentionally differs: reference repeatedly produced invalid-tool results until bounded timeout; Go emitted failed without tool execution. Raw comparisons remain changed/truncated.

## Exit coverage and unknowns

Go parser/failure tests and CLI acceptance are tracked by main project checks. Live endpoint compatibility, all provider finish/error forms in reference, rate-limit retry timing and stochastic outputs remain unverified. Reference EOF terminal capture is truncated.
