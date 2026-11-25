#!/bin/sh
# Runs goh on a Harbor dataset in Apple containers.
#   bench/harbor_agent/run.sh [harbor run options...]
# e.g. bench/harbor_agent/run.sh -d terminal-bench@2.0 -l 5 -n 2
# Needs: the container CLI (running), uv, a Harbor checkout (HARBOR_DIR,
# default research/harbor) and a ChatGPT sign-in on this machine (goh auth login).
# Container start, agent setup and test steps get twice the task's time (amd64
# images run under Rosetta here); the agent's own time limit is unchanged.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
GO=${GO:-$ROOT/research/tooling/go/bin/go}
HARBOR_DIR=${HARBOR_DIR:-$ROOT/research/harbor}
MODEL=${MODEL:-gpt-6.2-sol}
VERSION=$(git -C "$ROOT" describe --tags --always --dirty)
cd "$ROOT"
mkdir -p bench/harbor_agent/bin
for arch in arm64 amd64; do
  GOOS=linux GOARCH=$arch CGO_ENABLED=0 "$GO" build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" -o "bench/harbor_agent/bin/goh-linux-$arch" ./cmd/harness
done
CGO_ENABLED=0 "$GO" build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o bin/goh ./cmd/harness
cd "$HARBOR_DIR"
PYTHONPATH="$ROOT${PYTHONPATH:+:$PYTHONPATH}" exec uv run harbor run \
  -a bench.harbor_agent.goh_agent:Goh -m "$MODEL" -e apple-container \
  --environment-build-timeout-multiplier 2 --agent-setup-timeout-multiplier 2 \
  --verifier-timeout-multiplier 2 \
  -o "$ROOT/bench/results/harbor" "$@"
