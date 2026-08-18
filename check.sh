#!/usr/bin/env bash
# fails if Go is missing, then runs go run . check

set -u
cd "$(dirname "$0")"

failed=0

ok()   { printf "  ok    %-14s %s\n" "$1" "$2"; }
warn() { printf "  warn  %-14s %s\n" "$1" "$2"; }
fail() { printf "  FAIL  %-14s %s\n" "$1" "$2"; failed=$((failed + 1)); }

echo "cdnear preflight (is Go installed?)"
echo

if ! command -v go >/dev/null 2>&1; then
	fail "go binary" "not found on PATH"
	echo
	echo "Install Go, then reopen the terminal and run ./check.sh again."
	echo "  macOS (Homebrew):   brew install go"
	echo "  official installer: https://go.dev/dl/"
	echo
	exit 1
fi

ok "go binary" "$(command -v go)"

if ! go version >/dev/null 2>&1; then
	fail "go version" "go is on PATH but 'go version' failed"
else
	ok "go version" "$(go version)"
fi

if ver="$(go env GOVERSION 2>/dev/null)"; then
	minor="$(printf '%s\n' "$ver" | sed -n 's/^go1\.\([0-9][0-9]*\).*/\1/p')"
	if [ -n "$minor" ] && [ "$minor" -lt 21 ]; then
		fail "go too old" "$ver — need Go 1.21 or newer from https://go.dev/dl/"
	fi
fi

if [ ! -f go.mod ] || [ ! -f main.go ]; then
	fail "project files" "go.mod / main.go missing — run this from the CDNEAR folder"
else
	ok "project files" "in the CDNEAR folder"
fi

if [ ! -f go.mod ]; then
	echo
	exit 1
fi

echo
echo "running  go run . check"
echo

if ! go run . check; then
	exit 1
fi

if [ "$failed" -ne 0 ]; then
	exit 1
fi
