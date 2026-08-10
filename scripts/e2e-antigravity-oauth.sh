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
auth_url_file="$work_root/auth-url"
code_file="$work_root/auth-code"
fixture_pid=""
interop_pid=""
watch_pid=""
cleanup() {
  kill "${interop_pid:-}" "${watch_pid:-}" "${fixture_pid:-}" 2>/dev/null || true
  wait "${interop_pid:-}" "${watch_pid:-}" "${fixture_pid:-}" 2>/dev/null || true
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

go build -o "$interop_bin" ./cmd/mcp-interop || exit 1
go build -o "$fixture_bin" ./internal/e2e/oauthfixture || exit 1
before="$work_root/before"
after="$work_root/after"
snapshot "$before"
before_pids="$(pgrep -x agy 2>/dev/null | sort -n | tr '\n' ' ' || true)"

"$fixture_bin" --listen 127.0.0.1:0 --ready-file "$ready" --log-file "$trace" &
fixture_pid=$!
for _ in {1..100}; do
  [[ -s "$ready" ]] && break
  kill -0 "$fixture_pid" 2>/dev/null || { echo "OAuth fixture exited" >&2; exit 1; }
  sleep 0.05
done
[[ -s "$ready" ]] || { echo "OAuth fixture not ready" >&2; exit 1; }
endpoint="$(tr -d '\r\n' < "$ready")"
origin="${endpoint%/mcp}"

# The real agy 1.1.11 client launches the browser through macOS LaunchServices,
# so the short-lived launcher process is not a reliable observation point. This
# hosted-CI-only gate reads the current URL from the disposable Safari instance,
# accepts only this localhost fixture's /authorize URL, writes it to a private
# 0600 temp file, and never emits it to logs or mcp-interop diagnostics.
(
  for _ in {1..600}; do
    safari_url="$(/usr/bin/osascript -e 'tell application "Safari" to if (count of documents) > 0 then return URL of front document' 2>/dev/null || true)"
    if [[ -n "$safari_url" ]]; then
      python3 - "$safari_url" "$origin" "$auth_url_file" <<'PY'
import os,sys,urllib.parse
raw,origin,out=sys.argv[1:]
u=urllib.parse.urlparse(raw)
expected=urllib.parse.urlparse(origin)
if u.scheme=='http' and u.hostname==expected.hostname and u.port==expected.port and u.path=='/authorize':
    if not os.path.exists(out):
        fd=os.open(out,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
        with os.fdopen(fd,'w') as f:
            f.write(raw)
PY
    fi
    [[ -s "$auth_url_file" ]] && exit 0
    sleep 0.05
  done
) &
watch_pid=$!

mkfifo "$input_fifo"
set +e
"$interop_bin" test "$endpoint" --client antigravity --oauth --json < "$input_fifo" > "$result" 2> "$terminal" &
interop_pid=$!
exec 3> "$input_fifo"
set -e

for _ in {1..300}; do
  [[ -s "$auth_url_file" ]] && break
  kill -0 "$interop_pid" 2>/dev/null || break
  sleep 0.1
done
if [[ ! -s "$auth_url_file" ]]; then
  echo "Antigravity authorization launch URL was not observed" >&2
  fixture_trace
  exit 1
fi

python3 - "$auth_url_file" "$origin" "$code_file" <<'PY'
import os,sys,urllib.error,urllib.parse,urllib.request
url_path,origin,code_path=sys.argv[1:]
raw=open(url_path).read()
u=urllib.parse.urlparse(raw); expected=urllib.parse.urlparse(origin)
if u.scheme!='http' or u.hostname!=expected.hostname or u.port!=expected.port or u.path!='/authorize':
    raise SystemExit(65)
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,req,fp,code,msg,headers,newurl): return None
opener=urllib.request.build_opener(NoRedirect)
try:
    resp=opener.open(raw,timeout=15); location=resp.headers.get('Location','')
except urllib.error.HTTPError as e:
    location=e.headers.get('Location','')
code=urllib.parse.parse_qs(urllib.parse.urlparse(location).query).get('code',[''])[0]
if not code: raise SystemExit(66)
fd=os.open(code_path,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
with os.fdopen(fd,'w') as f: f.write(code)
PY

cat "$code_file" >&3
printf '\r' >&3
exec 3>&-
set +e
wait "$interop_pid"
rc=$?
set -e
interop_pid=""
kill "$watch_pid" 2>/dev/null || true
wait "$watch_pid" 2>/dev/null || true
watch_pid=""
/usr/bin/osascript -e 'tell application "Safari" to quit' >/dev/null 2>&1 || true

cat "$result"
if [[ "$rc" -ne 0 ]]; then
  echo "Antigravity OAuth test returned $rc" >&2
  fixture_trace
  exit 1
fi
[[ "$(grep -c '"status": "pass"' "$result" || true)" -eq 4 ]] || { echo "Antigravity did not pass all four stages" >&2; fixture_trace; exit 1; }
grep -Fq 'persisted MCP OAuth state only inside the isolated HOME' "$result" || { echo "isolated OAuth persistence evidence missing" >&2; exit 1; }
grep -Fq '"method":"POST","path":"/register"' "$trace" || { echo "DCR not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"GET","path":"/authorize"' "$trace" || { echo "authorization request not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"POST","path":"/token"' "$trace" || { echo "token exchange not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"POST","path":"/mcp"' "$trace" || { echo "MCP request not observed" >&2; fixture_trace; exit 1; }
if grep -Eq 'fixture-(code|token)-' "$result" "$trace" 2>/dev/null; then
  echo "OAuth secret material leaked into persisted E2E evidence" >&2
  exit 1
fi

sleep 0.3
snapshot "$after"
cmp -s "$before" "$after" || { echo "normal Antigravity/Keychain state changed" >&2; diff -u "$before" "$after" >&2 || true; exit 1; }
after_pids="$(pgrep -x agy 2>/dev/null | sort -n | tr '\n' ' ' || true)"
[[ "$before_pids" == "$after_pids" ]] || { echo "Antigravity process set changed: before=$before_pids after=$after_pids" >&2; exit 1; }

echo "READY: Antigravity OAuth real-client E2E passed."
