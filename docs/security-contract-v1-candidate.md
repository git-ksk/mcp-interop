# Security, privacy, cleanup, and release contract candidate

[English](security-contract-v1-candidate.md) | [日本語](security-contract-v1-candidate.ja.md)

This document records the security/operational guarantees intended for the future stable contract. These guarantees are minimums: a future release may reject more unsafe input or tighten isolation without that being considered a compatibility regression.

## Secret-safe evidence

Portable evidence must not persist credential material merely because a client or endpoint exposed it during execution.

- access/refresh tokens, authorization codes, OAuth state, PKCE secrets, client secrets, private keys, cookies, and credential-file contents are not portable evidence fields;
- unknown secret-bearing Runtime Evidence fields fail closed;
- free-form client/remote errors are redacted before they can become ordinary output;
- portable artifact files are written privately and replace symlinks rather than following them;
- schema-v2 protected-path identity never persists or hashes a protected endpoint path.

Schema v1 remains readable, but its endpoint path is part of identity and therefore must itself be non-secret. Credential-bearing paths should use normal authentication plus schema v2 protected-path identity.

**Credential-safe does not mean deployment-public.** Schema v2 still carries canonical origin. A private/sensitive hostname should not be published merely because its path was protected.

## Credential isolation

A real-client PASS must not require copying normal-user tokens, browser cookies, Keychain entries, or persistent credential files into a test profile.

Shipped adapter evidence paths use isolated temporary state or a deliberately accepted bounded client surface. Controlled real-client release gates compare relevant normal-user configuration/credential metadata before and after the run. Research candidates remain research-only when the required authenticated boundary cannot be isolated safely.

Dedicated test credentials may be used only in an explicitly isolated research/test flow. They do not justify copying ordinary user credentials.

## OAuth material

Interactive OAuth is opt-in. Authorization navigation information may be shown only as part of the explicit interactive flow. Tokens, callback codes, refresh tokens, PKCE verifier material, and client secrets are not persisted into portable evidence/log artifacts.

Tool OAuth release fixtures use synthetic credentials and assert that secret material does not leak into retained evidence.

## Owned cleanup

Cleanup is bounded by test ownership.

- temporary directories/configuration created by the run are removed on success and failure;
- process cleanup targets descendants/sessions owned by the test harness, not arbitrary matching user processes;
- normal user state must remain unchanged where the adapter claims isolation;
- a cleanup/isolation failure is a real test/release-gate failure even when the four core stages passed.

## Privileged real-client CI boundary

The self-hosted real-client workflow is deliberately separate from ordinary pull-request CI.

It must remain:

- manual `workflow_dispatch` only;
- repository/main/exact-workflow-SHA guarded;
- protected by the `real-client-e2e` GitHub Environment;
- restricted to the labeled self-hosted macOS arm64 runner;
- checked out at the exact trusted SHA with `persist-credentials: false`;
- passed through `guard-real-client-e2e.sh` before execution;
- limited to fixed shipped-client choices and controlled fixture execution.

Untrusted PR content must not redirect this privileged runner to arbitrary endpoints, commands, or credential state.

## Ordinary CI boundary

Pull-request CI uses GitHub-hosted runners with read-only repository contents permission. It may run unit/fixture/release-smoke tests but does not run the privileged real-client workflow or production targets.

CI retains formatting, vet, unit, race, vulnerability, fixture, trust-guard, and release archive smoke gates.

## Tagged release guarantee

A tagged release must retain all of these gates:

- tag syntax validation and proof that the tagged commit is contained in `origin/main`;
- SHA-pinned GitHub Actions;
- format, vet, unit, race, and `govulncheck` gates;
- real-client trust-guard and controlled OAuth fixture gates;
- deterministic six-target archive build plus checksums;
- embedded-version/packaged CLI smoke verification;
- GitHub artifact attestation using OIDC permission;
- verified-tag GitHub Release creation.

`release.yml`, `e2e-real-macos.yml`, and `ci.yml` are checked by `scripts/test-security-contract.sh` so accidental removal of these boundaries fails CI.

## Provenance boundary

GitHub release attestations authenticate the release workflow's published build artifacts according to GitHub's attestation model. This does not turn ordinary local live-result artifacts or baseline fingerprints into authenticated execution attestations.

Local baseline and result fingerprints retain the narrower content identity/consistency meaning documented in the schema contract.

## Security-driven compatibility exception

If an existing accepted input or behavior is shown to expose credentials, enable untrusted privileged execution, or create another concrete security defect, a stable release may tighten/reject that behavior without waiting for a major version. Such a change must be explicit in release notes and must fail closed; unsafe old evidence is never silently reinterpreted as valid PASS.
