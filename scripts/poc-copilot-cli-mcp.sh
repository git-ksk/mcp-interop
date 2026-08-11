#!/usr/bin/env bash

set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Copilot CLI real-client PoC currently targets macOS" >&2
  exit 2
fi

for command_name in go copilot python3 shasum stat pgrep comm; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "required command not found: $command_name" >&2
    exit 2
  fi
done

tmp_base="${TMPDIR:-/tmp}"
tmp_base="${tmp_base%/}"
work_root="$(mktemp -d "$tmp_base/mcp-copilot-poc.XXXXXX")" || exit 1
fixture_pid=""
copilot_pid=""
fixture_ready="$work_root/fixture.ready"
fixture_log="$work_root/fixture.jsonl"
fixture_bin="$work_root/mcp-interop-e2e-fixture"
copilot_home="$work_root/copilot-home"
copilot_cache="$work_root/copilot-cache"
workspace="$work_root/workspace"
result_json="$work_root/copilot-mcp-get.json"
result_stderr="$work_root/copilot-mcp-get.stderr"
mkdir -p "$copilot_home" "$copilot_cache" "$workspace"
chmod 700 "$copilot_home" "$copilot_cache" "$workspace" 2>/dev/null || true

cleanup() {
  if [[ -n "$copilot_pid" ]] && kill -0 "$copilot_pid" 2>/dev/null; then
    kill "$copilot_pid" 2>/dev/null || true
    wait "$copilot_pid" 2>/dev/null || true
  fi
  if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
  fi
  if [[ "${MCP_INTEROP_KEEP_COPILOT_POC_TMP:-0}" == "1" ]]; then
    echo "Copilot CLI PoC temporary state kept at: $work_root" >&2
  else
    rm -rf "$work_root"
  fi
}
trap cleanup EXIT INT TERM

snapshot_file() {
  local label="$1"
  local path="$2"
  local output="$3"
  if [[ -f "$path" ]]; then
    local size mtime hash
    size="$(stat -f '%z' "$path")"
    mtime="$(stat -f '%m' "$path")"
    hash="$(shasum -a 256 "$path" | awk '{print $1}')"
    printf '%s\tfile\t%s\t%s\t%s\n' "$label" "$size" "$mtime" "$hash" >> "$output"
  elif [[ -e "$path" ]]; then
    printf '%s\tother\t-\t-\t-\n' "$label" >> "$output"
  else
    printf '%s\tmissing\t-\t-\t-\n' "$label" >> "$output"
  fi
}

snapshot_user_state() {
  local output="$1"
  : > "$output"
  snapshot_file '~/.copilot/config.json' "$HOME/.copilot/config.json" "$output"
  snapshot_file '~/.copilot/settings.json' "$HOME/.copilot/settings.json" "$output"
  snapshot_file '~/.copilot/mcp-config.json' "$HOME/.copilot/mcp-config.json" "$output"
  snapshot_file '~/.copilot/permissions-config.json' "$HOME/.copilot/permissions-config.json" "$output"
  snapshot_file '~/.copilot/session-store.db' "$HOME/.copilot/session-store.db" "$output"
  if [[ "${MCP_INTEROP_SKIP_KEYCHAIN:-0}" == "1" ]]; then
    printf 'login-keychain-db\tskipped\t-\t-\t-\n' >> "$output"
  else
    snapshot_file 'login-keychain-db' "$HOME/Library/Keychains/login.keychain-db" "$output"
  fi
  sort -o "$output" "$output"
}

capture_copilot_pids() {
  local output="$1"
  pgrep -x copilot 2>/dev/null | sort -n > "$output" || true
}

method_seen() {
  local path="$1"
  local method="$2"
  grep -Fq "\"path\":\"$path\",\"method\":\"$method\"" "$fixture_log"
}

run_copilot_isolated() {
  env \
    -u GH_TOKEN \
    -u GITHUB_TOKEN \
    -u COPILOT_GITHUB_TOKEN \
    -u OPENAI_API_KEY \
    -u ANTHROPIC_API_KEY \
    -u GEMINI_API_KEY \
    -u GOOGLE_API_KEY \
    -u OPENROUTER_API_KEY \
    COPILOT_HOME="$copilot_home" \
    COPILOT_CACHE_HOME="$copilot_cache" \
    HTTP_PROXY='http://127.0.0.1:9' \
    HTTPS_PROXY='http://127.0.0.1:9' \
    ALL_PROXY='http://127.0.0.1:9' \
    http_proxy='http://127.0.0.1:9' \
    https_proxy='http://127.0.0.1:9' \
    all_proxy='http://127.0.0.1:9' \
    NO_PROXY='127.0.0.1,localhost,::1' \
    no_proxy='127.0.0.1,localhost,::1' \
    "$@"
}

before_state="$work_root/user-state.before"
after_state="$work_root/user-state.after"
before_pids="$work_root/copilot-pids.before"
after_pids="$work_root/copilot-pids.after"
new_pids="$work_root/copilot-pids.new"

snapshot_user_state "$before_state"
capture_copilot_pids "$before_pids"

echo "== Copilot CLI baseline =="
run_copilot_isolated copilot --version

if ! go build -o "$fixture_bin" ./internal/e2e/fixture; then
  echo "failed to build localhost MCP fixture" >&2
  exit 1
fi

"$fixture_bin" --listen 127.0.0.1:0 --ready-file "$fixture_ready" --log-file "$fixture_log" &
fixture_pid=$!

ready_attempt=0
while [[ ! -s "$fixture_ready" ]]; do
  if ! kill -0 "$fixture_pid" 2>/dev/null; then
    echo "localhost MCP fixture exited before becoming ready" >&2
    wait "$fixture_pid" 2>/dev/null || true
    fixture_pid=""
    exit 1
  fi
  ready_attempt=$((ready_attempt + 1))
  if [[ "$ready_attempt" -ge 100 ]]; then
    echo "localhost MCP fixture did not become ready" >&2
    exit 1
  fi
  sleep 0.05
done

fixture_url="$(tr -d '\r\n' < "$fixture_ready")"
endpoint="$fixture_url/copilot"
server_name="mcp-interop-fixture"
protocol_path="/mcp/copilot"

echo "Fixture: $endpoint"

cat > "$copilot_home/settings.json" <<'JSON'
{
  "autoUpdate": false,
  "banner": "never"
}
JSON

python3 - "$copilot_home/mcp-config.json" "$server_name" "$endpoint" <<'PY'
import json
import sys

path, name, url = sys.argv[1:]
with open(path, "w", encoding="utf-8") as f:
    json.dump({
        "mcpServers": {
            name: {
                "type": "http",
                "url": url,
                "headers": {},
                "tools": ["*"]
            }
        }
    }, f, indent=2)
    f.write("\n")
PY
chmod 600 "$copilot_home/settings.json" "$copilot_home/mcp-config.json" 2>/dev/null || true

echo "== Direct non-interactive MCP inventory =="
(
  cd "$workspace" || exit 1
  run_copilot_isolated copilot mcp get "$server_name" --json
) > "$result_json" 2> "$result_stderr" &
copilot_pid=$!

completed=0
for _ in $(seq 1 300); do
  if ! kill -0 "$copilot_pid" 2>/dev/null; then
    completed=1
    break
  fi
  sleep 0.1
done

if [[ "$completed" -ne 1 ]]; then
  echo "copilot mcp get did not finish within 30 seconds" >&2
  kill "$copilot_pid" 2>/dev/null || true
  wait "$copilot_pid" 2>/dev/null || true
  copilot_pid=""
  [[ -s "$result_stderr" ]] && cat "$result_stderr" >&2
  exit 1
fi

wait "$copilot_pid"
copilot_rc=$?
copilot_pid=""

if [[ -s "$result_stderr" ]]; then
  cat "$result_stderr" >&2
fi
cat "$result_json"

if [[ "$copilot_rc" -ne 0 ]]; then
  echo "copilot mcp get failed with rc=$copilot_rc" >&2
  exit 1
fi

if ! python3 - "$result_json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    data = json.load(f)

def contains_ping(value):
    if value == "ping":
        return True
    if isinstance(value, dict):
        return any(contains_ping(v) for v in value.values())
    if isinstance(value, list):
        return any(contains_ping(v) for v in value)
    return False

if not contains_ping(data):
    raise SystemExit("machine-readable Copilot MCP inventory did not contain fixture tool 'ping'")
PY
then
  exit 1
fi

protocol_ok=1
for method in initialize notifications/initialized tools/list; do
  if ! method_seen "$protocol_path" "$method"; then
    echo "fixture did not observe $method from Copilot CLI" >&2
    protocol_ok=0
  fi
done
if method_seen "$protocol_path" 'tools/call'; then
  echo "unexpected tools/call observed; inventory PoC must not invoke tools" >&2
  protocol_ok=0
fi

if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
  kill "$fixture_pid" 2>/dev/null || true
  wait "$fixture_pid" 2>/dev/null || true
fi
fixture_pid=""

snapshot_user_state "$after_state"
capture_copilot_pids "$after_pids"
comm -13 "$before_pids" "$after_pids" > "$new_pids"

config_gate=PASS
if ! cmp -s "$before_state" "$after_state"; then
  config_gate=FAIL
  echo "normal Copilot/Keychain metadata changed:" >&2
  diff -u "$before_state" "$after_state" >&2 || true
fi

process_gate=PASS
if [[ -s "$new_pids" ]]; then
  process_gate=FAIL
  echo "new Copilot process(es) remained after PoC; not killing by name:" >&2
  cat "$new_pids" >&2
fi

echo
echo "== Copilot CLI MCP PoC result =="
printf 'machine-readable fixture tool inventory\tPASS\n'
printf 'initialize + initialized + tools/list\t%s\n' "$([[ "$protocol_ok" -eq 1 ]] && echo PASS || echo FAIL)"
printf 'tools/call avoided\t%s\n' "$(! grep -Fq '\"method\":\"tools/call\"' "$fixture_log" && echo PASS || echo FAIL)"
printf 'normal user state unchanged\t%s\n' "$config_gate"
printf 'no leaked Copilot process\t%s\n' "$process_gate"

if [[ "$protocol_ok" -eq 1 && "$config_gate" == PASS && "$process_gate" == PASS ]]; then
  echo "READY: Copilot CLI direct MCP inventory boundary passed."
  exit 0
fi

echo "NOT READY: one or more Copilot CLI MCP PoC gates failed." >&2
exit 1
