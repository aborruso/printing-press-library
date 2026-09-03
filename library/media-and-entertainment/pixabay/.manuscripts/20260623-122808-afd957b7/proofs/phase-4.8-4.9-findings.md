# Phase 4.8/4.9 SKILL+README+AGENTS correctness — findings

Reviewer: general-purpose agent vs shipped binary.

## ERRORS fixed (regenerated from corrected research.json)
1. `config set-key <key>` — no `config` command exists; correct command is `auth set-token <key>`. Was in README, SKILL, research.json auth_narrative + troubleshoot, and quota.go hint. FIXED in research.json + quota.go; regenerated all surfaces.
2. Headline claimed "query with `sql`" — no `sql` subcommand exists. Removed from research.json value_prop; regenerated.

## WARNING fixed
3. harvest "category/color/order facets" overstated (harvest splits category+order only). Fixed in research.json novel_features + novel_features_built; regenerated. Also patched 3 preserved files (root.go, which.go, mcp/tools.go) that regen-merge kept.

## PASS
Flag accuracy (images/videos search use --per-page, media search uses --limit), novel-features alignment (8 built = 8 claimed), trigger phrases, anti-triggers (read-only/Pixabay-only/no mass-download), brand casing, no marketing fluff, no placeholder literals.
