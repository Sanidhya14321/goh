# GoHarness

The implementation is in [implementation/](implementation/README.md). It includes
a working headless CLI, terminal UI, approved file/shell tools, durable recovery,
context compaction and local stdio MCP support, with deterministic tests and
selected REA-backed OpenCode comparisons.

Start with the implementation README for commands and configuration. Current
release gates and remaining platform/parity limits are in [PROGRESS.md](PROGRESS.md)
and [specs/COMPATIBILITY_MATRIX.md](specs/COMPATIBILITY_MATRIX.md). Research,
artifacts and raw evidence remain separate from original implementation code.

The CLI command is `goh`. The release script stages development builds in
`implementation/dist/goh-{darwin,linux}-{arm64,amd64}`.
