#!/usr/bin/env bash
# Operator-initiated macOS arm64 OAuth E2E. Explicit loopback only.
set -euo pipefail
cd "$(dirname "$0")/.."
[[ "$(uname -s)" == "Darwin" ]] || { echo 'macOS is required' >&2; exit 2; }
for program in go codex python3 shasum pgrep; do
  command -v "$program" >/dev/null 2>&1 || { echo "missing required program $program" >&2; exit 2; }
done
root="$(mktemp -d "${TMPDIR:-/tmp}/mcp-interop-codex-oauth.XXXXXX")"
fixture_pid=""
cleanup() {
  if [[ -n "$fixture_pid" ]]; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
  fi
  rm -rf "$root"
}
trap cleanup EXIT
GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.9}" go build -o "$root/mcp-interop" ./cmd/mcp-interop
GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.9}" go build -o "$root/oauth-fixture" ./internal/e2e/oauthfixture
normal_state() {
  local output="$1" name file
  : > "$output"
  for name in '.codex/config.toml' '.codex/auth.json' '.codex/.credentials.json' 'Library/Keychains/login.keychain-db'; do
    file="$HOME/$name"
    if [[ -f "$file" ]]; then
      printf '%s\t%s\t%s\n' "$name" "$(stat -f '%m:%z' "$file")" "$(shasum -a 256 "$file" | awk '{print $1}')" >> "$output"
    else
      printf '%s\tmissing\n' "$name" >> "$output"
    fi
  done
}
normal_state "$root/before-state"
pgrep -x codex 2>/dev/null | sort > "$root/before-pids" || true
find "${TMPDIR:-/tmp}" -maxdepth 1 -type d -name 'mcp-interop-*' -print | sort > "$root/before-sessions" 2>/dev/null || true
"$root/oauth-fixture" --listen 127.0.0.1:0 --ready-file "$root/ready" --log-file "$root/trace" &
fixture_pid=$!
for _ in {1..100}; do
  [[ -s "$root/ready" ]] && break
  kill -0 "$fixture_pid" 2>/dev/null || { echo 'OAuth fixture died before ready' >&2; exit 1; }
  sleep 0.05
done
[[ -s "$root/ready" ]] || { echo 'OAuth fixture not ready' >&2; exit 1; }
endpoint="$(tr -d '\r\n' < "$root/ready")"
[[ "$endpoint" == http://127.0.0.1:*/mcp ]] || { echo 'Refusing non-loopback OAuth fixture' >&2; exit 2; }
rc=0
env -u OPENAI_API_KEY -u CODEX_API_KEY -u ANTHROPIC_API_KEY -u CURSOR_API_KEY -u GEMINI_API_KEY \
  HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 ALL_PROXY=http://127.0.0.1:9 \
  http_proxy=http://127.0.0.1:9 https_proxy=http://127.0.0.1:9 all_proxy=http://127.0.0.1:9 \
  NO_PROXY=127.0.0.1,localhost,::1 no_proxy=127.0.0.1,localhost,::1 \
  MCP_INTEROP_E2E_AUTO_AUTHORIZE_LOOPBACK=1 \
  "$root/mcp-interop" test "$endpoint" --client codex --oauth --timeout 45s --json \
  > "$root/result.json" 2> "$root/stderr" || rc=$?
python3 - "$root/result.json" "$root/trace" "$rc" <<'PY'
import json,sys
result,trace,rc=sys.argv[1:]
assert int(rc)==0,f'Codex OAuth exited {rc}'
reports=json.load(open(result,encoding='utf-8'))
assert len(reports)==1
run=reports[0]
assert run['client_id']=='codex'
assert run['client_version']
stages={row['stage']:row['status'] for row in run['stages']}
assert stages==dict(reach='pass',auth='pass',init='pass',tools='pass'),stages
requests=[json.loads(line) for line in open(trace,encoding='utf-8') if line.strip()]
paths={(str(r.get('method')),str(r.get('path'))) for r in requests}
print('Codex OAuth: exact-client version',run['client_version'],'4/4 PASS')
print('Controlled fixture request types:',','.join(sorted({path for _,path in paths})))
for path,method in (('/register','POST'),('/authorize','GET'),('/token','POST'),('/mcp','POST')):
 assert (method,path) in paths, f'missing direct {method} {path} observation'
for path in (result,trace):
 with open(path,'rb') as f:raw=f.read()
 assert b'fixture-code-' not in raw and b'fixture-token-' not in raw and b'fixture-refresh-' not in raw
print('Codex OAuth: DCR, authorization, token exchange and MCP traffic observed; no fixture secrets persisted')
PY
for _ in {1..35}; do
  pgrep -x codex 2>/dev/null | sort > "$root/after-pids" || true
  cmp -s "$root/before-pids" "$root/after-pids" && break
  sleep 0.1
done
normal_state "$root/after-state"
cmp -s "$root/before-state" "$root/after-state" || { echo 'Normal Codex/Keychain files changed' >&2; exit 1; }
cmp -s "$root/before-pids" "$root/after-pids" || { echo 'Codex subprocess remains' >&2; exit 1; }
find "${TMPDIR:-/tmp}" -maxdepth 1 -type d -name 'mcp-interop-*' -print | sort > "$root/after-sessions" 2>/dev/null || true
cmp -s "$root/before-sessions" "$root/after-sessions" || { echo 'A temporary client session remains' >&2; exit 1; }
echo 'READY: Codex OAuth isolated real-client E2E PASS; normal config/Keychain/process/session unchanged'
