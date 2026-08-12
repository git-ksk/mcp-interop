#!/usr/bin/env bash
set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1
[[ "$(uname -s)" == "Darwin" ]] || { echo "Antigravity OAuth E2E requires macOS" >&2; exit 2; }
command -v agy >/dev/null 2>&1 || { echo "Antigravity CLI missing" >&2; exit 2; }

work_root="$(mktemp -d "${TMPDIR:-/tmp}/mcp-antigravity-oauth.XXXXXX")" || exit 1
interop_bin="$work_root/mcp-interop"
fixture_bin="$work_root/oauth-fixture"
ready="$work_root/ready"
trace="$work_root/fixture.jsonl"
result="$work_root/result.json"
terminal="$work_root/terminal-private.log"
input_fifo="$work_root/input"
code_file="$work_root/auth-code"
fixture_pid=""
interop_pid=""
cleanup() {
  kill "${interop_pid:-}" "${fixture_pid:-}" 2>/dev/null || true
  wait "${interop_pid:-}" "${fixture_pid:-}" 2>/dev/null || true
  /usr/bin/osascript -e 'tell application "Safari" to quit' >/dev/null 2>&1 || true
  [[ "${MCP_INTEROP_KEEP_E2E_TMP:-0}" == "1" ]] || rm -rf "$work_root"
}
trap cleanup EXIT INT TERM

snapshot() {
  local out="$1"
  : > "$out"
  for path in \
    "$HOME/.gemini/config/mcp_config.json" \
    "$HOME/.gemini/antigravity/mcp_oauth_tokens.json" \
    "$HOME/.gemini/antigravity-cli/settings.json" \
    "$HOME/Library/Keychains/login.keychain-db"; do
    if [[ -f "$path" ]]; then
      printf '%s\t%s\t%s\n' "$path" "$(stat -f '%m:%z' "$path")" "$(shasum -a 256 "$path" | awk '{print $1}')" >> "$out"
    else
      printf '%s\tmissing\n' "$path" >> "$out"
    fi
  done
  sort -o "$out" "$out"
}

fixture_trace() {
  echo "--- secret-free OAuth fixture trace ---" >&2
  if [[ -s "$trace" ]]; then cat "$trace" >&2; else echo "(no fixture requests observed)" >&2; fi
}

agy_pids() {
  pgrep -x agy 2>/dev/null | sort -n | tr '\n' ' ' || true
}

wait_for_agy_pids() {
  local expected="$1"
  local deadline=$((SECONDS + 3))
  while true; do
    [[ "$(agy_pids)" == "$expected" ]] && return 0
    [[ "$SECONDS" -ge "$deadline" ]] && return 1
    sleep 0.05
  done
}

safe_terminal_state() {
  echo "--- secret-safe Antigravity terminal state ---" >&2
  python3 - "$terminal" <<'PY'
import re,sys
try:
    data=open(sys.argv[1],'rb').read().decode('utf-8','ignore')
except FileNotFoundError:
    raise SystemExit(0)
data=re.sub(r'\x1b\[[0-?]*[ -/]*[@-~]','',data)
data=re.sub(r'\x1b\][^\x07]*(?:\x07|\x1b\\)','',data)
data=re.sub(r'https?://\S+','[URL]',data)
data=re.sub(r'fixture-(?:code|token|refresh)-[A-Za-z0-9_-]+','[SECRET]',data)
data=re.sub(r'\b[A-Za-z0-9_-]{32,}\b','[LONG_VALUE]',data)
seen=set()
for raw in data.splitlines():
    line=' '.join(raw.split())
    low=line.lower()
    if line and any(k in low for k in ('mcp-interop-target','tool','authenticated','authentication','connected','ready','success')):
        line=line[:700]
        if line not in seen:
            print(line)
            seen.add(line)
PY
}

go build -o "$interop_bin" ./cmd/mcp-interop || exit 1
go build -o "$fixture_bin" ./internal/e2e/oauthfixture || exit 1
before="$work_root/before"
after="$work_root/after"
snapshot "$before"
before_pids="$(agy_pids)"

"$fixture_bin" --listen 127.0.0.1:0 --ready-file "$ready" --log-file "$trace" --authorization-code-file "$code_file" &
fixture_pid=$!
for _ in {1..100}; do
  [[ -s "$ready" ]] && break
  kill -0 "$fixture_pid" 2>/dev/null || { echo "OAuth fixture exited" >&2; exit 1; }
  sleep 0.05
done
[[ -s "$ready" ]] || { echo "OAuth fixture not ready" >&2; exit 1; }
endpoint="$(tr -d '\r\n' < "$ready")"

mkfifo "$input_fifo"
set +e
"$interop_bin" test "$endpoint" --client antigravity --oauth --json < "$input_fifo" > "$result" 2> "$terminal" &
interop_pid=$!
exec 3> "$input_fifo"
set -e

for _ in {1..300}; do
  [[ -s "$code_file" ]] && break
  kill -0 "$interop_pid" 2>/dev/null || break
  sleep 0.1
done
if [[ ! -s "$code_file" ]]; then
  echo "Antigravity/browser path did not reach the fixture authorization endpoint" >&2
  safe_terminal_state
  fixture_trace
  exit 1
fi

cat "$code_file" >&3
printf '\r' >&3
exec 3>&-
set +e
wait "$interop_pid"
rc=$?
set -e
interop_pid=""
/usr/bin/osascript -e 'tell application "Safari" to quit' >/dev/null 2>&1 || true

cat "$result"
python3 - "$result" "$trace" "$rc" <<'PY'
import json, re, sys
result_path, trace_path, rc_raw = sys.argv[1:]
rc = int(rc_raw)
results = json.load(open(result_path))
if len(results) != 1:
    raise SystemExit(f"expected one result, got {len(results)}")
result = results[0]
stages = {item["stage"]: item for item in result.get("stages", [])}
for name in ("reach", "auth", "init", "tools"):
    if name not in stages:
        raise SystemExit(f"missing stage {name}")
if stages["reach"]["status"] != "pass" or stages["auth"]["status"] != "pass":
    raise SystemExit(f"reach/auth must pass: {stages}")
terminal_pair = (stages["init"]["status"], stages["tools"]["status"])
if terminal_pair not in (("unknown", "unknown"), ("pass", "pass")):
    raise SystemExit(f"unexpected init/tools states: {terminal_pair}")
if terminal_pair == ("unknown", "unknown") and rc != 1:
    raise SystemExit(f"conservative UNKNOWN result should exit 1, got {rc}")
if terminal_pair == ("pass", "pass") and rc != 0:
    raise SystemExit(f"fully observed PASS result should exit 0, got {rc}")
auth_message = stages["auth"].get("message", "").lower()
if "isolated" not in auth_message or "oauth" not in auth_message:
    raise SystemExit("isolated OAuth persistence evidence missing from auth stage")

events=[]
with open(trace_path) as handle:
    for line in handle:
        line=line.strip()
        if line:
            events.append(json.loads(line))
def request(path, method):
    return any(e.get("path")==path and e.get("method")==method for e in events)
def observed(method):
    return any(e.get("event")=="mcp_observation" and e.get("authorized") is True and e.get("rpc_method")==method for e in events)
for path,method,label in (
    ("/register","POST","DCR"),
    ("/authorize","GET","authorization request"),
    ("/token","POST","token exchange"),
):
    if not request(path,method):
        raise SystemExit(f"{label} not observed")
for method in ("initialize","notifications/initialized","tools/list"):
    if not observed(method):
        raise SystemExit(f"authenticated {method} not observed")

persisted = open(result_path,'rb').read() + open(trace_path,'rb').read()
if re.search(rb'fixture-(?:code|token|refresh)-', persisted):
    raise SystemExit("OAuth secret material leaked into persisted E2E evidence")
PY
parser_rc=$?
if [[ "$parser_rc" -ne 0 ]]; then
  safe_terminal_state
  fixture_trace
  exit "$parser_rc"
fi

wait_for_agy_pids "$before_pids" || true
snapshot "$after"
cmp -s "$before" "$after" || { echo "normal Antigravity/Keychain state changed" >&2; diff -u "$before" "$after" >&2 || true; exit 1; }
after_pids="$(agy_pids)"
[[ "$before_pids" == "$after_pids" ]] || { echo "Antigravity process set changed: before=$before_pids after=$after_pids" >&2; exit 1; }

echo "READY: Antigravity OAuth real-client E2E passed."
