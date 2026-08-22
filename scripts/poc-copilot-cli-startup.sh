#!/usr/bin/env bash

set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Copilot CLI startup PoC currently targets macOS" >&2
  exit 2
fi

for command_name in go copilot python3 pgrep ps shasum stat comm; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "required command not found: $command_name" >&2
    exit 2
  fi
done

test_token="${MCP_INTEROP_COPILOT_TEST_TOKEN:-}"
auth_mode="unauthenticated"
if [[ -n "$test_token" ]]; then
  auth_mode="dedicated-env-token"
  if [[ "${MCP_INTEROP_COPILOT_ALLOW_NETWORK:-0}" != "1" ]]; then
    echo "authenticated Copilot PoC requires MCP_INTEROP_COPILOT_ALLOW_NETWORK=1" >&2
    exit 2
  fi
  if [[ "${MCP_INTEROP_KEEP_COPILOT_POC_TMP:-0}" == "1" ]]; then
    echo "authenticated Copilot PoC does not allow retaining temporary client state" >&2
    exit 2
  fi
fi

tmp_base="${TMPDIR:-/tmp}"
tmp_base="${tmp_base%/}"
work_root="$(mktemp -d "$tmp_base/mcp-copilot-startup.XXXXXX")" || exit 1
fixture_pid=""
fixture_ready="$work_root/fixture.ready"
fixture_log="$work_root/fixture.jsonl"
fixture_bin="$work_root/mcp-interop-e2e-fixture"
copilot_home="$work_root/copilot-home"
copilot_cache="$work_root/copilot-cache"
workspace="$work_root/workspace"
pty_output="$work_root/copilot-startup.txt"
before_state="$work_root/user-state.before"
after_state="$work_root/user-state.after"
mkdir -p "$copilot_home" "$copilot_cache" "$workspace" "$copilot_home/logs"
chmod 700 "$copilot_home" "$copilot_cache" "$workspace" 2>/dev/null || true

cleanup() {
  if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
  fi
  if [[ "${MCP_INTEROP_KEEP_COPILOT_POC_TMP:-0}" == "1" ]]; then
    echo "Copilot CLI startup PoC temporary state kept at: $work_root" >&2
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

method_seen() {
  local path="$1"
  local method="$2"
  grep -Fq "\"path\":\"$path\",\"method\":\"$method\"" "$fixture_log"
}

run_isolated() {
  if [[ "$auth_mode" == "dedicated-env-token" ]]; then
    env \
      -u GH_TOKEN \
      -u GITHUB_TOKEN \
      -u COPILOT_GITHUB_TOKEN \
      -u OPENAI_API_KEY \
      -u ANTHROPIC_API_KEY \
      -u GEMINI_API_KEY \
      -u GOOGLE_API_KEY \
      -u OPENROUTER_API_KEY \
      COPILOT_GITHUB_TOKEN="$test_token" \
      COPILOT_HOME="$copilot_home" \
      COPILOT_CACHE_HOME="$copilot_cache" \
      COPILOT_MCP_TOOL_CACHE=false \
      "$@"
  else
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
      COPILOT_MCP_TOOL_CACHE=false \
      HTTP_PROXY='http://127.0.0.1:9' \
      HTTPS_PROXY='http://127.0.0.1:9' \
      ALL_PROXY='http://127.0.0.1:9' \
      http_proxy='http://127.0.0.1:9' \
      https_proxy='http://127.0.0.1:9' \
      all_proxy='http://127.0.0.1:9' \
      NO_PROXY='127.0.0.1,localhost,::1' \
      no_proxy='127.0.0.1,localhost,::1' \
      "$@"
  fi
}

before_pids="$work_root/copilot-pids.before"
after_pids="$work_root/copilot-pids.after"
snapshot_user_state "$before_state"
pgrep -x copilot 2>/dev/null | sort -n > "$before_pids" || true

printf 'Copilot PoC auth mode\t%s\n' "$auth_mode"
run_isolated copilot --version

go build -o "$fixture_bin" ./internal/e2e/fixture || exit 1
"$fixture_bin" --listen 127.0.0.1:0 --ready-file "$fixture_ready" --log-file "$fixture_log" &
fixture_pid=$!

for _ in $(seq 1 100); do
  [[ -s "$fixture_ready" ]] && break
  if ! kill -0 "$fixture_pid" 2>/dev/null; then
    echo "localhost fixture exited before ready" >&2
    exit 1
  fi
  sleep 0.05
done
if [[ ! -s "$fixture_ready" ]]; then
  echo "localhost fixture did not become ready" >&2
  exit 1
fi

fixture_url="$(tr -d '\r\n' < "$fixture_ready")"
endpoint="$fixture_url/copilot-startup"
protocol_path="/mcp/copilot-startup"

cat > "$copilot_home/settings.json" <<'JSON'
{
  "autoUpdate": false,
  "banner": "never"
}
JSON
python3 - "$copilot_home/mcp-config.json" "$endpoint" <<'PY'
import json
import sys
path, url = sys.argv[1:]
with open(path, "w", encoding="utf-8") as f:
    json.dump({"mcpServers": {"mcp-interop-fixture": {
        "type": "http",
        "url": url,
        "headers": {},
        "tools": ["*"],
        "deferTools": "never"
    }}}, f, indent=2)
    f.write("\n")
PY
chmod 600 "$copilot_home/settings.json" "$copilot_home/mcp-config.json" 2>/dev/null || true

echo "== No-input PTY startup (30 second observation) =="
(
  cd "$workspace" || exit 1
  run_isolated python3 - "$pty_output" <<'PY'
import errno
import os
import pty
import select
import signal
import sys
import time

output_path = sys.argv[1]
argv = [
    "copilot",
    "--no-auto-update",
    "--no-custom-instructions",
    "--log-level", "debug",
    "--log-dir", os.path.join(os.environ["COPILOT_HOME"], "logs"),
]

pid, fd = pty.fork()
if pid == 0:
    os.execvp(argv[0], argv)

data = bytearray()
deadline = time.time() + 30.0
exited = False
while time.time() < deadline:
    try:
        ready, _, _ = select.select([fd], [], [], 0.2)
    except InterruptedError:
        continue
    if ready:
        try:
            chunk = os.read(fd, 65536)
            if chunk:
                data.extend(chunk)
        except OSError as exc:
            if exc.errno not in (errno.EIO, errno.EBADF):
                raise
    waited, _ = os.waitpid(pid, os.WNOHANG)
    if waited == pid:
        exited = True
        break

if not exited:
    try:
        pgid = os.getpgid(pid)
        if pgid == pid:
            os.killpg(pgid, signal.SIGTERM)
        else:
            os.kill(pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    for _ in range(20):
        waited, _ = os.waitpid(pid, os.WNOHANG)
        if waited == pid:
            exited = True
            break
        time.sleep(0.1)
if not exited:
    try:
        os.kill(pid, signal.SIGKILL)
    except ProcessLookupError:
        pass
    try:
        os.waitpid(pid, 0)
    except ChildProcessError:
        pass

try:
    os.close(fd)
except OSError:
    pass
with open(output_path, "wb") as f:
    f.write(data)
PY
)
startup_rc=$?

if [[ "$auth_mode" == "dedicated-env-token" ]]; then
  printf '%s\n' '-- startup output omitted in authenticated mode --'
else
  printf '%s\n' '-- startup output --'
  python3 - "$pty_output" <<'PY'
import re
import sys
raw = open(sys.argv[1], "rb").read().decode("utf-8", "replace")
raw = re.sub(r"\x1b\[[0-?]*[ -/]*[@-~]", "", raw)
print(raw)
PY
fi

printf '%s\n' '-- fixture wire log --'
cat "$fixture_log" 2>/dev/null || true

if [[ "$auth_mode" == "dedicated-env-token" ]]; then
  printf '%s\n' '-- Copilot debug log output omitted in authenticated mode --'
else
  printf '%s\n' '-- selected Copilot debug log lines --'
  for log_file in "$copilot_home"/logs/*; do
    [[ -f "$log_file" ]] || continue
    echo "### $(basename "$log_file")"
    grep -Eai 'mcp|auth|login|tool|error|warn|connect' "$log_file" | tail -n 120 || true
  done
fi

protocol_ok=1
for method in initialize notifications/initialized tools/list; do
  if ! method_seen "$protocol_path" "$method"; then
    echo "startup fixture did not observe $method" >&2
    protocol_ok=0
  fi
done
if method_seen "$protocol_path" 'tools/call'; then
  echo "unexpected tools/call observed during no-input startup" >&2
  protocol_ok=0
fi

if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
  kill "$fixture_pid" 2>/dev/null || true
  wait "$fixture_pid" 2>/dev/null || true
fi
fixture_pid=""

pgrep -x copilot 2>/dev/null | sort -n > "$after_pids" || true
new_pids="$work_root/copilot-pids.new"
comm -13 "$before_pids" "$after_pids" > "$new_pids"
process_gate=PASS
if [[ -s "$new_pids" ]]; then
  process_gate=FAIL
  echo "new Copilot process(es) remained after startup PoC; recording only, not killing by name:" >&2
  while IFS= read -r pid; do
    [[ -n "$pid" ]] || continue
    ps -p "$pid" -o pid=,ppid=,pgid=,etime=,command= >&2 || true
  done < "$new_pids"
fi

snapshot_user_state "$after_state"
state_gate=PASS
if ! cmp -s "$before_state" "$after_state"; then
  state_gate=FAIL
  echo "normal Copilot/Keychain metadata changed:" >&2
  diff -u "$before_state" "$after_state" >&2 || true
fi

echo
echo "== Copilot CLI no-input startup PoC result =="
printf 'auth mode\t%s\n' "$auth_mode"
printf 'PTY harness completed\t%s\n' "$([[ "$startup_rc" -eq 0 ]] && echo PASS || echo FAIL)"
printf 'initialize + initialized + tools/list\t%s\n' "$([[ "$protocol_ok" -eq 1 ]] && echo PASS || echo FAIL)"
printf 'tools/call avoided\t%s\n' "$(! grep -Fq '\"method\":\"tools/call\"' "$fixture_log" && echo PASS || echo FAIL)"
printf 'normal user state unchanged\t%s\n' "$state_gate"
printf 'no leaked Copilot process\t%s\n' "$process_gate"

if [[ "$startup_rc" -eq 0 && "$protocol_ok" -eq 1 && "$state_gate" == PASS && "$process_gate" == PASS ]]; then
  echo "READY: no-input Copilot startup reached MCP discovery without a model prompt."
  exit 0
fi

echo "NOT READY: no-input Copilot startup did not prove the direct MCP lifecycle boundary." >&2
exit 1
