#!/bin/sh
# Install goh on macOS or Linux:
#   curl -fsSL https://github.com/Sanidhya14321/goharness/releases/latest/download/install.sh | sh
# Environment:
#   GOH_VERSION   release tag to install (default: latest)
#   GOH_BASE_URL  directory that holds the binaries and SHA256SUMS (overrides GOH_VERSION)
#   GOH_INSTALL_DIR  destination directory (default: $HOME/.local/bin)
set -eu

repo="Sanidhya14321/goharness"
version="${GOH_VERSION:-latest}"
if [ -n "${GOH_BASE_URL:-}" ]; then
	base="$GOH_BASE_URL"
elif [ "$version" = latest ]; then
	base="https://github.com/$repo/releases/latest/download"
else
	base="https://github.com/$repo/releases/download/$version"
fi
dest="${GOH_INSTALL_DIR:-$HOME/.local/bin}"

fail() { echo "goh install: $*" >&2; exit 1; }

case "$(uname -s)" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) fail "unsupported system $(uname -s); on Windows use install.ps1" ;;
esac
case "$(uname -m)" in
	arm64 | aarch64) arch=arm64 ;;
	x86_64 | amd64) arch=amd64 ;;
	*) fail "unsupported architecture $(uname -m)" ;;
esac
artifact="goh-$os-$arch"

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL --proto '=https,http' -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -q -O "$2" "$1"; }
else
	fail "curl or wget is required"
fi
if command -v sha256sum >/dev/null 2>&1; then
	digest() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
	digest() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
	fail "sha256sum or shasum is required to verify the download"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
echo "Downloading $artifact from $base"
fetch "$base/$artifact" "$tmp/$artifact" || fail "download failed: $base/$artifact"
fetch "$base/SHA256SUMS" "$tmp/SHA256SUMS" || fail "download failed: $base/SHA256SUMS"

# Match the exact file name; refuse if it is missing or duplicated.
expected="$(awk -v name="$artifact" '$2 == name || $2 == "*" name { print $1 }' "$tmp/SHA256SUMS")"
[ -n "$expected" ] || fail "no checksum for $artifact in SHA256SUMS"
[ "$(printf '%s\n' "$expected" | wc -l | tr -d ' ')" = 1 ] || fail "ambiguous checksum entries for $artifact"
actual="$(digest "$tmp/$artifact")"
[ "$actual" = "$expected" ] || fail "checksum mismatch for $artifact (expected $expected, got $actual); nothing was installed"

mkdir -p "$dest"
chmod 0755 "$tmp/$artifact"
# Install atomically: a running goh is never left half-written.
mv -f "$tmp/$artifact" "$dest/goh.tmp.$$"
mv -f "$dest/goh.tmp.$$" "$dest/goh"
echo "Installed $("$dest/goh" version 2>/dev/null || echo goh) to $dest/goh (sha256 $actual)"
case ":$PATH:" in
	*":$dest:"*) ;;
	*) echo "Add $dest to your PATH, for example: export PATH=\"$dest:\$PATH\"" ;;
esac
