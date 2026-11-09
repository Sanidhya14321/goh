# Part 7: Sessions and recovery

## Question and provenance

Does resume retain conversation identity/history, and are uncertain effects blocked?

Reference identity and artifact digests: `research/reference-manifest.json`. Product files are independently authored under `cmd/` and `internal/`. Source inspection and local projections are explicitly distinguished from REA observations.

## Retained observations

- `reference-text.json` — `ev_35c087c15feb1082b7a7a6df1761de679a2c6e19be5a80ea019ebce3e4bf1db3`
- `reference-resume.json` — `ev_144632ab79a711a90de27db65a969bacc01c45c701e4220dc3059eaf5462ea03`
- `go-text.json` — `ev_0737231b1841c995b0b6387599a0024692063b793f74ebad043e57023b7d195b`
- `go-resume.json` — `ev_485552b4035ac97bfbb50dc320bc36e2d8c566a347eb4d1b6b0279809d26b356`
- `go-mcp-disconnect.json` — `ev_23b68454b044a6a7f7710fff5c0e5a2af3745b13cc90fd5ad1de3e319bafd047`
- `go-uncertain-resume.json` — `ev_8290150de7f8ab2f24c482b52fc5cf8e2d3b11d2e0b351ad8e9fd64e2d25b498`

## Original Go contract

Durable private versioned JSONL journal, exclusive writer lock, valid-prefix tail recovery, corruption/version refusal, synchronized starts/results. Never replay pending effects; explicit acknowledgment required and permissions reacquired.

## Comparison

Normal text resumes retain the same per-product session ID and include the prior user/assistant messages in later model requests. Go disconnected MCP leaves uncertain operation; a resume without acknowledgment refuses before new provider/tool execution. Reference uncertain-effect recovery is not established by the successful text resume.

## Exit coverage and unknowns

Reference journal truncation, conflicting writers, crash boundary semantics and uncertain-effect acknowledgment remain unverified. Go session tests are independent acceptance, not evidence of identical OpenCode persistence internals.
