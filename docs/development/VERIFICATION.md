# Verification log

What was run for each release candidate and what it showed, newest first. This is an evidence
log: entries are appended when a milestone lands and are not rewritten afterwards, so older
entries describe the code as it was then. For how to run the gates see [TESTING.md](TESTING.md);
for open gaps see [STATUS.md](STATUS.md).

Unless an entry says otherwise, local runs are on macOS ARM64 (Apple M5 Pro) with the Go version
pinned in `go.mod` at the time, and live inference used loopback fixtures.

## 1.1.0

- Gates: `gofmt -l`, `go vet ./...`, `go test ./...`, `go test -race` on runtime and TUI, `make fuzz`,
  `make acceptance` and all six PTY suites (`tui setup acp attach resume_setup salvage`) pass.
- Each fix has a regression test that fails without it, among them: recall bodies kept out of compaction summaries, compaction status by cause, fallback caps and
  calibration kept off the primary, discarded-attempt usage, image request bytes, handled check failures,
  approval review after resize, plain-text tool results on each wire format, large-file reads and line
  counts, sandbox caches, the reply line and timer.
- Measured: live `gpt-6.1-sol` runs of the seven prompt-range tasks (two attempts per build, four builds)
  and six hard bug fixes before and after; harness requests of goh, OpenCode and Claude Code recorded
  against a loopback mock. Results in [BENCHMARKS.md](../BENCHMARKS.md).
- Found by measuring: every sandboxed `go test` failed once on Go's build cache; returning the first 32 KB
  of a large file raised large-file input 5.7× and was replaced.

## 1.0.0: container benchmark tasks and release gates

- Docker (Colima) running: the 19 container tasks were dry-run. 15 passed at once; 4 were rejected
  because their failing test panicked (`go-gjson-eaae8c63`, `go-semver-441ff8e5`,
  `go-gotoml-79f82e3c`) or hit a fatal stack overflow (`go-cast-1b425f31`) on the starting tree,
  aborting the test binary so later tests never ran (reproduced by hand in `golang:1.23`). After
  the fix (`TestAbortedPackages`), all 19 pass and `-dry-run -record-identities` stored their
  identities: all 55 tasks now grade `identity`.
- `go vet ./...` and `go test -count=1 ./...` pass; CI for the release commit and the tag's
  release workflow are the remaining gates.

## 1.0.0: follow-up review fixes (reviewed at `1d0e29b`)

Go 1.27.4, macOS ARM64. `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...` with `GOOS=linux` and `GOOS=windows`, and `env -u NO_COLOR make GO=go e2e` (acceptance plus the TUI, setup, ACP, attach, resume-setup and salvage PTY suites) pass. The race-instrumented attach gate exits 0 with no race reports. The native benchmark dry run passes all 36 tasks, all graded by recorded test identities. Docker was not running, so the 19 container tasks remain unchecked.

- Release blockers: `TestMentionsAndEmptyRunsAreNotChecks` (`echo "go test"`, `printf 'pytest'` and zero-test output do not verify), `TestVerifierThatChangesFiles` (a verifier that keeps rewriting ends `unverified`; an idempotent formatter ends `verified`), `TestInterruptedVerifierIsPending` (the verifier's effect stays pending after cancellation and the next run needs acknowledgement), `TestAccessNeedsEnforceableBoundary` (container and temp-folder overlap ask), `TestFindUnicodeAndLongLines` (İ/ẞ history and a late match in a long line).
- Grader: `TestClassifyNonPasses` (zero tests, expected failure, XFAIL, ambiguous suffix, unrecognized output), `TestReproducerAudit` (an existing failing test is not credited to a no-op added test), `TestRetrievalTelemetry` (a failed read earns nothing). `--dry-run --record-identities` recorded identities for all 36 native tasks.
- Research: `TestEvidenceCallsInOneReply` (two calls in one reply, `-race`), `TestRecallMaskedObservation` (lossless pages over 3-byte runes, huge offset, a reused provider call id does not resolve to newer output).
- P2: `TestLockWaitIsCancellableAndLiveHolderKept` (`-race`, three runs), `TestDirectOpenReconcilesInterruptedTrash`, `TestConflictingResumeSelectors`, `TestBudgetCheckedBeforeCompaction` (fails on the previous code: a summary call after the allowance was spent, 124 of 100 tokens), `TestChildRetryRemovesOnlyItsOwnSpan`, `TestHeldQueueSurvivesSameChatNavigation`, `TestLatestIsMostRecentlyActive`, `TestRejectedAnswerIsAskedAgain`, `TestMCPRestartWithUncertainCleanupStops`.
- CI gains `attach-race` (Linux, macOS) and `bench-smoke` (offline dry run of two native tasks).
- Not run: live inference, interactive Windows and editor sessions, Docker-backed tasks.

## 1.0.0: review fixes (verification truth, access, UI state, benchmark integrity)

Implements the code and UI reviews prepared at `8417223`. Go 1.27.4, macOS ARM64.

- Gates: `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...` plus `GOOS=linux` and `GOOS=windows`, `make vulncheck` (no vulnerabilities) and `env -u NO_COLOR make GO=go e2e` (acceptance plus the TUI, setup, ACP, attach, resume-setup and salvage PTY suites) all pass. Fuzz (5–15 s each): `FuzzJournal`, `FuzzToolArguments`, `FuzzLSPFrame`, new `FuzzSplitArgs`, which found that arm flags rewrote invalid UTF-8 (fixed; the seed is kept in `cmd/bench/testdata`).
- Verification slice through the normal entry points: `TestHeadlessVerificationSlice` (`goh run` with a local scripted endpoint: edit then `pwd` → `unverified` with the note, exit 0; a valid check → `verified`; failing configured check with `repair_attempts: 0` → exit 6 and `Verification failed: grep -q gamma note.txt (exit 1)` in text; one repair → `verified`, exit 0; journal `run_finished` checked each time), `TestInspectionIsNotVerification`, `TestShellCheckEvidence`, `TestConfiguredVerifierJournal` (`verification_started` → `verification` → `run_finished`), `TestOperationIsolation` (failing run → typed write → typed read → new prompt; no model request from `!`), `TestRunFinishedHookOrder`, `TestAutoTitleUsesTypedPrompt`. TUI: `TestFailedVerificationHoldsQueue`, `TestQueuedPromptsHeldAfterFailure`, `TestReopenShowsLastVerification`; picker: `TestPreviewShowsLatestOutcome`; attach: `TestStaleInputReplyKeepsTerminalEvent`; ACP: `TestNeedsInputAndFailedVerificationReachTheEditor`.
- Attach race gate (handoff script, race-instrumented `goh`): passes with no race reports. The same gate on an `8417223` build fails: the attached process exits 99 and the race report names `reloadAttached` at `remote_attach.go:168`. `test_attach.py` now asserts the attached and server exits.
- Fail-before/pass-after checked by reverting the fix: `TestPositionalResumeUsesSessionProject`, `TestFittingFirstRequestIsSentUnchanged`, `TestRepeatedManualCompactIsNoOp`, `TestFailedFallbackCoolsDown`, `TestStaleInputReplyKeepsTerminalEvent`, `TestRepoMapCoverage`. Also new: `TestAutoEditRunsReadOnlyCommands`, `TestAutoKeepsStricterSubfoldersUnwritable`, `TestMCPStartWithUncertainCleanupStaysPending`, `TestReplaceAllSizePreflight` (a refused 1 GiB expansion allocates under 256 MiB), `TestConcurrentCaptureAndRestore` (with `-race`), `TestFileURIForWindows`, `TestContentChangeFollowsSyncKind`, `TestReadPrefixBoundsLongLine`, `TestReconcileInterruptedTrash`, `TestChildRetryRetractsOnlyItsText`, `TestComposerGraphemes`, `TestUpFollowsWrappedRows`, `TestFreeTextQuestionFocusedAndSubmits`, `TestFindSearchesEarlierJournalTurns`, `TestPickerTinyHeightShowsSelection`, `TestProgressScreenOwnsKeys`.
- PTY: the color suite no longer inherits `NO_COLOR`; setup waits for the changed model name; the question case types into a focused free-text field; `/compact` on a short chat is a recorded no-op.
- Benchmark: `goh-bench --tasks bench/tasks --only go-counter-race,py-cart-discount --dry-run` → both OK with observed test identities. All 36 native tasks pass the stricter dry run; it first rejected `hard-go-report-order` and `hard-go-combo-release`, whose starting tree did not compile (fixed in the tasks and in `gen_hard.py`, which regenerates all 14 hard tasks byte-identically). The 19 container tasks were not run: the Docker daemon was offline. Unit coverage: `TestParseArms`, `TestParseTestsAdapters`, `TestRunCheckKinds`, `TestProtectedChanges`, `TestReadEventsValidatesStream`, `TestSummarySeparatesInfraAndFalseVerification`, `TestRunTaskFoldersAreFresh`, `TestRetrievalTelemetry`, `TestReproducerAudit`.
- Microbenchmarks (3 runs each): `BenchmarkRepoMapWarm` 463 → 465 µs, `BenchmarkRepoMapCold` 939 → 939 µs, `BenchmarkSearchProject` 8.06 → 8.05 ms (allocations +27 per map). `BenchmarkRepoMapQuery` (256 files): base 8.41 ms; `retrieval-expansion` 8.63 ms warm, 9.03 ms cold.
- CI on push (run 44551523967): unit tests on Linux, macOS and Windows, terminal suites on Linux and macOS, installers on all three and vulncheck all pass. The first push (run 44549862681) failed `TestAccessIsScopedToFolders` on Linux and Windows: its stricter `web/vendor` folder now makes Auto ask where the sandbox cannot keep a nested folder read-only, as intended; the test now pins that capability, and `TestAutoKeepsStricterSubfoldersUnwritable` covers both backends.
- Not run: live inference (experiment arms), an interactive Windows session (LSP drive/UNC URIs are unit-tested natively in CI, not against a Windows language server), README screenshots (not refreshed).

## 1.0.0: access levels, live status line, latency

- Latency, measured live over ChatGPT (`gpt-6.2-sol`). goh's own share of a request is about 30 ms (token 1 ms, TLS 29 ms); the service took 1.5–1.8 s to the first byte and 2.4–3.1 s to the first word for a bare "hi" at every reasoning effort. In goh's repository, `goh run hi` spent 1.5 s before the request and 1.15 s after the reply walking ignored folders for the change fingerprint (`TestFingerprintHonoursRootGitignore`, which fails without the fix). After the fix: 0.08 s before the request, 3.1 s end to end (was 7.0 s).
- Prompt cache: replaying three captured agent requests, a `session_id` header gave cache hits on 3 of 4 follow-ups against 0 of 6 without it. A live 10-turn `goh run` then hit on 5 of 9 follow-up turns (was 1 of 9), up to 21,760 of 22,170 tokens (`TestResponsesSessionHeader`). Requesting reasoning summaries without an effort was tried and dropped: the summary arrived 0.07 s before the answer.
- Access levels: `TestAccessLevels` (each level against edits, commands, network and subagent edits), `TestAccessIsScopedToFolders` (deepest saved folder wins, patches reaching outside ask, commands confined to the granting folder, session level replaces only the project's own), `TestAccessKeepsSafetyRules` (no sandbox means commands ask under Auto; ask/deny rules and the loop guard win under Full), `TestAutoAccessCommandsAreSandboxed` (a real sandboxed command writes inside its folder and cannot write beside it; found that the sandbox compared `/var` and `/private/var` paths unresolved), `TestAutoEditRunsReadOnlyCommands`, `TestReadOnlyCommand` (33 commands each way), `TestShiftTabCyclesAccess`, `TestApprovalAllowAllRaisesAccess`.
- Status line: `TestToolCallProgress` (Responses, Chat Completions and Anthropic streams report the tool, its size and the path once complete) and `TestActivityLabels`. Live chat: "Waiting for gpt-6.2-sol · 15s" then "Writing page.html · 265 tokens (41/s)"; the file was created at Auto-edit without a prompt (`"source": "access", "level": "edits"`).

## 1.0.0: real-project A/B, overload retries

- `goh-bench` on the 19 real-project tasks, check vs `--completion-check=false`, `gpt-6.2-sol`, containerised ([report](../BENCHMARKS.md)). On the 13 tasks both arms finished: 10 resolved and 3 false completions each, the same tasks. The selected-tests reminder never fired, because the model ran `go test ./...` after every `-run`.
- Five runs ended on in-stream `server_is_overloaded`, which goh did not retry. Fixed: `TestBusyStreamErrors` (OpenAI Responses, Chat Completions and Anthropic overload codes are `ErrBusy`; usage limits, rate-limit codes and empty errors are not; messages are never shown; credential-shaped Anthropic types are dropped) and `TestBusyProviderIsRetriedAfterAPause` (two failures then success; persistent failure gives up after three retries waiting 5, 10 and 20 s, each announced as `provider_retry`).
- Three runs hit the plan's `subscription_sharing_usage_limit_exceeded`; those stop at once, as intended.

## 1.0.0: live ChatGPT checks after re-sign-in

The 12 Nov sign-in had been revoked; it was removed with `goh auth logout` and a new one made with `goh auth login` (model `gpt-6.2-sol`). All live, over the ChatGPT plan:

- `goh auth refresh` renewed the token against the real token endpoint.
- Token expiry: with the saved expiry moved into the past, a `goh run` renewed before its request (expiry 06:55 → 07:56) and completed. 401 renewal: with the saved access token replaced by an invalid one, the service refused it, goh renewed once and the retried request completed. The saved account was edited by a throwaway test, not committed.
- `scripts/e2e/test_openai_live.py` passed: streamed text, permission-gated read → edit → read, durable resume, and the terminal `/connect` → ChatGPT → model → send. Its first run failed: the terminal check starts the chat with `--config`, and the connected provider never reached that chat. Fixed (`TestConnectedRouteForConfigChats`), then passed.
- `scripts/e2e/test_openai_live_extra.py` passed: image input, `reasoning_effort: low`, end-of-request task state, plan handoff.
- Reasoning summaries: with `effort: high` the service echoed `summary: detailed` and streamed `reasoning_summary_text.delta`; goh relayed the text unchanged (one heading per summary part, e.g. "**Counting roots below 2000**", which is all the service sends). An easy question produced no reasoning and so no summary.
- Prompt cache: three identical 7.6k-token requests with one `prompt_cache_key` reported `cached_tokens` 0, 0, 7552; goh's usage showed `cache_read_tokens: 7552`. In a 10-turn tool run with `todowrite` updates (task state sent at the end of each request), turn 8 read 6656 cached tokens, covering the system prompt and the first file read across two task-list changes. Hits are sparse when requests are a few seconds apart, which is the service's cache warming, not a changed prefix.

## 1.0.0: Linux keychain, product name

- Linux Secret Service: `TestOSStoreRoundTrip` (`GOH_OS_STORE_TEST=1`) passed in a Debian bookworm ARM64 container with `gnome-keyring` unlocked in a D-Bus session and `secret-tool` from `libsecret-tools`: binary-unsafe bytes written, read back, replaced and deleted.
- `TestLSPInitializeQueriesUTF16SyncAndDiagnostics` checks that `initialize` names `goh`; the web-fetch test checks the `goh/` User-Agent.
- CI note: one Windows run (44125517188) failed `TestRealClangd` (a second start approval during a 37 s clangd warm-up) and `TestParentChildCancellationUnderLoad` (a child shell did not start within 30 s). The next Windows run passed both unchanged. Recorded as runner-load timing; not reproduced.

## 1.0.0: completion check for name-filtered tests

`go test -race ./internal/runtime`. `TestTestScope` classifies 22 commands across Go, pytest, unittest, Jest, npm, Cargo, Maven and Gradle (including `grep` of a test command, which is not a test run). `TestCompletionCheckSelectedTests`: one reminder after `go test -run` alone or followed by a build; none after a full run, with or without a filtered run first, or after a non-test check. Not yet measured on the benchmark: the ChatGPT sign-in used for benchmark runs has been revoked.

## 1.0.0: package managers, SBOM, provenance

- `scripts/packaging.sh` on a local `make release` (1.0.0-pkgtest): `ruby -c` accepts the formula and the Scoop manifest is valid JSON.
- Homebrew, `homebrew/brew` container (Linux ARM64): installed from a temporary tap with `file://` URLs; `brew test` passed; `goh version` printed 1.0.0-pkgtest from `…/Cellar/goh/1.0.0-pkgtest/bin/goh`; `goh upgrade` answered `goh was installed with Homebrew; upgrade it with: brew upgrade goh`. On this Mac, Homebrew fetched and verified the binary but refused to install because the Command Line Tools are out of date (a host issue). CI now runs `scripts/test_brew.sh` on macOS and a real Scoop install on Windows.
- SBOM: `cyclonedx-gomod@v1.10.0 app` wrote CycloneDX 1.6 with 26 components.
- `TestPackageManagerOwnedCopies`: Homebrew and Scoop paths on each platform; goh's own install folders are not mistaken for them.

## 1.0.0: live failover and remote MCP

- **Failover:** `goh run` with the primary route at a closed local port and `fallback_profiles: ["groq"]` (saved Groq key, `qwen/qwen3.8-27b`): two `provider_retry` events, `provider_failover` (`reason: connection`, `to: groq`), `model_route` with `failover: true`, and Groq streamed `FAILOVER_OK`. Two `goh resume` turns, then `goh compact`: `provider_failover` with `purpose: compaction`, then `context_compacted`.
- **Remote MCP:** DeepWiki's public server (`https://mcp.deepwiki.com/mcp`, streamable HTTP, no auth; serverInfo DeepWiki 4.0.2) with `--allow mcp_start:deepwiki,mcp__deepwiki__read_wiki_structure`. The server started, `read_wiki_structure` for `charmbracelet/bubbletea` returned its page list, and an unknown repository came back as the server's own message. A call with a wrong argument was refused by goh's schema check before it was sent (`missing property 'repoName'`).

## 1.0.0: live language servers (gopls, pyright, typescript-language-server)

The compiled `goh run` driven by the scripted model through all six `lsp` operations (`documentSymbol`, `hover`, `goToDefinition`, `findReferences`, `workspaceSymbol`, `diagnostics`) on a small project per language, with `--allow lsp,lsp_start:NAME`:

| Server | Version | Result |
| --- | --- | --- |
| gopls | v0.20.0 | all six; diagnostics reported the real `cannot use "oops"` type error (severity 1) |
| pyright-langserver | 1.1.406 | all six; diagnostics reported the `str` to `int` assignment |
| typescript-language-server | 5.0.1 (TypeScript 5.9.3) | all six; diagnostics reported `Type 'string' is not assignable to type 'number'` |
| rust-analyzer | rustup stub without the component | start failed, as expected |

- gopls first answered `no views` to every position query: the `go` command was not on the test's `PATH` (a shell function on this Mac), so gopls could not load the module. With the toolchain on `PATH` everything passed. Users see the same symptom without `go`; gopls's own `No active builds contain …` diagnostic is passed through.
- The rust-analyzer stub exited before `initialize`, and goh said only `LSP connection closed: EOF`. A start failure now ends with the server's last two error lines (escape sequences removed, at most 300 bytes): `…; the server said: help: run 'rustup default stable' …`. `TestLSPStartFailureIncludesServerError` covers it.

## 1.0.0: one terminal program per interactive session

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; all six terminal suites. Linux ARM64 container: all six terminal suites.

- `test_tui.py` now fails any scenario that enters the alternate screen more than once (editor handoff excepted). Every scenario entered it once, including `undo_redo`, `timeline_revert`, `provider_models` and `plan_exit`. The same check against a build of the previous commit failed with `undo_redo: entered the alternate screen 3 times`.
- `test_setup.py`: `/fork` before a turn opens the branch without leaving or re-entering the screen.
- New `test_salvage.py`: `goh chat --session` on a session damaged in its second turn asks on the normal terminal, recovers the first turn into a new session, opens it in the chat, and leaves the damaged file unchanged.
- SIGTERM to a hosted chat exits 0 with the terminal restored. The first version handed the waiting chat an error, and the launcher then opened a standalone provider picker that nothing could answer; found by `test_tui.py` `termination`.
- Unit tests: `TestHostCarriesKeysAcrossScreensAndDropsStaleMessages` (fails when message tagging is disabled: a closed screen's timer reached the next screen), `TestHostQuitInsideBatchClosesScreen`, `TestHostDialogBackdropFollowsChatResize` (fails without the live backdrop), `TestOwnMessage`.

## 1.0.0: CI back to green, credential boundary through symlinks

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; all five terminal suites; `govulncheck ./...` (v1.9.0, for darwin, linux and windows): no vulnerabilities; v1.1.4 panicked on the Linux build in CI (`unexpected expr: *ast.KeyValueExpr`). Linux ARM64 in a `python:3.12-bookworm` container (Colima): all five terminal suites passed with the cross-built binary.

- CI run 43748015459 (15 Nov) failed three ways. Linux `test_tui.py` `text` showed the first-run provider picker: the suite's private home lived inside the test project, which goh refuses because its saved keys would be inside the project. Reproduced in the container with the old script (fails) and the new layout (passes).
- The same layout passed on macOS, which exposed a real gap: `CheckProjectCredentialBoundary` compared the symlink-resolved project (`/private/var/…`) with the unresolved, not-yet-created key folder (`/var/…`), so the check was skipped. `TestCredentialBoundaryResolvesSymlinkedHome` fails without the fix.
- Windows: `TestCheckRoute` expected "not running at"; `errors.Is(err, syscall.ECONNREFUSED)` does not match Winsock's `WSAECONNREFUSED`, now `platform.ConnectionRefused`. `TestHomeFolderIsRefusedWithAFolderFix` read the runner's real profile because `USERPROFILE` was not isolated.

## 1.0.0: small per-request caps (Groq free tier)

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; `test_tui.py`, `test_setup.py`, `test_acp.py`, `test_attach.py` and `test_resume_setup.py` all passed.

- New tests:
  - **Provider:** `TestRequestTooLargeReportsStatedCap` (only the two numbers leave the body), `TestRequestCapParsing`, `TestQuotaResetFailsWithoutRetrying` (one request, reset time stated).
  - **Runtime:**
    - `TestRequestCapShrinksAndResends` and `TestRequestCapShrinksPreservedGroup`: an oversized result in the only group and in the newest of several.
    - `TestRequestCapLongRun`: page, lookups until a summary, the page again; one refusal, then nothing over the cap.
    - `TestRequestCapWithoutNumbersFails`, `TestUsageCalibratesEstimate`, `TestSummaryEstimateSkipsAnchor`, `TestOlderExcerptBecomesReceipt`, `TestLargeFetchedPageIsPageable`.
  - **Tools:** `TestArtifactPatternSearch`, `TestArtifactIDUsedAsPathGetsHint`.
  - **TUI:** `TestArtifactReadLabel`.
  - Disabling the anchor fails `TestRequestCapShrinksAndResends`.
- Live, Groq, "What's the latest stable Go release? Check the official Go site":
  - **Before:** failed at once with HTTP 413.
  - **Cap estimate fixed:** Groq's first request counted 1,961 prompt tokens for goh's 9.7 KB of instructions and tools, against the 9.7k that one token per byte assumed (the earlier "about 10.8k" note was that estimate).
  - **Checksum page:** go.dev/dl is mostly checksums, about 2.3 bytes a token. A single whole-request ratio inflated the tools to over 4k tokens until the anchor was split from the content ratio.
  - **After, with `openai/gpt-oss-20b`:** the cap was learned once per run ("The provider accepts at most 8000 tokens per request…"). Later requests stayed under it through summaries and re-fetches, and the model searched the saved page with `read_artifact` `pattern`.
  - **Not answered:** the 20B model never gave the answer within 290 s. It used wrong regexes, re-fetched the page and called `search` on artifact IDs, all while waiting 15–30 s on per-minute 429s. `openai/gpt-oss-120b` had used its daily quota: Groq's 429 said "try again in 28m", which led to the quota-reset fix.

## 1.0.0: end-to-end audit with real-world prompts

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; `test_tui.py` (25 scenarios), `test_setup.py`, `test_acp.py`, `test_attach.py` and `test_resume_setup.py` all passed.

- Live, Groq `openai/gpt-oss-120b` over `chat_completions` (ChatGPT could not be used: its saved sign-in was revoked, which found the renewal message fix). Real prompts on a small Python project: "I just inherited this repo…" completed after six tool calls and four 429 waits of 6–25 s (the same prompt previously failed at the fifth request); "how many stars and open issues does golang/go have" through the GitHub REST API returned live counts (previously HTTP 415); "total spend in EUR via the ops MCP tool" started the server, read the CSV and converted correctly. Fetching go.dev and Hacker News failed with HTTP 413: Groq's free tier caps a request at about 8k tokens, below one 64 KiB page; the error now says so. With `--context-window 8000` the run stopped before any request (`budget_exhausted`). That figure was the one-token-per-byte estimate: Groq counts goh's instructions and tools as about 1,960 tokens (see the per-request cap section below). An explore subagent prompt completed: both children failed (one `tool_use_failed` after re-sampling, one 429) and were reported, and the parent finished the audit itself.
- gpt-oss calls tools that do not exist (`repo_browser.read_file`) and invents file text (`<cstdio.h>`). Groq rejects some such calls mid-stream (`tool_use_failed`), which ended runs as "provider stream error"; the request is now re-sampled up to twice. Invented `old_text` is refused and the existing invalid-call limit stopped the loop after three identical failures.
- Groq's rejection messages, read once with a local debug build, showed gpt-oss calling `apply_patch` with `patch` instead of `patchText` (refused by Groq's schema check), plus invented tools and an out-of-range `limit`. After the alias, "the monthly report prints months in the wrong order and top categories look backwards; fix both and make sure the tests pass" produced a correct two-line fix and `unittest` passed; the run then hit its 300 s cap during rate-limit waits before the final summary.
- Components driven by the scripted model against real targets: `webfetch` of Hacker News, go.dev, example.com, a PDF, a 404, an `http://` → `https://` redirect and the cloud-metadata address; `lsp` against clangd (hover, references, document symbols and diagnostics without a position); a stdio MCP server (tool, error result, resources); skills, todos, `apply_patch`, 109 KB of shell output, `job_start` of a real HTTP server (stopped on exit), subagent with `output_schema`, question (exit 4), permission (exit 3), plan mode, resume, fork, Markdown export with tools, memory recall and a custom command.
- TUI in a PTY (110×40): Python source with `# TODO` and `**kwargs` is shown verbatim; grep and glob results are lines; the reply after tool output is a separate paragraph.
- New tests, each confirmed to fail without its fix where checked: `TestToolOutputRendersVerbatim`, `TestListingResultsAreReadable`, `TestJSONAPIsErrorsAndRedirectDestination`, LSP file-wide cases in `TestLSPAdmissionFailureAndBounds`, `TestPatchPreviewShowsChangeDeepInLongFile`, `TestShellPreviewKeepsHeadAndTail`, `TestRetryDelayWaitsOutRateLimits`, `TestStreamErrorCodes`, `TestRejectedOutputIsResampled`, `TestHeadlessRunSkipsUnapprovedMCPServer`, `TestPatchArgumentAlias`, `TestDroppedStreamIsInterrupted`.

## 1.0.0: project browser and folder trust

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- PTY, the user's real home (nothing chosen): started in `~`, the browser explains the home folder and lists its folders with project tags; `Desk` + Tab lists `~/Desktop` (goh tagged `git · Go · make`); `goh` + Enter shows the trust question; typing `~/Desktop/g` completes to three folders.
- `test_setup.py` now starts in an untrusted project: it opens the folder, answers the trust question and checks `workspaces.json` before the provider welcome. `TestWorkspacesTrustAndRecency`, `TestWorkspaceBrowserOpensAndTrusts`, `TestWorkspacesCommand`.

## 1.0.0: home-folder setup loop, saved /models choice, terminal colors

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- Reported: Groq key added and model chosen, yet chat kept showing "No model connected". With the user's own settings (read only), `goh providers list` and `goh doctor` were valid and `goh run hello` got a reply from Groq; a PTY run from `~/Desktop/goh` opened chat with the Groq model, while one from `~` showed the blocked screen with "choose a project directory outside the GoHarness credential storage scope". After the fix, the same run from `~` says "Chat can't open this folder"; Enter, typing `/tmp/gohproj` and Enter opened chat on the Groq model.
- `/models` on the user's settings (viewed, nothing selected) lists four Groq chat models and "Switch to ChatGPT"; `goh models` prints the same four instead of eleven. `test_setup.py` now checks that a `/models` choice is written to the provider in `config.json`. `TestHomeFolderIsRefusedWithAFolderFix`, `TestSameRoute`, `TestGroqHidesNonChatModels`.
- Default theme: the first chat frame has no background sequences (was 34 `48;2;10;10;10` and 4 panel sequences) and is 982 bytes instead of 2455; only the selected picker row uses a background. `theme_light` still renders its own background.
- `test_tui.py` previously ran goh with the caller's real home folder; it now uses a private one.

## 1.0.0: type-ahead across screens, setup screen, docs history

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- Reproduced in a PTY before the fix: `Tab` then `hello` sent as one write left only `h` in the composer; `build⏎hi there` sent to the agent picker in one write left nothing. After the fix both arrive intact, as does `/agents⏎build⏎hi there` in a single write. `TestKeysTypedWhileAScreenClosesReachTheNext` checks that late keys reach the next screen in order and that Enter keeps its meaning.
- `TestEnterOnBlockedChatOpensSetup`; the setup screen was rendered at 40×12 and 100×24.
- A first attempt word-wrapped every terminal row; `formatter_approval` then failed deterministically because the approval preview no longer fit at 80×24 and needed scrolling. Word wrapping is now limited to the setup screen, and the suite passes.

## 1.0.0: cleanup, CLI errors and repository layout

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` (acceptance plus `scripts/e2e/` TUI, setup, ACP, attach and resume-setup suites) all passed.

- `TestCLIErrorsSayWhatToDo` covers the no-provider message, `goh run` without a message, a missing `--project`, `goh resume` with flags before the ID, an unknown session ID and `goh compact` usage. A fresh-home probe of every top-level command found no remaining internal errors.
- `goh resume ID words…` was run against the scripted model with flags before and after the ID.
- `/settings` was rendered at 50 and 90 columns in a PTY; `test_setup.py` drives it with `s` and `q`.
- Dead code found with `golang.org/x/tools/cmd/deadcode` (with and without tests, for each OS) was removed; `licenses/` matches `go list -m all` exactly; a link check over all Markdown outside `research/` found no broken links.
- The README demo (`scripted-model` plus `goh run --config config.example.json`) was run as written.

## 1.0.0: third review fixes, onboarding and approval layout

Local, macOS ARM64, Go 1.27.4: `gofmt`, `go vet`, `go test -race ./...`, Linux and Windows cross-builds, `make acceptance`, `scripts/test_tui.py` (25 PTY scenarios), `test_setup.py`, `test_resume_setup.py`, `test_attach.py` and `test_acp.py` all passed.

- New regression tests, each confirmed to fail without its fix where it can run on macOS: `TestMCPRestartDuringCallIsSettled`, `TestSessionGrantsDoNotCoverInterpreterScripts`, `TestSandboxReadOnlyPathsInsideProject`, `TestMalformedSiblingInReadBatchIsNotExecuted`, `TestVerifyRepairShowsEndOfLongOutput`, `TestEncodeAnthropicRemapsForeignToolIDs`, `TestToolAssemblyWithRepeatedIDAndName`, `TestLogoutClearsCredentialsWhenRevokeFails`, `TestChatGPTLockWaitsForAnotherUpdater`, `TestPromptStartingWithPathIsNotACommand`, `TestLinkedParentDetectsSymlinkedFolders`, `TestDiffPreviewShowsOnlyTheChange`, `TestFitHintsDropsWholeHints`, `TestEditCustomProviderMovesSavedKey`, `TestCustomProviderWithUnsetKeyVariable`, `TestRunOutlivesServerReadTimeout`; extended `TestShellDecision`, `TestShellDescriptionReflectsSandbox`, `TestRunOutcomes` (empty reply not committed), the sandbox canary (parent listing) and the glob cases (nested base, for Windows CI).
- Not verifiable here: the Windows `api_key_env` case rule and glob separator fix (Windows CI), and the Landlock parent-listing rule (Linux CI).
- The review's `goh serve` ReadTimeout finding was a false positive: a probe and `TestRunOutlivesServerReadTimeout` show the request context survives the server's read deadline, so no change was made.
- PTY scripts updated for the new first run: the welcome picker, Enter-while-blocked opening the picker with the draft kept, and Esc from a key form returning to the picker.

## 1.0.0: /connect provider catalog

- Provider manager: `TestProviderRegistry` and `TestProviderRegistryRefusals` cover listing a hand-configured provider, saving and switching three providers including ChatGPT, removing the active one, refusing a provider an agent uses, refusing commented settings, and keeping unrelated settings. `TestProvidersEndToEnd` covers two providers keeping their own keys through `goh providers list/use/remove`, where removing a provider deletes its key. Also `TestChatGPTSavesAsAProvider`, `TestProviderManagerKeys` (two-press remove, Add row), `TestSettingsChoiceField` and `TestWordWrap`.
- `scripts/test_setup.py` (PTY) now runs setup through the manager:
  - status `p` → empty manager → Add → Custom provider;
  - a wrong key is refused before anything is written;
  - the right key saves the provider;
  - the fixture then revokes it, chat fails with 401, and `k` in `/providers` takes the new key after checking it;
  - the retry succeeds in the same session;
  - a second provider added with `a` becomes active, and Enter on the first switches back.
  
  `scripts/test_resume_setup.py` uses the same path. The suite passed three runs in a row after its wait was scoped to post-save output.
- `TestSwitchChatGPTAccount` covers the user-reported `3p_login_workspace_scope_denied`. The switch sends `dynamic_agent_client` with a new `ext_agent_host_id`, accepts a different account, saves the new registration and host identity, and revokes the old renewal token. An abandoned switch leaves the old sign-in and host identity untouched.
- `TestProviderCatalogRoutesValidate`, `TestPresetChatModels`, `TestKeyCheckURLsStayOnTheProviderHost`, `TestListModelInfoChecksAnUnsavedKey` (Bearer vs `x-api-key`, rejected key, reported windows), `TestKeyRefusalsAndKeyCheck` (Google's 400, OpenRouter-style key endpoint, key never sent off-host), `TestVerifyKey` and `TestSavePresetKeepsOtherSettings` (verified key saved under the preset's scope, other settings kept, old `api_key_env` dropped).
- `scripts/test_setup.py` (PTY): `/connect` lists the catalog, the Anthropic form shows its key page, saving with no key is refused, and Esc leaves settings and the request count unchanged.
- Live, with an invalid key and no quota used: Anthropic, OpenAI, OpenRouter, Gemini and Groq all report "did not accept this key". This found that OpenRouter's model list answers 200 to any key and that Gemini refuses keys with 400, both now handled. The key pages resolve; the Anthropic Console link now points at `platform.claude.com`, where the old address redirects.

## RC180 goh upgrade and first release

- `TestUpgradeReplacesTheExecutable` and `TestUpgradeRefusals` (tampered binary, missing or duplicated checksum, binary that does not run). Each refusal leaves the executable unchanged and no temporary files. Also `TestUpgradeCommand` (up to date, `--check`, remote plain HTTP refused) and `TestReplaceExecutableKeepsOldOnWindowsOnly`.
- End to end on macOS: `make release VERSION=9.9.9-e2e` was served on loopback. A `1.0.0-old` build upgraded itself to `9.9.9-e2e`. Running again reported "up to date". After the served binary was tampered with, a fresh old build refused it and still reported `1.0.0-old`.
- History rewrite: `git filter-branch --index-filter` removed `harness` from the 16 commits from rc.170 on. The final tree hash `cbe24da` is unchanged, no blob over 2 MB remains, and the local pack went from 47.6 MiB of loose objects to 6.4 MiB. A bundle of the old history was kept locally before the force-push.

## Earlier releases

Evidence for rc.1 – rc.179 is in [history/VERIFICATION-rc1-rc179.md](https://github.com/Sanidhya14321/goh/blob/v1.0.0/docs/development/history/VERIFICATION-rc1-rc179.md).
