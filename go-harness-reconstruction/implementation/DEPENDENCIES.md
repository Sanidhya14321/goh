# Dependency inventory

The three direct dependencies and the complete selected module graph are pinned in `go.mod` and verified through `go.sum`. The table includes test-only and graph-only modules; “host CLI” means the module appears in `go list -deps ./cmd/harness` on macOS ARM64. It does not assert that every file or symbol ships in a binary.

Exact upstream license, attribution and patent files are preserved under `licenses/`. `licenses/inventory.json` records paths and SHA-256 hashes. Original project material uses Apache-2.0 in `LICENSE`; third-party notices retain their original terms.

Go 1.27.3 is development tooling and supplies the standard library/runtime to release binaries. Its license and bundled third-party notices are in `licenses/go1.27.3/`. REA 6.8.0, Node, Bun and OpenCode are research tooling/reference artifacts, outside the product dependency graph and not runtime requirements.

| Module | Version | Host CLI | License text summary |
|---|---|---|---|
| `charm.land/bubbletea/v2` | `v2.1.1` | yes | MIT |
| `cloud.google.com/go/compute/metadata` | `v0.3.0` | no | Apache-2.0 |
| `github.com/aymanbagabas/go-udiff` | `v0.2.0` | no | MIT + BSD notices |
| `github.com/bits-and-blooms/bitset` | `v1.24.6` | no | BSD-style (see exact terms) |
| `github.com/charmbracelet/colorprofile` | `v0.4.3` | yes | MIT |
| `github.com/charmbracelet/ultraviolet` | `v0.0.0-20261008173134-6b8d4baf91b4` | yes | MIT |
| `github.com/charmbracelet/x/ansi` | `v0.11.9` | yes | MIT |
| `github.com/charmbracelet/x/exp/golden` | `v0.0.0-20241212170349-ad4b7ae0f25f` | no | MIT |
| `github.com/charmbracelet/x/term` | `v0.2.2` | yes | MIT |
| `github.com/charmbracelet/x/termios` | `v0.1.1` | yes | MIT |
| `github.com/charmbracelet/x/windows` | `v0.2.2` | yes | MIT |
| `github.com/clipperhouse/displaywidth` | `v0.11.0` | yes | MIT |
| `github.com/clipperhouse/stringish` | `v0.1.1` | no | MIT |
| `github.com/clipperhouse/uax29/v2` | `v2.7.0` | yes | MIT |
| `github.com/dlclark/regexp2` | `v1.11.0` | no | MIT |
| `github.com/golang-jwt/jwt/v5` | `v5.3.1` | no | MIT |
| `github.com/google/go-cmp` | `v0.7.0` | no | BSD-style (see exact terms) |
| `github.com/google/jsonschema-go` | `v0.4.3` | yes | MIT |
| `github.com/lucasb-eyer/go-colorful` | `v1.4.1` | yes | MIT |
| `github.com/mattn/go-runewidth` | `v0.0.30` | yes | MIT |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` | yes | Mixed Apache-2.0 / MIT code; CC-BY-4.0 documentation (upstream transition notice) |
| `github.com/muesli/cancelreader` | `v0.2.2` | yes | MIT |
| `github.com/rivo/uniseg` | `v0.4.7` | yes | MIT |
| `github.com/santhosh-tekuri/jsonschema/v6` | `v6.0.3` | yes | Apache-2.0 |
| `github.com/segmentio/asm` | `v1.1.3` | yes | MIT |
| `github.com/segmentio/encoding` | `v0.5.4` | yes | Apache-2.0 |
| `github.com/xo/terminfo` | `v1.0.0` | yes | MIT |
| `github.com/yosida95/uritemplate/v3` | `v3.0.2` | yes | BSD-style (see exact terms) |
| `golang.org/x/crypto` | `v0.58.0` | no | See preserved license texts |
| `golang.org/x/exp` | `v0.0.0-20220909182711-5c715a9e8561` | no | BSD-style (see exact terms) |
| `golang.org/x/mod` | `v0.41.0` | no | BSD-style (see exact terms) |
| `golang.org/x/net` | `v0.62.0` | yes | BSD-3-Clause; see exact license/patent notices |
| `golang.org/x/oauth2` | `v0.35.0` | yes | BSD-style (see exact terms) |
| `golang.org/x/sync` | `v0.24.0` | yes | BSD-style (see exact terms) |
| `golang.org/x/sys` | `v0.49.0` | yes | BSD-style (see exact terms) |
| `golang.org/x/term` | `v0.47.0` | no | See preserved license texts |
| `golang.org/x/text` | `v0.43.0` | yes | BSD-style (see exact terms) |
| `golang.org/x/time` | `v0.15.0` | yes | BSD-style (see exact terms) |
| `golang.org/x/tools` | `v0.51.0` | no | BSD-style (see exact terms) |

The summaries above identify the copied texts, rather than providing a legal conclusion. The MCP SDK explicitly records an ongoing licensing transition: Apache-2.0 for consented/new code, MIT for contributions without relicensing consent, and CC-BY-4.0 for documentation excluding specifications. The full transition notice is preserved. No MCP documentation is embedded into application UI or prompts.

Regenerate module selection with `go list -m -json all`, and package usage with `go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./cmd/harness`. Recollect notices whenever versions change.

Vulnerability audit: the local `govulncheck` result is retained under `research/evidence/govulncheck.json`; see that record for actual tool/database metadata and findings. It checks known Go advisories, rather than certifying absence of vulnerabilities.

The initial transitive `golang.org/x/text v0.14.0` selection produced module-only findings GO-2026-5970 and GO-2026-6629. It was raised to pinned `v0.41.0`, which also raised graph-only `golang.org/x/tools` to `v0.48.0` and `golang.org/x/mod` to `v0.38.0`. All three requested direct dependency pins remain unchanged. The subsequent source/symbol scan with govulncheck v1.9.0, Go 1.27.3 and database modification time 2026-11-02T22:01:43Z returned no findings. Exact scanner metadata is in the retained JSON record.

The web-fetch HTML tokenizer pins golang.org/x/net v0.62.0. Minimal version selection also advances x/sync to v0.24.0, x/sys to v0.49.0 and x/text to v0.43.0; the original three UI/MCP/schema dependency pins remain unchanged. Exact selected versions and copied notices are recorded in go.mod/go.sum and licenses/inventory.json.
