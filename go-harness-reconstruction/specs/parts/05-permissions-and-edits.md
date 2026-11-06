# Part 5: Permissions and edits

## Question and provenance

What happens with denial, ambiguity and external mutation between read and edit?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-ask.json` — `ev_ad92d8b5be6abbd00ae3cdcb42f27aa97fc106bb86b1bdbf80d5c889613b1225`
- `go-ask.json` — `ev_c710fb29facc513fca13a5e422eda435db5876d7aa896c9aee765cad6c6da721`
- `reference-deny.json` — `ev_24d445f64b6903f643262b9dd36deccc4813bc054a32a8489681788a8a5e1d13`
- `reference-ambiguous.json` — `ev_7bd107af1d7fb70e71c195b7be35154a962e4e2179433f0ffbc45c15f65d4829`
- `go-ambiguous.json` — `ev_76cef8472d1f2578ebbe83e788108d9264808dfceee7b50e4e54b29df75c775d`
- `reference-stale.json` — `ev_f7dddb77d0a8d8411cba1d8bdbd2a1b48a8b2ab1d6bfe82f24f7089b65fa78d1`
- `go-stale.json` — `ev_37ace88f3c14519001e0ea2cd4a2adc5dadebee61014497cc12206ce5856c5e6`
- `reference-stale-preserved.json` — `ev_1af4181458457e41947d42641a59d2342b7e413b5de09557c07ce4bc62b03d81`
- `go-stale-preserved.json` — `ev_c513c0355661fade4878e0a771203361a07074f871d1da4f6b5bf5f44627d6e9`

## Original Go contract

Require explicit authority before mutation. Exact edits need expected whole-file SHA256 and unique old_text. Recheck after approval; preserve unaffected bytes/mode; create without overwrite. Headless unavailable authority yields needs_permission.

## Comparison

Both reject ambiguous old text and leave the file unchanged. When the independent model fixture externally appends a line while retaining alpha, reference edits alpha to beta; Go rejects stale SHA and preserves the externally written file. This is intentional stricter behavior. If external mutation removes alpha altogether, both reject the edit. Reference headless ask auto-rejects and exits0; Go exits3 with needs_permission.

## Exit coverage and unknowns

Go ambiguous/stale captures initially used an earlier binary without validation-failure tool_finished events; those captures retain that identity. Root subsequently fixed/retested event emission. Multi-file transactions and universal atomic compare-and-swap are not promised. Interactive approval is not established by headless captures.
