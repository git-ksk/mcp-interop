#!/usr/bin/env python3
"""Local real-client cleanup acceptance; never use arbitrary remote endpoints.

The test server deliberately stalls loopback discovery, preventing a fabricated
PASS. It never handles OAuth, sends model prompts or invokes an MCP tool. Only
its own subprocesses may be signaled, and ordinary user profiles are read only.
"""
import hashlib
import http.server
import json
import os
import pathlib
import signal
import subprocess
import sys
import threading
import time

CLIENTS = ('codex', 'cursor', 'antigravity')
PID_NAMES = ('codex', 'cursor-agent', 'agy')
SECRET_ENV = ('OPENAI_API_KEY', 'CODEX_API_KEY', 'ANTHROPIC_API_KEY',
              'GEMINI_API_KEY', 'GOOGLE_API_KEY', 'GOOGLE_GENERATIVE_AI_API_KEY',
              'CURSOR_API_KEY', 'OPENROUTER_API_KEY')
PRIVATE_FILES = (
    '.codex/config.toml', '.codex/auth.json', '.codex/.credentials.json',
    '.config/cursor/cli-config.json', '.cursor/mcp.json',
    'Library/Application Support/Cursor/User/mcp.json',
    '.gemini/config/mcp_config.json', '.gemini/antigravity/mcp_oauth_tokens.json',
    '.gemini/antigravity-cli/mcp_oauth_tokens.json',
    '.gemini/antigravity-cli/settings.json',
    'Library/Keychains/login.keychain-db',
)

class DelayedHandler(http.server.BaseHTTPRequestHandler):
    count = 0
    def do_POST(self): self.respond()
    def do_GET(self): self.respond()
    def do_OPTIONS(self): self.respond()
    def respond(self):
        type(self).count += 1
        time.sleep(10)
        try:
            self.send_response(503)
            self.end_headers()
        except (BrokenPipeError, ConnectionResetError):
            pass
    def log_message(self, *_args): pass


def process_snapshot():
    pids = []
    for name in PID_NAMES:
        process = subprocess.run(['pgrep', '-x', name], capture_output=True,
                                 text=True, check=False, timeout=4)
        pids.extend((name, pid) for pid in process.stdout.splitlines())
    return sorted(pids)


def session_snapshot():
    tmp = pathlib.Path(os.environ.get('TMPDIR', '/tmp'))
    return sorted(str(p) for p in tmp.glob('mcp-interop-*') if p.is_dir())


def private_snapshot():
    home = pathlib.Path.home()
    entries = []
    for name in PRIVATE_FILES:
        path = home / name
        try:
            stat = path.stat()
            # Avoid reading real credentials; capture metadata only and hash
            # the login Keychain as an additional persistence signal.
            digest = None
            if name.endswith('login.keychain-db'):
                with path.open('rb') as stream:
                    h = hashlib.sha256()
                    for part in iter(lambda: stream.read(256 * 1024), b''):
                        h.update(part)
                    digest = h.hexdigest()
            entries.append((name, stat.st_size, stat.st_mtime_ns, digest))
        except FileNotFoundError:
            entries.append((name, 'absent'))
    return tuple(entries)


def main(binary):
    if sys.platform != 'darwin':
        raise RuntimeError('macOS is required')
    if not pathlib.Path(binary).is_file():
        raise RuntimeError('compiled mcp-interop binary is missing')
    server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), DelayedHandler)
    server.daemon_threads = True
    threading.Thread(target=server.serve_forever, daemon=True).start()
    endpoint = f'http://127.0.0.1:{server.server_port}/mcp/delayed'
    env = {k: v for k, v in os.environ.items() if k not in SECRET_ENV}
    env.update(HTTP_PROXY='http://127.0.0.1:9', HTTPS_PROXY='http://127.0.0.1:9',
               ALL_PROXY='http://127.0.0.1:9', http_proxy='http://127.0.0.1:9',
               https_proxy='http://127.0.0.1:9', all_proxy='http://127.0.0.1:9',
               NO_PROXY='127.0.0.1,localhost,::1',
               no_proxy='127.0.0.1,localhost,::1')
    before_pids = process_snapshot()
    before_sessions = session_snapshot()
    before_private = private_snapshot()
    results = []
    try:
        for client in CLIENTS:
            for mode in ('timeout', 'sigint'):
                timeout = '1s' if mode == 'timeout' else '20s'
                args = [binary, 'test', endpoint, '--client', client,
                        '--timeout', timeout, '--json']
                proc = subprocess.Popen(args, env=env, stdin=subprocess.DEVNULL,
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                        text=True, start_new_session=True)
                started = time.monotonic()
                if mode == 'sigint':
                    time.sleep(0.9)
                    if proc.poll() is None:
                        os.kill(proc.pid, signal.SIGINT)
                try:
                    stdout, _stderr = proc.communicate(timeout=27)
                except subprocess.TimeoutExpired as exc:
                    # Only kill our own isolated subprocess group, never the
                    # user's existing client processes.
                    try:
                        os.killpg(proc.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    proc.communicate(timeout=3)
                    raise RuntimeError(f'{client}/{mode} exceeded the 27s cleanup bound') from exc
                elapsed = time.monotonic() - started
                try:
                    report = json.loads(stdout)
                    statuses = {stage['stage']: stage['status'] for stage in report[0]['stages']}
                    complete_pass = all(statuses.get(key) == 'pass'
                                        for key in ('reach', 'auth', 'init', 'tools'))
                except (ValueError, LookupError, TypeError):
                    statuses, complete_pass = {'result': 'no usable JSON'}, False
                if proc.returncode == 0 or complete_pass:
                    raise RuntimeError(f'{client}/{mode} incorrectly returned PASS')
                if elapsed > 27:
                    raise RuntimeError(f'{client}/{mode} wall-clock cleanup exceeded bound')
                print(f'{client} {mode}: exit={proc.returncode}; statuses={statuses}; bounded={elapsed:.1f}s')
                results.append((client, mode))
            time.sleep(0.3)
        # Interrupt a real client's second repeated attempt after the first
        # result set is atomically committed. The wrapper must preserve the
        # first attempt and publish an incomplete (never clean) summary.
        work_root = pathlib.Path(binary).parent
        manifest = work_root / 'repeat-manifest.json'
        output = work_root / 'repeat-results'
        manifest.write_text(json.dumps({
            'schema_version': 1,
            'execution_context': 'trusted_real_client',
            'targets': [{'id': 'delayed', 'endpoint': {
                'source': 'environment',
                'variable': 'MCP_INTEROP_SUITE_ENDPOINT_DELAYED',
            }, 'deployment_id': 'delayed-local', 'clients': [
                {'id': 'codex', 'auth': 'none'}]}],
        }), encoding='utf-8')
        manifest.chmod(0o600)
        repeat_env = dict(env, MCP_INTEROP_SUITE_ENDPOINT_DELAYED=endpoint)
        repeat_cmd = [binary, 'suite', 'repeat', str(manifest),
                      '--output-dir', str(output), '--attempts', '3',
                      '--timeout', '3s', '--json']
        repeat = subprocess.Popen(repeat_cmd, env=repeat_env, stdin=subprocess.DEVNULL,
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                  text=True, start_new_session=True)
        first_index = output / 'attempt-01' / 'index.json'
        deadline = time.monotonic() + 18
        while not first_index.is_file() and time.monotonic() < deadline:
            if repeat.poll() is not None:
                raise RuntimeError('repeat exited before first evidence was committed')
            time.sleep(0.02)
        if not first_index.is_file():
            raise RuntimeError('repeat first attempt never wrote an index')
        if repeat.poll() is None:
            os.kill(repeat.pid, signal.SIGINT)
        try:
            repeat_stdout, _repeat_stderr = repeat.communicate(timeout=20)
        except subprocess.TimeoutExpired as exc:
            try:
                os.killpg(repeat.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            repeat.communicate(timeout=3)
            raise RuntimeError('interrupted suite repeat did not exit') from exc
        try:
            repeat_report = json.loads((output / 'repeat-report.json').read_text())
        except (OSError, ValueError) as exc:
            raise RuntimeError('interrupted suite repeat did not save a valid report') from exc
        if repeat.returncode != 1 or repeat_report['complete'] or repeat_report['decision'] != 'incomplete':
            raise RuntimeError(f'interrupted repeat was not fail-closed: rc={repeat.returncode} '
                               f'decision={repeat_report.get("decision")}')
        completed = repeat_report['completed_attempts']
        if completed < 1 or completed >= 3:
            raise RuntimeError('expected 1 or 2 retained attempts after interruption')
        for ref in repeat_report['attempt_indexes']:
            index_path = output / ref
            if not index_path.is_file():
                raise RuntimeError('completed attempt index disappeared after SIGINT')
            index = json.loads(index_path.read_text())
            if index['runs'][0]['outcome'] == 'pass':
                raise RuntimeError('deliberately stalled endpoint fabricated PASS')
        print(f'codex suite repeat SIGINT: exit=1; decision=incomplete; '
              f'retained={completed}/3; no attempt evidence lost')
        results.append(('codex', 'suite-repeat-sigint'))
        for _ in range(30):
            if process_snapshot() == before_pids and session_snapshot() == before_sessions:
                break
            time.sleep(0.1)
        if process_snapshot() != before_pids:
            raise RuntimeError('client PID set differs from the baseline')
        if session_snapshot() != before_sessions:
            raise RuntimeError('isolated mcp-interop session directory leaked')
        if private_snapshot() != before_private:
            raise RuntimeError('normal client configuration or login Keychain changed')
        if len(results) != 7:
            raise RuntimeError('not all cancellation cases completed')
        print(f'PASS: six real-client interruption cases plus incomplete real suite repeat; delayed fixture requests={DelayedHandler.count}; '
              'no false PASS; no client PID/session/config/Keychain changes')
    finally:
        server.shutdown()
        server.server_close()

if __name__ == '__main__':
    if len(sys.argv) != 2:
        raise SystemExit('usage: e2e-real-client-interruptions.py <compiled-binary>')
    try:
        main(sys.argv[1])
    except Exception as exc:
        raise SystemExit(f'Interruption acceptance failed: {exc}') from None
