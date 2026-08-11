#!/usr/bin/env bash

set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "VS Code Agent Plugin PoC currently requires macOS" >&2
  exit 2
fi

for command_name in go grep pgrep shasum stat mktemp; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "required command not found: $command_name" >&2
    exit 2
  fi
done

vscode_bin="${MCP_INTEROP_VSCODE_BIN:-code}"
if ! command -v "$vscode_bin" >/dev/null 2>&1; then
  echo "VS Code CLI not found: $vscode_bin" >&2
  exit 2
fi

timeout_seconds="${MCP_INTEROP_VSCODE_POC_TIMEOUT_SECONDS:-30}"
if ! [[ "$timeout_seconds" =~ ^[1-9][0-9]*$ ]]; then
  echo "MCP_INTEROP_VSCODE_POC_TIMEOUT_SECONDS must be a positive integer" >&2
  exit 2
fi

keep_tmp="${MCP_INTEROP_KEEP_VSCODE_POC_TMP:-0}"
tmp_base="${TMPDIR:-/tmp}"
tmp_base="${tmp_base%/}"
work_root="$(mktemp -d "$tmp_base/mcp-interop-vscode-poc.XXXXXX")" || exit 1
fixture_bin="$work_root/mcp-interop-e2e-fixture"
fixture_ready="$work_root/fixture.ready"
fixture_log="$work_root/fixture.jsonl"
plugin_root="$work_root/plugin"
user_data="$work_root/user-data"
extensions_dir="$work_root/extensions"
workspace="$work_root/workspace"
code_stdout="$work_root/vscode.stdout"
code_stderr="$work_root/vscode.stderr"
normal_state_before="$work_root/normal-code.before"
normal_state_after="$work_root/normal-code.after"
fixture_pid=""
launcher_pid=""

mkdir -p "$plugin_root" "$user_data/User" "$extensions_dir" "$workspace"

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

snapshot_normal_code_state() {
  local output="$1"
  : > "$output"
  snapshot_file 'Code/User/settings.json' "$HOME/Library/Application Support/Code/User/settings.json" "$output"
  snapshot_file 'Code/User/mcp.json' "$HOME/Library/Application Support/Code/User/mcp.json" "$output"
  sort -o "$output" "$output"
}

vscode_pids() {
  pgrep -f "$user_data" 2>/dev/null || true
}

stop_vscode_instance() {
  local pids pid remaining
  pids="$(vscode_pids)"
  if [[ -n "$pids" ]]; then
    while IFS= read -r pid; do
      [[ -z "$pid" ]] && continue
      kill "$pid" 2>/dev/null || true
    done <<< "$pids"

    local attempt=0
    while [[ "$attempt" -lt 30 ]]; do
      remaining="$(vscode_pids)"
      [[ -z "$remaining" ]] && break
      attempt=$((attempt + 1))
      sleep 0.1
    done

    remaining="$(vscode_pids)"
    if [[ -n "$remaining" ]]; then
      while IFS= read -r pid; do
        [[ -z "$pid" ]] && continue
        kill -KILL "$pid" 2>/dev/null || true
      done <<< "$remaining"
    fi
  fi

  if [[ -n "$launcher_pid" ]]; then
    wait "$launcher_pid" 2>/dev/null || true
    launcher_pid=""
  fi
}

cleanup() {
  stop_vscode_instance
  if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
  fi
  fixture_pid=""

  if [[ "$keep_tmp" == "1" ]]; then
    echo "VS Code PoC temporary state kept at: $work_root" >&2
  else
    rm -rf "$work_root"
  fi
}
trap cleanup EXIT INT TERM

method_seen() {
  local path="$1"
  local method="$2"
  [[ -f "$fixture_log" ]] && grep -Fq "\"path\":\"$path\",\"method\":\"$method\"" "$fixture_log"
}

run_network_isolated() {
  env \
    -u OPENAI_API_KEY \
    -u CODEX_API_KEY \
    -u ANTHROPIC_API_KEY \
    -u GEMINI_API_KEY \
    -u GOOGLE_API_KEY \
    -u GOOGLE_GENERATIVE_AI_API_KEY \
    -u CURSOR_API_KEY \
    -u OPENROUTER_API_KEY \
    -u GITHUB_TOKEN \
    -u GH_TOKEN \
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

echo "== VS Code Agent Plugin MCP PoC =="
"$vscode_bin" --version | head -n 3

if ! go build -o "$fixture_bin" ./internal/e2e/fixture; then
  echo "E2E fixture build failed" >&2
  exit 1
fi

snapshot_normal_code_state "$normal_state_before"

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
endpoint="$fixture_url/vscode-agent-plugin"
path="/mcp/vscode-agent-plugin"

echo "Fixture endpoint: $endpoint"

cat > "$plugin_root/plugin.json" <<'JSON'
{
  "name": "mcp-interop-vscode-poc",
  "description": "Temporary no-auth MCP interoperability fixture for mcp-interop",
  "version": "0.0.0",
  "mcpServers": ".mcp.json"
}
JSON

cat > "$plugin_root/.mcp.json" <<JSON
{
  "mcpServers": {
    "mcp-interop-vscode-poc": {
      "type": "http",
      "url": "$endpoint"
    }
  }
}
JSON

cat > "$user_data/User/settings.json" <<JSON
{
  "chat.plugins.enabled": true,
  "chat.pluginLocations": {
    "$plugin_root": true
  },
  "telemetry.telemetryLevel": "off",
  "update.mode": "none",
  "extensions.autoCheckUpdates": false,
  "extensions.autoUpdate": false,
  "workbench.enableExperiments": false,
  "security.workspace.trust.enabled": false
}
JSON

run_network_isolated "$vscode_bin" \
  --new-window \
  --disable-gpu \
  --user-data-dir "$user_data" \
  --extensions-dir "$extensions_dir" \
  "$workspace" \
  >"$code_stdout" 2>"$code_stderr" &
launcher_pid=$!

elapsed=0
protocol_ok=0
while [[ "$elapsed" -lt "$timeout_seconds" ]]; do
  if method_seen "$path" initialize \
    && method_seen "$path" notifications/initialized \
    && method_seen "$path" tools/list; then
    protocol_ok=1
    break
  fi
  sleep 1
  elapsed=$((elapsed + 1))
done

if method_seen "$path" tools/call; then
  echo "FAIL: unexpected tools/call observed; PoC must not invoke tools" >&2
  protocol_ok=0
fi

stop_vscode_instance
snapshot_normal_code_state "$normal_state_after"

state_ok=1
if ! cmp -s "$normal_state_before" "$normal_state_after"; then
  state_ok=0
  echo "FAIL: normal VS Code user settings changed during isolated PoC" >&2
  diff -u "$normal_state_before" "$normal_state_after" >&2 || true
fi

if [[ "$protocol_ok" -eq 1 && "$state_ok" -eq 1 ]]; then
  echo "PASS: real VS Code reached initialize -> notifications/initialized -> tools/list through a local Agent Plugin."
  echo "PASS: no tools/call was observed."
  echo "PASS: normal VS Code settings.json and mcp.json metadata remained unchanged."
  exit 0
fi

if [[ "$protocol_ok" -ne 1 ]]; then
  echo "FAIL: VS Code did not produce the required MCP wire evidence within ${timeout_seconds}s." >&2
  echo "Observed fixture traffic:" >&2
  if [[ -s "$fixture_log" ]]; then
    cat "$fixture_log" >&2
  else
    echo "  (none)" >&2
  fi
  echo >&2
  echo "This can mean Agent Plugins are unavailable/disabled by policy, the current VS Code build does not auto-start local plugin MCP servers, or the plugin HTTP configuration shape changed." >&2
  if [[ -s "$code_stderr" ]]; then
    echo >&2
    echo "VS Code stderr:" >&2
    tail -n 80 "$code_stderr" >&2
  fi
fi

exit 1
