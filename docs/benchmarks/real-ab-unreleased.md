# Real-project tasks: completion check on/off (after rc.180)

`gpt-6.2-sol`, ChatGPT sign-in, 18 Nov 2026. The same 19 real bug-fix tasks as
[real-rc162](real-rc162.md), with the corrected prompts from its run 2, one attempt per arm. Setup, agent and grading
ran in separate containers as before.

- **check**: defaults, which now include the reminder to run the full tests when only tests picked by name
  (`go test -run`, `pytest -k`, …) passed after the last change.
- **nocheck**: `--completion-check=false`.

## Results

Eight runs were cut off by the service rather than finishing: five "check" runs ended on OpenAI's in-stream
`server_is_overloaded` error, which goh did not retry at the time (fixed since; see below), and the last three
"nocheck" runs hit the ChatGPT plan's usage limit. Comparing only the 13 tasks both arms finished:

| Arm | Tasks | Resolved | False completions | Avg tool calls | Avg seconds |
|---|---|---|---|---|---|
| check | 13 | 10 | 3 | 15.3 | 120 |
| nocheck | 13 | 10 | 3 | 14.8 | 123 |

Including cut-off runs, `goh-bench` reported check 12/19 and nocheck 14/19. That difference comes from the
cut-off runs and is not a real gap between the arms.

| Task | check | nocheck |
|---|---|---|
| go-cast-1b425f31 | **false completion** | **false completion** |
| go-cast-6e9731d5 | resolved | resolved |
| go-cast-fa4ea64d | **false completion** | **false completion** |
| go-gjson-9378d3bb | resolved | resolved |
| go-gjson-a825fa56 | resolved | resolved |
| go-gjson-eaae8c63 | **false completion** | **false completion** |
| go-gotoml-071a36c2 | resolved | resolved |
| go-gotoml-686c980c | resolved | resolved |
| go-gotoml-79f82e3c | cut off (overloaded) | resolved |
| go-humanize-1b638b2c | cut off (overloaded) | resolved |
| go-humanize-21224679 | resolved | resolved |
| go-humanize-40736da3 | resolved | resolved |
| go-mux-2b030fc3 | cut off (overloaded) | resolved |
| go-mux-976b5364 | resolved | resolved |
| go-semver-441ff8e5 | resolved | resolved |
| go-sjson-1e706478 | resolved | resolved |
| go-sjson-9f1cbfdb | cut off (overloaded) | cut off (plan limit) |
| go-sjson-f0069f8a | cut off (overloaded) | cut off (plan limit) |
| go-toml-d10101f4 | resolved | cut off (plan limit) |

## What it shows

- **The arms gave the same outcome on every task both finished.** The name-filtered-tests reminder never fired:
  `gpt-6.2-sol` ran the full `go test ./...` itself after each `-run` check. The reminder costs nothing when it
  isn't needed. It stays on by default as a guard for models that stop after a filtered run.
- **The three false completions are the hard tasks from rc.162** (cast typed nils, cast `"+.25"` for unsigned
  types, a gjson change that broke an existing test). Each time, the project's own full test suite passed and only
  the hidden tests failed. No reminder that relies on the project's own tests can catch these.
- On finished tasks the resolve rate is 10/13 (77%), in line with rc.162's 74% on a different model version.

## Fixes found by this run

- `server_is_overloaded` (OpenAI) and the equivalent in-stream errors (`server_error`, Anthropic
  `overloaded_error`/`api_error`) now resend the turn after 5, 10 and 20 s, announced as a provider retry.
  Usage-limit and request errors are still not retried.
