GO ?= go
VERSION ?= 0.1.0-dev
PREFIX ?= $(HOME)/.local

.PHONY: build install test race vet fmt fuzz acceptance e2e release demo bench-check clean
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-s -w -X main.version=$(VERSION)' -o bin/goh ./cmd/harness
	ln -sf goh bin/harness
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-s -w' -o bin/scripted-model ./cmd/scripted-model

install: build
	mkdir -p $(PREFIX)/bin
	# Replace, never overwrite in place: macOS kills a binary whose file
	# changed under its cached code signature.
	rm -f $(PREFIX)/bin/goh
	cp bin/goh $(PREFIX)/bin/goh

test:
	$(GO) test ./...
race:
	$(GO) test -race ./...
vet:
	$(GO) vet ./...
fmt:
	gofmt -l -w cmd internal
fuzz:
	$(GO) test ./internal/provider -run '^$$' -fuzz FuzzAnthropicSSE -fuzztime 3s -parallel 2
	$(GO) test ./internal/structured -run '^$$' -fuzz FuzzStructuredResult -fuzztime 3s -parallel 2
	$(GO) test ./internal/provider -run '^$$' -fuzz FuzzSSE -fuzztime 3s -parallel 2
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzToolArguments -fuzztime 3s -parallel 2
	$(GO) test ./internal/session -run '^$$' -fuzz FuzzJournal -fuzztime 3s -parallel 2
	$(GO) test ./internal/skills -run '^$$' -fuzz FuzzSkillFrontmatter -fuzztime 3s -parallel 2
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzPatchGrammar -fuzztime 3s -parallel 2
	$(GO) test ./internal/lspclient -run '^$$' -fuzz FuzzLSPFrame -fuzztime 3s -parallel 2
	$(GO) test ./internal/webfetch -run '^$$' -fuzz FuzzWebfetchArguments -fuzztime 3s -parallel 2
	$(GO) test ./internal/core -run '^$$' -fuzz FuzzStripJSONCIdentityOnJSON -fuzztime 3s -parallel 2
	$(GO) test ./internal/webfetch -run '^$$' -fuzz FuzzHTMLConversion -fuzztime 3s -parallel 2
	$(GO) test ./internal/commands -run '^$$' -fuzz FuzzExpansion -fuzztime 3s -parallel 2
acceptance: build
	$(GO) run ./cmd/acceptance
# Drives the compiled binary in pseudo-terminals against the scripted model
# (macOS and Linux; needs python3, no credentials).
e2e: build acceptance
	for t in tui setup acp attach resume_setup; do python3 scripts/e2e/test_$$t.py || exit 1; done
release:
	GO='$(GO)' VERSION='$(VERSION)' sh scripts/release.sh
demo: build
	bin/scripted-model --scenario text

bench-check:
	$(GO) build -o bin/goh-bench ./cmd/bench
	PATH="$$(dirname $$(command -v $(GO))):$$PATH" bin/goh-bench --tasks bench/tasks --dry-run

clean:
	rm -rf bin dist
