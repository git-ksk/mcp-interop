# Changelog

All notable project changes will be summarized here. GitHub Releases remain the authoritative source for published release artifacts and checksums.

## Unreleased

### Added

- Optional machine-readable `reason_code` on stage results without changing the existing `reach` / `auth` / `init` / `tools` status contract.
- Initial Codex OAuth diagnostics for `DCR_UNSUPPORTED` and `DCR_FAILED`, classified from explicit real-client error evidence without exposing raw app-server error text.
- English and Japanese reason-code documentation and troubleshooting guidance.

### Planned

- Correlate client-observed OAuth failures with MCP Protected Resource Metadata and authorization-server metadata, including CIMD and DCR capability advertising, without guessing registration URLs.
- Complete Cursor OAuth through authenticated tool discovery.
- Establish a safe Antigravity OAuth completion boundary before enabling automated authorization/token exchange.
- Continue improving diagnostic output for inconclusive client results with additional conservative reason codes and sanitized traces.
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
