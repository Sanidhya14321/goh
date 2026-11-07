#!/bin/sh
# Exercise install.sh against a local server that mirrors a release layout:
# a clean install, then a tampered binary that must be refused.
set -eu
cd "$(dirname "$0")/.."
[ -f dist/SHA256SUMS ] || { echo "run make release first" >&2; exit 1; }
work="$(mktemp -d)"
trap 'kill $server 2>/dev/null || true; rm -rf "$work"' EXIT
cp -R dist "$work/good"
mkdir "$work/bad"; cp dist/* "$work/bad/" 2>/dev/null || true
printf 'tampered' >> "$work/bad/goh-$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
port=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 -m http.server "$port" --bind 127.0.0.1 --directory "$work" >/dev/null 2>&1 &
server=$!
for i in $(seq 50); do curl -fs "http://127.0.0.1:$port/good/SHA256SUMS" >/dev/null && break; sleep .1; done

GOH_BASE_URL="http://127.0.0.1:$port/good" GOH_INSTALL_DIR="$work/bin" sh scripts/install.sh
installed="$("$work/bin/goh" version)"
expected="$(dist/goh-$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') version)"
[ "$installed" = "$expected" ] || { echo "FAIL: installed $installed, expected $expected"; exit 1; }
echo "PASS clean install: $installed"

rm -rf "$work/bin2"
if GOH_BASE_URL="http://127.0.0.1:$port/bad" GOH_INSTALL_DIR="$work/bin2" sh scripts/install.sh 2>"$work/err"; then
	echo "FAIL: tampered binary was installed"; exit 1
fi
grep -q 'checksum mismatch' "$work/err" || { echo "FAIL: unexpected error: $(cat "$work/err")"; exit 1; }
[ ! -e "$work/bin2/goh" ] || { echo "FAIL: tampered binary left on disk"; exit 1; }
echo "PASS tampered download refused: $(cat "$work/err")"

# A binary that matches its checksum but cannot run must not replace a
# working installation.
mkdir "$work/broken"; cp dist/* "$work/broken/" 2>/dev/null || true
name="goh-$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
printf '#!/bin/sh\nexit 3\n' > "$work/broken/$name"
(cd "$work/broken" && { command -v shasum >/dev/null && shasum -a 256 "$name" || sha256sum "$name"; } > SHA256SUMS)
if GOH_BASE_URL="http://127.0.0.1:$port/broken" GOH_INSTALL_DIR="$work/bin" sh scripts/install.sh 2>"$work/err"; then
	echo "FAIL: non-running binary was installed"; exit 1
fi
[ "$("$work/bin/goh" version)" = "$installed" ] || { echo "FAIL: working installation was replaced"; exit 1; }
echo "PASS non-running binary refused; existing $installed kept"
