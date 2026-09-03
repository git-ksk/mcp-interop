# Codex stable-adapter acceptance

[English](codex-stable-acceptance.md) | [日本語](codex-stable-acceptance.ja.md)

This record closes the evidence gap for promoting the shipped Codex adapter from `beta` to `stable` for one deliberately narrow scope:

- client: Codex CLI;
- runner: macOS 26.5 (25F71), arm64;
- path: controlled localhost, non-OAuth core interoperability path;
- required stages: `reach/auth/init/tools=PASS`;
- safety gates: normal user config unchanged, login Keychain DB unchanged, no new Codex process, no leaked `mcp-interop` session, and no `tools/call`.

The stable claim does **not** include OAuth, macOS amd64, Linux, Windows, or an inferred Codex version range.

## Repeated exact-version evidence

Both runs used the same `scripts/e2e-real-clients.sh` real-client boundary on current pre-v1 main `8e12549f0563892b09b2ec6127eed2071fa376bd`.

| Exact Codex version | Result | Fixture protocol observation | Safety gates |
| --- | --- | --- | --- |
| `codex-cli 0.133.0` | `reach/auth/init/tools=PASS` | `initialize` proposed `2025-06-18` | all PASS; `tools/call` avoided |
| `codex-cli 0.152.1` | `reach/auth/init/tools=PASS` | `initialize`, `notifications/initialized`, and `tools/list` observed with `2025-06-18` | all PASS; `tools/call` avoided |

The `0.133.0` binary was retained before the Homebrew upgrade and executed from an isolated temporary PATH against a detached worktree of the same current main. The installed Homebrew Codex was then `0.152.1` and passed the same harness.

These are exact observed points only. `0.133.0` and `0.152.1` do not imply that versions between or after them are tested.

## Stable-gate decision

For this advertised stable scope:

- `repeat_path_version_coverage`: **met** — the same PASS-contributing non-OAuth core path has real-client evidence on two exact Codex versions;
- `advertised_platform_coverage`: **met** — the stable advertised scope is explicitly narrowed to macOS arm64, where retained exact evidence exists;
- all beta criteria, measurement-surface stability, and regression-maintenance criteria remain **met**.

OAuth remains an explicit opt-in feature outside this stable maturity scope and is tracked separately by the pre-v1 OAuth revalidation work.
