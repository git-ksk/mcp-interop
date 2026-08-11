# MCP Conformance vs. mcp-interop

[English](conformance-vs-interop.md) | [日本語](conformance-vs-interop.ja.md)

`mcp-interop` is **not** a replacement for the official [MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance). They answer different questions and are most useful as complementary test layers.

## Core distinction

| | MCP Conformance | `mcp-interop` |
| --- | --- | --- |
| Primary question | Does this MCP implementation behave according to the specification? | Does this deployed Remote MCP server actually work with this released client product/version? |
| Comparison axis | implementation × specification | deployment × client product × client version |
| Oracle | MCP specification, scenarios, and expected checks | observations exposed by the real installed client |
| Client-side topology | conformance test server → client under test | user-selected Remote MCP deployment ↔ real client product |
| Server-side topology | conformance client → server under test | real client products → the same Remote MCP deployment |
| Main result | scenario/check conformance PASS/FAIL | `reach` / `auth` / `init` / `tools` interoperability evidence |
| Product-specific behavior | Relevant when it violates a conformance scenario | A first-class part of the result |

The distinction is **not** “synthetic tests versus real software.” The official conformance framework can launch a real client command and can test a real server URL. The difference is the test topology and the source of truth.

## What the official conformance framework tests

For client testing, the official framework starts a scenario-controlled MCP test server, runs the client-under-test command, captures protocol interactions, and evaluates the observed behavior against specification-defined checks.

For server testing, the framework connects to the supplied server URL as a conformance client, exercises conformance scenarios, and evaluates the server against the specification.

That makes the official framework the right place for questions such as:

- Does this client or server satisfy the MCP wire/lifecycle requirements?
- Does it behave correctly for the selected MCP specification revision?
- Does it satisfy the required OAuth/MCP authorization scenarios?
- Does its JSON-RPC traffic conform to the expected schema and protocol behavior?

`mcp-interop` should not duplicate those generic checks.

## What mcp-interop tests

`mcp-interop test` starts from a different input: a Remote MCP deployment that a user actually intends to ship or consume.

It then asks released client products to use that deployment through their own supported or observed MCP surfaces. Current adapters include Codex CLI, Cursor CLI, and Antigravity CLI.

Conceptually:

```text
same Remote MCP deployment
        |
        +--> Codex CLI version X
        +--> Cursor CLI version Y
        +--> Antigravity CLI version Z
```

The result records whether each real client provided enough evidence for:

```text
reach -> auth -> init -> tools
```

A client-specific failure is useful even when both sides appear individually specification-compatible. Product configuration, OAuth discovery order, credential storage, callback handling, registration strategy, released-version regressions, or other implementation details can make a specific pairing fail in practice.

The client product and its version are therefore part of the interoperability evidence, not incidental metadata.

## Not a static client compatibility matrix

A client capability matrix answers a useful but different question: **what does product/version X generally support?** `mcp-interop` should not compete by manually curating another universal feature table.

Its strongest evidence is deployment-specific and reproducible:

```text
endpoint A + client X version 1 -> PASS
endpoint A + client X version 2 -> AUTH FAIL
endpoint A + client Y version 7 -> PASS
```

That makes version-to-version regression detection a first-class use case. A result must remain scoped to the endpoint, client product, client version, operating-system/runtime context where relevant, and evidence actually observed during that run. It must never be generalized into compatibility with products or versions that were not executed.

Static compatibility documentation can help decide **which tests to run**; it is not a substitute for a live result against the deployment under test.

## Evidence hierarchy

Keep these evidence layers distinct:

1. specification/conformance evidence;
2. direct server inspection/debugging;
3. product-profile preflight metadata;
4. sanitized Runtime Evidence supplied from a deployment;
5. **live deployment-specific real-client evidence**.

Only the fifth layer can produce an `mcp-interop` real-client `reach/auth/init/tools` PASS for the target deployment.

A fixture or localhost adapter self-test proves that the measurement path works. It does not prove that a different production deployment passed.

## Adapter graduation criteria

Adding more client names is less important than preserving trustworthy evidence. A live adapter should graduate from research/beta only when its measurement boundary is clear enough to document and reproduce.

At minimum, evaluate:

- **isolation** — normal user configuration/credentials are not reused or mutated;
- **supported or deliberately observed client surface** — no private/minified UI internals merely to increase coverage;
- **no-model core path** — core interoperability evidence does not depend on an LLM choosing a tool correctly;
- **machine-readable or conservatively interpretable evidence** — absence of evidence becomes `unknown`, not an invented PASS;
- **cleanup** — temporary credentials, config, processes, and state are removed or independently checked;
- **version context** — the shipping client version and relevant platform context are recorded;
- **deterministic fixture proof** — controlled E2E demonstrates that the adapter is observing the real client path it claims to measure.

If a client cannot meet this boundary, keep it research-only rather than weakening the project-wide meaning of PASS.

## Why both layers matter

A useful release pipeline can treat the two projects as sequential quality gates:

```text
1. protocol/specification correctness
   -> modelcontextprotocol/conformance

2. deploy the real Remote MCP endpoint

3. product-level interoperability
   -> mcp-interop against the clients users actually run
```

A conformance PASS does not by itself prove that every released client product will interoperate with a specific deployment. Conversely, an `mcp-interop` PASS does not prove full MCP specification conformance.

## OAuth and diagnostic boundary

OAuth is the area with the most potential overlap, so the project keeps an explicit boundary.

Generic MCP/OAuth protocol conformance belongs to the official conformance suite.

`mcp-interop diagnose --profile <product>` is instead a **product-compatibility profile**. A profile may compare published server metadata and sanitized runtime observations with the documented behavior of a specific client product, such as ChatGPT. It must not present those checks as generic MCP conformance.

Rules for diagnostic profiles:

- Keep product-specific expectations explicit.
- Do not reimplement generic MCP/OAuth conformance scenarios merely to create a second conformance suite.
- Keep `PREFLIGHT`, sanitized Runtime Evidence, and real-client interoperability as separate evidence layers.
- Never promote metadata compatibility to a real-client `reach/auth/init/tools` PASS.
- When a question is purely “is this MCP/OAuth implementation specification-conformant?”, prefer the official conformance framework.

## The localhost fixture is not a competing conformance suite

The repository includes a deterministic localhost MCP fixture used to validate that the adapters really observe the installed clients and that isolation/cleanup behavior remains correct.

That fixture is an **adapter self-test and release gate**. Its purpose is to prove the `mcp-interop` measurement path, not to certify arbitrary clients or servers as MCP-conformant.

## Non-claims

An `mcp-interop` PASS does not prove:

- complete MCP specification conformance;
- server or client security;
- correctness of every tool implementation;
- safety of destructive operations;
- correct model/tool selection;
- compatibility with client products or versions that were not actually run.

For the runtime design, see [Architecture](architecture.md).