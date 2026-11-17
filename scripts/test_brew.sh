#!/bin/sh
# Installs the generated Homebrew formula from a temporary local tap against
# the files in dist/ (file:// URLs), checks it, then removes it again.
# It changes the machine's Homebrew while it runs, so CI runs it; locally,
# use a throwaway machine or the homebrew/brew container.
#
# Usage: make release VERSION=X && sh scripts/test_brew.sh X
set -eu
version=${1:?usage: scripts/test_brew.sh VERSION}
export HOMEBREW_NO_AUTO_UPDATE=1 HOMEBREW_NO_INSTALL_FROM_API=1 HOMEBREW_NO_ANALYTICS=1 HOMEBREW_NO_ENV_HINTS=1
dist=$(cd dist && pwd)
sh scripts/packaging.sh "$version" "$dist" "file://$dist"
tap=goh-test/local
brew tap-new --no-git "$tap" >/dev/null
cleanup() {
    brew uninstall --force "$tap/goh" >/dev/null 2>&1 || true
    brew untap "$tap" >/dev/null 2>&1 || true
}
trap cleanup EXIT
cp "$dist/goh.rb" "$(brew --repository "$tap")/Formula/goh.rb"
brew install "$tap/goh"
brew test "$tap/goh"
installed=$(brew --prefix)/bin/goh
test "$("$installed" version)" = "$version"
echo "PASS brew install: $("$installed" version)"
if "$installed" upgrade >/dev/null 2>"$dist/upgrade.err"; then
    echo "goh upgrade replaced a Homebrew copy" >&2
    exit 1
fi
grep -q 'brew upgrade goh' "$dist/upgrade.err"
echo "PASS goh upgrade defers to brew"
