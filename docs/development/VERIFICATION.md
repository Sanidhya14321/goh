# Verification log

What was run for each release candidate and what it showed, newest first. This is an evidence
log: entries are appended when a milestone lands and are not rewritten afterwards, so older
entries describe the code as it was then. For how to run the gates see [TESTING.md](TESTING.md);
for open gaps see [STATUS.md](STATUS.md).

Unless an entry says otherwise, local runs are on macOS ARM64 (Apple M5 Pro) with the Go version
pinned in `go.mod` at the time, and live inference used loopback fixtures.

## Unreleased: CI back to green, credential boundary through symlinks

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; all five terminal suites; `govulncheck ./...` (v1.9.0, for darwin, linux and windows): no vulnerabilities; v1.1.4 panicked on the Linux build in CI (`unexpected expr: *ast.KeyValueExpr`). Linux ARM64 in a `python:3.12-bookworm` container (Colima): all five terminal suites passed with the cross-built binary.

- CI run 43748015459 (15 Nov) failed three ways. Linux `test_tui.py` `text` showed the first-run provider picker: the suite's private home lived inside the test project, which goh refuses because its saved keys would be inside the project. Reproduced in the container with the old script (fails) and the new layout (passes).
- The same layout passed on macOS, which exposed a real gap: `CheckProjectCredentialBoundary` compared the symlink-resolved project (`/private/var/…`) with the unresolved, not-yet-created key folder (`/var/…`), so the check was skipped. `TestCredentialBoundaryResolvesSymlinkedHome` fails without the fix.
- Windows: `TestCheckRoute` expected "not running at"; `errors.Is(err, syscall.ECONNREFUSED)` does not match Winsock's `WSAECONNREFUSED`, now `platform.ConnectionRefused`. `TestHomeFolderIsRefusedWithAFolderFix` read the runner's real profile because `USERPROFILE` was not isolated.

## Unreleased: small per-request caps (Groq free tier)

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

## Unreleased: end-to-end audit with real-world prompts

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make acceptance`; `test_tui.py` (25 scenarios), `test_setup.py`, `test_acp.py`, `test_attach.py` and `test_resume_setup.py` all passed.

- Live, Groq `openai/gpt-oss-120b` over `chat_completions` (ChatGPT could not be used: its saved sign-in was revoked, which found the renewal message fix). Real prompts on a small Python project: "I just inherited this repo…" completed after six tool calls and four 429 waits of 6–25 s (the same prompt previously failed at the fifth request); "how many stars and open issues does golang/go have" through the GitHub REST API returned live counts (previously HTTP 415); "total spend in EUR via the ops MCP tool" started the server, read the CSV and converted correctly. Fetching go.dev and Hacker News failed with HTTP 413: Groq's free tier caps a request at about 8k tokens, below one 64 KiB page; the error now says so. With `--context-window 8000` the run stopped before any request (`budget_exhausted`). That figure was the one-token-per-byte estimate: Groq counts goh's instructions and tools as about 1,960 tokens (see the per-request cap section below). An explore subagent prompt completed: both children failed (one `tool_use_failed` after re-sampling, one 429) and were reported, and the parent finished the audit itself.
- gpt-oss calls tools that do not exist (`repo_browser.read_file`) and invents file text (`<cstdio.h>`). Groq rejects some such calls mid-stream (`tool_use_failed`), which ended runs as "provider stream error"; the request is now re-sampled up to twice. Invented `old_text` is refused and the existing invalid-call limit stopped the loop after three identical failures.
- Groq's rejection messages, read once with a local debug build, showed gpt-oss calling `apply_patch` with `patch` instead of `patchText` (refused by Groq's schema check), plus invented tools and an out-of-range `limit`. After the alias, "the monthly report prints months in the wrong order and top categories look backwards; fix both and make sure the tests pass" produced a correct two-line fix and `unittest` passed; the run then hit its 300 s cap during rate-limit waits before the final summary.
- Components driven by the scripted model against real targets: `webfetch` of Hacker News, go.dev, example.com, a PDF, a 404, an `http://` → `https://` redirect and the cloud-metadata address; `lsp` against clangd (hover, references, document symbols and diagnostics without a position); a stdio MCP server (tool, error result, resources); skills, todos, `apply_patch`, 109 KB of shell output, `job_start` of a real HTTP server (stopped on exit), subagent with `output_schema`, question (exit 4), permission (exit 3), plan mode, resume, fork, Markdown export with tools, memory recall and a custom command.
- TUI in a PTY (110×40): Python source with `# TODO` and `**kwargs` is shown verbatim; grep and glob results are lines; the reply after tool output is a separate paragraph.
- New tests, each confirmed to fail without its fix where checked: `TestToolOutputRendersVerbatim`, `TestListingResultsAreReadable`, `TestJSONAPIsErrorsAndRedirectDestination`, LSP file-wide cases in `TestLSPAdmissionFailureAndBounds`, `TestPatchPreviewShowsChangeDeepInLongFile`, `TestShellPreviewKeepsHeadAndTail`, `TestRetryDelayWaitsOutRateLimits`, `TestStreamErrorCodes`, `TestRejectedOutputIsResampled`, `TestHeadlessRunSkipsUnapprovedMCPServer`, `TestPatchArgumentAlias`, `TestDroppedStreamIsInterrupted`.

## Unreleased: project browser and folder trust

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- PTY, the user's real home (nothing chosen): started in `~`, the browser explains the home folder and lists its folders with project tags; `Desk` + Tab lists `~/Desktop` (goh tagged `git · Go · make`); `goh` + Enter shows the trust question; typing `~/Desktop/g` completes to three folders.
- `test_setup.py` now starts in an untrusted project: it opens the folder, answers the trust question and checks `workspaces.json` before the provider welcome. `TestWorkspacesTrustAndRecency`, `TestWorkspaceBrowserOpensAndTrusts`, `TestWorkspacesCommand`.

## Unreleased: home-folder setup loop, saved /models choice, terminal colors

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- Reported: Groq key added and model chosen, yet chat kept showing "No model connected". With the user's own settings (read only), `goh providers list` and `goh doctor` were valid and `goh run hello` got a reply from Groq; a PTY run from `~/Desktop/goh` opened chat with the Groq model, while one from `~` showed the blocked screen with "choose a project directory outside the GoHarness credential storage scope". After the fix, the same run from `~` says "Chat can't open this folder"; Enter, typing `/tmp/gohproj` and Enter opened chat on the Groq model.
- `/models` on the user's settings (viewed, nothing selected) lists four Groq chat models and "Switch to ChatGPT"; `goh models` prints the same four instead of eleven. `test_setup.py` now checks that a `/models` choice is written to the provider in `config.json`. `TestHomeFolderIsRefusedWithAFolderFix`, `TestSameRoute`, `TestGroqHidesNonChatModels`.
- Default theme: the first chat frame has no background sequences (was 34 `48;2;10;10;10` and 4 panel sequences) and is 982 bytes instead of 2455; only the selected picker row uses a background. `theme_light` still renders its own background.
- `test_tui.py` previously ran goh with the caller's real home folder; it now uses a private one.

## Unreleased: type-ahead across screens, setup screen, docs history

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` all passed.

- Reproduced in a PTY before the fix: `Tab` then `hello` sent as one write left only `h` in the composer; `build⏎hi there` sent to the agent picker in one write left nothing. After the fix both arrive intact, as does `/agents⏎build⏎hi there` in a single write. `TestKeysTypedWhileAScreenClosesReachTheNext` checks that late keys reach the next screen in order and that Enter keeps its meaning.
- `TestEnterOnBlockedChatOpensSetup`; the setup screen was rendered at 40×12 and 100×24.
- A first attempt word-wrapped every terminal row; `formatter_approval` then failed deterministically because the approval preview no longer fit at 80×24 and needed scrolling. Word wrapping is now limited to the setup screen, and the suite passes.

## Unreleased: cleanup, CLI errors and repository layout

Local, macOS ARM64, Go 1.27.4: `gofmt` clean; `go vet` for darwin, linux and windows; `go test -race ./...`; `make e2e` (acceptance plus `scripts/e2e/` TUI, setup, ACP, attach and resume-setup suites) all passed.

- `TestCLIErrorsSayWhatToDo` covers the no-provider message, `goh run` without a message, a missing `--project`, `goh resume` with flags before the ID, an unknown session ID and `goh compact` usage. A fresh-home probe of every top-level command found no remaining internal errors.
- `goh resume ID words…` was run against the scripted model with flags before and after the ID.
- `/settings` was rendered at 50 and 90 columns in a PTY; `test_setup.py` drives it with `s` and `q`.
- Dead code found with `golang.org/x/tools/cmd/deadcode` (with and without tests, for each OS) was removed; `licenses/` matches `go list -m all` exactly; a link check over all Markdown outside `research/` found no broken links.
- The README demo (`scripted-model` plus `goh run --config config.example.json`) was run as written.

## Unreleased: third review fixes, onboarding and approval layout

Local, macOS ARM64, Go 1.27.4: `gofmt`, `go vet`, `go test -race ./...`, Linux and Windows cross-builds, `make acceptance`, `scripts/test_tui.py` (25 PTY scenarios), `test_setup.py`, `test_resume_setup.py`, `test_attach.py` and `test_acp.py` all passed.

- New regression tests, each confirmed to fail without its fix where it can run on macOS: `TestMCPRestartDuringCallIsSettled`, `TestSessionGrantsDoNotCoverInterpreterScripts`, `TestSandboxReadOnlyPathsInsideProject`, `TestMalformedSiblingInReadBatchIsNotExecuted`, `TestVerifyRepairShowsEndOfLongOutput`, `TestEncodeAnthropicRemapsForeignToolIDs`, `TestToolAssemblyWithRepeatedIDAndName`, `TestLogoutClearsCredentialsWhenRevokeFails`, `TestChatGPTLockWaitsForAnotherUpdater`, `TestPromptStartingWithPathIsNotACommand`, `TestLinkedParentDetectsSymlinkedFolders`, `TestDiffPreviewShowsOnlyTheChange`, `TestFitHintsDropsWholeHints`, `TestEditCustomProviderMovesSavedKey`, `TestCustomProviderWithUnsetKeyVariable`, `TestRunOutlivesServerReadTimeout`; extended `TestShellDecision`, `TestShellDescriptionReflectsSandbox`, `TestRunOutcomes` (empty reply not committed), the sandbox canary (parent listing) and the glob cases (nested base, for Windows CI).
- Not verifiable here: the Windows `api_key_env` case rule and glob separator fix (Windows CI), and the Landlock parent-listing rule (Linux CI).
- The review's `goh serve` ReadTimeout finding was a false positive: a probe and `TestRunOutlivesServerReadTimeout` show the request context survives the server's read deadline, so no change was made.
- PTY scripts updated for the new first run: the welcome picker, Enter-while-blocked opening the picker with the draft kept, and Esc from a key form returning to the picker.

## Unreleased: /connect provider catalog

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

Evidence for rc.1 – rc.179 is in [history/VERIFICATION-rc1-rc179.md](history/VERIFICATION-rc1-rc179.md).
