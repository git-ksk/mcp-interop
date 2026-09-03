#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release="$root/.github/workflows/release.yml"
real="$root/.github/workflows/e2e-real-macos.yml"
ci="$root/.github/workflows/ci.yml"

fail() {
  echo "security contract gate: $*" >&2
  exit 1
}

require() {
  local file="$1" pattern="$2"
  grep -Fq -- "$pattern" "$file" || fail "$(basename "$file") missing required contract: $pattern"
}

forbid() {
  local file="$1" pattern="$2"
  if grep -Fq -- "$pattern" "$file"; then
    fail "$(basename "$file") contains forbidden trigger/contract: $pattern"
  fi
}

require_pinned_actions() {
  local file="$1" line
  while IFS= read -r line; do
    [[ "$line" =~ uses:[[:space:]]+[^@[:space:]]+@[0-9a-f]{40}([[:space:]]|$) ]] || \
      fail "$(basename "$file") has non-SHA-pinned action: $line"
  done < <(grep -E '^[[:space:]]+uses:' "$file" || true)
}

# Tagged releases: trusted main ancestry, security gates, provenance and pinned actions.
require "$release" 'tags:'
require "$release" '- "v*"'
require "$release" 'contents: write'
require "$release" 'id-token: write'
require "$release" 'attestations: write'
require "$release" 'git merge-base --is-ancestor "$GITHUB_SHA" origin/main'
require "$release" 'govulncheck'
require "$release" 'bash scripts/test-real-client-e2e-guard.sh'
require "$release" "go test ./internal/e2e/fixture -run '^TestToolOAuthInsufficientScopeReleaseGate$' -count=1"
require "$release" 'bash scripts/test-security-contract.sh'
require "$release" 'bash scripts/build-release.sh'
require "$release" 'actions/attest@'
require "$release" '--verify-tag'
forbid "$release" 'pull_request_target:'
forbid "$release" 'pull_request:'
require_pinned_actions "$release"

# Privileged real-client E2E: manual, exact main workflow/SHA, environment gate, no checkout credentials.
require "$real" 'workflow_dispatch:'
require "$real" 'contents: read'
require "$real" "github.repository == 'git-ksk/mcp-interop'"
require "$real" "github.ref == 'refs/heads/main'"
require "$real" "github.workflow_ref == 'git-ksk/mcp-interop/.github/workflows/e2e-real-macos.yml@refs/heads/main'"
require "$real" 'github.workflow_sha == github.sha'
require "$real" 'environment: real-client-e2e'
require "$real" 'runs-on: [self-hosted, macOS, ARM64, mcp-interop-e2e]'
require "$real" 'persist-credentials: false'
require "$real" 'bash scripts/guard-real-client-e2e.sh'
require "$real" 'Run isolated real-client E2E'
forbid "$real" 'pull_request_target:'
forbid "$real" 'pull_request:'
require_pinned_actions "$real"

# Ordinary PR CI stays unprivileged and continuously checks the security contract.
require "$ci" 'pull_request:'
require "$ci" 'contents: read'
require "$ci" 'govulncheck'
require "$ci" 'bash scripts/test-real-client-e2e-guard.sh'
require "$ci" 'bash scripts/test-security-contract.sh'
require "$ci" 'native archive smoke ('
require "$ci" 'macos-15-intel'
require "$ci" 'windows-amd64'
require "$release" 'needs: [build, native-smoke]'
require "$release" 'macos-15-intel'
require "$release" 'Download natively verified release archives'
forbid "$ci" 'pull_request_target:'
require_pinned_actions "$ci"

echo "security contract gates: PASS"
