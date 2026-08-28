# Public contract candidate

[English](public-contract-v1-candidate.md) | [日本語](public-contract-v1-candidate.ja.md)

This document records the public meanings under review for the future `v1.x` compatibility promise. It is a **candidate contract** during v0.10; it does not retroactively make every internal implementation detail public API.

## CLI compatibility

The documented top-level commands and documented flags are the supported CLI surface. Across a stable `v1.x` line:

- an existing command/flag is not silently repurposed;
- removing or changing an existing meaning requires deprecation plus a documented migration, or a new major version;
- new commands and optional flags may be added compatibly;
- undocumented implementation details, temporary files, child-process arguments, and private adapter internals are not public CLI contracts.

## Exit codes

The CLI reserves three process exit classes:

| Code | Contract |
| ---: | --- |
| `0` | The requested command completed according to its success contract. For `test`, every core stage passed. For validators, the document is valid. A comparison without `--fail-on-regression` may still *report* a regression and exit `0` because reporting succeeded. |
| `1` | The invocation was valid, but execution/evaluation did not satisfy the requested gate, or an operational read/write/execution error prevented a successful result. Examples include a non-PASS live test, a failing preflight, or `--fail-on-regression` observing a regression. |
| `2` | Usage, option, trust-boundary, or pre-execution configuration validation failed. No interoperability PASS may be inferred. |

Future commands should use these classes rather than inventing command-specific numeric meanings.

## JSON compatibility classes

### Unversioned command JSON

Outputs such as `clients --json`, `test --json`, and diagnostic command JSON are command interfaces rather than portable evidence schemas. Within `v1.x`, existing field names, types, and meanings are preserved. Additive fields are allowed; consumers must ignore fields they do not understand. Removing, renaming, changing a field type, or changing its meaning is incompatible.

Local diagnostic JSON may contain local-machine facts such as an executable path when that is already part of the documented command output. Such output is not automatically safe to publish as portable evidence.

### Versioned reports and evidence

Portable/versioned artifacts use explicit `schema_version` and, where defined, `artifact_type`. Their evolution policy is stricter and is documented separately in the schema contract. A successful validator means the document satisfies that schema; it does not turn a non-PASS observation into PASS.

## Reason-code compatibility

`reason_code` is an **open string enum with stable existing values**:

- an existing code is not renamed, removed, or repurposed within `v1.x`;
- new reason codes may be added when a new directly evidenced classification is needed;
- consumers must tolerate an unknown non-empty future code and preserve it when possible;
- absence of a code means no narrower stable classification was made; consumers must not invent one from free-form text;
- project-authored messages may improve without becoming machine identifiers.

The current stable code names are listed in [Reason codes](reason-codes.md). Portable artifact readers intentionally do not reject an otherwise valid artifact merely because it contains a future reason-code string.

## Primary live-result JSON

The core live result retains these meanings:

- `client_id` — stable adapter identity;
- `client_name` — human-readable product label;
- `client_version` — exact observed version when safely available;
- `endpoint` — command result endpoint display; portable protected-path evidence uses its separate artifact identity model;
- `stages` — exactly the stable core order `reach`, `auth`, `init`, `tools`;
- `diagnostics` — supporting secret-safe evidence that never upgrades the four-stage verdict by itself.

A reason-code or diagnostic change cannot silently broaden core PASS.

## Compatibility review rule

Before changing a public command, JSON field, exit-code meaning, or existing reason code, classify the change as one of:

1. compatible addition;
2. behavior clarification with unchanged machine meaning;
3. versioned migration/deprecation;
4. major-version incompatibility.

If the classification is ambiguous, fail closed and do not silently change the public meaning.
