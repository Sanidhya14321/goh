GO ?= go
VERSION ?= 0.1.0-dev

.PHONY: build test race vet fuzz acceptance release demo
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-s -w -X main.version=$(VERSION)' -o bin/goh ./cmd/harness
	ln -sf goh bin/harness
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-s -w' -o bin/scripted-model ./cmd/scripted-model

test:
	$(GO) test ./...
race:
	$(GO) test -race ./...
vet:
	$(GO) vet ./...
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
	$(GO) test ./internal/webfetch -run '^$$' -fuzz FuzzHTMLConversion -fuzztime 3s -parallel 2
	$(GO) test ./internal/commands -run '^$$' -fuzz FuzzExpansion -fuzztime 3s -parallel 2
acceptance: build
	$(GO) run ./cmd/acceptance
release:
	GO='$(GO)' VERSION='$(VERSION)' sh scripts/release.sh
demo: build
	bin/scripted-model --scenario text
