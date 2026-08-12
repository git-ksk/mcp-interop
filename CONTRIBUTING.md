# Contributing to mcp-interop

[English](CONTRIBUTING.md) | [日本語](CONTRIBUTING.ja.md)

Thanks for helping improve real-client MCP interoperability testing.

## Before opening a pull request

1. Search existing issues and pull requests for related work.
2. For a new client adapter, open or reference an issue that documents the client's supported MCP management surfaces and the isolation strategy.
3. Do not add a live adapter that silently modifies a user's normal client configuration.

## Development

Requirements:

- Go version declared in `go.mod`

Run the same checks required by CI:

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

## Adapter requirements

A live client adapter should:

- invoke the real installed client rather than emulate client behavior;
- avoid model prompts when a client management/control surface can prove the result directly;
- isolate configuration and credentials in a temporary profile/home/config directory;
- return `unknown` instead of inventing success when the client surface cannot prove a stage;
- keep `reach`, `auth`, `init`, and `tools` as separate observations;
- clean up temporary state on both success and failure;
- redact bearer/OAuth credentials and other secret material from reports and errors;
- record the tested client version;
- include tests for success and relevant failure/inconclusive paths.

If safe isolation cannot be established for a client, keep the adapter experimental or research-only rather than mutating the user's existing configuration.

## OAuth changes

OAuth changes require extra care:

- authentication must remain explicit when it can trigger user interaction;
- do not silently open authorization URLs unless the CLI contract explicitly documents and opts into that behavior;
- do not persist test credentials in the user's normal OS keychain or client credential store;
- authorization URLs and callback state must not be included in machine-readable result output;
- use local/synthetic OAuth fixtures for automated tests rather than real production credentials.

## Pull requests

Keep pull requests focused. Include:

- what client/version was tested;
- the exact observable surface used to prove interoperability;
- isolation/cleanup behavior;
- local test results;
- known limitations or states that intentionally remain `unknown`.

Security vulnerabilities should be reported privately according to `SECURITY.md`, not through public issues.
