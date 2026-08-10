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