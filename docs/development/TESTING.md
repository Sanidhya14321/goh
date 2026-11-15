# Testing

Every test runs without credentials or network. Model behaviour comes from
`cmd/scripted-model`, a local endpoint that replays scripted replies over the real provider
protocols, so the same binary that users run is exercised end to end.

## Layers

| Layer | What it covers | Command |
| --- | --- | --- |
| Unit and integration | Every package, including runtime scenarios against the scripted model, real sandboxes and a real `clangd` when present | `make test`, `make race` |
| Fuzz | SSE and journal parsers, patch grammar, tool arguments, frontmatter, HTML conversion, command expansion | `make fuzz` (3 s per target) |
| CLI acceptance | `goh run`, sessions, export and recovery through the compiled binary | `make acceptance` |
| Terminal (PTY) | The compiled TUI in a real pseudo-terminal, read back through a VT100 screen model | `make e2e` (macOS and Linux) |
| Installers | Install scripts against a local release-layout server, including a tampered download | `make release && sh scripts/test_install.sh` |
| Benchmark tasks | Each task fails before and passes with its reference fix | `make bench-check` |

The terminal suites live in `scripts/e2e/`:

| Script | Covers |
| --- | --- |
| `test_tui.py` | 25 chat scenarios: approvals, undo, palette, sessions, jobs, layouts, editor handoff |
| `test_setup.py` | First run, provider setup and failure paths, draft preservation, model switching |
| `test_resume_setup.py` | `goh chat --session` through provider setup |
| `test_attach.py` | `goh serve` plus `goh attach` |
| `test_acp.py` | The editor protocol over stdio |

`test_openai_live.py` and `test_openai_live_extra.py` make real requests with the saved ChatGPT
sign-in. They are never run in CI; run them only when you intend to spend quota.

## Gates for a change

A change is ready when all of these pass locally:

```sh
gofmt -l cmd internal          # prints nothing
make vet race acceptance e2e
GOOS=linux go vet ./... && GOOS=windows go vet ./...
```

CI (`.github/workflows/ci.yml`) repeats the unit tests natively on Linux, macOS and Windows with
the shell sandbox required on Linux and macOS, runs the terminal suites on Linux and macOS, and
checks the installers on all three.

## Writing tests

- Reproduce a bug with a failing test before fixing it, and confirm the test fails without the fix.
- Prefer the scripted model over mocks: it exercises the real provider encoding and streaming.
- PTY scripts should wait for screen text rather than sleep, and drain output before sending
  a key after Esc (terminals read Esc followed quickly by a key as Alt+key).
- Platform-specific behaviour gets a build-tagged test; note anything only CI can run in the
  changelog entry.

## Recording a milestone

Add a short entry to the [changelog](../../CHANGELOG.md) (user-visible changes), a section to
[VERIFICATION.md](VERIFICATION.md) (what ran and what it showed), and update
[STATUS.md](STATUS.md) if a gap opened or closed.
