#!/usr/bin/env bash
set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1
[[ "$(uname -s)" == "Darwin" ]] || { echo "Cursor OAuth E2E requires macOS" >&2; exit 2; }

work_root="$(mktemp -d "${TMPDIR:-/tmp}/mcp-cursor-oauth.XXXXXX")" || exit 1
interop_bin="$work_root/mcp-interop"
fixture_bin="$work_root/oauth-fixture"
ready="$work_root/ready"
log="$work_root/fixture.jsonl"
result="$work_root/result.json"
browser_marker="$work_root/browser-invoked"
fixture_pid=""
cleanup() {
  [[ -n "$fixture_pid" ]] && kill "$fixture_pid" 2>/dev/null || true
  [[ -n "$fixture_pid" ]] && wait "$fixture_pid" 2>/dev/null || true
  [[ "${MCP_INTEROP_KEEP_E2E_TMP:-0}" == "1" ]] || rm -rf "$work_root"
}
trap cleanup EXIT INT TERM

snapshot() {
  local out="$1"
  : > "$out"
  for path in \
    "$HOME/.config/cursor/cli-config.json" \
    "$HOME/.cursor/mcp.json" \
    "$HOME/Library/Application Support/Cursor/User/mcp.json" \
    "$HOME/Library/Keychains/login.keychain-db"; do
    if [[ -f "$path" ]]; then
      printf '%s\t%s\t%s\n' "$path" "$(stat -f '%m:%z' "$path")" "$(shasum -a 256 "$path" | awk '{print $1}')" >> "$out"
    else
      printf '%s\tmissing\n' "$path" >> "$out"
    fi
  done
  if [[ -d "$HOME/.cursor/projects" ]]; then
    find "$HOME/.cursor/projects" -type f \( -name 'mcp-auth.json' -o -name 'mcp-approvals.json' \) -print 2>/dev/null | sort | while IFS= read -r path; do
      printf '%s\t%s\t%s\n' "$path" "$(stat -f '%m:%z' "$path")" "$(shasum -a 256 "$path" | awk '{print $1}')" >> "$out"
    done
  fi
  sort -o "$out" "$out"
}

fixture_trace() {
  echo "--- secret-free OAuth fixture trace ---" >&2
  if [[ -s "$log" ]]; then
    cat "$log" >&2
  else
    echo "(no fixture requests observed)" >&2
  fi
  if [[ -f "$browser_marker" ]]; then
    echo "browser launcher: invoked" >&2
  else
    echo "browser launcher: not observed" >&2
  fi
}

cursor_pids() {
  {
    pgrep -x cursor-agent 2>/dev/null || true
    pgrep -x agent 2>/dev/null || true
  } | sed '/^$/d' | sort -n | tr '\n' ' '
}

wait_for_cursor_pids() {
  local expected="$1"
  local deadline=$((SECONDS + 3))
  while true; do
    [[ "$(cursor_pids)" == "$expected" ]] && return 0
    [[ "$SECONDS" -ge "$deadline" ]] && return 1
    sleep 0.05
  done
}

command -v cursor-agent >/dev/null 2>&1 || command -v agent >/dev/null 2>&1 || { echo "Cursor CLI missing" >&2; exit 2; }
go build -o "$interop_bin" ./cmd/mcp-interop || exit 1
go build -o "$fixture_bin" ./internal/e2e/oauthfixture || exit 1

mkdir -p "$work_root/bin"
cat > "$work_root/bin/loopback-browser" <<'BROWSER'
#!/usr/bin/env bash
set -eu
url=""
for arg in "$@"; do
  case "$arg" in
    http://*) url="$arg" ;;
  esac
done
[[ -n "$url" ]] || exit 64
case "$url" in
  http://127.0.0.1:*/*|http://localhost:*/*|http://\[::1\]:*/*) ;;
  *) exit 65 ;;
esac
: > "${MCP_INTEROP_E2E_BROWSER_MARKER:?}"
/usr/bin/curl -fsS -L --max-time 20 "$url" >/dev/null
BROWSER
chmod 700 "$work_root/bin/loopback-browser"
ln -s loopback-browser "$work_root/bin/open"

before="$work_root/before"
after="$work_root/after"
snapshot "$before"
before_pids="$(cursor_pids)"

"$fixture_bin" --listen 127.0.0.1:0 --ready-file "$ready" --log-file "$log" &
fixture_pid=$!
for _ in {1..100}; do
  [[ -s "$ready" ]] && break
  kill -0 "$fixture_pid" 2>/dev/null || { echo "OAuth fixture exited" >&2; exit 1; }
  sleep 0.05
done
[[ -s "$ready" ]] || { echo "OAuth fixture not ready" >&2; exit 1; }
endpoint="$(tr -d '\r\n' < "$ready")"

set +e
env \
  -u CURSOR_API_KEY \
  PATH="$work_root/bin:$PATH" \
  BROWSER="$work_root/bin/loopback-browser" \
  MCP_INTEROP_E2E_BROWSER_MARKER="$browser_marker" \
  HTTP_PROXY='http://127.0.0.1:9' HTTPS_PROXY='http://127.0.0.1:9' ALL_PROXY='http://127.0.0.1:9' \
  NO_PROXY='127.0.0.1,localhost,::1' no_proxy='127.0.0.1,localhost,::1' \
  MCP_INTEROP_E2E_AUTO_AUTHORIZE_LOOPBACK=1 \
  "$interop_bin" test "$endpoint" --client cursor --oauth --json > "$result" 2> "$work_root/stderr"
rc=$?
set -e
cat "$result"
[[ -s "$work_root/stderr" ]] && cat "$work_root/stderr" >&2
if [[ "$rc" -ne 0 ]]; then
  echo "Cursor OAuth test returned $rc" >&2
  fixture_trace
  exit 1
fi
if [[ "$(grep -c '"status": "pass"' "$result" || true)" -ne 4 ]]; then
  echo "Cursor did not pass all four stages" >&2
  fixture_trace
  exit 1
fi
grep -Fq '"method":"POST","path":"/register"' "$log" || { echo "DCR not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"GET","path":"/authorize"' "$log" || { echo "authorization request not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"POST","path":"/token"' "$log" || { echo "token exchange not observed" >&2; fixture_trace; exit 1; }
grep -Fq '"method":"POST","path":"/mcp"' "$log" || { echo "authenticated MCP request not observed" >&2; fixture_trace; exit 1; }

wait_for_cursor_pids "$before_pids" || true
snapshot "$after"
cmp -s "$before" "$after" || { echo "normal Cursor/Keychain state changed" >&2; diff -u "$before" "$after" >&2 || true; exit 1; }
after_pids="$(cursor_pids)"
[[ "$before_pids" == "$after_pids" ]] || { echo "Cursor process set changed: before=$before_pids after=$after_pids" >&2; exit 1; }

if find "${TMPDIR:-/tmp}" -maxdepth 1 -type d -name 'mcp-interop-*' -newer "$before" -print 2>/dev/null | grep -q .; then
  echo "mcp-interop temporary session leaked" >&2
  exit 1
fi

echo "READY: Cursor OAuth real-client E2E passed."
