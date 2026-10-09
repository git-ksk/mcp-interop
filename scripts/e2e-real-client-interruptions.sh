#!/usr/bin/env bash
# Privileged, operator-initiated macOS arm64 acceptance only. Not a hosted PR test.
set -euo pipefail
cd "$(dirname "$0")/.."
[[ "$(uname -s)" == "Darwin" ]] || { echo 'macOS is required' >&2; exit 2; }
for program in go python3 codex cursor-agent agy; do
  command -v "$program" >/dev/null 2>&1 || { echo "Missing required executable: $program" >&2; exit 2; }
done
root="$(mktemp -d "${TMPDIR:-/tmp}/mcp-interop-interrupt-e2e.XXXXXX")"
trap 'rm -rf "$root"' EXIT
GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.9}" go build -o "$root/mcp-interop" ./cmd/mcp-interop
python3 scripts/e2e-real-client-interruptions.py "$root/mcp-interop"
