# Contributing to mcp-interop

[English](CONTRIBUTING.md) | [日本語](CONTRIBUTING.ja.md)

Thanks for helping improve real-client MCP interoperability testing.

## Before opening a pull request

1. Search existing issues and pull requests for related work.
2. For a new client adapter, open or reference an issue that documents the client's supported or safely observable MCP surface and the isolation strategy.
3. New client work should progress in this order: issue/research -> safe observable-surface proof -> bounded PoC -> implementation. Do not implement a live adapter first and decide its evidence model later.
4. Do not add a live adapter that silently modifies a user's normal client configuration or credential state.
5. Keep unsupported, blocked, experimental, and research-only clients labeled as such. Partial observations must not be presented as shipped support.

## Development

Requirements:

- Go version declared in `go.mod`

Run the same basic checks required by CI:

```console
gofmt -w .
git diff --exit-code
go vet ./...
go test ./...
go build ./cmd/mcp-interop
```

For changes involving process lifecycle, OAuth, shared state, or release gates, also run:

```console
go test -race ./...
govulncheck ./...
```

If `govulncheck` is not installed, note that in the pull request and rely on CI's pinned scan.

The required pull-request status checks are the Linux, macOS, and Windows `test (...)` jobs. A change is not ready to merge while a required check is failing or missing.

## Evidence and adapter requirements

A live client adapter should:

- invoke the real installed client rather than emulate client behavior;
- avoid model prompts when a client management/control surface can prove the result directly;
- isolate configuration and credentials in a temporary profile/home/config directory;
- return `unknown` instead of inventing success when the client surface cannot prove a stage;
- keep `reach`, `auth`, `init`, and `tools` as separate observations;
- clean up temporary state and owned client processes on both success and failure;
- redact bearer/OAuth credentials and other secret material from reports and errors;
- record the tested client version;
- include tests for success and relevant failure/inconclusive paths.

A fixture proves the measurement path; it does **not** by itself prove deployment-specific live interoperability. Fixture-only success, configuration presence, metadata compatibility, or a configured tool allowlist must not be promoted into a real-client live PASS.

Cleanup failure is test-significant. If an isolated run leaves temporary credential/configuration state or owned processes behind, the test/harness should surface that failure rather than silently treating the interoperability result as clean.

If safe isolation cannot be established for a client, keep the adapter experimental or research-only rather than mutating the user's existing configuration.

## OAuth changes

OAuth changes require extra care:

- authentication must remain explicit opt-in when it can trigger user interaction;
- do not silently open authorization URLs unless the CLI contract explicitly documents and opts into that behavior;
- do not persist test credentials in the user's normal OS keychain or client credential store;
- do not copy normal-user browser/client credentials into a test profile;
- authorization URLs, authorization codes, callback state, access/refresh tokens, cookies, client secrets, and private keys must not be included in machine-readable result output or public evidence;
- use local/synthetic OAuth fixtures for automated tests rather than real production credentials.

## Pull request workflow

Development is PR-first. Changes intended for `main` should be made on a focused branch and merged through a pull request; do not use direct pushes to `main` as the normal development workflow.

Keep pull requests focused. Include:

- the related issue/research context where applicable;
- scope and explicit non-goals;
- what client/version was tested when client behavior is involved;
- the exact observable surface used to prove interoperability;
- isolation/cleanup behavior;
- local test results and relevant CI/E2E results;
- secret-safety considerations;
- documentation sync status;
- known limitations or states that intentionally remain `unknown`.

The repository uses squash merge for normal PR integration. Required CI must be green before merge. English/Japanese document pairs should be updated together when the same contract, safety boundary, or user-facing behavior changes; the English security policy remains canonical where `SECURITY.md` says so.

For product-direction or roadmap changes, keep the roles of `docs/project-direction*.md` and `docs/roadmap*.md` distinct. A milestone change should state its goal, exit criteria, explicit non-goals, and impact on existing evidence/security invariants, and the English/Japanese pair should remain synchronized. Do not present a future roadmap capability in the README as if it were already shipped.

## Release and versioning

Release preparation is also PR-first. The normal release sequence is:

1. prepare a focused release-prep PR;
2. update `CHANGELOG.md` and current-release references in the English/Japanese READMEs as applicable;
3. merge only after required CI is green;
4. create the release tag on a commit already contained in `main`;
5. let `.github/workflows/release.yml` run the authoritative publication gate;
6. verify generated archives, `checksums.txt`, embedded version output, and the packaged CLI regression smoke before treating the release as complete.

The release workflow rejects a release tag whose commit is not contained in `origin/main` and reruns source quality/security gates before publishing artifacts.

Version numbers follow SemVer intent while the project remains in `v0.x`:

- **patch**: bug/security fixes, documentation, and maintenance that do not intentionally break the public contract;
- **minor**: backward-compatible features or meaningful capability additions;
- **major**: reserved for an intentionally breaking public contract once the project reaches a maturity point where that distinction is useful.

Because `v0.x` denotes an evolving pre-1.0 contract, compatibility may still need to change before `v1.0`. Any known breaking behavior must be explicit in scope and release notes and must not be hidden inside an otherwise routine patch release.

Pre-1.0 minor versions are ordinary SemVer integers, not decimal fractions: `v0.10.0` may follow `v0.9.0`, followed by `v0.11.0` and later releases as needed. Do not promote the project to `v1.0.0` merely because a particular `v0.x` number was reached; use the stable-contract exit criteria in [docs/roadmap.md](docs/roadmap.md).

## Reporting security and support

Security vulnerabilities should be reported privately according to [SECURITY.md](SECURITY.md), not through public Issues or PRs.

For bug reports, interoperability reports, feature requests, and usage questions, see [SUPPORT.md](SUPPORT.md) and the repository Issue templates. Public reports must be secret-safe and must not disclose private production service identity, sensitive endpoint values, or credential material.
