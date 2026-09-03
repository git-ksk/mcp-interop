#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 || ! -x "$1" ]]; then
  echo "usage: $0 <mcp-interop-binary> <expected-version>" >&2
  exit 2
fi

binary="$1"
expected="$2"
"$binary" version | grep -F "$expected" >/dev/null
bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/cli-regression-smoke.sh" "$binary"
echo "native packaged archive smoke: PASS ($expected)"
