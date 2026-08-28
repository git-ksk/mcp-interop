# Capability profile v1

[English](capability-profile-v1.md) | [日本語](capability-profile-v1.ja.md)

Capability profile v1 is a **separate optional-capability evidence contract**. It does not change the existing Remote Tool Interoperability result:

```text
reach -> auth -> init -> tools
```

A capability profile can describe Resources, Prompts, Tasks, MRTR, controlled tool-call profiles, or future capabilities only after that capability has its own precise evidence contract. Merely naming a capability here does **not** mean any shipped adapter currently supports it.

Current main does not emit a capability PASS for Resources, Prompts, Tasks, MRTR, or any other optional capability. This issue establishes the evidence boundary first.

## CLI

Validate a capability-profile file without executing a client or contacting a Remote MCP deployment:

```console
mcp-interop capability validate capability-profile.json
mcp-interop capability validate capability-profile.json --json
```

The command returns `0` only when the document is structurally valid. That exit code means **valid capability evidence document**, not core interoperability PASS and not "all capabilities pass". Human output prints every capability state. `--json` re-emits only the validated profile and does not include the input filesystem path.

## Schema

A profile uses:

```json
{
  "schema_version": 1,
  "artifact_type": "mcp-interop/capability-profile",
  "context": {
    "observed_at": "2026-08-28T12:00:00Z",
    "deployment_id": "production-a",
    "deployment_fingerprint": "sha256:...",
    "client": {
      "id": "codex",
      "product": "Codex CLI",
      "version": "codex-cli 0.133.0"
    },
    "platform": {
      "os": "darwin",
      "arch": "arm64"
    },
    "runtime": {
      "mcp_interop_version": "dev",
      "mcp_interop_commit": "...",
      "go_version": "go1.26.6"
    },
    "auth_mode": "default",
    "evidence_provenance": {
      "kind": "real_client_adapter",
      "adapter_id": "codex"
    }
  },
  "capabilities": [
    {
      "capability_id": "resources",
      "state": "pass",
      "evidence_kind": "client_protocol",
      "evidence_id": "resources.list.response"
    }
  ]
}
```

`capability_id` and `evidence_id` are short project-defined identifiers, not raw protocol payloads, client logs, UI text, URLs, or diagnostic messages.

The capability array is deterministic and sorted by `capability_id`; duplicate capability IDs are rejected.

## Exact context

Every profile is tied to one exact context:

- operator-chosen non-secret `deployment_id`;
- exact secret-safe `deployment_fingerprint`;
- exact client ID/product/version;
- runner/process OS and architecture;
- `mcp-interop` runtime identity;
- auth mode (`default` or `oauth`);
- real-client adapter provenance;
- UTC observation time.

No endpoint URL, endpoint path, query string, bearer token, authorization code, cookie, client secret, executable path, or raw human/client output is part of the schema.

Future emitters should derive context from an already validated schema-v2 protected-path live run when possible. `ContextFromLiveRunV2` copies the non-secret deployment ID/fingerprint and exact client/platform/runtime/auth/provenance context without receiving the protected endpoint path.

## States

The states are deliberately non-overlapping:

- `pass` — the exact capability-specific success condition was directly observed through the real client on this exact context.
- `fail` — a direct real-client attempt produced an explicit negative capability outcome on this exact context.
- `unknown` — a direct real-client attempt occurred, but the evidence was ambiguous or incomplete.
- `unsupported` — the adapter's documented policy/boundary says the capability path is unavailable for this exact client/profile. This is not a tested failure.
- `untested` — no capability attempt/evidence exists for this exact context. This is not `unknown`, `unsupported`, or `fail`.

A nearby client version, platform, auth mode, or deployment never inherits one of these states automatically.

## Evidence kinds

`pass`, `fail`, and `unknown` require one of these **direct client** evidence kinds:

- `client_protocol` — direct client-originated/consumed protocol evidence for the capability-specific operation;
- `client_control_surface` — a supported or deliberately accepted real-client management/control surface that directly reports the capability operation/result;
- `client_observed_state` — bounded client-owned state produced by actual capability execution/discovery and documented as the accepted observation boundary.

`client_observed_state` does not include configuration presence, an enabled flag, an allowlist, cached static metadata, or UI text that merely advertises a feature.

`unsupported` requires `adapter_policy` plus a stable non-secret `evidence_id` naming the documented policy boundary.

`untested` requires `none` and must not carry an `evidence_id`.

The schema has no valid evidence kind for server metadata, client configuration presence, browser/UI presence, model output, fixture-only server observations, or generic feature advertising. Those inputs therefore cannot validate as capability PASS evidence.

A controlled fixture may participate in a capability test only when the **real client itself** provides one of the accepted direct evidence surfaces. Fixture-side observation alone cannot become a deployment-specific capability PASS.

## PASS is capability-specific

A capability PASS must have a separately documented success condition. Examples of capability names such as `resources`, `prompts`, `tasks`, or `mrtr` are not enough. Before a shipped adapter emits one of those PASS states, its implementation/documentation must define:

1. the exact client operation or observed state that constitutes success;
2. the accepted direct evidence kind and stable `evidence_id`;
3. what produces `fail` versus `unknown`;
4. what is explicitly `unsupported`;
5. the isolation/cleanup/secret-safety boundary;
6. controlled real-client fixture coverage for the claimed path.

This prevents a static client capability matrix or protocol/server metadata from being converted into live deployment evidence.

## Core interoperability remains unchanged

Capability profiles are not embedded into live-result artifact v1/v2 and are not read by the current `test`, `compare`, suite, baseline, or compatibility PASS/regression decisions.

Therefore:

```text
capability pass != core live PASS
core live PASS != optional capability pass
```

A valid capability profile may contain `fail`, `unknown`, `unsupported`, or `untested` entries and still pass `capability validate`, because validation checks the evidence contract rather than inventing an aggregate verdict.

## v0.8 compatibility

Capability profile v1 is additive and separate. It does **not** change or migrate:

- live-result artifact schema v1;
- live-result artifact schema v2;
- suite manifest/result-set schemas;
- baseline schema;
- compatibility envelope/query semantics;
- the public `reach/auth/init/tools` result.

Existing v0.8 artifacts remain readable by their existing readers. Capability-profile creation from an existing run should use a validated schema-v2 protected-path run so the new profile never needs to persist a raw endpoint path. Existing schema-v1 artifacts remain valid historical evidence; they are not silently rewritten into capability profiles.

## Secret and input boundary

Profile files are strictly decoded with unknown fields rejected and input size bounded. This intentionally rejects ad-hoc fields such as `endpoint`, `path`, `token`, `authorization_code`, `cookie`, or raw output blobs instead of trying to redact an open-ended capability document after the fact.

Writers use the existing private JSON replacement path. Evidence identifiers must remain short, lowercase, non-secret project identifiers.
