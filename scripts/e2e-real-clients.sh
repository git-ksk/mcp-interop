#!/usr/bin/env bash

set -u
set -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root" || exit 1

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "real-client E2E currently requires macOS" >&2
  exit 2
fi

for command_name in go shasum stat pgrep find comm; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "required command not found: $command_name" >&2
    exit 2
  fi
done

tmp_base="${TMPDIR:-/tmp}"
tmp_base="${tmp_base%/}"
work_root="$(mktemp -d "$tmp_base/mcp-e2e.XXXXXX")" || exit 1
fixture_pid=""
fixture_ready="$work_root/fixture.ready"
fixture_log="$work_root/fixture.jsonl"
interop_bin="$work_root/mcp-interop"
fixture_bin="$work_root/mcp-interop-e2e-fixture"
result_dir="$work_root/results"
mkdir -p "$result_dir"

cleanup() {
  if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
  fi
  if [[ "${MCP_INTEROP_KEEP_E2E_TMP:-0}" == "1" ]]; then
    echo "E2E temporary state kept at: $work_root" >&2
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
  snapshot_file '~/.codex/config.toml' "$HOME/.codex/config.toml" "$output"
  snapshot_file '~/.codex/.credentials.json' "$HOME/.codex/.credentials.json" "$output"
  snapshot_file '~/.codex/auth.json' "$HOME/.codex/auth.json" "$output"
  snapshot_file '~/.config/cursor/cli-config.json' "$HOME/.config/cursor/cli-config.json" "$output"
  snapshot_file '~/.cursor/mcp.json' "$HOME/.cursor/mcp.json" "$output"
  snapshot_file '~/Library/Application Support/Cursor/User/mcp.json' "$HOME/Library/Application Support/Cursor/User/mcp.json" "$output"
  snapshot_file '~/.gemini/config/mcp_config.json' "$HOME/.gemini/config/mcp_config.json" "$output"
  snapshot_file '~/.gemini/antigravity/mcp_oauth_tokens.json' "$HOME/.gemini/antigravity/mcp_oauth_tokens.json" "$output"
  snapshot_file '~/.gemini/antigravity-cli/mcp_oauth_tokens.json' "$HOME/.gemini/antigravity-cli/mcp_oauth_tokens.json" "$output"
  snapshot_file '~/.gemini/antigravity-cli/settings.json' "$HOME/.gemini/antigravity-cli/settings.json" "$output"

  if [[ -d "$HOME/.cursor/projects" ]]; then
    while IFS= read -r path; do
      [[ -z "$path" ]] && continue
      local relative="${path#$HOME/}"
      snapshot_file "~/$relative" "$path" "$output"
    done < <(find "$HOME/.cursor/projects" -type f \( -name 'mcp-approvals.json' -o -name 'mcp-auth.json' \) -print 2>/dev/null | sort)
  fi

  if [[ "${MCP_INTEROP_SKIP_KEYCHAIN:-0}" == "1" ]]; then
    printf 'login-keychain-db\tskipped\t-\t-\t-\n' >> "$output"
  else
    snapshot_file 'login-keychain-db' "$HOME/Library/Keychains/login.keychain-db" "$output"
  fi

  sort -o "$output" "$output"
}

capture_client_pids() {
  local output="$1"
  : > "$output"
  local name pid
  for name in codex cursor-agent agy; do
    while IFS= read -r pid; do
      [[ -z "$pid" ]] && continue
      printf '%s\t%s\n' "$name" "$pid" >> "$output"
    done < <(pgrep -x "$name" 2>/dev/null || true)
  done
  sort -o "$output" "$output"
}

wait_for_client_pids_to_match() {
  local expected="$1"
  local scratch="$2"
  local deadline=$((SECONDS + 3))
  while true; do
    capture_client_pids "$scratch"
    if cmp -s "$expected" "$scratch"; then
      return 0
    fi
    if [[ "$SECONDS" -ge "$deadline" ]]; then
      return 1
    fi
    sleep 0.05
  done
}

capture_session_dirs() {
  local output="$1"
  find "$tmp_base" -maxdepth 1 -type d -name 'mcp-interop-*' -print 2>/dev/null | sort > "$output"
}

client_command_exists() {
  case "$1" in
    codex)
      command -v codex >/dev/null 2>&1
      ;;
    cursor)
      command -v cursor-agent >/dev/null 2>&1 || command -v agent >/dev/null 2>&1
      ;;
    antigravity)
      command -v agy >/dev/null 2>&1
      ;;
    *)
      return 1
      ;;
  esac
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

method_seen() {
  local path="$1"
  local method="$2"
  grep -Fq "\"path\":\"$path\",\"method\":\"$method\"" "$fixture_log"
}

method_protocol_seen() {
  local path="$1"
  local method="$2"
  local version="$3"
  grep -F "\"path\":\"$path\",\"method\":\"$method\"" "$fixture_log" \
    | grep -Fq "\"protocol_version\":\"$version\""
}

fixture_protocol_readiness() {
  local path="$1"

  if method_protocol_seen "$path" 'tools/list' '2026-07-28'; then
    printf '%s\n' modern
    return 0
  fi

  if method_seen "$path" initialize \
    && method_seen "$path" notifications/initialized \
    && method_seen "$path" tools/list; then
    printf '%s\n' legacy
    return 0
  fi

  printf '%s\n' unknown
  return 1
}

print_protocol_observations() {
  local path="$1"
  local client="$2"
  local observed=0
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    observed=1
    printf '%s\t%s\n' "$client" "$line"
  done < <(
    grep -F "\"path\":\"$path\"" "$fixture_log" \
      | sed -n 's/.*"method":"\([^"]*\)".*"protocol_version":"\([^"]*\)".*"protocol_source":"\([^"]*\)".*/method=\1 protocol=\2 source=\3/p'
  )
  if [[ "$observed" -eq 0 ]]; then
    printf '%s\t%s\n' "$client" 'no explicit protocol version observed'
  fi
}

before_state="$work_root/user-state.before"
after_state="$work_root/user-state.after"
before_pids="$work_root/client-pids.before"
after_pids="$work_root/client-pids.after"
before_sessions="$work_root/sessions.before"
after_sessions="$work_root/sessions.after"
new_pids="$work_root/client-pids.new"
new_sessions="$work_root/sessions.new"
status_file="$work_root/status.tsv"
: > "$status_file"

echo "== Build and unit checks =="
if ! go test ./...; then
  echo "go test ./... failed" >&2
  exit 1
fi
if ! go vet ./...; then
  echo "go vet ./... failed" >&2
  exit 1
fi
if ! go build -o "$interop_bin" ./cmd/mcp-interop; then
  echo "mcp-interop build failed" >&2
  exit 1
fi
if ! go build -o "$fixture_bin" ./internal/e2e/fixture; then
  echo "E2E fixture build failed" >&2
  exit 1
fi
"$interop_bin" version
"$interop_bin" clients

snapshot_user_state "$before_state"
capture_client_pids "$before_pids"
capture_session_dirs "$before_sessions"

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
echo "Fixture: $fixture_url"

clients_csv="${MCP_INTEROP_CLIENTS:-codex,cursor,antigravity}"
IFS=',' read -r -a clients <<< "$clients_csv"
overall_fail=0

for client in "${clients[@]}"; do
  client="$(printf '%s' "$client" | tr '[:upper:]' '[:lower:]' | tr -d ' ')"
  case "$client" in
    codex|cursor|antigravity) ;;
    *)
      echo "unsupported E2E client: $client" >&2
      printf '%s\tFAIL\tunsupported client\n' "$client" >> "$status_file"
      overall_fail=1
      continue
      ;;
  esac

  if ! client_command_exists "$client"; then
    echo "required real client is not installed: $client" >&2
    printf '%s\tFAIL\tclient not installed\n' "$client" >> "$status_file"
    overall_fail=1
    continue
  fi

  endpoint="$fixture_url/$client"
  result_json="$result_dir/$client.json"
  result_stderr="$result_dir/$client.stderr"
  echo
  echo "== Real-client E2E: $client =="
  run_network_isolated "$interop_bin" test "$endpoint" --client "$client" --json > "$result_json" 2> "$result_stderr"
  rc=$?
  cat "$result_json"
  if [[ -s "$result_stderr" ]]; then
    cat "$result_stderr" >&2
  fi

  path="/mcp/$client"
  protocol_ok=1
  protocol_era="$(fixture_protocol_readiness "$path")" || protocol_ok=0
  if [[ "$protocol_ok" -ne 1 ]]; then
    echo "$client: fixture did not observe a complete legacy or modern protocol-readiness path" >&2
  fi
  if method_seen "$path" 'tools/call'; then
    echo "$client: unexpected tools/call observed; core E2E must not invoke tools" >&2
    protocol_ok=0
  fi

  echo "Protocol observations (controlled fixture only):"
  print_protocol_observations "$path" "$client"

  if [[ "$rc" -eq 0 && "$protocol_ok" -eq 1 ]]; then
    printf '%s\tPASS\tfixture %s protocol readiness + real-client adapter PASS\n' "$client" "$protocol_era" >> "$status_file"
  else
    printf '%s\tFAIL\trc=%s protocol_ok=%s protocol_era=%s\n' "$client" "$rc" "$protocol_ok" "$protocol_era" >> "$status_file"
    overall_fail=1
  fi
done

if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
  kill "$fixture_pid" 2>/dev/null || true
  wait "$fixture_pid" 2>/dev/null || true
fi
fixture_pid=""

wait_for_client_pids_to_match "$before_pids" "$after_pids" || true
snapshot_user_state "$after_state"
capture_client_pids "$after_pids"
capture_session_dirs "$after_sessions"
comm -13 "$before_pids" "$after_pids" > "$new_pids"
comm -13 "$before_sessions" "$after_sessions" > "$new_sessions"

config_gate=PASS
if ! cmp -s "$before_state" "$after_state"; then
  config_gate=FAIL
  overall_fail=1
  echo >&2
  echo "User config/credential metadata changed:" >&2
  diff -u "$before_state" "$after_state" >&2 || true
fi

process_gate=PASS
if [[ -s "$new_pids" ]]; then
  process_gate=FAIL
  overall_fail=1
  echo >&2
  echo "New real-client process(es) remained after E2E; not killing by name for safety:" >&2
  cat "$new_pids" >&2
fi

session_gate=PASS
if [[ -s "$new_sessions" ]]; then
  session_gate=FAIL
  overall_fail=1
  echo >&2
  echo "New mcp-interop session directory/directories remained after E2E:" >&2
  cat "$new_sessions" >&2
fi

keychain_gate=PASS
if [[ "${MCP_INTEROP_SKIP_KEYCHAIN:-0}" == "1" ]]; then
  keychain_gate=SKIP
elif ! grep -Fq $'login-keychain-db\t' "$before_state" || ! grep -Fq $'login-keychain-db\t' "$after_state"; then
  keychain_gate=FAIL
  overall_fail=1
elif ! diff <(grep -F $'login-keychain-db\t' "$before_state") <(grep -F $'login-keychain-db\t' "$after_state") >/dev/null; then
  keychain_gate=FAIL
  overall_fail=1
fi

echo
echo "== E2E result =="
printf 'CLIENT\tRESULT\tDETAIL\n'
cat "$status_file"
echo
printf 'RELEASE GATE\tRESULT\n'
printf 'real-client protocol E2E\t%s\n' "$([[ "$overall_fail" -eq 0 ]] && echo PASS || echo FAIL)"
printf 'user config unchanged\t%s\n' "$config_gate"
printf 'login Keychain DB unchanged\t%s\n' "$keychain_gate"
printf 'no new client processes\t%s\n' "$process_gate"
printf 'no leaked mcp-interop sessions\t%s\n' "$session_gate"
printf 'tools/call avoided\t%s\n' "$(! grep -Fq '"method":"tools/call"' "$fixture_log" && echo PASS || echo FAIL)"

echo
if [[ "$overall_fail" -eq 0 ]]; then
  echo "READY: real-client E2E gates passed."
  exit 0
fi

echo "NOT READY: one or more real-client E2E gates failed." >&2
exit 1
