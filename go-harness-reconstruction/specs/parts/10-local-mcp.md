# Part 10: Local MCP

## Question and provenance

Can a configured stdio server be discovered/called and disconnected without duplicate execution?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `implementation/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-mcp.json` — `ev_82321967238bdec9ed81047543d481a2e4488391fc5a646c6e5d1f833f40a81d`
- `go-mcp.json` — `ev_7ed3eefea04b6f3092e431d3331b00a55d541040ad7aa5cff7ea570eb4347327`
- `go-mcp-admitted.json` — `ev_6d5510bc7a3c452553581d186eddf0e1b7fbf82f53b1923356add7e43ca5dd96`
- `go-mcp-final.json` — `ev_cd1673f73eb9d8f1a88dc25537193a6acec9bd3678a0bf349f6f09692cda1c8f`
- `reference-mcp-disconnect.json` — `ev_e8f24e034b2521a25dfc344ebe2186e76739a16f19b21a3753b44ba13fcd21d8`
- `go-mcp-disconnect.json` — `ev_23b68454b044a6a7f7710fff5c0e5a2af3745b13cc90fd5ad1de3e319bafd047`
- `go-uncertain-resume.json` — `ev_8290150de7f8ab2f24c482b52fc5cf8e2d3b11d2e0b351ad8e9fd64e2d25b498`

## Original Go contract

Use official Go SDK with explicit configured command admission and per-tool authority; namespace tools, compile supported inline schemas without remote refs, bound time/output and close processes. Disconnect is uncertain; never replay automatically.

## Comparison

Both discovered the independently authored SDK fixture and returned its echo text. Reference names fixture_echo; Go uses mcp__fixture__echo. Go startup/tool approvals were explicit. A controlled disconnect produced reference tool error/continuation exit0, whereas Go stopped failed with uncertain-effect recovery required. Raw pair comparisons report changed. Earlier grant failures are build-specific: an old binary used mcp_start__fixture; final binary correctly accepts documented mcp_start:fixture.

## Exit coverage and unknowns

Fixture binary was built with go test -c from original implementation/internal/mcpclient/TestMCPHelper using the pinned official SDK; it is not copied from OpenCode. Reference schema-failure, timeout, cancellation and remote features remain unverified; Go tests independently cover its cancellation/disconnect/schema behavior. No automatic remote installs or credential calls were used.
