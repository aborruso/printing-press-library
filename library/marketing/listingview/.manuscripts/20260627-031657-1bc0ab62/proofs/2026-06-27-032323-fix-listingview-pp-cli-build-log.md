# ListingView CLI Build Log

Manifest transcendence rows: 7 planned, 7 built. Phase 3 passes.

## Foundation
- Fixed generator cookie-auth bug: `client.go` wrapped the whole cookie jar in a single cookie named "Cookie" (`AddCookie{Name:"Cookie"}`), which the backend 401'd. Replaced with `applyListingViewAuth` (hand-authored `internal/client/listingview_csrf.go`): sends the raw Cookie header, derives `X-CSRF-Token` from the `csrf_token` cookie (CSRF double-submit), and sets optional `shopid`. RETRO CANDIDATE: generator emits broken cookie handling for `auth.header: Cookie` specs. Re-apply this 7-line edit if regenerated.
- Auth validated end-to-end: `auth login --chrome` imports 4 cookies, `account me` / `keywords search` return live data.

## Absorbed (generator-emitted, all live-verified)
keywords search; listings search; shops search/listings; tags search/analyze/analytics/listings/shops/generate/extract; watchlist list/toggle; discover popular/recent; account me. Framework: sync/search/sql/analytics/doctor/auth.
- Removed 2 unusable endpoints (listing-explorer/analytics, shop-analyzer/analytics): require a sync-first/async input shape that could not be determined; all 7 novel commands work without them.

## Transcendence (7, hand-coded, behaviorally verified live)
1. niche <term> — GO/CAUTION/AVOID verdict from keyword demand/competition + top-seller winnability. Verified: "cat sticker" -> GO (ratio 141, 65% winnable).
2. listings audit <id> — extract + grade a listing's tags. Verified: 13 tags graded.
3. tags consensus <term> — revenue-weighted consensus tags across top sellers. Verified.
4. tags rising <term> — velocity-ranked uncrowded tags (competitionScore is high-is-uncrowded). Verified.
5. drift [--since] — diffs snapshot history. Verified: detects injected +35% volume change; --since window filter works.
6. gaps <my> <competitor> — revenue-weighted tag gaps. Verified.
7. opportunities — local shortlist over researched snapshots, deduped. Verified: 795 researched, ranked.

## Store
- Hand-authored snapshot table `lv_snapshots` (lazy CREATE in `listingview_research.go`); live commands (niche, tags rising) write keyword/tag metrics; drift/opportunities read history. Data-source annotations: live (niche/audit/consensus/rising/gaps), local (drift/opportunities).

## Tests
- internal/client/listingview_csrf_test.go (CSRF derivation, cookie header)
- internal/cli/listingview_research_test.go (nicheVerdict, gradeTag, parsing, stringsOf — caught & fixed an object-array parse bug)

## Deferred
- listing-explorer & shop-analyzer/analytics endpoints (unknown input DTO).
