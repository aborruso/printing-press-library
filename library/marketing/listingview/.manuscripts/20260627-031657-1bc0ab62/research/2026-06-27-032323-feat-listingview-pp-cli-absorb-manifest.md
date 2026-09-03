# ListingView CLI Absorb Manifest

Ecosystem note: **No ListingView-specific tool exists** (no SDK/MCP/CLI/wrapper). Absorb targets = ListingView's own feature surface + competitor parity (eRank, EverBee, Marmalead, Alura). Public-library peers `everbee` and `erank` (different data vendors) set the transcendence bar.

## Absorbed (match or beat every feature that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | Keyword research (volume, competition, demand, recommendation score) | ListingView Search Term Analyzer; eRank/EverBee/Marmalead | (generated endpoint) keywords search | offline cache, `--json`/`--select`, SQL-composable, batch |
| 2 | Listing DB search/rank (140M listings by sales/revenue/trend) | ListingView Database; EverBee product analytics | (generated endpoint) listings search | offline ranking, $/competition sort, dedupe quota stretch |
| 3 | Single-listing analytics (SEO score, sales est, tags) | ListingView Listing Explorer | (generated endpoint) listings explore | cached, agent-native |
| 4 | Shop DB search/rank | ListingView Database Shops | (generated endpoint) shops search | offline |
| 5 | Shop performance analytics | ListingView Shop Analyzer; EverBee | (generated endpoint) shops analyze | cached |
| 6 | Shop top listings | ListingView Shop Analyzer | (generated endpoint) shops listings | offline rank |
| 7 | Tag DB search/scoring (opportunity/demand/competition/velocity) | ListingView Database Tags; eRank | (generated endpoint) tags search | offline |
| 8 | Tag analyzer (single tag) | ListingView Tag Analyzer; eRank | (generated endpoint) tags analyze | cached |
| 9 | Generate market-validated tags from keyword | ListingView Tag Generator; Alura | (generated endpoint) tags generate | scriptable batch |
| 10 | Extract a listing's tags | ListingView Tag Extractor; eRank competitor tags | (generated endpoint) tags extract | scriptable |
| 11 | Watchlist list/add/remove | ListingView Watchlist | (generated endpoint) watchlist list / watchlist toggle | offline |
| 12 | Popular terms + recent research history | ListingView discovery | (generated endpoint) discover popular / discover recent | cached |
| 13 | Account/plan/connected shops | ListingView account | (generated endpoint) account me | feeds doctor |
| 14 | Tag analyzer sub-views (analytics, top listings, top shops) | ListingView Tag Analyzer tabs | (generated endpoint) tags analytics / tags listings / tags shops | composable |
| 15 | Local SQLite store of every fetched result + FTS + raw SQL | Unique to this CLI | (behavior in listingview-pp-cli sync) framework sync/search/sql over cached research | offline, FTS5, snapshot history the web UI never keeps |

Every absorbed row is covered by the generator's typed endpoint surface (`spec-emits`) or the framework store. No stubs.

## Transcendence (only possible with our approach)

| # | Feature | Command | Buildability | Score | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|-------|-------------------------|------------------|
| 1 | Niche go/no-go verdict | `niche "<term>"` | hand-code | 9 | Joins keywords search (demand/competition) + listings search (best-seller density, price band, sales-recency=winnability) into one composite; ListingView's real sales/revenue + trend data scores *winnability*, not just volume. | Use for a single-term go/no-go before committing designs. To rank across many already-researched terms use 'opportunities'; for whether a term is moving over time use 'drift'. |
| 2 | Listing SEO + tag teardown | `listings audit <id\|url>` | hand-code | 9 | Combines listings explore (per-listing **SEO score**, unique to ListingView) with tags analyze on each of the listing's tags, flags dead-weight tags, proposes replacements from tags generate; eRank/EverBee have no SEO score. `--shop` batches the seller's whole catalog. | Works on any listing (yours or a rival's). For the consensus tag set across a term's top sellers use 'tags consensus'; for missing tags vs a competitor shop use 'gaps'. |
| 3 | Velocity early-mover | `tags rising` | hand-code | 8 | Ranks tags by ListingView's **velocity score** (accelerating demand) while competition is still low; neither eRank nor EverBee exposes velocity; local store confirms acceleration across snapshots. | none |
| 4 | Revenue-weighted tag consensus | `tags consensus "<term>"` | hand-code | 8 | listings search returns top sellers for a term *with revenue estimates*; tags extract pulls each one's tags; ranks tags by frequency weighted by revenue, so consensus reflects what actually sells, not a flat count. | Term-level "what do winners tag with." To audit one specific listing's tags use 'listings audit'. |
| 5 | Snapshot drift diff | `drift` | hand-code | 9 | The API returns only point-in-time estimates; the local store keyed by query+timestamp is the only place history exists, so diffing saved keyword/listing/shop snapshots (volume change, new best-seller entrants, price drops, SEO/rank moves) is impossible without this CLI. | Detects change across cached entities. For a one-shot verdict on a new term use 'niche'; for best static opportunities use 'opportunities'. |
| 6 | Revenue-weighted shop gaps | `gaps <myshop> <competitor>` | hand-code | 8 | Cross-entity join: tags extracted from my listings vs a rival's, ranked by the revenue those tags drive for the rival; deeper than EverBee/eRank flat gap lists because it prioritizes gaps that convert. | Two-shop comparison. For tag quality on a single listing use 'listings audit'; for term-level winning tags use 'tags consensus'. |
| 7 | Local opportunity shortlist | `opportunities` | hand-code | 7 | Pure local SQLite aggregation over every cached keyword/tag result; ranks untapped plays (high demand, low competition, rising velocity) with zero new API calls, deduping repeat queries to stretch the 50/mo free quota. | Ranks across everything already researched. For a deep single-term verdict use 'niche'; for what's changed use 'drift'. |

**Hand-code count: 7 transcendence features** (all require SQLite joins / cross-endpoint synthesis / snapshot history). 0 stubs. Absorbed features are generator-emitted.

## Build risks / notes
- Auth: cookie + X-CSRF-Token (cookie-derived) + shopid header. Requires a hand-authored `listingview_csrf.go` injecting headers into `Config.Headers` (proven pattern: kdpnichefinder). Foundation for all commands.
- Data is directional (public-Etsy-derived estimates) — transcendence features lean on *relative* ranking and *change over time*, which turns this limitation into a strength.
- `drift`, `opportunities`, `gaps` depend on the local store accruing snapshots; they degrade gracefully (honest "no history yet / run sync") when empty.
