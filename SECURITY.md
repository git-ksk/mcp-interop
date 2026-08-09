# Security Policy

[English](SECURITY.md) | [日本語](SECURITY.ja.md)

The English version of this policy is the canonical security-reporting policy for this repository.

## Reporting a vulnerability

Please **do not open a public issue** for a suspected security vulnerability.

Use GitHub's private vulnerability reporting for this repository instead:

1. Open the repository's **Security** tab.
2. Choose **Report a vulnerability**.
3. Include the affected version or commit, reproduction steps, expected impact, and any suggested mitigation if known.

Please avoid including live credentials, OAuth authorization codes, access or refresh tokens, cookies, or other secrets in a report. Use synthetic or revoked credentials when a reproduction requires credential-shaped data.

## Scope

Security reports are especially useful for issues involving:

- leakage of OAuth or bearer credentials;
- unintended mutation of a user's MCP client configuration;
- failure to clean up temporary credentials or configuration;
- command or argument injection through Remote MCP metadata or URLs;
- unsafe handling of client-generated authorization URLs;
- incorrect isolation between a test session and the user's normal client profile;
- report/log redaction bypasses.

## Supported versions

Security fixes are developed against the latest `main` branch.

For the current `v0.x` series, maintainers will assess whether a reported vulnerability requires a patched tagged release based on severity, exploitability, and whether released users are affected. The latest published release is the primary supported release line; older `v0.x` releases may not receive backports.

If a vulnerability affects a released artifact, the advisory or release notes will identify the fixed version when a patched release is published.
