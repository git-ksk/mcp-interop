# Offline suite HTML and CI summary (v0.11 development)

[English](offline-suite-report.md) | [日本語](offline-suite-report.ja.md)

The opt-in `report suite` command produces a **single static offline HTML page** and/or a concise Markdown CI summary from existing validated suite regression evidence. It does not execute clients, contact a Remote MCP server, promote a baseline, or change the strict artifact/compare contracts.

## Example

```console
mcp-interop report suite baseline-results \
  repeat-results/attempt-01 repeat-results/attempt-02 \
  --html review.html --ci-summary ci-summary.md --fail-on-regression
```

The baseline and every attempt must be readable suite result sets under the same exact manifest fingerprint and trusted execution context. Every referenced artifact is validated by the existing suite loader, including symlink/path and protected-deployment identity checks. Up to 20 attempt sets and 192 logical runs can be rendered; larger input is rejected rather than generating unbounded HTML.

`--html` and `--ci-summary` are independently optional, but at least one must be supplied. Output filenames must be distinct and absent before execution. Each output file is created exclusively with owner-private file permissions (0600 on POSIX), so an existing file/symlink is never overwritten. Input/option errors return `2`, file write errors `1`. With `--fail-on-regression`, any regression or unstable evidence returns `1` **after** writing both reports. A clean decision returns `0`. Without the gate, validated reports return `0` regardless of decision.

## Privacy and security policy

The renderer uses a deliberately **reduced view** of the validated regression model. It includes the suite decision, regression/unstable flags, target ID, shipped client ID, auth mode, baseline outcome, **all attempt outcomes and evidence states**, and stage status transitions. It does not render the deployment label, client version strings (only a changed/not-changed indicator), endpoint fingerprints, any raw endpoint URLs or protected paths, filesystem paths, client messages/logs, OAuth state/token values, arbitrary reason-code strings, raw unexpected tool names, or arbitrary metadata. Unknown, untested/missing, skipped, and execution-error evidence is never shown as PASS.

The HTML uses Go `html/template` automatic escaping, no dynamic HTML fragments, no images, JavaScript, inline handlers, fonts, external CSS, external links, or network requests. A restrictive meta Content Security Policy (`default-src 'none'`) blocks all remote asset loading. The only CSS is embedded, constant styling (`style-src 'unsafe-inline'`). The Markdown CI view contains only a fixed-format table of validated enum values and constrained suite IDs: arbitrary client-supplied strings cannot create Markdown links, HTML, or injection content.

**Limitations:** This is not a cryptographically authenticated report; it inherits the trust boundary of the source suite artifacts. The optional `test --tool-evidence` snapshot and inventory drift are separate from core suite comparisons, and are **not** silently included in an HTML report. The report omits exact client version strings intentionally; consult the protected underlying evidence in a trusted environment when deeper provenance is needed. If you need a CI job summary, append `ci-summary.md` to `$GITHUB_STEP_SUMMARY` in your own **trusted** workflow after review; the project PR CI does not accept arbitrary remote endpoint manifests.
