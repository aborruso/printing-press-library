# ListingView CLI Brief

## API Identity
- **Domain:** Etsy seller intelligence / SEO / keyword & listing research SaaS.
- **Product:** ListingView (listingview.io; app at app.listingview.io). "Easily find best-selling products on Etsy."
- **Users:** Etsy sellers and print-on-demand shops doing product research, keyword/SEO optimization, competitor analysis, and listing management.
- **Data profile:** Proprietary database of **134M Etsy listings, 3M shops, 1.8M tags** with sales/revenue/trend/conversion estimates. Per-listing SEO scores, per-keyword search volume/competition/demand, per-shop performance.
- **API:** No public/developer API, no docs, no API keys, no webhooks. A real internal REST backend exists at **`api.listingview.io`** (NestJS-style envelope `{"statusCode","message","data"}`), consumed by the Next.js SPA at `app.listingview.io`. Routes are undocumented and auth-gated.

## Reachability Risk
- **None.** `api.listingview.io` responds with clean JSON for unknown routes (`{"statusCode":404,"message":"Route not found!","data":null}`). No Cloudflare/WAF/DataDome/bot-protection signals. No `cf-mitigated`/`x-vercel-challenge`. `app.listingview.io` is plain Next.js (`x-powered-by: Next.js`), HTTP 200.
- **Discovery blocker is auth, not reachability:** endpoints require a logged-in session token; routes are unknown without watching the SPA's XHR.
- Probe-safe endpoint used: `GET https://api.listingview.io/` (404 envelope, read-only).

## Top Workflows
1. **Keyword research** — given a search term, get search volume, competition (competing listings/shops), buyer demand, recommendation score, price breakdown. (Search Term Analyzer)
2. **Listing analysis** — given an Etsy listing (URL/id), get sales/revenue estimates, SEO score, tag analytics, similar listings. (Listing Explorer)
3. **Shop analysis** — given a shop, get sales, top products, engagement, strategy signals. (Shop Analyzer)
4. **Marketplace database search** — filter 134M listings / 3M shops / 1.8M tags by sales/revenue/trend/conversion across 19+ filters; find best-sellers; CSV export.
5. **Tag intelligence** — generate market-validated tags, extract competitor tags, score tags by Opportunity/Demand/Competition/Velocity. (Tag Generator/Extractor/Analyzer)
6. **Watchlist** — save/track listings, shops, keywords over 2/6/12-month timeframes.

## Table Stakes (vs eRank, EverBee, Marmalead, Alura)
- Keyword search volume + competition + related/long-tail terms (eRank, EverBee, Marmalead all have this).
- Listing/shop sales & revenue estimates (EverBee, eRank).
- Tag generation and competitor tag extraction (eRank, Alura).
- Trend/seasonality data (Marmalead forecasting; EverBee trends).
- Best-seller / product opportunity discovery (EverBee product analytics).
- Bulk listing edit + optimizer (Alura, eRank).

## Data Layer
- **Primary entities:** keyword/search-term, listing, shop, tag, database-row (listing record), watchlist-item, trend, connected-shop.
- **Sync cursor:** research results are query-shaped (per term / per id), not a global feed — store each fetched result keyed by query + timestamp for offline reuse, drift, and gap analysis.
- **FTS/search:** local SQLite FTS over cached listings/keywords/tags for offline filtering, ranking by $/competition, and cross-query comparison the web UI doesn't offer.

## Codebase Intelligence
- Backend: `api.listingview.io` — NestJS REST (envelope `{statusCode,message,data}`). Frontend: Next.js SPA.
- Auth: app session token (Bearer JWT or cookie — to confirm via sniff). Etsy shop connection uses Etsy's OAuth handled server-side by ListingView (not exposed to a CLI).
- A separate `backend.listingview.io` host also resolves (DNS only; role TBD).
- Discovery requires authenticated browser-sniff of the SPA → `api.listingview.io` XHR to learn routes, params, response shapes, and auth header construction.

## Auth & Economics
- **Free tier ($0):** ~50 uses/mo for most research tools, 100 listing audits/mo, 1 connected shop, database access, watermarked mockups, 1GB. The **entire headline research surface (keyword/listing/shop/database/tag/watchlist) is usable on free tier.**
- **Plus ($24.99/mo):** higher quotas, CSV export, AI writing (2,000 credits), mockups, bulk editor, 50GB.
- **Scale:** "launching soon," unlimited.
- CLI implication: read/research commands work for any logged-in account (free included). AI-generation and bulk-write commands are paid/quota-gated and should be honestly labeled.

## Ecosystem
- **No existing tool of any kind.** No npm/PyPI wrapper, no SDK, no MCP server, no Claude skill, no reverse-engineering repo (web/GitHub searches returned only generic Etsy API libs). This CLI would be the first programmatic ListingView client.
- Public-library peers (different data vendors, informational): `everbee`, `erank`. User chose to build ListingView as a distinct product.

## User Pain Points
1. Estimates are directional (derived from public Etsy signals) — good for comparison, not exact financials. A CLI that caches results and computes *relative* rankings/drift turns this limitation into a strength.
2. Browser extension settings don't sync across devices; extension-only access to overlay data. A CLI gives scriptable, device-independent access.
3. Research quotas (50/mo free) + no bulk/programmatic access — power users can't batch keyword/listing research. A local-cache CLI that dedupes repeat queries and runs offline analysis stretches quota and adds batch workflows.

## Product Thesis
- **Name:** ListingView CLI (`listingview-pp-cli`).
- **Why it should exist:** The first programmatic, agent-native interface to ListingView's 134M-listing Etsy database. Brings keyword/listing/shop/tag research to the terminal and to AI agents, with a local SQLite store that enables offline filtering, $/competition ranking, saved-search drift, and cross-query comparison the web UI never built — while stretching the free-tier quota by caching and deduping.

## Build Priorities
1. **Auth + transport** — capture/replay app session token to `api.listingview.io`; `doctor`, `auth` flow. Confirm Bearer vs cookie via sniff.
2. **Core research commands** (Priority 1, free-tier usable): keyword/search-term analyze, listing explore, shop analyze, database search (with filters), tag generate/extract/analyze, watchlist list.
3. **Local store** (Priority 0): SQLite mirror of every fetched keyword/listing/shop/tag result; FTS; query+timestamp keying.
4. **Transcendence** (Priority 2): offline cross-query ranking, saved-search drift/price-drop detection, keyword-gap analysis, $/competition opportunity scoring, quota-stretching dedupe — features only possible with a local store.
