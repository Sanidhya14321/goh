# Part 6: Shell lifecycle

## Question and provenance

Do commands complete, and does cancellation settle their owned processes?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-edit.json` — `ev_7dfd2590059d9606a94fd7a843caec96b881d3693def21b4b1717c92838475de`
- `go-edit.json` — `ev_83dc064570a4507894a4cfe9c0c61a694c6fdf93447f191914db1809dfc3e1d0`
- `reference-cancel.json` — `cleanup_incomplete`
- `go-cancel.json` — `ev_5648c5b395c78af9f0cf8b03777d00544e1a9f389187721deaf8229af475e7de`

## Original Go contract

Own a Unix process group, drain both pipes concurrently, cap preview/retained artifacts, terminate on cancellation, wait up to two seconds, escalate and report incomplete cleanup. Cancellation cannot roll back effects.

## Comparison

Both complete the fixture shell test successfully. Go SIGINT capture emitted cancelled/exit130 with verified REA group cleanup. Reference SIGINT yielded cleanup_incomplete plus partial observation; its platform sleep child was orphaned, then explicitly terminated through ordinary shell followup. That manual action is not successful REA evidence.

## Exit coverage and unknowns

Reference cancellation gate remains blocked by macOS ownership-token observation. Tool timeouts differed (Go 5s/reference 35s), but identical 3.5s SIGINT arrived before either. Detached processes outside owned groups remain outside guarantee. Linux runtime is untested.
