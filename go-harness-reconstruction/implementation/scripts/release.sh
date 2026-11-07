#!/bin/sh
set -eu
: "${GO:=go}"
: "${VERSION:=0.1.0-dev}"
mkdir -p dist
set --
for platform in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64 windows/arm64; do
    target_os=${platform%/*}
    target_arch=${platform#*/}
    artifact="goh-$target_os-$target_arch"
    if [ "$target_os" = windows ]; then artifact="$artifact.exe"; fi
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" "$GO" build -trimpath \
      -ldflags "-s -w -X main.version=$VERSION" -o "dist/$artifact" ./cmd/harness
    set -- "$@" "$artifact"
done
cp LICENSE THIRD_PARTY_NOTICES DEPENDENCIES.md README.md VERIFICATION.md BENCHMARKS.md config.example.json dist/
cp ../ORIGIN_LEDGER.md dist/
cp ../specs/COMPATIBILITY_MATRIX.md dist/
cp -R licenses dist/
cp scripts/install.sh scripts/install.ps1 dist/
# shasum on macOS, sha256sum on Linux; identical output format.
if command -v shasum >/dev/null 2>&1; then sum="shasum -a 256"; else sum="sha256sum"; fi
(cd dist && $sum "$@" > SHA256SUMS && $sum -c SHA256SUMS >/dev/null)
"$GO" version -m dist/goh-darwin-arm64 > dist/BUILD_INFO.txt
