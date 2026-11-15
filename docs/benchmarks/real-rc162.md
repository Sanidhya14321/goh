# Real-project tasks on rc.162 (containerised)

`gpt-6.1-sol`, ChatGPT sign-in. 19 tasks mined from real bug-fix commits in eight Go libraries
(gjson, cast, BurntSushi/toml, gorilla/mux, go-humanize, go-toml v2, sjson, Masterminds/semver).
Setup ran in a container with network; the agent's shell commands and grading ran in separate
containers with no network (confirmed: test traces show the container's `/usr/local/go`).

## Run 1: 19 tasks, 1 attempt

| Runs | Resolved | False completions | Budget stops | Avg tool calls | Avg seconds | Input tokens | Cached | Output tokens |
|---|---|---|---|---|---|---|---|---|
| 19 | 14 (74%) | 5 | 0 | 11.8 | 77 | 2354726 | 250624 | 33876 |

Failure review: 4 of the 5 failures were task defects. The hidden tests required details the prompt
did not state: the sjson padding limit and exact error messages, the go-toml exact error text,
gjson clamping of numeric *strings*, and the contract of cast's internal `indirect` helper (where the
first prompt pointed the wrong way). Those prompts now state the observable requirements. The fifth,
`go-cast-fa4ea64d`, was a genuine miss: `"+.25"` must convert to 0 for unsigned types.

## Run 2: the five unsolved tasks, corrected prompts, 2 attempts

| Task | Resolved | Cause of misses |
|---|---|---|
| go-sjson-9f1cbfdb | 2/2 | — |
| go-gjson-eaae8c63 | 1/2 | broke an existing test (TestPlus53BitInts) |
| go-gotoml-79f82e3c | 1/2 | incomplete fix |
| go-cast-1b425f31 | 0/2 | fixed bool/string but not time conversion of typed nils |
| go-cast-fa4ea64d | 0/2 | rejects `"+.25"` for unsigned types (0/3 overall) |

All 6 misses in run 2 were reported `verified`: the model's own checks passed, so these are false
completions caused by incomplete fixes rather than skipped checks. Unlike the synthetic suite, this
set separates outcomes; it is the right place for further A/B tests.
