from pathlib import Path


def replace_one(path: str, old: str, new: str) -> None:
    p = Path(path)
    text = p.read_text(encoding="utf-8")
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one match, found {count}: {old[:120]!r}")
    p.write_text(text.replace(old, new), encoding="utf-8")


def replace_count(path: str, old: str, new: str, expected: int) -> None:
    p = Path(path)
    text = p.read_text(encoding="utf-8")
    count = text.count(old)
    if count != expected:
        raise SystemExit(f"{path}: expected {expected} matches, found {count}: {old[:120]!r}")
    p.write_text(text.replace(old, new), encoding="utf-8")


# CHANGELOG: keep an empty Unreleased section and promote its current contents to v0.4.0.
replace_one(
    "CHANGELOG.md",
    "## Unreleased\n\n### Fixed",
    "## Unreleased\n\n## v0.4.0 — 2026-08-11\n\n### Fixed",
)

# README (English)
replace_one("README.md", "**v0.3.0 is the current published release.**", "**v0.4.0 is the current published release.**")
replace_one(
    "README.md",
    "Release: [v0.3.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.3.0)",
    "Release: [v0.4.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.4.0)",
)
replace_one(
    "README.md",
    "Current main contains additional **unreleased post-v0.3.0 OAuth work**. The live adapters on current main are:",
    "The live adapters in v0.4.0 are:",
)
replace_one(
    "README.md",
    "These Cursor and Antigravity OAuth paths are **not included in the v0.3.0 release artifact**. Use the v0.3.0 release notes for published behavior and [CHANGELOG.md](CHANGELOG.md) for current unreleased changes.\n\n",
    "",
)
replace_one(
    "README.md",
    "v0.3.0 includes **ChatGPT OAuth/server preflight, Runtime Evidence v3, secret-free evidence utilities, controlled insufficient-scope OAuth fixtures, and versioned OpenAI reference-pattern diagnostics**. Runtime Evidence v3 separates static tool metadata from runtime reauthorization challenges while preserving v1/v2 input compatibility. These diagnostics validate published metadata and explicitly supplied sanitized runtime observations without claiming that the real ChatGPT client ran.",
    "v0.4.0 adds **Cursor OAuth completion, Antigravity OAuth completion on the tested macOS baseline, secret-free real-client OAuth capability enrichment, stricter deployment-specific live-evidence boundaries, and hardened release provenance gates**. v0.3.0 introduced ChatGPT OAuth/server preflight, Runtime Evidence v3, secret-free evidence utilities, controlled insufficient-scope OAuth fixtures, and versioned OpenAI reference-pattern diagnostics.",
)
replace_one(
    "README.md",
    "GitHub Copilot CLI is a follow-up candidate. Claude Code support is intentionally deferred.",
    "GitHub Copilot CLI remains research-only: current testing proves real-client MCP initialization but has not yet proven `tools/list` under the project's no-model evidence contract ([#48](https://github.com/git-ksk/mcp-interop/issues/48)). Claude Code support is intentionally deferred.",
)
replace_count("README.md", "@v0.3.0", "@v0.4.0", 1)
replace_count(
    "README.md",
    "[v0.3.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.3.0)",
    "[v0.4.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.4.0)",
    1,
)
replace_one("README.md", "OAuth flows are always explicit opt-in. On current main:", "OAuth flows are always explicit opt-in:")
replace_one("README.md", "### Completed on main after v0.3.0 (unreleased)", "### Shipped in v0.4.0")
replace_one("README.md", "### Open after v0.3.0", "### Open after v0.4.0")
replace_one(
    "README.md",
    "- [ ] Evaluate additional real MCP clients when they expose stable automatable lifecycle/tool-discovery surfaces.",
    "- [ ] Complete GitHub Copilot CLI tool-discovery/auth-isolation research before any live adapter ([#48](https://github.com/git-ksk/mcp-interop/issues/48)).\n- [ ] Evaluate additional real MCP clients when they expose stable automatable lifecycle/tool-discovery surfaces.",
)

# README (Japanese)
replace_one("README.ja.md", "**現在の公開releaseはv0.3.0です。**", "**現在の公開releaseはv0.4.0です。**")
replace_one(
    "README.ja.md",
    "Release: [v0.3.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.3.0)",
    "Release: [v0.4.0](https://github.com/git-ksk/mcp-interop/releases/tag/v0.4.0)",
)
replace_one(
    "README.ja.md",
    "current mainには、v0.3.0以降の**未release OAuth対応**も入っています。current mainのlive adapterは次の通りです。",
    "v0.4.0で提供するlive adapterは次の通りです。",
)
replace_one(
    "README.ja.md",
    "これらCursor/Antigravity OAuth pathは**v0.3.0の公開artifactには含まれません**。公開版の挙動はv0.3.0 releaseを、current mainの未release変更は[CHANGELOG.md](CHANGELOG.md)を参照してください。\n\n",
    "",
)
replace_one(
    "README.ja.md",
    "v0.3.0には、**ChatGPT OAuth/server preflight、Runtime Evidence v3、secret-free evidence utilities、controlled insufficient-scope OAuth fixture、versioned OpenAI reference-pattern diagnostics**が含まれます。Runtime Evidence v3はstatic tool metadataとruntime reauthorization challengeを分離しつつ、v1/v2 input互換を維持します。公開metadataと明示的に渡されたsanitized runtime observationを診断しますが、実ChatGPT clientを動かしたとは主張しません。",
    "v0.4.0では、**Cursor OAuth完遂、tested macOS baselineでのAntigravity OAuth完遂、secret-free real-client OAuth capability enrichment、deployment固有のlive-evidence境界の明確化、release provenance gateの強化**を追加します。v0.3.0ではChatGPT OAuth/server preflight、Runtime Evidence v3、secret-free evidence utilities、controlled insufficient-scope OAuth fixture、versioned OpenAI reference-pattern diagnosticsを導入しました。",
)
replace_one(
    "README.ja.md",
    "GitHub Copilot CLIは今後の候補です。Claude Code対応は現時点では優先していません。",
    "GitHub Copilot CLIはresearch-onlyです。現在の検証では実clientのMCP initializationまでは証明できましたが、projectのno-model evidence contractで`tools/list`までは未証明です ([#48](https://github.com/git-ksk/mcp-interop/issues/48))。Claude Code対応は現時点では優先していません。",
)
replace_count("README.ja.md", "@v0.3.0", "@v0.4.0", 1)
replace_count(
    "README.ja.md",
    "[v0.3.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.3.0)",
    "[v0.4.0 GitHub Release](https://github.com/git-ksk/mcp-interop/releases/tag/v0.4.0)",
    1,
)
replace_one("README.ja.md", "OAuth flowは常に明示的opt-inです。current mainでは次を使えます。", "OAuth flowは常に明示的opt-inです。")
replace_one("README.ja.md", "### v0.3.0以降のcurrent mainで完了（未release）", "### v0.4.0で提供")
replace_one("README.ja.md", "### v0.3.0以降もopen", "### v0.4.0以降もopen")
replace_one(
    "README.ja.md",
    "- [ ] stable automatable lifecycle/tool-discovery surfaceを持つ追加real MCP clientを評価",
    "- [ ] GitHub Copilot CLIのtool discovery/auth isolation researchを完了してからlive adapter化を判断 ([#48](https://github.com/git-ksk/mcp-interop/issues/48))\n- [ ] stable automatable lifecycle/tool-discovery surfaceを持つ追加real MCP clientを評価",
)

# Architecture release boundary.
replace_one(
    "docs/architecture.md",
    "The published stable release is still v0.3.0. The Cursor and Antigravity OAuth paths below describe **current main after v0.3.0** and are not present in the v0.3.0 release artifact.",
    "The current stable release is v0.4.0. The Cursor and Antigravity OAuth paths below are included in v0.4.0.",
)
replace_count("docs/architecture.md", "On current main, explicit `--oauth`", "In v0.4.0, explicit `--oauth`", 2)
replace_one(
    "docs/architecture.md",
    "GitHub Copilot CLI remains a follow-up candidate if a stable automatable MCP inventory/lifecycle surface can provide the same black-box evidence without model prompts.",
    "GitHub Copilot CLI remains research-only. Current PoC evidence shows real-client `initialize` / `notifications/initialized` on no-input startup, but not `tools/list` without an authenticated/model backend; see #48.",
)
replace_one(
    "docs/architecture.ja.md",
    "現在のstable releaseはv0.3.0です。以下のCursor/Antigravity OAuth pathは**v0.3.0以降のcurrent main（未release）**を説明しており、v0.3.0の公開artifactには含まれません。",
    "現在のstable releaseはv0.4.0です。以下のCursor/Antigravity OAuth pathはv0.4.0に含まれます。",
)
replace_count("docs/architecture.ja.md", "current mainでは、明示的な`--oauth`", "v0.4.0では、明示的な`--oauth`", 2)
replace_one(
    "docs/architecture.ja.md",
    "model promptなしで同等のblack-box evidenceを得られる安定したMCP inventory/lifecycle surfaceが確認できれば、今後の候補になります。",
    "GitHub Copilot CLIはresearch-onlyです。現在のPoCではno-input startupで実clientの`initialize` / `notifications/initialized`までは確認できましたが、authenticated/model backendなしの`tools/list`は未証明です。詳細は#48を参照してください。",
)

# Troubleshooting release boundary.
replace_one(
    "docs/troubleshooting.md",
    "The published stable release is still v0.3.0. Cursor and Antigravity OAuth support described below is available on **current main after v0.3.0** and is not part of the v0.3.0 release artifact.",
    "The current stable release is v0.4.0. Cursor and Antigravity OAuth support described below is included in v0.4.0.",
)
replace_one("docs/troubleshooting.md", "On current main, Cursor OAuth is explicit opt-in:", "In v0.4.0 and later, Cursor OAuth is explicit opt-in:")
replace_one("docs/troubleshooting.md", "On current main, Antigravity OAuth is explicit opt-in on macOS:", "In v0.4.0 and later, Antigravity OAuth is explicit opt-in on macOS:")
replace_one(
    "docs/troubleshooting.md",
    "- whether the checkout is the published v0.3.0 release or current main when reporting Cursor/Antigravity OAuth behavior;",
    "- the exact `mcp-interop` version when reporting Cursor/Antigravity OAuth behavior;",
)
replace_one(
    "docs/troubleshooting.ja.md",
    "現在のstable releaseはv0.3.0です。以下のCursor/Antigravity OAuth対応は**v0.3.0以降のcurrent main**で利用でき、v0.3.0の公開artifactには含まれません。",
    "現在のstable releaseはv0.4.0です。以下のCursor/Antigravity OAuth対応はv0.4.0に含まれます。",
)
replace_one("docs/troubleshooting.ja.md", "current mainではCursor OAuthを明示的にopt-inできます。", "v0.4.0以降ではCursor OAuthを明示的にopt-inできます。")
replace_one("docs/troubleshooting.ja.md", "current mainではmacOS上でAntigravity OAuthを明示的にopt-inできます。", "v0.4.0以降ではmacOS上でAntigravity OAuthを明示的にopt-inできます。")
replace_one(
    "docs/troubleshooting.ja.md",
    "- Cursor/Antigravity OAuthを報告する場合、公開v0.3.0かcurrent mainか",
    "- Cursor/Antigravity OAuthを報告する場合、正確な`mcp-interop` version",
)

print("v0.4.0 release documentation prepared successfully")
