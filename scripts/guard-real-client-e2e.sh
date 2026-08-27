#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "real-client E2E trust guard: $*" >&2
  exit 2
}

require_equal() {
  local name="$1" expected="$2" actual="${!1:-}"
  [[ "$actual" == "$expected" ]] || fail "$name must be $expected"
}

require_equal GITHUB_EVENT_NAME workflow_dispatch
require_equal GITHUB_REPOSITORY git-ksk/mcp-interop
require_equal GITHUB_REF refs/heads/main
require_equal GITHUB_WORKFLOW_REF git-ksk/mcp-interop/.github/workflows/e2e-real-macos.yml@refs/heads/main
require_equal RUNNER_ENVIRONMENT self-hosted
require_equal RUNNER_OS macOS
require_equal RUNNER_ARCH ARM64

[[ "${GITHUB_SHA:-}" =~ ^[0-9a-f]{40}$ ]] || fail "GITHUB_SHA must be a full lowercase commit SHA"
[[ "${GITHUB_WORKFLOW_SHA:-}" =~ ^[0-9a-f]{40}$ ]] || fail "GITHUB_WORKFLOW_SHA must be a full lowercase commit SHA"
[[ "$GITHUB_WORKFLOW_SHA" == "$GITHUB_SHA" ]] || fail "workflow SHA must equal checked-out event SHA"
[[ "${GITHUB_RUN_ID:-}" =~ ^[0-9]+$ ]] || fail "GITHUB_RUN_ID is required"
[[ "${GITHUB_RUN_ATTEMPT:-}" =~ ^[0-9]+$ ]] || fail "GITHUB_RUN_ATTEMPT is required"
[[ -n "${GITHUB_ACTOR:-}" ]] || fail "GITHUB_ACTOR is required"
[[ -n "${GITHUB_TRIGGERING_ACTOR:-}" ]] || fail "GITHUB_TRIGGERING_ACTOR is required"

clients_csv="${MCP_INTEROP_CLIENTS:-}"
[[ -n "$clients_csv" ]] || fail "MCP_INTEROP_CLIENTS is required"
IFS=',' read -r -a clients <<< "$clients_csv"
seen=","
for client in "${clients[@]}"; do
  case "$client" in
    codex|cursor|antigravity) ;;
    *) fail "unsupported client selection" ;;
  esac
  case "$seen" in
    *,"$client",*) fail "duplicate client selection" ;;
  esac
  seen="${seen}${client},"
done
[[ "${#clients[@]}" -gt 0 ]] || fail "at least one client is required"

if [[ "${1:-}" == "--provenance" ]]; then
  [[ "$#" -eq 2 && -n "$2" ]] || fail "--provenance requires an output path"
  output="$2"
  umask 077
  mkdir -p "$(dirname "$output")"
  python3 - "$output" <<'PY'
import json, os, shutil, subprocess, sys

selected = os.environ["MCP_INTEROP_CLIENTS"].split(",")
commands = {
    "codex": ["codex", "--version"],
    "antigravity": ["agy", "--version"],
}
versions = {}
for client in selected:
    if client == "cursor":
        exe = shutil.which("cursor-agent") or shutil.which("agent")
        if not exe:
            raise SystemExit("selected Cursor CLI is not installed")
        cmd = [exe, "--version"]
    else:
        exe = shutil.which(commands[client][0])
        if not exe:
            raise SystemExit(f"selected {client} CLI is not installed")
        cmd = [exe, "--version"]
    proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=10, check=False)
    if proc.returncode != 0:
        raise SystemExit(f"{client} version check failed")
    first = (proc.stdout.strip().splitlines() or [""])[0][:512]
    if not first:
        raise SystemExit(f"{client} version output is empty")
    versions[client] = first

value = {
    "schema_version": 1,
    "artifact_type": "mcp-interop/real-client-e2e-provenance",
    "repository": os.environ["GITHUB_REPOSITORY"],
    "ref": os.environ["GITHUB_REF"],
    "sha": os.environ["GITHUB_SHA"],
    "workflow_ref": os.environ["GITHUB_WORKFLOW_REF"],
    "workflow_sha": os.environ["GITHUB_WORKFLOW_SHA"],
    "run_id": os.environ["GITHUB_RUN_ID"],
    "run_attempt": int(os.environ["GITHUB_RUN_ATTEMPT"]),
    "actor": os.environ["GITHUB_ACTOR"],
    "triggering_actor": os.environ["GITHUB_TRIGGERING_ACTOR"],
    "clients": selected,
    "client_versions": versions,
}
with open(sys.argv[1], "w", encoding="utf-8") as f:
    json.dump(value, f, indent=2, sort_keys=True)
    f.write("\n")
PY
elif [[ "$#" -ne 0 ]]; then
  fail "unsupported guard arguments"
fi
