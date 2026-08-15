## Related issue

<!-- Link the issue or research thread when applicable. -->

## Scope

<!-- Keep the PR focused. State what changes and what is intentionally out of scope. -->

## Observable evidence

<!-- For client/interoperability work: identify the real-client observable surface that proves each claimed stage. Do not use fixture/configuration/metadata-only success as deployment-specific live PASS. -->

## Tests

<!-- Include local checks and any bounded real-client E2E that was applicable. -->

## Isolation / cleanup

<!-- Describe temporary profile/HOME/config isolation, cleanup on success/failure, and owned-process cleanup where relevant. -->

## Secret safety

- [ ] No access/refresh token, OAuth authorization code/state, raw authorization URL, cookie, client secret, private key, credential file, normal-user browser credential, or production-sensitive endpoint value is included in the diff or evidence.
- [ ] Security-sensitive findings are routed through `SECURITY.md`, not a public PR discussion when disclosure would expose a vulnerability.

## Documentation sync

- [ ] English/Japanese paired docs were kept in sync where this change affects both, or the PR explains why a paired update is not applicable.
- [ ] Research-only / experimental client status is not overstated.

## Known unknowns / limitations

<!-- Record evidence gaps, version/OS limitations, intentionally unknown stages, or follow-up work. -->
