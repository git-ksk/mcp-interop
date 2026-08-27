# Self-hosted real-client CI security boundary

[English](self-hosted-ci-security.md) | [日本語](self-hosted-ci-security.ja.md)

`mcp-interop` is a public repository. GitHub explicitly warns that self-hosted runners are not ephemeral clean machines and are especially risky for public repositories. The project therefore treats the self-hosted macOS real-client workflow as a **privileged trusted path**, never as ordinary pull-request CI.

## Current boundary

Ordinary `pull_request` CI runs only on GitHub-hosted Ubuntu/macOS/Windows runners. It never targets a `self-hosted` label.

`.github/workflows/e2e-real-macos.yml` is manual-only and has layered gates:

1. `workflow_dispatch` is the only trigger.
2. The self-hosted job has a GitHub-evaluated `if` requiring the canonical repository, `refs/heads/main`, the workflow file on `main`, and `github.workflow_sha == github.sha`. GitHub evaluates a job-level `if` before sending the job to a runner.
3. The job references the `real-client-e2e` GitHub Environment. Repository settings restrict that environment to the custom deployment branch policy `main` only.
4. Checkout is pinned to the exact `github.sha` and sets `persist-credentials: false`.
5. `scripts/guard-real-client-e2e.sh` repeats the repository/ref/workflow/SHA checks on the runner and also requires the expected self-hosted macOS ARM64 runner context.
6. The `clients` dispatch input is a fixed `choice`, not arbitrary text. The runner guard independently accepts only unique `codex`, `cursor`, and `antigravity` selections.
7. The workflow uses only the controlled localhost real-client fixture. It does not accept an endpoint URL, suite manifest path, shell command, executable path, environment override, OAuth option, or credential input.
8. The workflow records a private provenance JSON containing the exact repository/ref/SHA, workflow ref/SHA, run ID/attempt, actors, selected clients, and observed client versions. It is uploaded as a 30-day workflow artifact.

The Environment branch policy is repository configuration rather than a file in Git. Maintainers should periodically verify that `real-client-e2e` still permits only `main`.

## Deliberate non-connection to remote suite execution

`mcp-interop suite run` can execute `trusted_real_client` manifests locally, but v0.7 does **not** wire arbitrary remote suite execution into the self-hosted GitHub Actions workflow. This is intentional.

A future workflow that resolves `MCP_INTEROP_SUITE_ENDPOINT_*` values or uses production-equivalent credentials would expand the privileged network/credential boundary and requires a new security review. Untrusted pull-request content must never be allowed to choose those values or a manifest that can redirect the runner.

## OAuth

The manual self-hosted release gate does not pass `--oauth`. OAuth remains an explicit local/operator opt-in and retains the existing isolated credential boundaries. Adding OAuth credentials or browser/session state to self-hosted CI is outside the v0.7 contract.

## Runner operations

Repository workflow controls do not make a persistent self-hosted machine a sandbox. The runner should be dedicated/minimal, should not contain normal-user browser/client credentials or unrelated secrets, and should have only the network access required for the controlled test. Ephemeral or disposable runners are preferable when practical.

If a runner is shared at organization/enterprise scope, use a runner group restricted to the intended repository/workflow where the GitHub plan supports it.

## Threat-model limit

These controls are designed to stop **untrusted pull-request/branch content** from reaching the privileged runner by accident or through ordinary CI. They do not protect against a malicious or compromised repository administrator/write actor who can change trusted `main`, workflow files, environment policies, or runner configuration.

## Audit/test gates

`scripts/test-real-client-e2e-guard.sh` tests rejection of pull-request events, non-main refs, wrong repositories, mismatched workflow refs/SHAs, hosted-runner contexts, and invalid client input. It also checks the workflow remains manual-only/main-gated and emits the expected provenance shape. Hosted CI and release workflows run this test without using a self-hosted runner.
