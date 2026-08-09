# Changelog

All notable project changes will be summarized here. GitHub Releases remain the authoritative source for published release artifacts and checksums.

## Unreleased

### Added

- Optional machine-readable `reason_code` on stage results without changing the existing `reach` / `auth` / `init` / `tools` status contract.
- Initial Codex OAuth diagnostics for `DCR_UNSUPPORTED` and `DCR_FAILED`, classified from explicit real-client error evidence without exposing raw app-server error text.
- Profile-based `mcp-interop diagnose <url> --profile chatgpt` preflight diagnostics for Protected Resource Metadata, authorization-server discovery, CIMD/DCR registration compatibility, ChatGPT token endpoint auth methods, PKCE S256, refresh-token advisory evidence, and optional observed ChatGPT CIMD/redirect/JWKS validation.
- Versioned secret-free ChatGPT Runtime Evidence v2 with explicit `cimd` / `dcr` / `predefined` registration strategy, separate authorization/token/resource/tool-auth observations, and backward-compatible legacy v1 input.
- OpenAI Reference Pattern correlation based on the documented ChatGPT OAuth flow and the structural authenticated-MCP pattern demonstrated by `openai/openai-mcpkit`, without treating Auth0-specific setup as a protocol requirement.
- Runtime diagnostics for PKCE, resource consistency, token endpoint authentication/errors, bearer delivery, resource-server signature/issuer/audience/expiry/scope verification, and tool-level OAuth linking signals.
- Conservative runtime reason codes including `TOKEN_AUTH_METHOD_MISMATCH`, `CLIENT_AUTH_REJECTED`, `TOKEN_AUDIENCE_MISMATCH`, `ACCESS_TOKEN_NOT_ATTACHED`, and tool OAuth metadata/challenge failures.
- Conservative handling for ambiguous multiple authorization servers; token-auth expectations remain unknown until the selected issuer is observable.
- English and Japanese reason-code, ChatGPT diagnostic, and troubleshooting documentation.

### Planned

- Correlate additional real-client OAuth failures with metadata/runtime diagnostic evidence while keeping server preflight, observed runtime evidence, OpenAI reference-pattern comparison, and real-client interoperability verdicts separate.
- Complete Cursor OAuth through authenticated tool discovery.
- Establish a safe Antigravity OAuth completion boundary before enabling automated authorization/token exchange.
- Continue improving diagnostic output for inconclusive client results with additional conservative reason codes and sanitized traces.
- Research a supported ChatGPT real-client automation surface without browser DOM scraping before adding any ChatGPT live adapter.
- Treat ChatGPT mTLS client-certificate evidence as a future advisory/runtime observation rather than a current interoperability requirement.
- Revisit additional real MCP clients when they expose a supported, automatable lifecycle/tool-discovery surface.

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
- VS Code remains research-only pending a supported direct lifecycle/tool-discovery surface.
