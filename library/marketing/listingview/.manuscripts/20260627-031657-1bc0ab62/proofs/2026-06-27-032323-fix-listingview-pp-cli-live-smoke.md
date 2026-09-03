# ListingView CLI Live Smoke / Dogfood

Account: PLUS_YEARLY plan. Session via `auth login --chrome` (4 cookies, .app.listingview.io).

## Phase 5 live dogfood: 91/91 PASS (status: pass, level: full)
Every command exercised against the real API with the session injected (LISTINGVIEW_CONFIG).

## Novel commands (behaviorally verified live)
- `niche "cat sticker"` → **GO** (volume 675K, 4767 competing, ratio 141, 65% winnable).
- `listings audit 1581100221` → 13 tags extracted + graded (strong/ok/weak).
- `tags consensus "vinyl sticker"` → revenue-weighted consensus tags with share_pct.
- `tags rising "sticker"` → velocity-ranked uncrowded tags (competitionScore is high-is-uncrowded).
- `gaps StickerTX StickerPacked` → revenue-weighted tag gaps.
- `drift` → detected an injected +35% volume change; `--since 30d` window filter works.
- `opportunities` → 795 researched terms/tags ranked, deduped.

## Absorbed surface (live)
keywords/listings/shops/tags search, tags analyze/analytics/listings/shops/generate/extract, watchlist list/toggle, discover popular/recent, account me, sync — all return real data.

## Gate: PASS
