#!/usr/bin/env bash
set -euo pipefail

# Fixture-only layer-5 smoke: this exercises the shipped CLI and existing Codex
# adapter boundary with a synthetic app-server. It is not deployment-specific
# real-client evidence and must never be used to claim a live interoperability PASS.

if [[ $# -ne 1 || ! -x "$1" ]]; then
  echo "usage: $0 <mcp-interop-binary>" >&2
  exit 2
fi

binary="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
workdir="$(mktemp -d)"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT
mkdir -p "$workdir/bin"

cat >"$workdir/bin/codex" <<'FIXTURE'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  --version)
    printf '%s\n' "${MCP_INTEROP_FIXTURE_VERSION:-codex-fixture 1.0.0}"
    exit 0
    ;;
  app-server)
    ;;
  *)
    exit 2
    ;;
esac

while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      printf '%s\n' '{"id":1,"result":{"userAgent":"codex-cli-regression-fixture"}}'
      ;;
    *'"method":"mcpServerStatus/list"'*)
      if [[ "${MCP_INTEROP_FIXTURE_MODE:-pass}" == "pass" ]]; then
        printf '%s\n' '{"id":2,"result":{"data":[{"name":"mcp-interop-target","authStatus":"unsupported","tools":{"fixture_tool":{}}}]}}'
      else
        printf '%s\n' '{"id":2,"result":{"data":[{"name":"mcp-interop-target","authStatus":"unsupported","tools":{}}]}}'
      fi
      ;;
  esac
done
FIXTURE
chmod 700 "$workdir/bin/codex"
export PATH="$workdir/bin:$PATH"

"$binary" maturity --json >"$workdir/maturity.json"
grep -F '"artifact_type": "mcp-interop/adapter-maturity"' "$workdir/maturity.json" >/dev/null
grep -F '"client_id": "codex"' "$workdir/maturity.json" >/dev/null
grep -F '"client_id": "cursor"' "$workdir/maturity.json" >/dev/null
grep -F '"client_id": "antigravity"' "$workdir/maturity.json" >/dev/null
grep -F '"maturity": "beta"' "$workdir/maturity.json" >/dev/null

cat >"$workdir/capability-profile.json" <<'CAPABILITY'
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/capability-profile",
  "context": {
    "observed_at": "2026-08-28T12:00:00Z",
    "deployment_id": "fixture-a",
    "deployment_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "client": {
      "id": "codex",
      "product": "Codex CLI",
      "version": "codex-fixture 1.0.0"
    },
    "platform": {"os": "darwin", "arch": "arm64"},
    "runtime": {
      "mcp_interop_version": "dev",
      "mcp_interop_commit": "deadbeef",
      "go_version": "go1.26.6"
    },
    "auth_mode": "default",
    "evidence_provenance": {"kind": "real_client_adapter", "adapter_id": "codex"}
  },
  "capabilities": [
    {
      "capability_id": "resources",
      "state": "pass",
      "evidence_kind": "client_protocol",
      "evidence_id": "resources.list.response"
    },
    {
      "capability_id": "tasks",
      "state": "untested",
      "evidence_kind": "none"
    }
  ]
}
CAPABILITY
"$binary" capability validate "$workdir/capability-profile.json" --json >"$workdir/capability-validated.json"
grep -F '"artifact_type": "mcp-interop/capability-profile"' "$workdir/capability-validated.json" >/dev/null
grep -F '"state": "pass"' "$workdir/capability-validated.json" >/dev/null
grep -F '"state": "untested"' "$workdir/capability-validated.json" >/dev/null

endpoint='https://example.com/mcp?api_key=cli-smoke-secret&tenant=fixture'
old="$workdir/old.json"
new="$workdir/new.json"
regression="$workdir/regression.json"
malformed="$workdir/malformed.json"
protected_old="$workdir/protected-old.json"
protected_new="$workdir/protected-new.json"
protected_secret='cli-protected-path-secret'
protected_endpoint="https://example.com/mcp/$protected_secret?token=protected-query-secret"

MCP_INTEROP_FIXTURE_VERSION='codex-fixture 1.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" test "$endpoint" --client codex --output "$old" >"$workdir/old.txt"
MCP_INTEROP_FIXTURE_VERSION='codex-fixture 2.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" test "$endpoint" --client codex --output "$new" >"$workdir/new.txt"

"$binary" compare "$old" "$new" >"$workdir/version-compare.txt"
"$binary" compare "$old" "$new" --fail-on-regression >"$workdir/version-gate.txt"

grep -F 'codex-fixture 1.0.0' "$old" >/dev/null
grep -F 'codex-fixture 2.0.0' "$new" >/dev/null
if grep -F 'cli-smoke-secret' "$old" "$new" >/dev/null; then
  echo "portable artifact leaked endpoint query secret" >&2
  exit 1
fi

MCP_INTEROP_FIXTURE_VERSION='codex-fixture 1.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" test "$protected_endpoint" --client codex --output "$protected_old" \
  --deployment-id production-a >"$workdir/protected-old.txt"
MCP_INTEROP_FIXTURE_VERSION='codex-fixture 2.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" test "$protected_endpoint" --client codex --output "$protected_new" \
  --deployment-id production-a --json >"$workdir/protected-new.txt"

grep -F '"schema_version": 2' "$protected_old" >/dev/null
grep -F '"identity_kind": "deployment_id"' "$protected_old" >/dev/null
grep -F '"identity": "production-a"' "$protected_old" >/dev/null
"$binary" compare "$protected_old" "$protected_new" >"$workdir/protected-compare.txt"
if grep -R -F "$protected_secret" "$protected_old" "$protected_new" "$workdir/protected-old.txt" "$workdir/protected-new.txt" "$workdir/protected-compare.txt" >/dev/null; then
  echo "schema v2 protected-path output leaked endpoint path secret" >&2
  exit 1
fi

set +e
"$binary" compare "$old" "$protected_new" >"$workdir/schema-mismatch.out" 2>"$workdir/schema-mismatch.err"
schema_mismatch_exit=$?
set -e
if [[ $schema_mismatch_exit -ne 2 ]]; then
  echo "cross-schema compare exit=$schema_mismatch_exit, want 2" >&2
  exit 1
fi
grep -F 'artifact schema mismatch' "$workdir/schema-mismatch.err" >/dev/null

MCP_INTEROP_FIXTURE_VERSION='codex-fixture 2.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" test "$endpoint" --client codex --json >"$workdir/legacy.json"
if ! grep -Eq '^[[:space:]]*\[' "$workdir/legacy.json"; then
  echo "legacy test --json output is no longer an array" >&2
  exit 1
fi
if grep -F '"schema_version"' "$workdir/legacy.json" >/dev/null; then
  echo "portable artifact fields leaked into legacy test --json output" >&2
  exit 1
fi
if grep -F 'cli-smoke-secret' "$workdir/legacy.json" >/dev/null; then
  echo "legacy test --json output leaked endpoint query secret" >&2
  exit 1
fi

set +e
MCP_INTEROP_FIXTURE_VERSION='codex-fixture 3.0.0' MCP_INTEROP_FIXTURE_MODE=unknown \
  "$binary" test "$endpoint" --client codex --output "$regression" >"$workdir/regression.txt" 2>"$workdir/regression.err"
regression_test_exit=$?
set -e
if [[ $regression_test_exit -ne 1 ]]; then
  echo "incomplete fixture test exit=$regression_test_exit, want 1" >&2
  exit 1
fi

"$binary" compare "$old" "$regression" >"$workdir/regression-compare.txt"
set +e
"$binary" compare "$old" "$regression" --fail-on-regression >"$workdir/regression-gate.txt" 2>"$workdir/regression-gate.err"
regression_gate_exit=$?
set -e
if [[ $regression_gate_exit -ne 1 ]]; then
  echo "regression gate exit=$regression_gate_exit, want 1" >&2
  exit 1
fi

grep -F 'PASS_TO_UNKNOWN' "$workdir/regression-gate.txt" >/dev/null

suite_manifest="$workdir/suite.json"
cat >"$suite_manifest" <<'SUITE'
{
  "schema_version": 1,
  "execution_context": "trusted_real_client",
  "targets": [
    {
      "id": "production-a",
      "endpoint": {
        "source": "environment",
        "variable": "MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A"
      },
      "deployment_id": "production-a",
      "clients": [{"id": "codex", "auth": "none"}]
    }
  ]
}
SUITE
export MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A="$protected_endpoint"
suite_baseline_source="$workdir/suite-baseline-source"
suite_attempt="$workdir/suite-attempt"
accepted_baseline="$workdir/accepted-baseline"

MCP_INTEROP_FIXTURE_VERSION='codex-fixture 4.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" suite run "$suite_manifest" --output-dir "$suite_baseline_source" >/dev/null
"$binary" baseline create "$suite_baseline_source" --output-dir "$accepted_baseline" \
  --json >"$workdir/baseline-create.json"
MCP_INTEROP_FIXTURE_VERSION='codex-fixture 5.0.0' MCP_INTEROP_FIXTURE_MODE=pass \
  "$binary" suite run "$suite_manifest" --output-dir "$suite_attempt" >/dev/null
"$binary" baseline compare "$accepted_baseline" "$suite_attempt" --fail-on-regression \
  >"$workdir/baseline-compare.txt"
grep -F '"artifact_type": "mcp-interop/suite-baseline"' "$workdir/baseline-create.json" >/dev/null
grep -F 'DECISION' "$workdir/baseline-compare.txt" >/dev/null
grep -F 'CLEAN' "$workdir/baseline-compare.txt" >/dev/null

MCP_INTEROP_FIXTURE_VERSION='codex-fixture 5.0.0' \
  "$binary" compatibility query --client codex --target production-a \
  --deployment-id production-a --baseline "$accepted_baseline" \
  --observation "$suite_attempt" --json >"$workdir/compatibility-tested.json"
grep -F '"state": "tested"' "$workdir/compatibility-tested.json" >/dev/null
MCP_INTEROP_FIXTURE_VERSION='codex-fixture 6.0.0' \
  "$binary" compatibility query --client codex --target production-a \
  --deployment-id production-a --baseline "$accepted_baseline" \
  --observation "$suite_attempt" --json >"$workdir/compatibility-untested.json"
grep -F '"state": "untested"' "$workdir/compatibility-untested.json" >/dev/null
if grep -F "$protected_secret" "$workdir/compatibility-tested.json" "$workdir/compatibility-untested.json" >/dev/null; then
  echo "compatibility query leaked protected endpoint path" >&2
  exit 1
fi

set +e
"$binary" baseline create "$suite_attempt" --output-dir "$accepted_baseline" \
  >"$workdir/baseline-overwrite.out" 2>"$workdir/baseline-overwrite.err"
baseline_overwrite_exit=$?
set -e
if [[ $baseline_overwrite_exit -ne 2 ]]; then
  echo "baseline overwrite exit=$baseline_overwrite_exit, want 2" >&2
  exit 1
fi
grep -F 'already exists' "$workdir/baseline-overwrite.err" >/dev/null

printf '%s\n' '{"schema_version":1' >"$malformed"
set +e
"$binary" compare "$malformed" "$old" >"$workdir/malformed.out" 2>"$workdir/malformed.err"
malformed_exit=$?
set -e
if [[ $malformed_exit -ne 2 ]]; then
  echo "malformed artifact exit=$malformed_exit, want 2" >&2
  exit 1
fi

if grep -R -F 'cli-smoke-secret' "$workdir" >/dev/null || grep -R -F "$protected_secret" "$workdir" >/dev/null; then
  echo "CLI smoke output/log leaked endpoint secret" >&2
  exit 1
fi

printf '%s\n' 'CLI regression artifact smoke: PASS'
