#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
guard="$repo_root/scripts/guard-real-client-e2e.sh"
sha="0123456789abcdef0123456789abcdef01234567"

if [[ "${1:-}" == "__single" ]]; then
  shift
  env \
    GITHUB_EVENT_NAME="${GITHUB_EVENT_NAME_TEST:-workflow_dispatch}" \
    GITHUB_REPOSITORY="${GITHUB_REPOSITORY_TEST:-git-ksk/mcp-interop}" \
    GITHUB_REF="${GITHUB_REF_TEST:-refs/heads/main}" \
    GITHUB_WORKFLOW_REF="${GITHUB_WORKFLOW_REF_TEST:-git-ksk/mcp-interop/.github/workflows/e2e-real-macos.yml@refs/heads/main}" \
    GITHUB_SHA="${GITHUB_SHA_TEST:-$sha}" \
    GITHUB_WORKFLOW_SHA="${GITHUB_WORKFLOW_SHA_TEST:-$sha}" \
    GITHUB_RUN_ID=12345 GITHUB_RUN_ATTEMPT=1 GITHUB_ACTOR=trusted-actor GITHUB_TRIGGERING_ACTOR=trusted-actor \
    RUNNER_ENVIRONMENT="${RUNNER_ENVIRONMENT_TEST:-self-hosted}" RUNNER_OS="${RUNNER_OS_TEST:-macOS}" RUNNER_ARCH="${RUNNER_ARCH_TEST:-ARM64}" \
    MCP_INTEROP_CLIENTS="${MCP_INTEROP_CLIENTS_TEST:-codex,cursor,antigravity}" \
    "$guard" "$@"
  exit $?
fi

run_guard() {
  env \
    GITHUB_EVENT_NAME="${GITHUB_EVENT_NAME_TEST:-workflow_dispatch}" \
    GITHUB_REPOSITORY="${GITHUB_REPOSITORY_TEST:-git-ksk/mcp-interop}" \
    GITHUB_REF="${GITHUB_REF_TEST:-refs/heads/main}" \
    GITHUB_WORKFLOW_REF="${GITHUB_WORKFLOW_REF_TEST:-git-ksk/mcp-interop/.github/workflows/e2e-real-macos.yml@refs/heads/main}" \
    GITHUB_SHA="${GITHUB_SHA_TEST:-$sha}" \
    GITHUB_WORKFLOW_SHA="${GITHUB_WORKFLOW_SHA_TEST:-$sha}" \
    GITHUB_RUN_ID=12345 \
    GITHUB_RUN_ATTEMPT=1 \
    GITHUB_ACTOR=trusted-actor \
    GITHUB_TRIGGERING_ACTOR=trusted-actor \
    RUNNER_ENVIRONMENT="${RUNNER_ENVIRONMENT_TEST:-self-hosted}" \
    RUNNER_OS="${RUNNER_OS_TEST:-macOS}" \
    RUNNER_ARCH="${RUNNER_ARCH_TEST:-ARM64}" \
    MCP_INTEROP_CLIENTS="${MCP_INTEROP_CLIENTS_TEST:-codex,cursor,antigravity}" \
    "$guard" "$@"
}

expect_reject() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "guard unexpectedly accepted: $label" >&2
    exit 1
  fi
}

run_guard >/dev/null
expect_reject pull-request env GITHUB_EVENT_NAME_TEST=pull_request bash "$0" __single
expect_reject feature-ref env GITHUB_REF_TEST=refs/heads/feature bash "$0" __single
expect_reject wrong-repo env GITHUB_REPOSITORY_TEST=attacker/fork bash "$0" __single
expect_reject workflow-feature-ref env GITHUB_WORKFLOW_REF_TEST=git-ksk/mcp-interop/.github/workflows/e2e-real-macos.yml@refs/heads/feature bash "$0" __single
expect_reject workflow-sha-mismatch env GITHUB_WORKFLOW_SHA_TEST=1111111111111111111111111111111111111111 bash "$0" __single
expect_reject hosted-runner env RUNNER_ENVIRONMENT_TEST=github-hosted bash "$0" __single
expect_reject invalid-client env MCP_INTEROP_CLIENTS_TEST='codex;uname' bash "$0" __single

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin"
for name in codex cursor-agent agy; do
  cat > "$work/bin/$name" <<SCRIPT
#!/usr/bin/env bash
printf '%s\\n' '$name test-version'
SCRIPT
  chmod +x "$work/bin/$name"
done
PATH="$work/bin:$PATH" run_guard --provenance "$work/provenance.json"
python3 - "$work/provenance.json" "$sha" <<'PY'
import json, sys
value=json.load(open(sys.argv[1]))
assert value["sha"] == sys.argv[2]
assert value["workflow_sha"] == sys.argv[2]
assert value["ref"] == "refs/heads/main"
assert value["clients"] == ["codex", "cursor", "antigravity"]
assert set(value["client_versions"]) == {"codex", "cursor", "antigravity"}
PY

workflow="$repo_root/.github/workflows/e2e-real-macos.yml"
grep -Fq 'workflow_dispatch:' "$workflow"
! grep -Eq '^[[:space:]]*(pull_request|pull_request_target|push):' "$workflow"
grep -Fq "github.ref == 'refs/heads/main'" "$workflow"
grep -Fq "github.repository == 'git-ksk/mcp-interop'" "$workflow"
grep -Fq "github.workflow_sha == github.sha" "$workflow"
grep -Fq 'environment: real-client-e2e' "$workflow"
grep -Fq 'persist-credentials: false' "$workflow"
grep -Fq 'type: choice' "$workflow"
grep -Fq 'bash scripts/guard-real-client-e2e.sh' "$workflow"
! grep -Fq 'self-hosted' "$repo_root/.github/workflows/ci.yml"

echo "real-client E2E trust guard tests: PASS"
exit 0

# Recursive single-case entry point used by expect_reject.
