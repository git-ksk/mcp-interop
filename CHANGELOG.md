# Changelog

All notable project changes will be summarized here. GitHub Releases remain the authoritative source for published release artifacts and checksums.

## Unreleased

## v0.3.0 — 2026-08-10

### Fixed

- OpenAI Reference Pattern tool OAuth aggregation now reports partial evidence as WARN instead of N/A when static tool metadata was not observed.

### Added

- Versioned OpenAI Reference Pattern metadata (`profile_revision`, `observed_date`, `source`) and a documented manual real-ChatGPT secret-free dogfood workflow.
- Controlled loopback OAuth scope fixture with `fixture.read` / `fixture.write`, insufficient-scope tool challenges, and an explicit release gate.
- `mcp-interop evidence validate`, `summary`, and `merge` utilities using the same strict secret-free Runtime Evidence decoder as `diagnose`.
- Conflict-safe evidence merging that emits canonical schema v3 JSON and structural summaries that never echo observed values.
- Runtime Evidence schema v3 with independent `tool_metadata` and `tool_challenge` sections, while preserving v1/v2 input compatibility.
- v2 `tool_auth` normalization to the same evaluation boundaries used by v3, with mixed v2/v3 tool shapes rejected.
- Runtime Evidence coverage counters (`observed`, `passed`, `failed`, `unknown`, `not_applicable`) in JSON and text diagnostics.
- Explicit `not_applicable` runtime-check status, rendered as `N/A`, for observations whose trigger condition did not occur in the captured flow.

### Changed

- Tool-level OAuth diagnostics now evaluate per-tool OAuth `securitySchemes` independently from runtime reauthorization challenges. An explicitly missing OAuth security scheme is a failure even when the current grant already satisfies the tool.
- `mcp/www_authenticate` is `N/A` rather than `WARN` when sanitized evidence explicitly says no reauthorization challenge was expected. OpenAI reference aggregation ignores `N/A` checks when deciding whether all applicable supplied observations passed.

## v0.2.0 — 2026-08-10

### Added

- Structured OAuth `reason_code` values while preserving the existing `reach` / `auth` / `init` / `tools` result contract, including real-client `DCR_UNSUPPORTED` / `DCR_FAILED` and Runtime Evidence v2 reason codes.
- `mcp-interop diagnose <url> --profile chatgpt` OAuth/server preflight for Protected Resource Metadata, authorization-server discovery, CIMD/DCR registration compatibility, token endpoint authentication, PKCE S256, and refresh-token advisory evidence.
- Exact observed ChatGPT CIMD validation for the stable client metadata URL, redirect URI, token endpoint authentication method intersection, and JWKS reachability.
- Secret-free ChatGPT Runtime Evidence v2 with explicit `cimd` / `dcr` / `predefined` registration strategy and separate authorization-request, token-request, resource-request, and tool-auth observations.
- Runtime correlation for OAuth `resource` consistency across authorization/token requests, PKCE verifier presence, token endpoint authentication/errors, and Bearer delivery to the MCP resource server.
- Resource-server verification evidence for token signature, issuer, audience/resource, expiry, and required scopes.
- OpenAI authenticated MCP reference-pattern diagnostics covering registration, PKCE, token auth, Bearer delivery, resource-server verification, and tool-level OAuth signals without treating provider-specific Auth0 setup as a protocol requirement.
- Tool-level OAuth signal diagnostics for `securitySchemes` and `_meta["mcp/www_authenticate"]`, evaluated only when corresponding sanitized Runtime Evidence is explicitly available.
- Conservative handling of multiple advertised authorization servers; issuer-specific token-auth expectations remain unknown until the selected issuer is observable.
- Backward-compatible legacy Runtime Evidence v1 input, normalized internally to the v2 diagnostic model.
- Expanded English and Japanese documentation for ChatGPT diagnostics, reason codes, troubleshooting, architecture, and conformance/interoperability boundaries.

### Current limitations

- ChatGPT real-client automation remains research-only. v0.2.0 provides ChatGPT OAuth/server preflight, observed Runtime Evidence, and OpenAI reference-pattern diagnostics, not a ChatGPT live adapter or a real-client `reach/auth/init/tools` PASS.
- Cursor OAuth completion and authenticated tool discovery are not shipped yet.
- Antigravity OAuth completion remains intentionally disabled until credential isolation from the normal macOS Keychain can be proven safe.
- VS Code remains research-only pending a supported direct lifecycle/tool-discovery automation surface.
- Tool-level OAuth signal checks are only meaningful when those signals were actually observed and supplied as sanitized Runtime Evidence.
- Automatic end-to-end correlation between real-client OAuth failures and profile/runtime capability evidence is not yet complete; the evidence layers remain deliberately separate.

## v0.1.0 — 2026-08-09

First public release of `mcp-interop`.

### Added

- Real-client interoperability testing for Remote MCP servers across Codex CLI, Cursor CLI (beta), and Antigravity CLI on macOS (beta).
- Four-stage `reach` / `auth` / `init` / `tools` result model with conservative `pass` / `fail` / `skip` / `unknown` semantics.
- Codex OAuth flow with isolated file-backed credential storage.
- Isolated temporary client configuration and state with secret redaction.
- Cross-client text summary and machine-readable JSON output.
- Repeatable real-client macOS E2E harness using a localhost MCP fixture and cleanup/isolation gates.
- Three-OS CI plus release-build smoke testing.
- Automated checksummed release archives for macOS, Linux, and Windows on amd64 and arm64.

### Current limitations

- Cursor OAuth completion and authenticated tool discovery are not enabled yet.
- Antigravity OAuth completion is intentionally disabled until credential storage can be proven safely isolated from the macOS Keychain.
- The Antigravity live adapter is currently macOS-only.
- VS Code remains research-only pending a supported direct lifecycle/tool-discovery automation surface.
