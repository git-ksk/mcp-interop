# Usage guide

[English](usage.md) | [日本語](usage.ja.md)

Recipes for live tests, saved evidence, regression checks, OAuth, and diagnostics. Start with the [README](../README.md) for installation and your first run.

## Choose a command

For ordinary use, start with `clients` and `test`. The remaining commands are for regression, evidence, maturity, or preflight workflows.

If you are not sure which command to use:

| Goal | Command |
| --- | --- |
| See which supported clients are installed | `mcp-interop clients` |
| Test one or more real clients now | `mcp-interop test` |
| Save one live run for later comparison | `mcp-interop test --output` |
| Check a suite file without running clients | `mcp-interop suite validate` |
| Run the same declared targets across multiple clients | `mcp-interop suite run` |
| Compare raw result sets with one or more attempts | `mcp-interop suite compare` |
| Accept an immutable baseline snapshot | `mcp-interop baseline create` |
| Compare an accepted baseline with retained attempts | `mcp-interop baseline compare` |
| Classify the installed exact client version from observed evidence | `mcp-interop compatibility query` |
| Review evidence-based maturity for shipped adapters | `mcp-interop maturity` |
| Review research-client graduation blockers | `mcp-interop graduation` |
| Validate separate optional-capability evidence | `mcp-interop capability validate` |
| Run metadata-only preflight diagnostics | `mcp-interop diagnose` |

Detect known clients on the local machine:

```console
mcp-interop clients
mcp-interop clients --json
```

The `tier` shown by `clients` is roadmap/delivery placement, not evidence maturity. In v0.9.0, use `mcp-interop maturity` for the separately reviewed `research_only` / `beta` / `stable` decision; this does not execute or detect a client.

Capability profile v1 is included in v0.9.0. `mcp-interop capability validate` validates a separate evidence document and never changes core `reach/auth/init/tools` PASS. See [Capability profile v1](capability-profile-v1.md) ([日本語](capability-profile-v1.ja.md)).

v0.9.0 also adds `mcp-interop graduation`, a no-execution common gate for research candidates. Copilot CLI, VS Code, ChatGPT, and Claude web/Desktop all remain `research_only`; none is eligible to ship yet. Live test, suite, and compatibility client selection are wired to the validated shipped-adapter maturity catalog so a research candidate cannot enter through a separate allowlist. See [Real-client adapter graduation gate](adapter-graduation-gate.md) ([日本語](adapter-graduation-gate.ja.md)).

Run one live adapter:

```console
mcp-interop test https://example.com/mcp --client codex
mcp-interop test https://example.com/mcp --client cursor
mcp-interop test https://example.com/mcp --client antigravity
```

Run multiple implemented adapters sequentially:

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

When more than one client is selected, the text output starts with a cross-client summary and then prints the detailed result for each client:

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   codex-cli 0.133.0
Cursor CLI       PASS   PASS  PASS  PASS   2026.08.04-aaa8809
Antigravity CLI  PASS   PASS  PASS  PASS   1.1.11
```

JSON output remains an array, preserving the existing machine-readable contract.

### Portable regression artifacts

Export the same live run into a separate versioned, secret-safe local artifact without changing the existing result shape:

```console
mcp-interop test https://example.com/mcp --client codex --output result.json
```

This default remains artifact schema v1. It records the exact detected client version, the `mcp-interop` runner/process OS and architecture, runner/runtime context, invocation auth mode, evidence provenance, and the existing four stage status/reason results. `platform.arch` is **not** proof of the real client binary architecture. Before emitting new portable evidence, mcp-interop non-invasively inspects Mach-O/ELF/PE executable metadata where available and fails closed on a known runner/client architecture mismatch; script/wrapper launchers remain explicitly unknown rather than inferred. The raw endpoint URL is not persisted; query values are excluded before deriving the endpoint fingerprint. Human stage messages and diagnostic payloads are also excluded.

If the endpoint path itself contains credential material, use schema v2 protected-path identity instead of exporting v1:

```console
mcp-interop test 'https://example.com/mcp/<protected-path>' \
  --client codex \
  --output result.json \
  --deployment-id production-a
```

The deployment ID is persisted verbatim and must be a stable non-secret operator label, never a value derived from the protected path. In this mode the artifact persists only the canonical origin plus that deployment ID; the path/query/userinfo/fragment are neither persisted nor hashed. Ordinary text/JSON output also avoids echoing the protected path. v1↔v2 comparison is rejected explicitly rather than guessing an identity mapping.

Compare two artifacts across client versions or repeated runs:

```console
mcp-interop compare old.json new.json
mcp-interop compare old.json new.json --json
mcp-interop compare old.json new.json --fail-on-regression
```

The comparison explicitly reports `PASS_TO_FAIL`, `PASS_TO_UNKNOWN`, `PASS_TO_SKIP`, reason-code changes, and missing baseline evidence. A client-version change by itself is not a regression. `--fail-on-regression` exits `1` only when one of those regression/evidence-loss conditions is present; malformed or unsupported artifacts are usage/input errors and exit `2`.

See [Live interoperability result artifact schema v1](live-result-schema-v1.md) ([日本語](live-result-schema-v1.ja.md)) and [schema v2 protected-path identity](live-result-schema-v2.md) ([日本語](live-result-schema-v2.ja.md)) for the exact compatibility, secret-safety, pairing, and migration contracts.

### Repeatable multi-client suites

For repeatable multi-client checks, declare the targets and clients in `suite.json`. Start by validating the file; validation does not launch clients or read endpoint values:

```console
mcp-interop suite validate suite.json
mcp-interop suite validate suite.json --json
```

A `trusted_real_client` manifest can then run every declared target/client combination and save the results as one coherent schema-v2 artifact set:

```console
export MCP_INTEROP_SUITE_ENDPOINT_PRODUCTION_A='https://example.com/mcp/<protected-path>'
mcp-interop suite run suite.json --output-dir suite-results
mcp-interop suite run suite.json --output-dir suite-results-json --json
```

The suite resolves every endpoint before launching the first client, executes each run through the same live-test path used by `mcp-interop test`, and writes `index.json` plus one protected-path schema-v2 artifact per run. The index never stores endpoint URLs or endpoint environment-variable names. A non-PASS/missing-client result remains represented in the set and makes the command exit `1`; invalid manifests, unresolved endpoints, or an existing output directory fail before execution with exit `2`.

Manifest v1 never stores a Remote MCP endpoint URL. Hosted fixture declarations cannot select network targets or OAuth and remain validation-only in v0.9.0; repository PR CI continues to use controlled localhost fixture gates separately rather than executing arbitrary suite manifests. Trusted real-client suites reference a target-specific `MCP_INTEROP_SUITE_ENDPOINT_*` variable and require a non-secret `deployment_id`. See [Suite manifest v1](suite-manifest-v1.md) ([日本語](suite-manifest-v1.ja.md)) and [Suite result set v1](suite-result-set-v1.md) ([日本語](suite-result-set-v1.ja.md)).

Compare a baseline result set with one or more retained attempts:

```console
mcp-interop suite compare baseline-results attempt-1 attempt-2
mcp-interop suite compare baseline-results attempt-1 attempt-2 --json
mcp-interop suite compare baseline-results attempt-1 attempt-2 --fail-on-regression
```

Every attempt remains in the report. A failed/unknown first attempt followed by a passing retry is `regression_and_unstable`, not a clean PASS. The gated form exits `1` for regression or unstable evidence, `0` for a clean report, and `2` for invalid/unreadable input. See [Suite regression report v1](suite-regression-report-v1.md) ([日本語](suite-regression-report-v1.ja.md)).

Accept a result set as an immutable baseline when every declared run contains complete real-client evidence and an exact client version:

```console
mcp-interop baseline create suite-results --output-dir baselines/current
mcp-interop baseline create suite-results-new --output-dir baselines/next --supersedes baselines/current
mcp-interop baseline compare baselines/current attempt-1 attempt-2 --fail-on-regression
```

Compatibility is modeled from exact observed client-version/platform points only; unobserved versions remain `untested`, and version changes alone are never regressions. See [Compatibility envelope v1](compatibility-envelope-v1.md) ([日本語](compatibility-envelope-v1.ja.md)) for the `tested` / `untested` / `stale` / `known_broken` / `regressed` / `unknown` semantics.

To list every exact version/platform point from retained result-set/baseline evidence without detecting or executing a client, use `mcp-interop compatibility matrix`. Retry history and evidence gaps remain visible. The repository-retained current observations are documented in [Exact observed client coverage](observed-coverage.md) ([日本語](observed-coverage.ja.md)).

To classify the currently installed exact client version without changing the existing `clients --json` schema:

```console
mcp-interop compatibility query --client codex --target production-a --deployment-id production-a --baseline baselines/current --observation attempt-1 --json
```

For version-change staleness, repeated `--observation` inputs are explicit oldest-to-newest collection order rather than a wall-clock sort. Age-based staleness requires `--max-age-seconds N --trust-executed-at-clock`; future timestamps are treated conservatively as stale.

Baseline creation never overwrites an existing directory. The result set is copied into the bundle, bound by a deterministic digest, and selected explicitly by path; retries and client auto-updates cannot silently replace the accepted anchor. This is workflow-local immutability/tamper-consistency, **not cryptographic authenticity**. `mcp-interop baseline verify` revalidates a bundle and can verify an explicitly supplied predecessor link without claiming authenticated provenance. A superseding baseline keeps the prior baseline unchanged and records its fingerprint. Baseline comparison fails closed on manifest, execution-context, deployment-fingerprint, or platform mismatch while preserving suite regression report v1 output. See [Suite baseline v1](suite-baseline-v1.md) ([日本語](suite-baseline-v1.ja.md)).

Self-hosted real-client GitHub Actions are a separate privileged path: the manual macOS workflow is main-only, uses a main-only Environment policy, records exact run provenance, and never accepts remote suite endpoints or OAuth credentials. See [Self-hosted real-client CI security boundary](self-hosted-ci-security.md) ([日本語](self-hosted-ci-security.ja.md)).

### OAuth

OAuth flows are always explicit opt-in:

```console
mcp-interop test https://example.com/mcp --client codex --oauth
mcp-interop test https://example.com/mcp --client cursor --oauth
mcp-interop test https://example.com/mcp --client antigravity --oauth
```

For Codex, `mcp-interop` prints the authorization URL to stderr and waits for the real Codex OAuth callback. The URL contains short-lived OAuth state and should not be shared.

Cursor uses the real Cursor MCP login path inside an isolated temporary HOME/workspace and proves authenticated discovery with `mcp list-tools`. Callback details are version-specific and are not hard-coded.

Antigravity enters the real `/mcp` manager inside an isolated PTY. OAuth token persistence is confined to the isolated temporary HOME; authorization codes and token contents are not persisted in `mcp-interop` evidence. See [Antigravity OAuth live-test boundary](antigravity-oauth.md) ([日本語](antigravity-oauth.ja.md)).

### ChatGPT OAuth/server preflight

Check whether a Remote MCP server's published OAuth metadata has a known blocking mismatch with ChatGPT's documented MCP authentication path:

```console
mcp-interop diagnose https://example.com/mcp --profile chatgpt
```

The profile follows Protected Resource Metadata and authorization-server discovery, then checks CIMD/DCR registration strategy, token endpoint authentication methods, PKCE S256, and refresh-token related metadata.

A server with `client_id_metadata_document_supported: true` can pass registration preflight without a `registration_endpoint`: ChatGPT can use CIMD and does not require DCR for that path.

If sanitized authorization-server logs expose the exact non-secret ChatGPT `client_id` CIMD URL and `redirect_uri`, they can be checked too:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --client-id 'https://chatgpt.com/oauth/.../client.json' \
  --redirect-uri 'https://chatgpt.com/connector/oauth/...'
```

That extended check validates the CIMD document, redirect URI, client/server token auth method intersection, and JWKS when `private_key_jwt` is available.

This command does **not** operate the ChatGPT UI, complete OAuth, or claim a real ChatGPT client PASS. See [ChatGPT connection diagnostics](chatgpt-diagnostics.md) ([日本語](chatgpt-diagnostics.ja.md)).

### Secret-free ChatGPT Runtime Evidence

When sanitized Authorization Server logs can observe token-request **presence/match signals rather than values**, `diagnose` can correlate those observations as a separate Runtime Evidence layer:

```console
mcp-interop diagnose https://example.com/mcp \
  --profile chatgpt \
  --runtime-evidence runtime-evidence.json
```

Minimal v3 evidence:

```json
{
  "schema_version": 3,
  "registration": {
    "strategy": "cimd",
    "client_metadata_url": "https://chatgpt.com/oauth/.../client.json"
  },
  "token_request": {
    "resource_matches": true,
    "client_assertion_present": false
  },
  "tool_metadata": {
    "oauth2_security_scheme_present": true
  },
  "tool_challenge": {
    "expected": false
  }
}
```

Additional authorization/token/resource/tool observations are optional. Schema v3 keeps `tool_metadata` and `tool_challenge` independent; schema v2 `tool_auth` and legacy v1 input remain accepted for compatibility. Missing observations remain `WARN / unknown`; they are never inferred. Unknown JSON fields are rejected, so tokens, authorization codes, PKCE verifier values, raw client assertions, cookies, and credentials are not accepted.

Preflight, Runtime Evidence, and real-client interoperability remain separate evidence layers. A server can therefore show `PREFLIGHT PASS` and Runtime Evidence `FAIL` with `TOKEN_AUTH_METHOD_MISMATCH`.

### Evidence utilities

Use the same strict secret-free decoder outside `diagnose` to validate, summarize, or combine evidence fragments:

```console
mcp-interop evidence validate runtime-evidence.json
mcp-interop evidence summary runtime-evidence.json
mcp-interop evidence merge authorization.json resource.json tool.json -o runtime-evidence.json
```

`summary` prints only structural coverage (section names and supplied-field counts), never observed values or metadata URLs. `merge` fails on conflicting observations instead of using last-write-wins and emits canonical schema v3 JSON. Unknown fields remain rejected, so these commands do not create a second path for ingesting tokens, authorization codes, raw client assertions, cookies, or other credentials.

## Codex adapter (stable: macOS arm64 non-OAuth core path)

The Codex adapter:

1. creates an isolated temporary `CODEX_HOME`;
2. forces MCP OAuth credential storage to a file inside that isolated home;
3. writes only the test Remote MCP endpoint into the isolated config;
4. starts the real `codex app-server` process;
5. initializes the app-server control connection;
6. asks Codex for `mcpServerStatus/list` with tool inventory;
7. optionally runs Codex's own OAuth flow when `--oauth` is explicitly enabled;
8. reports what Codex itself observed;
9. removes the temporary session, including OAuth credentials.

It does **not** send a model prompt and does not require model/API usage for the live MCP inventory or OAuth test.

### OAuth isolation

The temporary Codex config sets:

```toml
mcp_oauth_credentials_store = "file"
```

This prevents the test from using the normal automatic/keyring MCP OAuth storage mode. During an OAuth test, Codex stores credentials under the temporary `CODEX_HOME`; the whole test session is deleted during cleanup.

### Current Codex limitations

- OAuth is interactive and explicit. Without `--oauth`, a server that reports `notLoggedIn` remains an incomplete test and exits non-zero.
- Current Codex app-server versions can expose an unreachable server and a legitimate zero-tool server in the same empty-inventory shape. `mcp-interop` therefore reports those stages as `unknown` instead of inventing a pass/fail result.
- The adapter relies on the installed Codex app-server MCP status and OAuth surfaces. Older or future Codex versions that do not expose the required methods may return an inconclusive/error result until the adapter is updated.

## Cursor adapter (stable on macOS arm64 core path)

The Cursor adapter:

1. creates an isolated temporary `HOME` and workspace;
2. writes only the target Remote MCP endpoint to `<workspace>/.cursor/mcp.json`;
3. invokes the installed Cursor CLI's MCP management surface;
4. attempts `mcp enable`, then queries `mcp list` and `mcp list-tools`;
5. when `--oauth` is explicit, invokes the real Cursor MCP login path inside the isolated session;
6. treats successful authenticated `mcp list-tools` as direct evidence from Cursor's real MCP client;
7. removes all temporary Cursor state during shared session cleanup.

The adapter never sends a Cursor model prompt. A fresh isolated HOME prevents the test from reusing normal Cursor MCP auth/config state.

### Current Cursor limitations

- OAuth is explicit opt-in; the adapter does not silently start login for an OAuth-required target without `--oauth`.
- Callback addresses are version-specific and must not be treated as a permanent fixed port.
- MCP management output is human-readable rather than a dedicated JSON contract, so the adapter keeps interpretation deliberately conservative.
- Real OAuth validation has been completed on macOS for the tested Cursor CLI version; additional client-version/OS evidence should be added as the adapter matures.

## Antigravity adapter (stable on macOS arm64 core path)

The Antigravity adapter currently has a live implementation for macOS only:

1. creates an isolated temporary `HOME` and workspace;
2. writes the target Remote MCP endpoint to the temporary `~/.gemini/config/mcp_config.json` using the current `serverUrl` field;
3. writes `modelProvider: "gemini"` to isolated Antigravity CLI settings, removes ambient Gemini credential/endpoint overrides, and injects a fixed non-secret `GEMINI_API_KEY` sentinel so `agy` uses its documented no-account mode instead of a normal-user Keychain account session;
4. starts the installed `agy` process under a PTY without a model prompt;
5. in no-auth mode, observes machine-readable tool schema state under the isolated `~/.gemini/antigravity-cli/mcp/<server>/` cache;
6. when `--oauth` is explicit, enters the real Antigravity `/mcp` manager and forwards authorization-code input directly to the isolated PTY;
7. observes OAuth token persistence only through metadata for isolated `~/.gemini/antigravity/mcp_oauth_tokens.json`, never by opening the token file;
8. captures and reaps only descendants of the test PTY wrapper before shared session cleanup, then removes the temporary HOME/workspace.

### Current Antigravity limitations

- The live adapter remains macOS-only until equivalent real-client evidence exists on other operating systems.
- The login Keychain before/after comparison proves non-mutation, not non-read by itself. Normal-user account-session non-reuse relies on Antigravity's documented `modelProvider: "gemini"` + API-key mode and is release-gated with the real client; the core credential-isolation path was revalidated with `agy 1.1.22`.
- OAuth is explicit opt-in and still depends on the tested Antigravity interactive `/mcp` surface.
- On the tested `agy 1.1.11` OAuth path, authenticated `initialize` and `tools/list` can complete without materializing the same client-side tool cache used by no-auth mode. The generic result therefore keeps `init/tools=unknown` rather than inferring pass from authentication alone.
- The controlled localhost OAuth E2E independently requires authenticated `initialize`, `notifications/initialized`, and `tools/list` server-side evidence. See [Antigravity OAuth live-test boundary](antigravity-oauth.md) ([日本語](antigravity-oauth.ja.md)).
- The tool cache is an observed Antigravity client surface rather than a stable cross-vendor protocol API, so version information must remain part of every result.

## Safety and isolation

- **Real clients, not emulators.** Client-specific checks invoke the installed client wherever practical.
- **No model benchmark required.** The core interoperability path does not ask a model to choose or call tools.
- **Do not mutate user configuration.** Live adapters must use isolated/temporary profiles or return `skip`/`unknown`.
- **Private temporary state.** Session directories are created with owner-only permissions where the OS supports them, and test configuration/credential files use owner-only permissions on POSIX systems where applicable.
- **Credential redaction.** Bearer/OAuth material and credential-like Remote MCP URL query parameters are redacted from reports.
- **OAuth is explicit.** Authorization only starts when the caller opts in and the selected adapter has a verified isolated OAuth implementation.
- **Preflight is not live evidence.** Profile diagnostics may fetch public metadata, but they never promote server metadata compatibility into a real-client `reach/auth/init/tools` PASS.
- **No hosted service required.** The core tool runs locally and in CI without a project-operated backend.

## Real-client E2E on macOS

The repository includes a deterministic localhost MCP fixture plus a release-gate runner for the installed real clients:

```console
bash scripts/e2e-real-clients.sh
```

By default it tests Codex, Cursor, and Antigravity. A subset can be selected explicitly:

```console
MCP_INTEROP_CLIENTS=codex,cursor bash scripts/e2e-real-clients.sh
```

The harness:

- builds and tests the current checkout before E2E;
- starts a Go fixture bound only to `127.0.0.1` with one deterministic `ping` tool;
- runs each selected real client against a distinct fixture path;
- requires the fixture to observe `initialize`, `notifications/initialized`, and `tools/list` for every selected client;
- fails if any `tools/call` occurs;
- removes common model/API key environment variables and points normal outbound HTTP(S) proxy variables at an unreachable loopback port while exempting localhost;
- compares curated user MCP/config/credential metadata before and after the run, including the login Keychain database by default;
- detects newly leaked `codex`, `cursor-agent`, or `agy` processes without killing processes by name;
- detects newly leaked `mcp-interop-*` temporary session directories.

OAuth-specific Cursor and Antigravity harnesses additionally exercise their real OAuth paths against controlled loopback fixtures and keep authorization codes/tokens out of persisted evidence.

The network controls are defense in depth, not a packet-capture proof that a client implementation cannot bypass proxy settings. The core adapter paths used by this harness do not submit model prompts.

For a shared development Mac where unrelated Keychain writes would make the database hash noisy, the Keychain check can be explicitly skipped:

```console
MCP_INTEROP_SKIP_KEYCHAIN=1 bash scripts/e2e-real-clients.sh
```

Do not skip that gate for a release-candidate validation unless the equivalent Keychain comparison is performed separately.

A manual GitHub Actions workflow is also included at `.github/workflows/e2e-real-macos.yml`. It intentionally targets a self-hosted runner labeled:

```text
self-hosted, macOS, ARM64, mcp-interop-e2e
```

The runner is expected to have the real Codex, Cursor, and Antigravity CLIs installed. GitHub-hosted CI does **not** install those external clients; normal CI validates the fixture, the harness syntax/build path, adapter regression tests, and release build path without making external client availability a pull-request dependency.

