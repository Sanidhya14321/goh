# Verification record — 7 November 2026

This is a working development alpha. Full release acceptance in the supplied plan
is not complete. The observed host is macOS 26.5.1, ARM64, Apple M5 Pro, Go 1.27.3.
Live inference was tested on loopback. Source milestones are published to Sanidhya14321/goharness.

## RC23 editor and Windows filesystem

Native CI at RC22 (run 42386722466): `go vet` and `go test ./...` passed on
`ubuntu-latest` and `macos-latest`. On `windows-latest` the build and vet
passed and tests failed, mostly from directory fsync ("Access is denied") and
permission-bit privacy checks; a formatter test also depends on Unix scripts.
RC23 addresses the first two. Locally `1.0.0-rc.23` passed `go vet`, the race
suite, seven CLI scenarios and seventeen PTY scenarios twice. The new
`editor_handoff` PTY scenario sets `$VISUAL` to a script, opens it with Ctrl+X E,
and checks the journal records the rewritten prompt.

## Windows session delete (after RC33)

Native CI for RC32 and RC33 failed on Windows only: renaming a journal while
holding its lock handle is not allowed there, so trashing a session failed.
Commit `29bf905` keeps the lock held through the move on Unix and closes the
handle just before the journal moves on Windows. CI on `29bf905` passed all six
jobs.

## RC177 second review, runtime findings

Regression tests reproduce each reviewed scenario:
- `TestFingerprintSeesRewritesWithRestoredTimes`: same size, mode and modification time, different content.
- `TestAnthropicUntaggedThinkingIsNeverReplayed`: legacy thinking reaches neither the same route nor another endpoint, and a fresh prompt can think again.
- `TestResponsesStreamErrorDropsCredentialShapedCodes`.
- `TestShownTextMatchesTheFinalReply` (suffix, divergent, identical and unstreamed replies after a retry) and `TestRetryRemovesPartialReplyFollowedByReasoning`.
- `TestLegacyMalformedToolArgumentsForkAndReimport`: fork, export and reimport, plus an archive exported before the repair.
- `TestLSPUnversionedStaleDiagnosticsAreNotTaken`: a new fixture mode republishes the previous text's diagnostics unversioned, then the current ones. The test fails with the settle window removed.

Gates passed on macOS with Go 1.27.4:
- gofmt, `go vet` (also with `GOOS=windows` and `GOOS=linux`) and `go test -race ./...`;
- `make acceptance`;
- the TUI, setup, ACP and attach PTY suites.

## RC176 second review, P1 and attach findings

Each finding has a regression test that reproduces the reviewed scenario:
- `TestAttachAcknowledgmentStaysWithItsSession`: an acknowledgment for session A does not apply to B and covers only A's first prompt. `TestAttachAcceptsFlagsAfterTheAddress` checks flags after the address.
- `TestCancelBeforeResponseHeaders`: a fixture holds the run's response headers. Cancelling returns `cancelled` at once, and the server sees the request end.
- `TestRelativizePreviewLeavesHeaderLookalikesInBodies`: a heredoc `Directory:` line, a diff body `--- ` line and a patch excerpt `File 2:` line are left unchanged.
- `TestDenyCoversNewFilesAndUnicodeSpellingsOnFoldingFilesystems` (macOS APFS, case-insensitive): denies `PRIVATE/new.txt` under `private/*`, both by `denied` and by `CheckWritable`, and denies a decomposed spelling of a file denied in composed form. `TestCanonicalRelResolvesExistingPrefix` covers a symlinked prefix with a missing tail, and a symlink out of the project.
- `TestStripFileBlocksCountsInsteadOfSearching` and `TestAttachmentClosingTagCannotSmuggleMentions`: a `</file>` line in an attachment exposes no mentions, and forged blocks in the user's own text hide nothing.
- `TestRunnerGradesCleanlyAndFlagsFalseCompletion`: a task without an image is refused unless `-host-grading` is set. `TestAgentContainerCannotReachGradeTree` (Docker Desktop, alpine) has an agent container write to `../grade` and to the grade tree's absolute path. Neither reaches the host's grade tree, and the container is gone after it is stopped. The test fails when the container mounts the whole attempt folder, as before this change. It skips where Docker or the image is unavailable.

Gates passed on macOS with Go 1.27.4:
- gofmt, `go vet` (also with `GOOS=windows`) and `go test -race ./...`;
- `make acceptance`;
- the TUI, setup, ACP and attach PTY suites.

## RC175 goh attach

- `internal/remote` tests run a real local API server over a scripted model, under the race detector:
  - `TestRunRelaysEventsApprovalsAndQuestions`: an approved write lands in the server project, a question is answered, events stream, and the read-only view reads the server's history and refuses writes;
  - `TestDeniedApprovalAndCancellation`: a denied write is not applied, and cancelling the client cancels the server's run;
  - `TestNewRefusesNonLoopbackAndMissingToken`, `TestWrongTokenIsReported`.
- `scripts/test_attach.py` runs the real binaries: `goh serve` with the scripted model, and `goh attach` in an emulated 100×30 terminal. The edit is approved and applied on the server, the shell check is denied, both decisions are in the server's journal, and `!command` is refused while attached. It is added to the CI terminal job.
- The full gates pass: vet (macOS and Windows), race suite, acceptance, and the TUI, setup, ACP and attach suites.

## RC174 Go 1.27.4

The official go1.27.4 darwin-arm64 archive was checked against the SHA-256 published at go.dev/dl (`91a44a2b…bd0bb4`) and installed as `research/tooling/go`; 1.27.3 is kept beside it as `go1.27.3`. `go mod tidy -diff` is clean. The full gates pass on 1.27.4: vet (macOS and Windows), race suite, acceptance, the TUI, setup and ACP suites, and the bounded fuzz targets. `govulncheck` was not rerun: downloading it was not permitted in this session.

## RC173 Linux sandbox thread race

CI run 43065205413 failed `TestInjectedInstructionsCannotExfiltrateOrPlant` once on ubuntu-latest: the sandboxed shell read a protected file, and a rerun passed. Cause: `landlock_restrict_self` and `PR_SET_NO_NEW_PRIVS` act on the calling thread, and the Go scheduler could move the helper goroutine between `restrict` and `syscall.Exec`. Reproduced in a Linux 6.8 container (arm64, Landlock ABI ≥ 4) under parallel load: the old test binary failed 1 of 4,800 runs with the same message; the fixed binary failed 0 of 4,800. The race is too rare to bound statistically. The fix follows the kernel semantics (`runtime.LockOSThread` in the helper).

The PTY suite now also checks an emulated 80×24 screen; with it, the macOS and Ubuntu terminal jobs passed in CI.

## RC172 external review: terminal UI

New tests:
- `TestTinyTerminalCannotApproveUnseenOperation` (45×3), `TestCompactViewHidesThinkingAndScrolls` (52×15), `TestQuestionSelectionStaysOnScreen` (64×18), `TestSettingsFormKeepsFocusAndErrorVisible` (62×16);
- `TestPaletteCommandsKeepTheDraft`, `TestQuotedMentionsAttachPathsWithSpaces`;
- `TestRetryRemovesDiscardedPartialReply`, `TestRunNoticesShowBesideSpinner`;
- the keymap test rejects `ctrl+l`.

`scripts/test_setup.py` asserted that a quoted picker path was not attached, which was the reported bug; it now checks that picking inserts only the path and that sending attaches it. The TUI suite passes with `COLORTERM` unset. The full gates pass: vet, race suite, acceptance, and the TUI (25 scenarios), setup and ACP suites.

Not done: moving to Go 1.27.4 for GO-2026-6859 needs a toolchain download, which this session did not do.

## RC171 external review: P2/P3 fixes outside the TUI

New tests reproduce each finding:
- `TestRunOutcomes` (shell edit, shell edit checked, cache only), `TestFingerprintIgnoresStateAndCaches`;
- `TestUserShellReportsFailureAndKeepsCancelledEffectUncertain`;
- `TestLargeIntegersReachTheServerExactly`, `TestMCPPromptConnectsBeforeExpanding`;
- `TestLSPDiagnosticsDescribeTheCurrentText` (fails without the fix; delayed-diagnostics fixture mode);
- `TestAnthropicThinkingStaysOnItsRoute`, `TestAnthropicEarlyEOFIsInterrupted`, `TestStreamedRefusalIsTheReply`, `TestResponsesStreamErrorIsSanitised`, `TestJSONCompletionKeepsCachedTokens`;
- `TestDamagedCredentialsDoNotBlockLoginOrLogout`;
- `TestMalformedToolCallKeepsSessionPortable` (fails without the fix), `TestTrashConflictKeepsSessionTogether`, `TestPreviewUsesSavedTitleAndEmptyListIsArray`, `TestSessionCommandsUseConfiguredSessionDir`;
- `TestRetryAndFinalSuffixReachTheEditor`, `TestRunWorktree` (unrelated repository), `TestDamagedSessionPointsToSalvage`.

The PTY suite first caught journal writes in a session folder inside the project being counted as changes (paths compared before resolving `/var` → `/private/var`); fixed. The full gates pass: vet, race suite, acceptance, and the TUI (25 scenarios), setup and ACP suites.

## RC170 external review: P1 fixes

Each fix has a test that reproduces the reported failure:
- `TestRestoreRefusesToDeleteUnrecordedFiles` (ignored file kept; restore then succeeds);
- `TestAttachmentsHonourFileRulesInEverySpelling`, `TestDenyCoversOtherSpellings`, `TestAttachFilesRefusesDeniedAndSpecialFiles` (case, symlinked folder, image, nested mention, FIFO);
- `TestRelativizePreviewLeavesCommandsAndContentExact`;
- `TestRepairDoesNotReuseResultOfEarlierCallWithSameID`;
- `TestWithoutStartupFlags`;
- `TestFileCompletionSurvivesShrinkingRescan`;
- `TestHostGradingDropsSecrets`, `TestShortCommitIsAnError`;
- `TestShareJobsKeepsJobsAcrossRebuiltChat`.

The full gates pass: vet, race suite, acceptance, the TUI (25 scenarios), setup and ACP PTY suites, and `make bench-check` (55/55 tasks valid). The container cache change was not exercised against Docker in this run.

## RC169 technical, licensing and documentation audit

- Build, `go vet` and the full test suite pass; `licenses/` matches `go list -m all` exactly, and every "Host CLI" entry in DEPENDENCIES.md matches `go list -deps ./cmd/harness`.
- A reference-count scan of all product functions found one test-only wrapper (`parseResponses`), now removed.
- Tracked research evidence contains no reference prompts; product code contains no copied upstream source (OpenCode is named only in behavioural comments).
- New test: `TestShellDescriptionReflectsSandbox`.
- The full gates pass: vet, race suite, acceptance, and the TUI (25 scenarios), setup and ACP PTY suites.
- Not rerun: `govulncheck` (downloading the tool was not permitted in this session); the last retained result is in `research/evidence/govulncheck.json`.

## RC168 queue and sidebar audit

- New tests:
  - a queued prompt survives an open panel;
  - the sidebar shows open tasks after 20 finished ones.
- The full gates pass: vet, race suite, acceptance, and the TUI, setup and ACP PTY suites.

## RC167 scrolling and queue audit

- New tests in `component_audit_test.go`:
  - the scroll anchor holds while output streams (this test fails when the anchor is disabled);
  - the scrolled-up hint appears and Esc returns to the latest output;
  - approval review does not overshoot;
  - ↑ takes back a queued prompt.
- The full gates pass: vet, race suite, acceptance, and the TUI, setup and ACP PTY suites.

## RC166 component audit

- `go test ./internal/tui` passes. That includes the new `component_audit_test.go`, which has one test per fix:
  - paste during a run;
  - empty question;
  - custom-answer Esc;
  - uppercase approval;
  - Tab cycling;
  - one-line picker paste;
  - settings discard prompt, paste trimming and typed spaces;
  - find after a trim.
- `jobs_test` now requires a confirming second Ctrl+K.
- The full gates pass: vet on macOS and Windows, the race suite, acceptance, and the TUI, setup and ACP PTY suites.
- The setup PTY suite caught a regression during this work (typed spaces were trimmed in single-line fields). It was fixed before the commit. The suite now expects the new "already exists" export message.

## RC165 event audit

- `internal/tui/event_fuzz_test.go`:
  - covers 30 in-session states and 6 setup screens;
  - sends every event once from a fresh state, then a seeded random walk;
  - each event has a 2 s deadline;
  - checks for panics, blocked handlers, and frames taller or wider than the
    terminal.
- `TestIdleCtrlCClearsDraftThenNeedsDoublePress`.
- Real PTY checks:
  - SIGTERM restores the terminal (alternate screen and mouse tracking off);
  - SIGHUP exits;
  - a 300 KB bracketed paste stays responsive.
- Latency with a 128 KB transcript at 200x60: 0.5 ms per keystroke, 2.3 ms per
  wheel notch, 2.6 ms per resize (update plus render).
- All gates pass.

## RC164 mouse and first frame

- `internal/tui/mouse_test.go`:
  - the wheel scrolls the transcript and stops at the bottom;
  - the wheel moves the palette selection;
  - clicks, right clicks, releases, motion, focus and blur change nothing;
  - the first frame before the size is known is blank and uses the
    alternate screen.
- Real PTY at 100x20:
  - the alternate screen (`?1049h`), mouse tracking (`?1002h`) and SGR mouse
    reporting (`?1006h`) are enabled;
  - after injected wheel and right-click sequences, the process stays
    responsive and exits cleanly with Ctrl+D.
- All gates pass: vet, race, acceptance and the PTY suites.

## RC163 TUI polish

- `go test ./internal/tui` (new `spinner_test.go`: humanized counts, status
  states, project-relative approval previews, path truncation).
- `scripts/test_tui.py` PTY suite passes with the new approval banner.
- Real PTY captures at 120x36 (session, approval, palette) refreshed in
  `docs/images/`.
- vet (macOS and Windows), race suite and acceptance gates pass.

## RC161 token budget

`TestMaxRunTokens`: with a 150-token budget and 100 tokens per turn, the run stops before the third
turn with `budget_exhausted`, after both tool calls returned results. All gates passed.

## RC160 MCP prompts

`TestPrompts` (fixture server with a prompt: listed with required argument, rendered with arguments,
unknown server refused, servers without prompts list none) and `TestMCPPromptExpansion` (free text to
first argument, `name=value` pairs, server label present, malformed form fails). All gates passed.

Benchmark (rc.159): line-range edit A/B 20/20 with line ranges used once in 13 edits (kept off);
compound three-bug tasks 8/8.

## RC158 test-double check and opt-in stream retry

New tests: `TestCompletionCheckTestDouble` (own fake → one reminder; plain test or fake in source → none),
`TestWritesTestDoublePatch`, `TestRetryInterruptedStreams` (off → failed with one request; on → completed
with two; the broken reply's tool call never ran; 3 retries rejected). All gates passed: gofmt, vet
(macOS/Linux/Windows), race suite, acceptance, TUI PTY, setup and ACP suites.

Stage 2 A/B on rc.157 (`docs/bench/stage2-ab-rc157.md`): base 9/10, nocheck 10/10; the completion check
fired in 0 of 20 runs. Hard-set total so far: 44/46 resolved, 2 false completions (same task).

Root cause of those false completions: a task defect. The hidden test for `hard-py-double-charge` calls
`gateway.lookup()`, which nothing in the task repository named. With the old check (name patterns only)
the reminder fired in 0/3 runs (model fake: `IdempotentGateway` plus `unittest.mock`); with broadened
detection it fired in 3/3 but could not help (1/3 resolved), because the model correctly reported that no
real gateway existed to check against. After documenting the interface in the task: 3/3 resolved,
0 false completions (rc.158).

## RC157 stream error codes and partial stage 2

`TestResponsesStreamErrorShowsCode` covers four cases: a top-level error event, a nested `error` object, and a `response.failed` error each keep their code and a cleaned message, and an event with neither falls back to the old hint. Long messages are bounded.

**Stage 2 (hard set) was cut short.** After about 26 runs the ChatGPT plan returned `subscription_sharing_usage_limit_exceeded` inside the stream. Until RC157 this showed only as "request failed", which is how the problem was found.
- **Before the limit (base arm):**
  - 25 of the 26 real attempts resolved;
  - 1 genuine false completion: on `hard-py-double-charge`, the model invented an `idempotency_key` parameter the gateway does not accept, tested it against its own fake gateway and reported `verified`, while the hidden tests fail with `TypeError`.
- **Lost to the limit:** the remaining base runs and the whole `nocheck` arm failed within 2 s and are not results.

The A/B comparison must be rerun after the limit resets (`docs/bench/stage2-partial-rc156.md`).

## RC156 live benchmark findings (stage 1)

First live run: 22 tasks, one attempt each, gpt-6.1-sol through the ChatGPT sign-in.

| Measure | Result |
| --- | --- |
| Resolved | 21/22 |
| False completions | 1 |
| Budget stops | 4 |
| Average tool calls | 6.5 |
| Average time | 50 s |
| Tokens | 320,212 input (28,672 cached), 21,725 output |

**Four budget stops:** all said "compaction did not reduce context enough", within about a minute. Cause: the ChatGPT default window was a fixed 32,768, about 7% of the catalog's 500,000 for this model, which the byte-based estimator filled after roughly 10K real tokens. `goh config` now reports 500,000. `TestModelCatalogContextWindowParsed` covers the catalog field.

**The false completion** (`py-mutable-default`) was a task defect: the hidden test required not modifying a passed-in list, which the prompt never said. The model's `None`-default fix answered the prompt as written.

**Rerun on RC156:** 22/22 resolved, 0 false completions, 0 budget stops, 6.0 average tool calls, 42 s average, 312,011 input tokens (69,248 cached), 22,612 output. Every task ended `verified`. The suite is now at its ceiling for this model, so it cannot separate A/B arms (`docs/bench/stage1-rc156.md`).

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC155 TUI snapshots

`TestSnapshotsCommandListsRunChanges` checks two cases. With snapshots off, `/snapshots` explains the `project_snapshots` setting. After a real run that creates `made.txt` with snapshots on, it lists `run 1  1 file(s)` and `A made.txt`, along with the restore command for that session. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC154 expanded benchmark suite

`goh-bench --dry-run` reports all 22 tasks OK: each fails on its starting tree and passes, including pass-to-pass, with its reference solution. The dry run caught one broken fixture while the suite was being written (an unused import in a hidden Go test), which was fixed. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC153 benchmark runner

- `make bench-check` (dry run, no model) confirms all 8 tasks fail on the starting tree and pass with their reference solution, including pass-to-pass.
- `TestRunnerGradesCleanlyAndFlagsFalseCompletion` drives the real runner with stand-in `goh` scripts:
  - an agent applying the fix is resolved, with tool calls, tokens and cache counted;
  - an agent changing nothing but reporting `completed` is flagged as a false completion;
  - an agent rewriting the hidden test is graded against the real one and flagged `hidden_tests_overwritten`.

`bench/go.mod` keeps the task fixtures out of `./...`. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC152 headless run flags

End-to-end with the real `goh` binary and `scripted-model`:
- `goh run --allow all --verify "test -f done.txt"` creates `work.txt`, fails the check (exit 1), receives the failure, creates `done.txt`, passes (exit 0) and finishes with `"outcome":"verified"`;
- the same script with `--max-tool-calls 1` exits 5 with `budget_exhausted: run stopped at the max_tool_calls limit (1)`.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC151 MCP resources

`TestResourcesListAndRead` (with `-race`) checks four things:
- a server without resources gets no resource tools;
- a fixture server with a `fixture://docs/api` resource lists it;
- reading it returns `RESOURCE-TEXT-55` with an untrusted label;
- an unknown server is refused.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC150 Responses prompt cache key

- `TestResponsesPromptCacheKey` checks that there is no key without a session and that the session ID is sent when set.
- `TestResponsesDropsRejectedCacheKeyOnce` has a server answer the first request with a structured 400 naming `prompt_cache_key`; the second request omits it and the reply is parsed.

Whether the live ChatGPT endpoint accepts the field is **unverified** (the fallback covers rejection). Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC149 multi-language repository map

`TestRegexSymbolsAcrossLanguages` extracts the expected declarations from Python, TypeScript, Rust, Java, C and Ruby samples. The samples include traps: a commented-out `def`, a commented `function`, a C prototype and an `if (` call, none of which may be listed. Writing the test found a bug: a C keyword filter dropped Rust's `fn new`; the filter now applies only to C-like languages. Existing Go repo_map tests pass unchanged. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC148 masking window

`TestMaskingKeepsRecentWindow` fills a 200K window with 15 large reads and triggers masking. RC147 masked 14 and kept 1; RC148 masks 5 and keeps the newest 10. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC147 completion check and automatic messages

- `TestCompletionCheck` checks that an unchecked change gets exactly one reminder and ends `verified` after the model runs a check, while a run that already checked gets none.
- `TestAutomaticMessagesAreNotUserTurns` checks that only the real prompt counts as a turn.
- The rewind suite (5 tests) failed while reminders were counted as turns, which exposed the bug; it passes now.

Scripted runtime tests opt out of the reminder by default in `testAgent`. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC146 tolerant unique-match edits

`TestFlexibleReplace` checks four cases:
- a de-indented `old_text` still edits the right function and is re-indented with tabs, while a near-identical second function is untouched;
- an input that matches twice ignoring whitespace is refused, listing lines 1 and 2;
- a not-found error names line 2, where the first line appears;
- a file without a final newline keeps that state.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC145 shared run budgets

- `TestMaxRunSecondsBudget` checks that a `sleep 5` under a 1 s limit returns `budget_exhausted` within about 1 s.
- `TestChildTasksShareToolCallBudget` gives the parent a limit of 3 and a child that never stops. The test asserts the child really ran, and that parent plus child stayed within 3 calls.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC144 verify/repair, run budget, adversarial canary

- `TestVerifyRepairCycle`:
  - a failing `test -f done.txt` check is handed back once, the model fixes it, and the outcome is `verified`;
  - with `repair_attempts: 0` the outcome is `failed_verification` and nothing is handed back;
  - a model that never fixes it gets exactly 2 attempts, then `failed_verification`.
- `TestMaxToolCallsBudget` stops at `budget_exhausted` after 2 of 3 allowed calls when the next batch would exceed the limit.
- `TestInjectedInstructionsCannotExfiltrateOrPlant`: a planted `AGENTS.md` instructs exfiltration, and a scripted model with shell, webfetch and write grants obeys it. The test checks:
  - no `CANARY-TOKEN` or `CANARY-HTTP` reaches the model;
  - the local server gets 0 requests;
  - no git hook is created;
  - `CANARY-ENV` is blocked on macOS (it leaked there through the sandboxed shell before this release; the Linux limit is documented).

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC143 provider capabilities and guidance trust

- `TestReasoningEffortValidation` checks that OpenAI accepts `xhigh` and refuses `max`, and Anthropic accepts `low`/`high`/`xhigh`/`max` and refuses `minimal`.
- `TestAnthropicThinkingRoundTrip` now requires stored thinking blocks (signature `sig-abc`) to be replayed even without a thinking setting, and checks that adaptive mode encodes `{"type":"adaptive","display":"summarized"}` with `output_config.effort`.
- `TestGuidanceFilesLoaded` checks that user guidance stays in the system message, project guidance is absent from it and arrives as the next, untrusted user message, and `AGENTS.md` still takes precedence over `CLAUDE.md`.

Live behaviour against current models was **not** tested (no live keys). Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC142 constraints and memory replay

- `TestUserConstraintsSurviveRepeatedCompaction` **fails on RC141**, where the constraint is lost after the first compaction. It now finds `CONSTRAINT-42` verbatim after two compactions.
- `TestRecalledNotesAreRevalidatedOnReplay` recalls a note tied to `config.go`, changes the file mid-run, and checks the next model request contains no `NOTE-777` and reports it withdrawn.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC141 scoped session grants

`TestSessionGrantsAreScopedToTheReviewedAction` approves `go test ./...` for the session and then checks which later requests ask again:

| Request | Asks again? |
| --- | --- |
| `go test ./internal/x` | No |
| `rm -rf build` | Yes |
| `go test ./...; curl evil` | Yes |
| a second `docs.example.com` URL | No |
| `other.example.com` | Yes |
| a second `edit_file` | No (tool-wide as before) |

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC140 run outcomes and MCP oversized results

`TestRunOutcomes` covers three cases plus an empty reply:
- a run with no edits records `no_changes`;
- a `create_file` alone records `unverified`;
- `create_file` followed by a passing `test -f` records `verified`;
- an empty assistant reply no longer completes.

`TestBoundedResults` now requires an oversized MCP result to come back non-error and bounded, with status `completed_output_truncated` and "Do not repeat". Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC139 usage attribution

`TestFallbackUsageAttributedToServingModel` fails the primary route with a 503 and serves the reply from the `backup` profile with 800 cache-read and 100 cache-write tokens. It checks:
- the journaled usage names `backup-model` (RC138 named the primary);
- `CollectStats` keeps the cache split;
- the cost estimate prices fresh, cached-read, cached-write and output tokens separately.

Usage recording moved into the single routing function every model call uses, so summary and title calls are counted with their purpose. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC138 recovery and context fixes

Two tests fail on RC137 and pass on RC138:
- `TestRepairUsesJournaledResults` journals a finished shell result and a never-started call, then resumes. RC137 replaced the recorded `BUILD-OK-7731` result with "outcome may be uncertain". RC138 reuses it exactly and marks the other call "Not executed".
- `TestLargeEarlyResultDoesNotExhaustBudget` reads a ~45 KB file in a 32K window. RC137 returned `budget_exhausted` (62,047 needed, 20,480 available). RC138 gives the model an excerpt with `start_line` paging advice and the run completes.

`TestMaskReceiptsKeepFactsAndDiscourageRepeats` checks that a masked shell result keeps its exit code and artifact ID and says "do not run it again", while a masked read keeps its SHA-256 and is marked re-runnable. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC137 external review P0 fixes (paths, web, MCP)

- `TestWritesIntoGitMetadataAreRefused` **failed on RC136**: a granted `create_file` planted `.git/hooks/pre-commit`. It now passes for that case and for a nested `sub/.git/config` patch.
- `TestFileRulesDenyCoversReadsSearchAndMentions` puts synthetic secrets in a denied `.env` and `secrets/`, then sends `read_file`, `grep`, `glob`, `list_files` and `@`-mentions. No secret text reaches the model, and allowed files still appear. RC136 had no deny check on these paths (confirmed by reading the code; the test needs the new field, so it cannot compile there).
- `TestDefaultClientRefusesNonPublicAddresses` checks that a loopback test server is refused by URL and via `localhost`, and reachable with the opt-in client. Ten private, loopback, link-local, CGNAT and unspecified IPs are classified non-public, and two public ones public.
- `TestDuplicateKeysRefusedAndValidatedObjectSent` refuses `{"text":"safe","text":"rm -rf /"}` before the server sees it, along with nested duplicates.

Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC136 external review P0 fixes

`review_p0_test.go` reproduces three reported failures, and each test **fails on the RC135 code** (checked by stashing the fix):
- `TestHooksOnParallelReadsAreRaceFree` reports 4 data races there;
- `TestNoExecutionAfterHaltInBatch` shows a shell marker created after three invalid calls halted the run;
- `TestRepeatGuardCoversBatchedReads` shows four identical reads, one with reordered keys, running unchecked.

All three pass with the fix. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the setup and ACP suites. The PTY suite failed once under load straight after the race suite, then passed on three reruns; it is noted as possibly flaky.

## RC135 MCP reconnect

`TestReconnectAfterServerDropsWithoutReplay` (with `-race`) uses the fixture MCP server's `disconnect` tool, which kills the server mid-call. For three rounds it checks:
- the dropped call is reported as "not replayed";
- the next `echo` call restarts the server and succeeds;
- each restart went through approval.

After a fourth drop the call fails with "could not be restarted", so restarts are bounded. Existing cancellation, disconnect, remote and failed-server tests still pass. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC134 shell sandbox

`TestSandboxConfinesWritesReadsAndNetwork` runs a real shell through the sandbox with a project, a protected secret folder and a sibling folder:
- the project write succeeds;
- the sibling write fails and no file appears;
- the secret cannot be read, but a normal file outside can;
- a `curl` to a local test server is blocked, then succeeds with `network: true`;
- on unsupported systems `Wrap` returns `ErrUnsupported`.

`TestShellToolRunsInsideSandbox` repeats the write check through the real shell tool. On Linux both tests re-run the test binary as the Landlock helper via `TestMain`. CI sets `GOH_REQUIRE_SANDBOX=1` on Linux and macOS so a missing sandbox fails rather than skips. The tests passed on ubuntu-latest (Landlock with network rules) and macos-latest (sandbox-exec), and Windows confirms the refusal. Gates passed: gofmt, vet (macOS, Linux and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC133 whole-project snapshots

`TestTrackDiffRestoreLeavesUserRepoAlone` uses a real Git repository with a `.gitignore`, a committed base and an uncommitted edit. It simulates shell changes (edit, delete, create in a new folder, and an ignored build output), then checks:
- the diff is exactly `M a.txt, A new/c.txt, D src/b.txt` and the patch contains the new line;
- the restore brings back the uncommitted edit and the deleted file, removes the new file and its empty folder, and leaves the ignored file alone;
- restoring the returned "before" tree redoes the changes;
- the user's HEAD and staged state are unchanged;
- invalid or unknown tree IDs are refused, and so is storage inside the project.

`TestProjectSnapshotsRevertShellChanges` runs a real agent run whose approved shell command clobbers a file and creates a folder. `goh snapshots list` reports both and `diff` shows the change. `restore` without `--yes` changes nothing, `restore --yes` reverts, `undo --yes` brings the run's changes back, and an unknown run number is refused. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC132 hooks

Windows CI follow-up to RC130: `TestParentChildCancellationUnderLoad` timed out because PowerShell children rebuilt their module cache without `PSModuleAnalysisCachePath`. The standard Windows system variables and the PowerShell cache and telemetry settings are now in the safe set. A temporary CI diagnostic confirmed PowerShell starts with the filtered environment while `PGPASSWORD` and the runner variables are withheld. CI is green on Linux, macOS and Windows.

`TestHooksBlockAddFeedbackAndNotify` drives a real agent run with shell scripts as hooks. A `sh*` pre-tool hook refuses `rm -rf`, and the model receives the hook's stderr. A second shell call passes and a post-tool hook's output arrives as `hook_feedback`. A `read_file` hook that was never triggered does not run. The `run_finished` hook receives `{"event":"run_finished","status":"completed"}`. The hooks saw `GOH_TOOL=shell` but not `OPENAI_API_KEY`. `TestHooksCannotGrantAndFailClosed` checks that without a shell grant an allowing hook neither runs nor lets the command run (`needs_permission`), and that a hook exiting 1 stops an approved tool. `TestValidateHooks` rejects unknown events, missing commands, timeouts over 120 s, `tools` on `run_finished`, and bad patterns. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC131 Anthropic prompt caching

`TestEncodeAnthropicPromptCacheBreakpoints` encodes two tools, two system blocks and a tool round trip, then checks the last tool, the last system block and the final `tool_result` carry `cache_control`, earlier ones do not, and there are exactly 3 breakpoints (the API allows 4). It also checks `disable_prompt_cache` removes every marker. Prefix stability was reviewed: the system prompt is fixed at session start (no timestamps) and MCP tool order is sorted. Cache hit rates against the live API are **not** measured (no live keys); `cache_read_input_tokens` is already parsed into usage. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC130 child-process environment

`TestFilterEnv` checks the default set keeps PATH, HOME, `LC_*`, `XDG_*`, GOPATH and Windows `Path`, and drops `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, `DATABASE_URL`, a credential-bearing `HTTPS_PROXY` and malformed entries. It also checks that `env_passthrough` exact and `PREFIX_*` names are added and that `*` passes everything. `TestShellDoesNotInheritSecrets` runs `env` through the real shell tool with `OPENAI_API_KEY` and `GITHUB_TOKEN` set: neither value appears, while PATH works and a passthrough variable is visible. MCP and LSP servers use the same `platform.ChildEnvMap`, and their existing tests (including the real clangd run) pass. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC129 repository restructure

Moves only, plus the module path rename. `go.mod` and `go.sum` are byte-identical apart from the module line. `research/go.mod` keeps `./...` out of local reference material. Gates passed from the new root: gofmt, vet (macOS and Windows), race suite, acceptance, the PTY, setup and ACP suites, `make release` (all six targets plus documents) and `scripts/test_install.sh` (clean install, tampered download refused, non-running binary refused). All relative Markdown links in the root and `docs/` were checked to resolve.

## RC128 git worktree

`TestRunWorktree` builds a real Git repository in a temp folder, rejects unsafe names (`a/b`, `../x`, `-x`, `a..b`), creates `repo-exp` with the committed file present and the UI opened there, reuses it on a second call, and refuses a plain folder that is not a worktree. Gates passed: gofmt, vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC127 sessions list --project

`TestSessionsListProjectFilter` creates sessions for two projects and checks `list --format ids --project A` prints exactly A's ID. `goh stats --project` shares the same path resolution and its tests still pass. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC126 completion of subcommands and session IDs

`TestCompletionScripts` runs the bash script in a real bash with a stand-in `goh` that prints two IDs: `goh auth l` → `login logout list`, `goh resume 2026110` → both IDs, `goh sessions show 20261102` → the matching ID, plus the earlier top-level checks; the zsh script still passes `zsh -n`. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC125 sessions prune

`TestSessionsPruneListsThenTrashes`: with a 40-day-old and a 1-hour-old session and a 30-day cutoff, the listing names only the old one and removes nothing; `--yes` trashes only the old one; it is then restored; `prune` without `--days` is refused. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC124 tool statistics

`TestCollectStatsCountsTools` journals four tool calls (one failed) and checks the counts, failure attribution by call ID and ordering (most calls first, then name). Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC123 piped run input

`TestPipedInputDetection` accepts a pipe and a redirected file and rejects `/dev/null`. End to end, `echo "PIPED-LOG-LINE 42" | goh run explain this` (unreachable local endpoint, temporary session folder) journalled the message `explain this\n\nPIPED-LOG-LINE 42`. The acceptance, PTY, setup and ACP suites, which start goh with non-terminal stdin, all still pass without hanging. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC122 goh config

`TestConfigShowRemovesSecrets` puts secrets in a URL's user info and query, an MCP environment value and a formatter argument; none appear in the output, while the host and path, the variable name and the model remain, and the output is valid JSON. Run against the real configuration it printed the ChatGPT default model and no secrets. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC121 command agent

`TestCommandAgentDelegatesLikeAMention`: `/scan internal` with `"agent": "explore"` reaches the model as `@explore Scan internal for dead code` plus the delegation note; a command naming an unknown subagent fails before the provider is called. Repository command files are read by `commands.Discover`, which has no agent field, so they cannot set it. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC120 goh uninstall

`TestUninstallListsThenRemovesOnlyOwnFolders` (temporary home): the target list is the program, its `harness` link and the two GoHarness folders; `--keep-data` drops the folders; a program not named goh is never listed; listing removes nothing; `--yes` removes both folders and leaves a neighbouring tool's folder. The built binary was also copied into a temporary prefix with a temporary HOME: the listing, then `--yes`, removed exactly the program and the two folders. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC119 goh auth list

`TestAuthListNamesSourcesNotSecrets` sets an official key and a gateway key and checks: neither value appears in the output; the official route reports `environment OPENAI_API_KEY`; the gateway profile reports `environment GW_KEY`; a third-party profile without its own variable reports `none` (the OpenAI key is not offered to it); ChatGPT reports not signed in. Run against the real configuration, it printed one row, `chatgpt  chatgpt.com  ChatGPT sign-in`. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC118 Ctrl+Z suspend

`TestCtrlZSuspends` checks the default binding, that the action sends Bubble Tea's suspend message, and that `"suspend": "none"` frees the key. A real check ran the built `goh` inside an interactive bash under a PTY: Ctrl+Z left it listed by `jobs` as `Stopped`, and `fg` resumed it and redrew the screen. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC117 goh pr

`TestRunPR` uses a fake `gh` on PATH: malformed references, extra arguments, flag-looking input and non-GitHub URLs are refused before gh runs; `#42` runs exactly `gh pr checkout 42` and then opens the UI; a failed checkout does not open the UI; a missing gh gives an install hint. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC116 shell completion

`TestCompletionScripts` checks every command appears in all three scripts, rejects an unknown shell, runs the bash function in a real bash (`goh se` → `sessions setup settings serve`; `goh completion z` → `zsh`) and syntax-checks the zsh script with `zsh -n`. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC115 formatter auto-detection

`TestWithDetectedFormatters` checks: nothing is added when the option is off; installed gofmt and prettier are added; ruff is skipped because a configured formatter owns `.py`; a disabled configured shfmt is kept; rustfmt (not on PATH) is skipped; the input config is not mutated; and the result passes formatter validation. RC114's Windows CI failure (`TestChatGPTDefaultNeverOverridesExplicitProvider`, which read the runner's real home) is fixed in `d95f30c`. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC114 HTML from /export

`TestExportFormatByExtension` covers the extension choice, including upper case and names without an extension. The setup suite's Markdown export check still passes. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC113 HTML export

`TestHTMLTranscriptEscapesAndRedacts` checks that a `<script>` prompt is escaped, an OpenAI-style key is masked, ampersands are escaped, and the restrictive Content-Security-Policy is present. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC112 run --fork

A scripted-model run made a session, and a second `--session ID --fork` run then:
- reported the fork;
- left the original journal byte-for-byte unchanged (same checksum);
- produced a second journal.

`--fork` without a session is refused. Forking itself is the tested `session.Fork` path. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC111 sidebar cost

`TestSidebarCostFromConfiguredPrice` checks the priced estimate (0.5M input tokens at $2 plus 0.125M output tokens at $8 is $2.00) and that nothing is shown without a price. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC110 cost estimates

`TestCostEstimates` checks the estimate arithmetic (2M input tokens at $1.50 plus 0.5M output tokens at $6 is $6.00), that unpriced models have no cost, the total line, and the configuration hint. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC109 grep ignore rules

`TestGrepRespectsIgnoreFiles` covers these cases:
- a root `.gitignore` directory rule and glob rule;
- a nested `.ignore`;
- explicitly searching an ignored directory.

The existing glob test, which expects gitignored files to be included as the reference does, is unchanged and passes. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC108 file mentions

`TestFileMentionsInlineTextFiles` checks that a text file is attached once and that these are not:
- binaries, directories and missing files;
- a symlink pointing outside the project;
- agent names and images.

It also checks that the original prompt text is unchanged. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC107 command suggestions

`TestUnknownCommandSuggestsNearest` covers a typo (suggestion within edit distance 2) and an unrelated name (no suggestion, but still the `/commands` pointer). Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC106 skill diagnostics

`TestSkillDiagnosticsNoticeOnce` checks that the notice appears once and that no raw JSON is written. `/status` lists `Skills.Diagnostics()`. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC105 provider cooldowns

`TestFailedPrimaryCoolsDownWhileFallbacksExist` uses an injected clock to check:
- the primary is skipped during the cooldown;
- it is tried last when every fallback fails;
- it is tried first again after the minute.

The existing failover tests are unchanged and pass. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC104 retry visibility

- `TestRetryAndNoPartialReplay` now also checks that two retry notices arrive (attempts 2 and 3 of 3, with a positive delay) before the third request succeeds.
- `TestRetryNoticeInStatus` checks the TUI status wording.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC103 damaged sessions in the picker

`TestChooserShowsDamagedSessionsForRecovery` checks that damaged entries pass the project filter and are labelled. Choosing one opens the session through `chat --session`, which reaches the RC102 `offerSalvage` path. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC102 guided salvage

- `TestOfferSalvage` damages a real journal and checks the decline path (error and hint kept) and the accept path (new session, success message).
- It also checks that the damaged journal is byte-for-byte unchanged.
- CI for RC101 is green on all three platforms, including the real clangd test on Windows.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC101 slow diagnostics

- Windows CI failed RC99's `TestRealClangd`: clangd needed more than 2 seconds to publish diagnostics, and the server was then marked broken.
- New fixture mode `silent-diagnostics`. `TestLSPSlowDiagnosticsKeepServer` and `TestLSPDiagnosticsTimeoutKeepsServer` check that the server stays usable after a pending result or a caller timeout.
- The race detector found a concurrent-writer bug in the first version of the fix. The waiting goroutine is now drained before the server is reused, and the race suite passed three times.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC100 language server restart

`TestRealClangd` now kills the running clangd mid-run. The next request fails without replay, the following one restarts the server behind exactly one more approval, and the restarted server answers go-to-definition. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC99 real language server

`TestRealClangd` passed locally against `/usr/bin/clangd`:
- the server started after one approval;
- the `-Wint-conversion` error arrived with severity 1 on line 2;
- `EditErrors` returned it for the project-relative path the edit tools use;
- `goToDefinition` resolved to line 1.

While writing the test, an absolute temp path under macOS `/var` returned no edit errors. This does not affect GoHarness, because the edit tools refuse absolute paths. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC98 language server detection

`TestWithDetectedLSP` covers the default-off case, detection, configured extensions taking precedence, missing binaries and an unchanged input map. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites. Real language servers have still not been exercised end to end.

## RC97 compact --focus

The flag passes through the same `CompactWith` path that RC96 tests; `goh compact --help` lists it. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC96 compaction focus

`TestCompactWithFocus` checks that the focus reaches the summarizer instructions and is cleared afterwards. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC95 ChatGPT 401 renewal

- The ChatGPT authorization test now covers forced renewal of an unexpired token, and that a stale rejection reuses the already renewed token.
- The provider retry (401, then renew, then resend) is not unit-tested at the HTTP level because the endpoint is bound to the official host. It has not been exercised live.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC94 sidebar title

`TestSidebarShowsSessionTitle` checks that an automatic title appears in the sidebar and is terminal-safe. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC93 automatic titles

- `TestAutoTitleAfterFirstRun` covers the request content, cleaning of quotes, extra lines and trailing punctuation, no renaming on later runs, and that a user title is never replaced.
- `TestAutoTitleOffByDefault` checks that no extra request is made by default.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC92 ChatGPT default everywhere

- All entry points share `withChatGPTDefault`, which RC91 tests.
- The ACP suite passes unchanged.
- `goh models` against the saved sign-in was not run, since it would make a live request.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC91 ChatGPT default for run and doctor

- `TestChatGPTDefaultNeverOverridesExplicitProvider` checks that an explicit endpoint and model are kept, and that ChatGPT is not chosen without a saved sign-in.
- A manual `goh doctor` on this machine shows the saved sign-in and a valid configuration (`gpt-6.1-sol`, 32k), with no network requests.
- `goh version` prints `1.0.0-rc.91`.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC90 agent completion

`TestCompletionOffersAgentsBeforeFiles` checks ordering and that agents are matched by prefix only. The setup suite's path-only file-completion check still passes. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC89 agent mentions

`TestAgentMentions` covers known and custom subagents, punctuation, unknown names, email-like text, file mentions, and that child agents get no note. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC88 session table

`TestSessionTable` covers saved titles, terminal-safe output and the empty case. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC87 reasoning-effort flag

- The acceptance `stdin-words` scenario passes `--reasoning-effort off`.
- A manual run confirmed that an invalid level is refused before any request.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC86 effort command

- `TestSetReasoningEffort` checks validation, provider rebuild and turning effort off.
- `TestEffortCommand` covers case handling, refusal of invalid levels, refusal during a run, and off.
- Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC85 export and import

`TestExportAliasDefaultsToLatestSession` exports this directory's latest session as Markdown and checks the error when there is no session. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC84 MCP list

`TestWriteMCPHidesSecrets` checks that env values, URL user info, query strings and fragments never appear, and that unset header variables are flagged. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC83 agents list

`TestWriteAgents` covers ordering, defaults, terminal-safe output and the skipped-file report. The usage test confirms that `agents` appears in the help. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites. CI is green on all three platforms after the load-test hardening.

## RC82 review command

`TestReviewBuiltin` checks the argument and default forms. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC81 ACP reasoning

`TestReasoningBecomesThoughtChunk` checks the ACP update. Gates passed: vet (macOS and Windows), race suite, acceptance, and the PTY, setup and ACP suites.

## RC80 reasoning display

- `TestResponsesReasoningSummaryStreamsToSink` covers Responses summary streaming.
- The Anthropic round-trip test now checks the thinking stream.
- `TestReasoningBlockAndToggle` checks block ordering before the answer, that NUL markers cannot be forged, and hiding with `/thinking`.
- Gates passed: vet (macOS and Windows), race suite, acceptance, the PTY suite and the setup suite.
- Live acceptance of `reasoning.summary` was not run (no live keys in this loop).

## RC79 command help

`TestUsageListsDispatchedCommands` compares the help with the dispatcher. `scripts/test_setup.py` had been failing since RC59 because it expected an old dialog title. It is fixed, passes, and now runs with the gates. Gates passed: vet (macOS and Windows), race suite, acceptance, the PTY suite and the setup suite.

## RC78 run --command

`TestCommandPrompt` covers the name forms and refusals. Expansion itself is the existing runtime path covered by the command tests. Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC77 run --title

The acceptance `stdin-words` scenario now passes `--title` and checks the saved metadata. Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC76 run --file

`TestAttachFiles` covers inline text, image mentions, absolute in-project paths and refusal of `../`, binary and missing files. Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC75 transcript redaction

`TestRedactSecrets` covers each pattern, prefix retention and a non-match (`sk-short`). Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC74 exit command

`TestExitCommandQuitsWhenIdle` checks that `/quit` quits when idle. Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC73 session rename

`TestRenameCommandSavesAndResetsTitle` checks that the title is cleaned of control characters, saved, and reset. Gates passed: vet (macOS and Windows), race suite, acceptance and the PTY suite.

## RC72 live ChatGPT verification

With the user's approval, live checks ran against their ChatGPT account
(model `gpt-6.1-sol`) through the official Sign in with ChatGPT flow.

- The first `scripts/test_openai_live.py` run failed: resuming with a 32k
  window ended `budget_exhausted` because the fixed 16k compaction margin
  plus ~9 KB of tool definitions (estimated at one token per byte) left too
  little room. The margin is now `min(16k, window/4)`; the run then passed:
  streamed text, permission-gated read/edit/verify, durable resume and TUI
  connect-and-send.
- New `scripts/test_openai_live_extra.py` passed all four checks: an
  attached PNG was identified as red; `reasoning_effort: low` was accepted;
  a turn using `todowrite` then reading a file completed with task state
  sent at the end of the request; the plan agent called `plan_exit` after
  investigating.
- Not measured: prompt-cache hit rates (usage reports no cache fields on
  this endpoint) and long-run cost.

Local gates after the fix: vet (macOS and Windows), race suite, eight CLI
and twenty-five PTY scenarios.

## RC71 web search

`1.0.0-rc.71` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-five PTY scenarios. A test against a local SearXNG-shaped server
checks the query and `format=json` parameters, whitespace-cleaned snippets,
dropped non-http(s) links, the result limit, refused redirects, argument
validation, the approval preview and URL rules. Runtime tests check the tool
exists only with `websearch_url` and always needs approval. No public search
service was contacted.

## RC70 Anthropic extended thinking

`1.0.0-rc.70` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-five PTY scenarios. A parser test streams a thinking block (text and
signature in deltas), a redacted thinking block and a tool call; the encoder
then sends `thinking: {type: enabled, budget_tokens}` and replays both
blocks verbatim ahead of the tool call, and omits them when thinking is off.
Chat Completions wire messages never carry them. Validation limits the
budget to Anthropic, at least 1024 and below `output_budget`. Live API
acceptance is unverified.

## RC69 fuzzy pickers

`1.0.0-rc.69` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-five PTY scenarios (which type filters into the palette and pickers).
Tests check in-order letter matching, out-of-order rejection, exact
substrings ranking first, and an empty query keeping the original order.

## RC68 more LSP operations

`1.0.0-rc.68` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-five PTY scenarios. The fixture language server now advertises type
definitions and call hierarchy; the LSP test runs all eleven operations
against it and checks incoming and outgoing calls use the item returned by
`prepareCallHierarchy`. Behavior against real language servers is untested.

## RC67 plan handoff

`1.0.0-rc.67` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-five PTY scenarios (two consecutive runs). The new `plan_exit`
scenario starts in plan mode, approves the scripted `plan_exit`, and checks
the chat restarts in Build in the same session with the go-ahead in the
composer. Runtime tests: the approval shows the plan; approval records the
handoff, a decline does not; the tool is absent and refused in build mode
and for subagents; a headless run stops with `needs_permission`.

## RC66 orphaned background jobs

`1.0.0-rc.66` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios. A Unix test starts a real job, rewrites its
registry so the owner is an exited process (simulating a hard kill), and
checks a second engine lists the orphan, refuses to stop it when the
recorded command does not match (process left running), stops it when it
matches (process group gone) and prunes the registry. Windows builds and
vets; orphan stopping there is refused by design.

## Windows cancellation under load

`TestParentChildCancellationUnderLoad` (six concurrent parents cancelled
while each child runs a long shell command) now also builds for Windows,
using a PowerShell command that writes its PID and `tasklist` to confirm the
process is gone. Native CI for `555fae0` passed all six jobs, including this
test on Windows: every child process ended within the deadline and each
interrupted effect stayed pending.

## RC65 compaction threshold

`1.0.0-rc.65` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios. A runtime test fills about 60% of a 100k window
and checks no summary is requested at the default 75% but one is at 55%;
validation rejects values outside 50–95.

## RC64 tool details toggle

`1.0.0-rc.64` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios (two consecutive runs). Tests check full and
summarized output, that marker-like content cannot forge either form, and
that `/find` sees each line once and only what is displayed.

## RC63 file edit rules

`1.0.0-rc.63` passed vet (macOS and Windows), the race suite, eight CLI
scenarios and twenty-four PTY scenarios (three consecutive runs after the
harness change below). Decision tests cover deny winning across paths,
allow requiring every path, ask, nested paths and no rules. Runtime tests
check denials without prompting for `create_file`, `child:create_file` and a
two-file `apply_patch`, auto-allow for an allowed path, and an `ask` rule
prompting despite a session grant.

PTY runs failed twice in the undo scenarios waiting for `fixture complete.`
on screen (split redraw); those waits now also accept the journaled reply.
CI for `ad25fff` failed on Windows because the ACP test used `/usr/bin/tool`,
which is not absolute there; fixed in `18d2531`.

## RC62 context meter

`1.0.0-rc.62` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios. A TUI test feeds two usage events and checks the
meter uses the latest reply (50% of 200k), the running total still sums
both, the sidebar shows the line, nothing appears before usage, and the
percentage caps at 100.

## RC61 ACP remote MCP servers

`1.0.0-rc.61` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios, and `scripts/test_acp.py`. New ACP unit tests
cover merging an HTTP and a stdio client server with configured ones, and
refusing a missing URL, invalid header names, header values with line
breaks, unknown transport types, duplicates and relative commands. A remote
MCP test checks a literal client header reaches the server but not the
approval preview.

## RC60 usage statistics

`1.0.0-rc.60` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios. `CollectStats` is tested across two projects and
ages with the project and day filters, per-model totals ordered by tokens,
and negative counts ignored. Two runs of the built binary against the
scripted model produced `2 replies, 20 in, 10 out, 30 total` in text and
JSON output.

## RC59 provider models in /models

`1.0.0-rc.59` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-four PTY scenarios (two consecutive runs). The new
`provider_models` scenario opens `/models`, filters to `scripted-large` from
the scripted provider's list, sends a prompt, and checks the journaled run
used `scripted-large`.

## RC58 provider model list

`1.0.0-rc.58` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-three PTY scenarios. Provider tests check the derived `/v1/models`
URL for all three endpoint shapes, the bearer key from the configured
variable, sorted de-duplicated IDs, a refused redirect and the ChatGPT
refusal. The scripted model now serves `/v1/models`, and the built binary
listed its two models. CI for `5fc2e3c` passed all six jobs, including the
Windows jobs test.

## Windows jobs test cleanup

Native CI for `4a9785a` failed on Windows only: `TestJobsPanelShowsOutputAndStops`
passed its checks, but removing the test project folder failed because
Windows still held a handle from the just-killed `ping` job. The test now
stops jobs first and removes the folder with retries. Product behavior is
unchanged; the same test passed on earlier Windows runs.

## RC57 remote MCP servers

`1.0.0-rc.57` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-three PTY scenarios. A test runs an in-process MCP SDK server over
HTTP: the manager connects after approval, discovers and calls a tool, sends
the `Authorization` header read from an environment variable, never shows
its value in the approval preview, and closes cleanly. Refusal tests cover
non-loopback plain HTTP, other schemes, embedded credentials, fragments, a
missing header variable, `command` and `url` together, and a redirecting
server. No third-party remote MCP service was contacted.

## RC56 default checkpoints

`1.0.0-rc.56` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-three PTY scenarios. Config tests: no `snapshots` key means on and
best effort, `true` means strict, `false` means off, and `Defaults()` stays
off for tests. A runtime test makes checkpoint storage unusable and checks
that a default edit succeeds with `checkpoint_skipped` while a strict edit is
refused with the file unchanged. Writing that test found that a skipped
capture still attempted to seal an empty checkpoint and reported the edit
as interrupted; fixed.

## RC55 changes panel

`1.0.0-rc.55` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-three PTY scenarios. A TUI test runs a real agent turn that creates a
file with checkpoints on, then checks the `/changes` list, the diff panel
(`made.txt · added` with the new content), the explanation when checkpoints
are off, and refusal during a run.

## RC54 continue the last session

`1.0.0-rc.54` passed vet (macOS and Windows), the race suite, eight CLI and
twenty-three PTY scenarios. `session.Latest` is tested with sessions of
different ages across two projects. A manual run of the built binary against
the scripted model sent `first prompt`, then `--continue second prompt`,
and both landed in one journal; `--continue` with `--session` and
`--continue` in a project without sessions were refused with messages.

## RC53 run with words and piped input

`1.0.0-rc.53` passed vet (macOS and Windows), the race suite, eight CLI
scenarios (new `stdin-words`: piped input, message words and flags
interleaved) and twenty-three PTY scenarios. Unit tests cover word joining,
stdin before and after words with blank-line separation, a second `-`,
empty and oversized input. A manual run against the scripted model
journaled `say hello` and `piped context\n\nsummarize` as the prompts.

## RC52 agent files everywhere

`1.0.0-rc.52` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests check skipped agent files are recorded on
the configuration and shown on `/status`. Serve and ACP use the same
`WithProjectAgents` merge; there is no dedicated serve/ACP agent-file test.

## RC51 project agent files

`1.0.0-rc.51` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests cover frontmatter description and plan
mode, default build mode, an ignored `model` key, folder precedence for
duplicates, reserved names, empty bodies, configured agents winning, and
resolving a file-defined agent into plan mode with its instructions. Agent
files load for `goh`/`goh chat`; `goh serve` and ACP still use configured
agents only, and skipped files are not yet shown on `/status`.

## RC50 image attachments

`1.0.0-rc.50` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Runtime tests: an `@shot.png` mention reaches
the model as an image with the prompt text unchanged, the context estimate
counts it as a fixed cost rather than its base64 length, and summaries get a
path note instead of image data. Refusals: content not matching the
extension, over 5 MiB, outside the project, symlinked (Unix), missing, more
than four, and a failed run before any model call. Encoder tests check
`image_url` (Chat Completions), `input_image` (Responses) and base64 `image`
blocks (Anthropic), and that messages without images are unchanged. Live
vision acceptance is unverified.

## RC49 reasoning effort

`1.0.0-rc.49` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests: Chat Completions sends
`reasoning_effort`, Responses sends `reasoning.effort` only when configured,
validation accepts the four levels and refuses others and Anthropic.
Live acceptance of the parameter by OpenAI models is unverified.

## RC48 job visibility

`1.0.0-rc.48` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. The jobs TUI test now checks that an exited job
is not counted, a running one appears in the sidebar count and in the
`/status` section, using `ping` instead of `sleep` on Windows.

## RC47 /init

`1.0.0-rc.47` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests: `/init` expands with arguments, a user
command replaces it, and built-ins never push the command table past 256.
RC46 native CI (`b4013fc`) passed all six jobs.

## RC46 shell command rules

`1.0.0-rc.46` passed vet (macOS and Windows), the race suite, seven CLI
scenarios and twenty-three PTY scenarios (three consecutive runs after the
harness change below). Matcher tests cover allow and deny precedence, a later
allow carving an exception from a deny, and bypass attempts through `;`,
`&&`, pipes, backticks, `$(…)`, redirects, newlines and subshells. Runtime
tests check that rules decide without prompting and are journaled with
source `shell_rules`, that `ask` prompts even with a session grant, that
`job_start` and `child:shell` follow the rules, and that compound commands
fall back to normal approval. Known limit (documented): patterns match text,
so `rm *` misses `FOO=1 rm` or `/bin/rm`.

One PTY run failed in `editor_handoff` because a redraw split the reply on
screen. All scenarios that wait for a model reply now accept it from the
screen or the journal (`wait_reply`).

## RC45 loop guard

`1.0.0-rc.45` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. With a model that repeats one `read_file` call:
a headless run executes it twice, then stops with `needs_permission`;
declining blocks every later repeat while the run completes; a session
approval asks once and allows all repeats. Changing arguments resets the
count.

## RC44 instruction files

`1.0.0-rc.44` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. Tests isolate the user config folder and check
that user and project guidance both reach the system message, that
`AGENTS.md` replaces `CLAUDE.md`, that project text stays labelled untrusted,
and that an oversized personal file is skipped. RC42 native CI (`cd052cd`)
passed all six jobs, including background job tests on Windows.

## RC43 commented configuration

`1.0.0-rc.43` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios. JSONC tests cover line and block comments,
comment-like text and escapes inside strings, `//` in URLs, trailing commas
in objects and arrays, unterminated comments, preserved line numbers and
continued rejection of unknown fields. A new fuzz target
(`FuzzStripJSONCIdentityOnJSON`, 2.4 million executions in 10 s locally)
checks that valid JSON passes through byte-for-byte; it is in `make fuzz`.

## RC42 background jobs

`1.0.0-rc.42` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-three PTY scenarios (two consecutive runs). The new `background_job`
scenario approves a scripted `job_start`, confirms the job's grandchild
process is alive while the chat is open and gone after GoHarness exits.
Tool tests cover output capture and exit codes, stopping a job together with
a grandchild (Unix), the eight-job limit, `StopJobs`, argument and
directory validation, and that only `job_output` skips approval. Policy
tests: plan mode may read but not start or stop jobs; subagents cannot
start them. A TUI test lists a job, shows its output under its own title,
keeps `r` from replacing it and stops the job. Windows job stopping is
covered only by vet and the shared process-group code, not a native test.
A hard kill of GoHarness (SIGKILL) cannot run cleanup and may orphan jobs.

## RC41 typed shell commands

`1.0.0-rc.41` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-two PTY scenarios (two consecutive runs). The new `user_shell`
scenario types `!echo pty-shell-$((40+2))` and checks the shell-evaluated
output, a `user_typed` permission record, a journaled tool result and that
no model reply was requested. Runtime tests check the user/call/result
history, that the next turn's model input contains the output, that no
session grant is created, and refusals for empty commands and plan mode.

## RC40 command files and find highlighting

`1.0.0-rc.40` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). Discovery tests cover
frontmatter, CRLF, reserved and invalid names, empty and unterminated
frontmatter, oversized files, non-Markdown files, subfolders, symlinks and
duplicate names across folders; merge tests cover config precedence and the
256-command limit. A runtime test expands a project file command into the
exact prompt the model receives. Find tests check the marked match row,
that non-matches stay unmarked and that marks clear on Esc. A first race run
failed because file commands were merged once at startup, missing commands
configured later; they are now merged at use.

## RC39 transcript search and CLI revert

`1.0.0-rc.39` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). Find tests use a transcript
whose matches start off screen and check on the rendered view that the
newest match, then an older one 400 lines up, become visible; that keys wrap
between matches, typing never edits the draft, and Esc returns to the
bottom. `RevertTurn` rejects turn 0 and nonexistent turns and reverts turn 1
of 2 with both files restored.

## RC38 revert to an earlier turn

`1.0.0-rc.38` passed vet (macOS and Windows), the race suite, seven CLI and
twenty-one PTY scenarios (two consecutive runs). The new `timeline_revert`
scenario creates a file in a turn, opens the timeline, reverts with Ctrl+U
and checks the file is gone and the prompt is back. Rewind tests revert turn
2 of 3 (files from turns 2 and 3 removed, turn 1's kept, branch ends before
turn 2), redo restores both turns, an unknown prompt is refused without
changes, and repeated prompts select the right occurrence. Undo PTY
scenarios now use per-case session folders.

## RC37 task-state placement

`1.0.0-rc.37` passed vet (macOS and Windows), the race suite, seven CLI
scenarios and twenty PTY scenarios (three consecutive runs). New tests: all
earlier request messages are byte-identical across todo updates, task state
is never journaled or placed after a pending tool call, and the Anthropic and
Responses encoders accept the trailing state message after tool results.
One PTY run failed in `queued_prompt` because a redraw split "Second reply."
on screen; the scenario now confirms the reply from the journal. This may
also explain the unidentified RC34 flake. Cache-hit effect and model
behavior are unverified until a live run.

## RC36 record times and timestamps

`1.0.0-rc.36` passed vet (macOS and Windows), the race suite, seven CLI and
twenty PTY scenarios (two consecutive runs). The new `timestamps` scenario
toggles `/timestamps` in the real binary, checks a rendered turn time and
that every journal record is version 2 with a time. Unit tests: a version 1
journal opens, accepts version 2 appends and forks keeping both the original
times and their absence; inconsistent version/time pairs are rejected; NUL
in content cannot forge a timestamp line; repeated prompts get times in
order. A first PTY run failed because the hidden marker added a blank line
that changed the layout; fixed before commit.

## RC35 copy picker

`1.0.0-rc.35` passed vet (macOS and Windows), the race suite, seven CLI and
nineteen PTY scenarios. Unit tests check that system and tool content is never
offered, empty replies are skipped, order is newest first, filtering and Enter
copy the exact content, the draft is untouched and Esc closes the picker.
Timestamps were deferred: journal records carry no time and salvage rejects
unknown fields, so adding them is a journal-format change.

## RC34 status diagnostics

`1.0.0-rc.34` passed vet (macOS and Windows), the race suite, seven CLI
scenarios, nineteen PTY scenarios and the install test. One PTY suite run
failed and two immediate reruns passed; the failing scenario was not
captured, so it is recorded as an unexplained flake. New tests check the
validation reason, failover summary, theme line and integration states on
`/status`, and that `DegradedIntegrations` returns a copy.

## RC33 queued prompts

`1.0.0-rc.33` passed vet (macOS and Windows), the race suite, seven CLI and
nineteen PTY scenarios. The new `queued_prompt` scenario types a second prompt
while a 2.5 s reply is in flight, checks the `[queued]` marker, waits for the
second reply and confirms both prompts are journaled in order. Unit tests
cover slash-command and queue-limit refusals that keep the draft, blocked
dialog shortcuts during a run, the return of queued prompts after a cancelled
run, and scheduling after a completed one.

## RC32 session management

`1.0.0-rc.32` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. RC30 and RC31 native CI passed all six jobs. New tests
cover title cleaning (control and bidi characters removed, whitespace kept,
120-character limit) and clearing; trash and restore of journal, title and
artifacts; refusal to trash an open session and to overwrite on restore; the
browser's rename, two-step delete and current-session refusal; and the CLI
commands.

## RC30 credentials and cancellation under load

`1.0.0-rc.30` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. `TestParentChildCancellationUnderLoad` ran 15 times
with `-race` (90 parent/child cancellations): each returned within the 6 s
bound, each child shell process was gone (checked by PID; `pgrep` found no
leftover `sleep 30` afterwards), and each follow-up run refused with the
acknowledgement requirement without calling the model. The test is Unix-only
(it uses POSIX shell and signals); Windows process-tree cleanup is covered by
the native shell timeout test. `TestCorruptCredentialsAreReplacedBySignIn`
covers corrupt and world-readable credential files.

## RC29 damaged-session recovery

`1.0.0-rc.29` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. RC28 native CI (run 42391210737) passed all six jobs.
New tests: a damaged header no longer hides a healthy session; salvage keeps
the turns before mid-journal damage and leaves the file byte-identical; an
unanswered tool call at the cut is dropped; intact and headerless journals are
refused; the CLI path classifies damage, lists, and salvages.

## RC28 themes

`1.0.0-rc.28` passed vet (macOS and Windows), the race suite, seven CLI and
eighteen PTY scenarios. The new `theme_light` PTY scenario starts chat with
`"theme": "light"` and checks the terminal output carries the light background
and not the dark one. Unit tests cover the picker (Esc keeps the theme, Enter
applies and drops cached rows) and rejection of unknown theme names.

## RC27 transcript navigation

`1.0.0-rc.27` passed vet (macOS and Windows), the race suite, seven CLI and
seventeen PTY scenarios. A new test renders a four-turn transcript and checks
the visible top row after previous, previous, next and first jumps, and that
last returns to the bottom. Clipboard delivery depends on the terminal's OSC 52
support and was not tested in a real terminal.

## Native Windows shell (after RC26)

New Windows-only tests run the `cmd.exe` shell path natively: output and
stderr, verbatim quoting, nonzero exit, and timeout cleanup. The first run
(42390079332) found that a non-forced `taskkill` cannot stop console process
trees, so every Windows timeout reported `cleanup_incomplete`; commit
`fa7c660` forces the tree kill. CI on `fa7c660` passed all six jobs (tests and
installers on Ubuntu, macOS and Windows). The installers now also refuse a
checksum-valid binary that does not run, keeping the existing install
(local test; idea from the Harvest installer).

## RC26 shortcuts and first all-green native CI

Native CI run 42389704720 (RC25): `go vet` and `go test ./...` passed on
`ubuntu-latest`, `macos-latest` and `windows-latest`, and the installer job
passed on all three. This is the first fully native pass on Windows; some
Windows tests skip POSIX-shell fixtures and permission-bit assertions, so the
Windows shell tool and ACL-based privacy still lack dedicated native tests.
Windows CI is no longer allowed to fail. Locally `1.0.0-rc.26` passed vet
(macOS and Windows), the race suite, seven CLI and seventeen PTY scenarios;
new tests cover keymap overrides, rejection rules and remapped keys in the UI.

## RC25 edit feedback

Native CI run 42389421084 (RC24): tests passed on Ubuntu and macOS; the
installer job passed on Ubuntu, macOS and Windows (clean install, tampered
binary refused, nothing installed). Windows tests had one remaining
permission-bit assertion, guarded in RC25. Locally `1.0.0-rc.25` passed vet
(macOS and Windows), the race suite, seven CLI and seventeen PTY scenarios.
New tests: errors from a running fixture server appear as `lsp_errors` with
warnings filtered out, and an edit with no running server neither requests
`lsp_start` nor attaches errors.

## RC24 installers and portability

Native CI run 42388824277 (portability fixes): macOS passed; Windows had four
remaining failures in internal/tools (permission-bit assertions, 8.3 temp
paths, control-character filenames); Ubuntu hit an LSP refusal race that is
now fixed in the client (0 failures in 60 local `-race` runs). Those were
addressed for RC24. Locally `1.0.0-rc.24` passed vet for macOS and Windows,
the race suite, seven CLI and seventeen PTY scenarios, and
`scripts/test_install.sh` (clean install; tampered binary refused with
nothing installed). The RC24 native CI run is the Windows and installer check.

RC23 native CI (run 42387111774): Ubuntu and macOS passed. Windows still
fails in cmd/harness, core, formatters, runtime, session, skills, snapshots
and tools. The directory-sync errors are gone; remaining causes include path
separators in glob/skill matching, Unix shell test fixtures, and file mode
assertions. Windows is not yet a supported platform.

## RC22 Windows build

`1.0.0-rc.22` passed `go vet`, the race suite, seven CLI fixture scenarios,
sixteen PTY scenarios (three consecutive runs) and the ACP suite on this Mac.
`go vet` (including test compilation) passed for windows/amd64,
windows/arm64, linux/amd64, linux/arm64 and darwin/amd64. `make release`
produced six binaries; `file` confirms PE32+ and ELF outputs, and SHA256SUMS
self-verifies. Not yet exercised natively: Linux and Windows runtime, the
Windows shell, lock and process cleanup. The CI workflow is the first native
check. The RC20 undo PTY scenario was timing-sensitive and is fixed here.

## RC21 compaction masking

`1.0.0-rc.21` passed `go vet`, the race suite, seven CLI fixture scenarios and
sixteen PTY scenarios. New tests cover compaction by masking alone with no
model call, placeholder identity, recent groups left verbatim, the
`observation_masking` journal record and its restoration on resume, the
fall-back to summarizing a masked view, and manual compaction still
summarizing. The existing summarizer-path test now disables masking
explicitly. Live providers were not used.

## RC20 undo and redo

`1.0.0-rc.20` passed `go vet`, the race suite, seven CLI fixture scenarios and
sixteen PTY scenarios. The new PTY scenario approves a file creation, types
`/undo`, confirms the file is gone and the prompt is back, then selects Redo in
the palette and confirms the file is restored. Fixture tests cover a two-turn
round trip, refusal after an external edit with rollback of the already
reverted file, shell effects reported as unrestorable, redo refused after a
new prompt, and forking and continuing an undo branch. Live providers were not
used.

## RC19 integration degradation

`1.0.0-rc.19` passed `go vet`, the race suite, seven CLI fixture scenarios and
fifteen PTY scenarios. New fixture tests start a healthy, a broken and a
missing MCP server together and invoke the healthy tool; run an agent whose
only MCP server fails discovery to completion with nothing pending; and run an
LSP query against a missing executable, then a follow-up turn that is not
blocked. The existing bad-initialize LSP test now expects completion with the
server stopped. No third-party servers were tested.

## RC18 provider failover

`1.0.0-rc.18` passed `go vet`, the full race suite, seven CLI fixture scenarios
and fifteen PTY scenarios on this Mac. New tests cover failover after 503,
connection and 429 failures; chained fallbacks; refusal to reroute after
authentication errors, partial streams, or with no list configured; config
validation; and a loopback HTTP run where three 503 attempts on the primary
route are followed by a streamed reply from the fallback endpoint. Failover was
not exercised against live providers.

## V1 release candidate

The installed macOS ARM64 `goh` binary reports `1.0.0-rc.17` and matches the
local build byte for byte. The race suite, `go vet`, seven CLI fixture scenarios,
fifteen PTY scenarios, and compiled ACP integration suite passed on this
candidate. Eleven bounded fuzz targets passed on RC1. A TUI regression covers a
narrow viewport with wide text, tabbed previews, and long input.

RC17 evidence is under `research/evidence/v1-chatgpt-*`. The live
OpenAI test used a user-approved GoHarness ChatGPT grant and the account's
`gpt-6.1-sol` model. It verified streamed text, a permission-gated file edit,
read-back, durable resume, and a real model turn through the terminal
`/connect` picker. Fixture tests cover PKCE and callback state, signed identity,
private storage, refresh rotation, exact endpoint binding, and refusal to
execute tool calls from a partial Responses stream.

RC16 evidence is under `research/evidence/v1-audit-*`. Compiled setup tests
cover provider draft recovery and the gateway fallback. The shell cancellation
terminal test uses Esc. Unit checks cover credential routing, JSON reply bounds
and malformed replies. The native OpenAI probe reached HTTP 401 without a key.
A separate signed-in Codex CLI probe completed. The
[end-to-end audit](END_TO_END_AUDIT.md) records the remaining account,
integration, UI and platform checks.

RC15 evidence is under `research/evidence/v1-ui-export-*`. The terminal setup
flow tests cancellation, successful export, private file mode, collision
handling, unchanged journal bytes and request counts, and matching CLI Markdown.
Focused tests verify optional tool details, history retained after compaction,
and cleanup after a failed write. The paired PTY capture also saves a transcript
with tool details enabled and verifies the resulting tool calls and results.

RC14 evidence is under `research/evidence/v1-ui-fork-*`. The terminal setup
flow cancels a fork without losing the source draft, branches before a numbered
turn, verifies the separate journal and unchanged source, and sends an edited
draft from the new chat. It also exercises the CLI boundary before the first
turn. Focused journal tests verify repeated prompts, retained compaction state,
branch provenance, and invalid boundaries. Paired PTY captures show both fork
menus.

RC13 evidence is under `research/evidence/v1-ui-files-*`. The terminal setup
flow finds a file with spaces using a fuzzy abbreviation. It checks that
selection leaves provider request counts and journal bytes unchanged, then
sends the quoted relative path with a separate Enter. The provider fixture
receives the exact prompt text. Focused checks cover Unicode, cursor placement,
narrow terminals, draft preservation, and hidden and symlink exclusions.
Paired PTY captures show the GoHarness and OpenCode composer menus.

RC12 evidence is under `research/evidence/v1-ui-timeline-*`. The executable
setup flow searches a prompt from a resumed conversation, opens its preview,
and recalls it. Request counts and journal byte comparisons verify that browsing
and recall do not send or persist a turn; a separate Enter sends the exact selected
prompt. Unit regressions cover draft/cursor preservation, newest-first ordering,
multiline and Unicode search/recall, empty results, the 200-prompt bound, terminal
control sanitization, and viewport bounds down to 8×4. Captures include the Go
list and preview and the reference OpenCode timeline.

RC11 evidence is under `research/evidence/v1-ui-status-*`. The PTY flow opens
status before provider setup, uses its provider shortcut, then inspects and
refreshes status after a run. Provider request counts and journal byte comparisons
verify that opening and refreshing the snapshot create no model request or
journal mutation. Unit regressions check key/path/query/environment redaction,
draft preservation, Plan-mode labels, terminal sizes, and scrolling.
Resumed tool-message regressions check readable saved envelopes, preserved error
labels, and legacy payloads. The same tool formatter serves live and resumed
results.

RC10 evidence is under `research/evidence/v1-ui-sessions-*`. The PTY flow
searches an older conversation by title, resumes its original journal, and
verifies that a second journal is untouched. It also cancels the browser back
to chat and invokes the CLI preview while the chat writer is active.
Focused tests check prefix limits, incomplete tails, symlink/traversal refusal,
unfinished-run status, workspace scope, empty results, and terminal dimensions.
Preview byte comparisons verify that it leaves the journal contents intact.

RC9 evidence is under `research/evidence/v1-ui-pickers-*`. Its executable PTY
test selects a named model profile, observes the changed model ID in an actual
provider request, returns to the default profile, and verifies the original
model ID. It selects Plan and checks that shell and mutation tools are absent
from the provider request. It also cycles to Build and verifies /new creates a
second journal while retaining the original. Race, vet, CLI, ACP, and the
isolated 15-case terminal suite pass.

`goh-ui-models.png` and `goh-ui-agents.png` capture the new selectors. Source,
test evidence, and project stage documents are published through the project
Git repository. Local toolchains and reference checkouts are excluded by the
repository ignore rules.

RC8 evidence is under `research/evidence/v1-ui-rich-*`. The race suite, vet,
CLI, ACP, setup, and isolated 15-case PTY run passed. One PTY run concurrent
with reference captures returned `cleanup_incomplete` in child cancellation.
The subsequent isolated pass leaves the cause and reliability under load
unresolved. See `research/evidence/ui-rich-cancellation-note.txt`.

Streaming regressions check every byte prefix of a mixed code/table response,
12/24/60/90-column panes, escaped and inline-code pipes, column alignment,
multiline comments, wrapped strings, unknown-language fallback, NO_COLOR, and
terminal-control sanitization. Approval previews bypass Markdown interpretation.
The OpenCode executable was also sent the same rich-text fixture response.
`opencode-ui-session.png` and `goh-ui-session.png` retain the rendered captures.

The `v1-chat-first-*` evidence includes a first-run PTY test. With a fresh
home, it opens chat before configuration, verifies blocked submissions make no
requests, saves a masked provider key, recovers from HTTP 401 by updating the
key in the UI, and verifies that the session is retained. It also checks saved
credentials in CLI requests, settings return to chat, and terminal restoration. The
`goh` and `harness` commands resolve to the same installed build and session
storage.

The UI redesign is covered by `research/evidence/v1-opencode-ui-*`. Layout
regressions cover 24×10, 60×18, 80×24, and 120×36 terminals, wide input, permission
previews, and command palette navigation. Actual PTY frames rendered into PNGs
are retained as `goh-ui-home.png`, `goh-ui-palette.png`,
`goh-ui-session.png`, and `goh-ui-approval.png`. OpenCode v1.19.5 was run
with an isolated home and a loopback provider configuration for the startup
comparison and `opencode-ui-home.png`. These frame captures use a terminal
emulator and a fixed font rather than an OS screenshot.

The `v1-ui-composer-*` records cover RC7, including the full race suite and
CLI, ACP, and terminal integration checks. The setup PTY test checks that
editing inside a prompt changes the actual provider request and that recalled
history can be resubmitted. Focused regressions cover Unicode input, multiline
cursor movement, draft restoration, configured command completion, literal
code/diff preservation, and code styling when scrolling into a fenced block.
`goh-ui-slash.png` and `goh-ui-composer.png` capture the composer changes.

RC2 streamed a real Qwen3-0.6B response from a loopback llama.cpp server at a
16,384-token context. The CLI base URL override and a custom bearer token
environment variable succeeded. The same server rejected a wrong token.
The 8,192-token trial stopped before inference because the default tool schemas
exceeded the harness's conservative budget. See
`research/evidence/v1-live-local-provider.json` and the `v1-base-url-*` logs.
The Windows AMD64 cross-build currently fails on Unix-specific filesystem and
subprocess operations. See
`research/evidence/v1-windows-crossbuild.txt` for compiler diagnostics.

## Executed checks

| Check | Result | Retained record |
|---|---|---|
| `go test -race ./...` | Pass across all packages, including 47 top-level tests and their subcases | research/evidence/go-test-race.txt |
| `go vet ./...` | Pass, no diagnostics | research/evidence/go-vet.txt |
| Bounded SSE fuzzing | Pass, 96,707 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Bounded tool arguments fuzzing | Pass, 145,879 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Bounded journal fuzzing | Pass, 131,383 executions in 3 s in recorded run | Agent tool output; reproducible `make fuzz` |
| Built executable driver | Seven cases pass: text, parallel reads, read/edit/shell, permission, drop, malformed stream, HTTP failure | research/evidence/cli-acceptance.json |
| Pseudo-terminal integration | Five cases pass: text, refusal, edit approval, active shell cancellation, termination; app remains open after first cancellation | research/evidence/tui-acceptance.json |
| Primary paired semantic comparison | Five selected cases pass | research/evidence/semantic-comparisons.json |
| Extended paired semantic comparison | Nine selected cases pass | research/evidence/extended-semantic-comparisons.json |
| Vulnerability scan | govulncheck 1.9.0 source/symbol scan exits0, no findings in final module graph; database 2026-11-02 | research/evidence/govulncheck.json and dependency-audit.md |
| Cross-builds | CGO_ENABLED=0 for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64 | dist/SHA256SUMS and BUILD_INFO.txt |

Retained paths above are relative to the reconstruction project root except
`dist/`, which is relative to implementation/. Raw REA comparisons are preserved
and report changed/truncated output. Independent semantic checks establish only
the declared observed facts, not universal equivalence.

The terminal integration test compares termios flags, speeds and control
characters before/after. It masks only Darwin's PENDIN flag, which the kernel sets
when restoring canonical mode after raw mode; an independent Python raw/restore
probe reproduced that single-bit change. Resize is covered by state tests and
REA PTY captures. Terminal restore escapes alone are not treated as proof of
termios restoration; the independent test checks the actual slave attributes.

## Acceptance coverage

Parser tests cover one-byte network fragmentation (including UTF-8), split tool
arguments, empty-choice usage chunks, early EOF, bad JSON, duplicate tool IDs,
unknown finish reasons, late content and cancelled requests. Partially consumed
responses are never transparently replayed. Length termination has a separate
budget outcome and cannot execute incomplete tool calls.

File tests cover exact ambiguous/stale edits, intervening mutations, CRLF, Unicode,
mode preservation, new-file collisions, escaped/symlink/FIFO/binary paths,
argument validation, complete tool schemas, bounded two-sided previews and
cancellation during admission. Runtime tests verify parallel read results remain
in call order and mutation/shell operations are barriers.

Process tests cover successful/failing commands, bounded retained output,
large output, timeout, cancellation and children in owned groups. Journals cover
conflicting writers, private permissions, torn tail versus corruption, unsupported
version, and uncertain-effect acknowledgment without replay. Context tests cover
successful/failed/ineffective compaction, active request and recent groups,
unresolved calls and explicit exhaustion. MCP fixtures verify discovery,
validation, references rejection, cancellation, disconnect, bounded results,
cleanup, admission, durable startup and recovery without authority restoration.

## Outstanding release gates

- Linux runtime tests: cross-builds passed, but no Linux execution environment was
  available on this host. No Docker/Podman/QEMU executable was found on PATH.
- macOS AMD64 and Linux architecture runtime coverage; macOS ARM64 is the tested host.
- Live model endpoint/profile compatibility: intentionally untested without an
  explicitly configured endpoint, credentials and authorization for live calls.
- Reference compaction and journal-corruption behavior, plus full interactive
  reference approval/cancellation coverage, remain unverified. Go deterministic
  behavior is tested; this does not imply reference parity.
- Reference shell cancellation returned REA cleanup_incomplete; its orphaned
  experiment child was manually cleaned up and recorded. Go cancellation cleanup
  was verified separately.
- Full comparative performance: only local microbenchmarks and version-process
  startup were measured. No reliability/speed superiority claim is made.
- Public-release ownership, branding, contribution and legal review remain open.

See PROGRESS.md, research/FEATURE_MAP.md (local) and
plan/specs/COMPATIBILITY_MATRIX.md for exact scoped evidence and next actions.


## Expanded-feature verification (same pinned reference)

The full reference checkout is detached at 810c0eba1a7348036f8bdf61e6632d0c36c6876c:
6,627 tracked files across 32 top-level packages, indexed under research/. This
is whole-repository acquisition and surface mapping, not an assertion that every
source file or behavior was reviewed.

Current test inventory: 68 top-level tests plus their subcases. The full current
race suite and vet pass (feature-tests-race.txt, feature-vet.txt). Original seven
built-CLI acceptance cases still pass (feature-cli-acceptance.json). Six PTY
scenarios now pass, adding task approval, task overview/local reader and manual
compaction to the prior five cases (feature-tui-acceptance.json). Actual terminal
settings are checked; the inherited Darwin PENDIN exception remains documented.

New reference/Go evidence covers recursive glob/include grep, session task list
snapshots and durable reset, manual compaction, and session portability/fork.
`research/compare_new_features.py` passes its six selected semantic checks;
`grep-semantic-comparison.json` confirms the declared seven-case Go contract,
while documenting different default eligible files and named-file semantics.
`go-session-portable.json` captures fresh identities/equal state/no authority copy.
Glob records retain their truncation limits and changed raw comparison status.
These checks do not imply universal parity. Exact contracts and evidence IDs are
in specs/features/. Full Go snapshot/manifest identity remains in each record.

Security: replay always re-anchors the current trusted system message; historical
compaction systems become untrusted assistant data. Portable archives reject
unsupported formats, uncertain effects, incomplete tool groups, invalid task
schemas and oversized state, and never import permission grants. Manual compaction
starts no integrations. Named todowrite grants authorize task-state updates only.

The expanded whole-repository objective remains unfinished: skills, patch engine,
agent/model profiles, subagents, structured questions, additional providers/auth,
LSP, formatters, plugins, server/ACP, web/desktop and integrations are backlog
items, alongside existing Linux/live-provider/public-release gates. The manual
reference compaction gap is partially resolved; automatic thresholds and pruning
remain unverified. See PROGRESS.md and FEATURE_BACKLOG.md for the next exact slice.

Final added glob-parser fuzz run: 67,442 executions, no failures. Final source/symbol vulnerability scan has 0 findings (feature-govulncheck.json). All four CGO-free platform builds/checksums were regenerated after feature integration.

## Further feature verification

Skills, exact multi-file patches, JSON model/agent profiles, strict build/plan mode,
pure custom commands and structured questions are now integrated over the same
runtime. Detailed contracts/capture identities: plan/specs/features/{skills,
apply-patch,profiles,commands,questions}.md. All observed reference data remains in
research; original product prompts and implementation remain separate.

Current full race and vet logs are rest-tests-race.txt/rest-vet.txt. Seven CLI
acceptance scenarios remain passing. Nine PTY scenarios now include actual
structured choices/multiple/custom text, dismissal and cancellation with terminal
restoration, in addition to the earlier task/compaction and shell cases. Runtime
checks cover plan-mode forced-tool refusal despite grants, profile/env/CLI
precedence, metadata-only skill admission, lazy resources, explicit custom-command
provenance without effects, no question-to-permission transfer, and repaired input
relationships on safe continuation. Actual real patch tests cover approval,
denial, intervening writes, CRLF/modes, partial cancellation, durable observations,
uncertain recovery blocking and no replay. New schema tests compile all runtime
tool schemas with the pinned validator.

The patch semantic projection passes 10 bounded reference/Go cases; selected new
skill/plan/command/question checks are in rest-semantic-comparisons.json. Captures
and tests document stricter deliberate matching/authority/input differences.
Source-only surfaces remain distinguished. Skill parser fuzzing passed 270,248
executions; patch grammar fuzzing passed 180,851 in the final worker run. No new
runtime module dependencies were introduced. The renewed source/symbol advisory
scan is retained in rest-govulncheck.json.

Scope remains partial: controlled subagents, webfetch/search, LSP, additional
providers/auth, server/ACP, remote MCP, plugin compatibility and web/desktop
interfaces are among the remaining whole-repository backlog. Existing Linux and
other-architecture runtime/live-provider/public-release gates remain open.


Final batch inventory: 112 top-level tests and their subcases pass under race;
vet passes, seven CLI/nine PTY cases pass, and fresh govulncheck has 0 findings.
All four CGO-free binaries/checksums were refreshed. Existing local-only performance
measurements predate this feature batch and are not performance claims about the
new binaries; benchmark full workloads separately when those gates are reached.

## Foreground child/web/LSP verification

The next slice adds private child sessions/owned continuation with independent
child authority, explicit web fetching and configured local LSP. Current retained
logs are next-tests-race.txt and next-vet.txt; seven CLI acceptance cases and eleven
PTY cases pass. New PTY cases separately approve a child mutation and cancel an
executing child shell (command-entry pid fixture), while preserving terminal
settings and leaving the application open after cancel.

Unit/integration tests cover child ownership/type/limits/grants, strict explore/
parent-plan caps, safe continuation, no recursive delegation, web credential/proxy/
cookie isolation and redirect destination non-contact, response/HTML/result bounds,
LSP initialize/sync/UTF16/diagnostics, malformed/large frames, out-of-order replies,
blocked writes/cancellation, startup admission/failure, server-requested edit
rejection, special files and uncertain recovery without replay. Original helper
servers frame JSON-RPC independently of the client's parser.

Actual reference/Go captures and selected assertions are retained in
next-batch-driver.json and next-semantic-comparisons.json. Child creation/resume,
eleven fetch cases and six LSP operations pass their declared selected contracts;
interface, authority, byte/charset/redirect and Unicode-column differences are
explicit. Webfetch/LSP rendered outputs are truncated; webfetch includes unverified
ambient-process observations despite verified owned-group cleanup. No global
process/network absence or universal equivalence claim follows.

Bounded fuzz runs: 207,743 LSP-frame executions and 40,395 web-argument executions,
no failures. Renewed govulncheck source/symbol scan has no findings after pinning
x/net v0.62.0 and its selected transitive graph; exact 39-module notices/digests were
refreshed. No live providers, production language servers, remote fetch targets,
automatic server downloads or remote publication were used for this acceptance.
Native/advanced LSP, background/nested children, richer web search/auth, all broader
backlog and platform/live-provider/public-release gates remain open.

Final inventory for this batch: 133 top-level tests plus subcases pass under race, clean vet, seven CLI and eleven PTY acceptance cases. Four rebuilt Darwin/Linux AMD64/ARM64 artifacts pass SHA256 verification. Platform cross-builds remain distinct from native runtime acceptance; no whole-repository parity claim follows.

## Explicit formatting and file checkpoint verification

This batch adds original formatter configuration/discovery, literal argv
substitution, separately admitted supervised processes, post-edit final identity,
and opt-in durable before/after checkpoints. Snapshot list/diff/undo/redo are
available through the runtime and `changes` CLI. Tests cover separate write versus
formatter authority, headless skip/denial, failure after mutation, final formatted
bytes, child/plan boundaries, cancelled formatter checkpoints, patch moves and
creation, all-file conflict prevalidation, CLI recovery and concurrent read tools.
The snapshot package additionally checks binary bytes, Git-index preservation,
private storage, corrupt blobs/metadata, modes, bounds, locks, links/FIFO and
partial multi-file cancellation without replay. Restores preserve unrelated files;
full Git-tree snapshots/message-history revert are unimplemented.

Fourteen actual PTY cases now include formatter approval, denial and running
formatter cancellation; terminal settings are restored and cancellation leaves the
application open. Seven existing compiled-CLI scenarios also pass. Logs are
changes-tests-race.txt, changes-integration-race.txt, changes-vet.txt,
changes-cli-acceptance.json and formatter-tui-acceptance.json in research/evidence.
Research-only formatter and snapshot scripts verify real pinned reference
observations, including exit7 preservation and revert/unrevert of externally
modified touched files. Go REA captures go-formatters/go-snapshot-{undo,redo,conflict}
plus changes-semantic-comparisons.json retain selected original assertions.
These projections are not canonical REA equivalence. Process observations have
incomplete shutdown sampling; the formatter capture additionally contains
unverified ambient-process identities. Owned-group cleanup was verified. No
universal absence-of-effects or runtime-platform equivalence claim follows.

Webfetch HTML raw-text handling now also preserves suppression for self-closing
script/style tokens; regression and bounded argument/HTML fuzz checks cover that
parser behavior. Dependencies and their selected39-module graph are unchanged.
All broader repository parity, native/live-provider, platform runtime and public
release review gates remain open. Existing benchmark measurements predate this
batch and are not measurements of these new features.

Final batch inventory:153 top-level tests plus subcases, seven CLI and fourteen
PTY scenarios pass; race/vet are clean. Web argument fuzz60,219 and HTML fuzz26,293
executions pass. The renewed source/symbol scan contains0 findings. All Go REA
captures in this batch have normalized_result.truncated=true because of incomplete
process observations, even when terminal output itself fits the render budget.
Four CGO-free Darwin/Linux AMD64/ARM64 binaries and checksums are regenerated;
platform cross-builds do not establish runtime parity. No new dependencies.

##6 November2026: native protocol, Harvest roles/results and context efficiency

Native Anthropic Messages text/client-tool encoding and streaming, explicit
compaction/task/explore roles, strict optional task JSON results, Go declaration
maps and scoped shell/formatter output retrieval are implemented. Full source
pins/license digests for Harvest/Aider/Goose/Crush are retained under
research/comparisons; their builds/tests were not executed and no upstream runtime
claims follow from source inspection. OpenCode native complete/drop observations
are separately retained and checked by research/compare_native_provider.py.

New tests cover native request history/tool-result identity, fragmented Unicode/
JSON SSE, usage/cache counts, finish/errors/lifecycle/bounds/unsupported blocks,
credential separation, redirect non-contact, HTTP classification/redaction,
backoff cancellation and partial-call no-effect. Runtime tests exercise a real
mixed chat/native child role, preflight invalid schema no-dispatch, strict child
output failures without repeated effects, plan/custom-role precedence and failed
summary admission preserving history. Tool tests cover content-cache invalidation,
concurrent maps, syntax/body exclusion, limits, retained-artifact byte/base64/tail
paging, session ownership, links/FIFO, escaped results and cancellation.

184 top-level tests plus subcases pass under race, with clean vet. Seven existing
CLI acceptance cases, seven additional native/Harvest CLI REA cases and15 actual
PTY cases (including typed child approval/result) are checked independently.
Fuzz runs:19,334 structured output cases and67,105 native SSE cases, no failure.
Renewed govulncheck source/symbol scan reports0 findings in the unchanged selected
39-module graph. No extra module/runtime is linked for repository mapping.
Logs: harvest-tests-race.txt, harvest-vet.txt, harvest-integration-race.txt,
efficiency-integration-race.txt, harvest-tui-acceptance.json, harvest-govulncheck.json,
structured-fuzz.txt, native-provider-fuzz.txt and repomap-benchmark.txt.

Actual Go CLI captures indexed by harvest-batch-driver.json exercise valid/invalid
typed child results, mixed protocol, a dedicated compaction role, native read/text,
partial stream,401 and redirect refusal. harvest-semantic-checks.json and
native-provider-comparison.json are original selected assertions, not canonical
REA equivalence. In the pinned reference dropped stream, the read is interrupted
then the run continues and exits0; Go fails once and commits no incomplete tool.
Each capture retains its own build identity, truncation/cleanup/ambient-process
unknowns; process capture is not network tracing or universal absence-of-effects.

The bounded declaration/output fixture and local microbenchmark are reported in
BENCHMARKS.md, efficiency-report.json and go-efficiency.json. Byte reduction is
fixture-specific, not a tokenizer/live-cost or cross-agent benchmark. Native
live-account acceptance, server/API/ACP, richer protocol content/auth, general
artifact storage/portability/GC, multilanguage/dependency-ranked context and all
other broader compatibility/platform/release gates remain incomplete.

## Local HTTP API verification

The race suite now includes 187 top-level tests. The HTTP tests exercise
permission approval, question validation and retry, stale challenge rejection,
overlapping run rejection, explicit cancellation, disconnect cancellation,
project scope, bearer authentication, and journal recovery. The compiled
Darwin ARM64 smoke test checks startup on an assigned loopback port, streaming
events with matching run IDs, an approved shell command without the server
token in its environment, and graceful shutdown. Vet and all four rebuilt
release checksums passed.

## ACP stdio verification

Run `python3 scripts/test_acp.py bin/harness` from the implementation directory.
The fixture launches the compiled agent with real pipes and verifies version
negotiation, workspace scope, approval before a file write, prompt overlap
rejection, cancellation followed by a late approval, durable resume after a
restart, client MCP startup denial, and EOF and SIGTERM cleanup. The full Go
race suite and vet pass. The same stdio fixture also passes against a binary
built with the race detector. Evidence is under research/evidence/acp-*.

Generated-file protection tests cover preview and direct execution, explicit
configuration override, whole-patch rejection on a generated deletion or move,
and a marker after source code. Before/after exact-edit benchmarks use the
same fixture and are retained in research/evidence/generated-edit-{before,after}.txt.

Compaction receipt tests cover projected file and exit-code evidence, the exact
context estimate, portable export and import, and rejection of a modified
working view. Existing tests continue to cover legacy compaction, preserved
tool groups, and uncertain effects. Logs are retained under
research/evidence/compaction-receipt-*.

Project memory verification covers cross-project isolation, changed evidence,
deletion, symlink rejection, opt-in and explicit recall through the real
runtime loop in plan mode. The CLI smoke test exercises the compiled binary
against a temporary project. Logs are under research/evidence/project-memory-*.
