# Schema evolution v1 candidate

[English](schema-evolution-v1-candidate.md) | [日本語](schema-evolution-v1-candidate.ja.md)

This document defines the v1-candidate evolution policy for portable evidence, suite state, baselines, and derived machine reports. It complements the [Public contract candidate](public-contract-v1-candidate.md).

## Schema families

### Strict portable/input schemas

These are persisted documents that mcp-interop reads back with fail-closed validation:

| Family | Current schema | Artifact type / identity |
| --- | ---: | --- |
| Live result artifact | v1 / v2 | `mcp-interop/live-results` |
| Suite manifest | v1 | manifest contract |
| Suite result set index | v1 | `mcp-interop/suite-results` |
| Suite baseline descriptor | v1 | `mcp-interop/suite-baseline` |
| Capability profile | v1 | `mcp-interop/capability-profile` |
| Runtime Evidence | v1 / v2 / v3 | diagnostic evidence contract |

For these schemas, a structural change that an older strict reader would reject requires a new schema version. Do not silently add a required or unknown field to the same schema version and call it compatible.

A new schema version must document:

- what semantic problem required the new version;
- which older versions remain readable;
- whether a deterministic migration exists;
- what information cannot be derived and therefore requires re-observation/re-generation;
- whether comparison identity changed.

### Versioned derived reports

Regression, compatibility, maturity, graduation, baseline-verification, and similar reports carry a schema version even when they are primarily outputs. Existing field names, types, and meanings are stable within a schema. Additive optional fields may be introduced when they do not change existing meanings; removal, rename, type changes, or semantic repurposing require a report schema bump.

## Current live-result v1/v2 boundary

Live-result v1 and v2 intentionally share `artifact_type=mcp-interop/live-results` but have different endpoint-identity semantics:

- v1 includes the canonical endpoint identity path;
- v2 uses protected-path identity with an explicit non-secret `deployment_id` and origin binding.

They are both readable, but they are **not implicitly comparable**. `compare` rejects v1-v2 pairing because comparison identity is schema-specific.

There is no generic automatic v1 -> v2 migration. A protected deployment ID cannot be inferred safely from a v1 artifact. When v2 identity is required, regenerate/re-observe with an explicit non-secret deployment ID.

## Cross-schema comparison policy

Cross-schema comparison is allowed only when an explicit comparison/migration rule proves semantic equivalence for the compared identity. Otherwise the operation fails closed.

Never:

- pair runs only because client IDs or endpoint strings look similar;
- infer a protected deployment ID from an endpoint path;
- merge schema-specific identity fields by best effort;
- treat failed migration as `untested`, `pass`, or `regressed` evidence.

A migration tool, if added later, is a separate explicit operation and must preserve the source artifact. It must report any lossy or non-derivable field rather than fabricate it.

## Baseline contract

Baseline v1 provides local workflow immutability and content-consistency binding:

- output directories are no-clobber;
- the result-set snapshot is copied and digested;
- descriptor/result-set mismatch is rejected;
- `supersedes` binds an exact predecessor baseline fingerprint when explicitly supplied.

A baseline fingerprint proves the identity of the descriptor content under the documented algorithm. It is **not** a signature, authenticated reviewer identity, timestamp authority, or proof of who executed/accepted the evidence.

Team/CI authenticity may externally bind the exact fingerprint to a reviewed signature/attestation. Any future native authenticated-provenance format must use an explicit versioned envelope instead of changing the meaning of baseline v1.

## Compatibility-envelope contract

Compatibility reports are derived only from retained exact evidence. They do not create a continuous client-version range.

- `tested` means matching exact evidence exists under the selected policy;
- `untested` means that exact query point has not been observed;
- stale/known-broken/regressed states require their documented evidence conditions;
- version-only change is context, not regression;
- retries/evidence gaps remain retained rather than collapsed into a clean PASS.

Changing these meanings requires an explicit report/schema contract revision, not a documentation-only reinterpretation.

## Deprecation and removal policy

For a stable `v1.x` line:

- readers continue to accept every portable schema documented as supported at v1.0 unless a concrete security issue makes that unsafe;
- a newer schema may become the default writer without making older supported evidence unreadable;
- public schema/CLI deprecation is documented before removal;
- removal or incompatible reinterpretation normally waits for the next major version;
- a security-driven exception must be explicit in release notes and fail closed rather than silently reinterpret old evidence.

Historical evidence is never rewritten in place by default.

## Evidence authenticity boundary

Portable artifacts are evidence records, not cryptographic attestations by default. A digest/fingerprint establishes documented content identity or consistency only. It does not establish actor identity, machine identity, trusted time, or execution authenticity unless a separate authenticated provenance mechanism explicitly does so.
