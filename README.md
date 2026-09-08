# mcp-interop

[![CI](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml/badge.svg)](https://github.com/git-ksk/mcp-interop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/git-ksk/mcp-interop)](https://github.com/git-ksk/mcp-interop/releases/latest)
[![License](https://img.shields.io/github/license/git-ksk/mcp-interop)](LICENSE)

[English](README.md) | [日本語](README.ja.md)

**Find out whether your Remote MCP server works in the clients your users run.**

Test the same endpoint with real Codex, Cursor, and Antigravity CLIs. See where each client gets stuck, save the result, and catch regressions after a server deployment or client update. Runs locally, with no hosted service required.

[Quick start](#quick-start) · [Supported clients](#supported-clients) · [Usage guide](docs/usage.md) · [Contribute](CONTRIBUTING.md)

```console
mcp-interop test https://example.com/mcp --client codex,cursor,antigravity
```

Illustrative output format; versions are placeholders, not a compatibility claim:

```text
SUMMARY
CLIENT           REACH  AUTH  INIT  TOOLS  VERSION
Codex CLI        PASS   PASS  PASS  PASS   <exact version>
Cursor CLI       PASS   PASS  PASS  PASS   <exact version>
Antigravity CLI  PASS   PASS  PASS  PASS   <exact version>
```

## When to use it

- **Before shipping a Remote MCP server:** check that real clients can connect, authenticate when needed, and discover tools.
- **After a client update:** compare saved runs to see whether behavior changed. A new version number alone is not a regression.
- **When a user reports “it won't connect”:** narrow the problem to reachability, authentication, protocol readiness, or tool discovery.
- **For repeatable release checks:** run a declared suite across clients and keep every attempt, including failed retries.

Protocol conformance and real-client interoperability answer different questions. Use this alongside your conformance tests to check the client-facing result. [How they fit together →](docs/conformance-vs-interop.md)

## Quick start

Start on **macOS with Apple Silicon**, the evidenced stable platform for the three live adapters. You need **Go 1.24+** and at least one supported client executable on your `PATH` (`codex`, `cursor-agent`, or `agy`).

### 1. Install

```console
go install github.com/git-ksk/mcp-interop/cmd/mcp-interop@latest
```

Prefer a binary? Download an archive for your OS and architecture from [GitHub Releases](https://github.com/git-ksk/mcp-interop/releases/latest), verify it against `checksums.txt`, and put the executable on your `PATH`. Go is only needed for a source install.

If the shell cannot find a Go-installed binary, add `$(go env GOPATH)/bin` to your `PATH` (or your custom `GOBIN`).

**Published release: [v0.10.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.10.0).** `main` contains preparation for v1.0.0 and newer fixes; v1.0.0 has not been published. `@latest` installs the published module, not the current checkout. See the [changelog](CHANGELOG.md) for release history.

### 2. Check your clients

```console
mcp-interop version
mcp-interop clients
```

### 3. Test your endpoint

Replace the example URL with your Remote MCP endpoint and select an installed client:

```console
mcp-interop test https://example.com/mcp --client codex
```

A complete success has **PASS in all four stages** and exits `0`:

| Stage | What the real client proves |
| --- | --- |
| `reach` | Live interaction with the endpoint |
| `auth` | Required authentication completed, or discovery succeeded without it |
| `init` | Protocol readiness to continue the MCP exchange |
| `tools` | Discovery of the server's tools |

`FAIL`, `SKIP`, or `UNKNOWN` makes a live test exit non-zero. `UNKNOWN` means the available evidence cannot prove the result; it is useful diagnostic information. [Troubleshooting →](docs/troubleshooting.md)

For an OAuth-protected server, start the interactive flow explicitly:

```console
mcp-interop test https://example.com/mcp --client cursor --oauth
```

OAuth is opt-in and outside the stable non-OAuth scope. [OAuth details →](docs/usage.md#oauth)

## Supported clients

The current `main` maturity review uses the same criteria for all three adapters:

| Client | How it observes the real client | Stable scope |
| --- | --- | --- |
| Codex CLI | `codex app-server` MCP status and tool inventory | macOS arm64, non-OAuth core path |
| Cursor CLI | MCP management commands, including `mcp list-tools` | macOS arm64, non-OAuth core path |
| Antigravity CLI | Isolated PTY and client-generated tool cache | macOS arm64, non-OAuth core path |

Stable describes the reviewed adapter scope. It does not guarantee every client version or every deployment. See [exact observed coverage](docs/observed-coverage.md) and the [maturity criteria](docs/adapter-maturity.md). Binary availability for an OS does not establish live-client coverage on that OS.

VS Code, GitHub Copilot CLI, ChatGPT, and Claude web/Desktop remain research-only. ChatGPT has a separate metadata-based `diagnose --profile chatgpt` command; its preflight result is not a real ChatGPT live PASS. [Research and graduation status →](docs/adapter-graduation-gate.md)

## Turn a check into a regression workflow

Save a run before a change, repeat it afterward, then compare:

```console
mcp-interop test https://example.com/mcp --client codex --output before.json
# Deploy your server change or update the client, then run again.
mcp-interop test https://example.com/mcp --client codex --output after.json
mcp-interop compare before.json after.json --fail-on-regression
```

The comparison catches lost PASS evidence, including transitions to `FAIL`, `UNKNOWN`, or `SKIP`. For endpoints with credentials in their path, use [`--deployment-id` and protected-path artifacts](docs/usage.md#portable-regression-artifacts).

When you need more than a single run, [suites and immutable baselines](docs/usage.md#repeatable-multi-client-suites) let you declare targets, run multiple clients, and retain retry history. A passing retry does not erase an earlier failure.

## What makes the result useful

- **Direct client evidence.** The core test observes the installed client without sending model prompts or invoking server tools.
- **Isolated runs.** Adapters use temporary configuration and auth state, with bounded cleanup of owned processes and files.
- **Honest uncertainty.** Metadata checks, server-side observations, and incomplete client evidence do not become live PASS by inference.
- **Portable results.** Save versioned artifacts for comparison; secret-bearing values are rejected or redacted according to the [security contract](docs/security-contract-v1.md).

This tests connection and discovery. Tool correctness, model tool selection, and a server's overall security need their own tests.

## Go further

| You want to… | Read |
| --- | --- |
| Run more commands, OAuth, or preflight diagnostics | [Usage guide](docs/usage.md) |
| Understand an unexpected result | [Troubleshooting](docs/troubleshooting.md) · [Reason codes](docs/reason-codes.md) |
| Review client versions and evidence | [Observed coverage](docs/observed-coverage.md) · [Adapter maturity](docs/adapter-maturity.md) |
| Integrate repeatable checks into CI | [Suite manifests](docs/suite-manifest-v1.md) · [Self-hosted CI](docs/self-hosted-ci-security.md) |
| Understand or extend the implementation | [Architecture](docs/architecture.md) · [Project direction](docs/project-direction.md) |
| Find schemas, contracts, and research | [Documentation index](docs/README.md) |

## Help make MCP work across clients

A reproducible connection report, a clearer example, or an English/Japanese documentation fix is a useful contribution. You do not need to build a new adapter to help.

[Report an interoperability issue](https://github.com/git-ksk/mcp-interop/issues/new/choose) with the client version, OS, command, and sanitized stage results. For code changes, start with the [contributing guide](CONTRIBUTING.md); for questions, see [support](SUPPORT.md). Read the [roadmap](docs/roadmap.md) to see where the project is heading.

Report vulnerabilities privately through the [security policy](SECURITY.md). Participation follows the [code of conduct](CODE_OF_CONDUCT.md).

## License

[Apache License 2.0](LICENSE).
