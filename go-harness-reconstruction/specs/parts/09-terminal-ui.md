# Part 9: Terminal UI

## Question and provenance

Can the terminal UI accept text, resize, exit and send restoration sequences?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-tui.json` — `ev_1fb7d7d95efb2e598a0835f51cca84f4f9ab631a25298923d7823b7941896f1e`
- `go-tui.json` — `ev_6830a2c599a42c24889ae2bb105d995cb8be7e77ed323a18e91d0224e251f48b`
- `reference-tui-ready.json` — `ev_c2749aa684819c9c225d282d1c090a0391fdb7bf2b819277d1a3dc7226b04de1`
- `go-tui-ready.json` — `ev_5ac137d9296c0dd19d755699946c16d02bf20a05c364046b00b4750ee73be242`
- `go-tui-selected.json` — `ev_74461fc39acd30bbc2a0560fde74080181e6a1ad402485cc0e842055d92da636`

## Original Go contract

Bubble Tea UI consumes the same runtime events, renders bounded safe content, exposes approval/cancel controls, session selection, profile and usage. Resizing changes view only. First cancellation stops the run; exit is separate.

## Comparison

Initial reference input arrived before readiness; a later 5.5s input experiment produced a local text response, observed resize, Ctrl+D exit0 and restoration escapes. New Go binary added session selection; its first input selected New Session rather than submitting text. An explicit Enter selection followed by the same text produced local response, resize and exit0/restoration escapes. These are declared interaction differences, not visual equivalence.

## Exit coverage and unknowns

Several rendered-terminal budgets truncated while raw terminal bytes, interactions and exit were retained; see each truncation_details. Restoration escape sequences are observed, but no claim is made that every host terminal mode was independently verified. Reference approval dialogs, active-run cancellation and crash restoration remain unverified. Go UI unit tests do not close those reference gaps.
