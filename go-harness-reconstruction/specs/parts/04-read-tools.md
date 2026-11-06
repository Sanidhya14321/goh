# Part 4: Read tools

## Question and provenance

Can multiple validated reads preserve tool pairings and avoid effects?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-edit.json` — `ev_7dfd2590059d9606a94fd7a843caec96b881d3693def21b4b1717c92838475de`
- `go-edit.json` — `ev_83dc064570a4507894a4cfe9c0c61a694c6fdf93447f191914db1809dfc3e1d0`
- `reference-multi.json` — `ev_7f4eadc4d3884515a250e1e60e2789579097ce6ec519daa93266b4de4dc6b355`
- `go-multi.json` — `ev_b0da711d3f262c79a2f018f7a64d2588c545c3296ada4dcf83b0d7e6995af44f`

## Original Go contract

Project-scoped bounded UTF8 reads expose file identity/hash; list/search are Go built-ins. Validate arguments, reject escapes, bound scheduling, retain original result order/IDs.

## Comparison

The scripted read/edit flow and two simultaneous read requests preserve IDs and content (reference completion events can arrive out of requested order; Go reports results in requested order); file snapshots remain unchanged for read-only calls. Go stricter filesystem/schema limits are original policy.

## Exit coverage and unknowns

Reference list/search edge cases, symlink races and complete filesystem traversal parity are unverified. Paired fixtures observe only selected roots and may miss transient effects.
