# Pixabay CLI Brief

## API Identity
- **Domain:** Free stock media search — photos, illustrations, vectors, and videos (CC0-style Pixabay Content License).
- **Endpoints:** `GET https://pixabay.com/api/` (images) and `GET https://pixabay.com/api/videos/` (videos). Two endpoints, read-only.
- **Auth:** `key` query param (required), free per-account API key. Rate limit ~100 req/60s (default), surfaced via `X-RateLimit-Limit` / `X-RateLimit-Remaining` / `X-RateLimit-Reset` response headers.
- **Users:** Developers, designers, content creators, bloggers, and AI agents needing royalty-free imagery/footage by search query.
- **Data profile:** JSON. Each image/video hit is a rich, stable, ID-keyed record (id, tags, dimensions, URLs at multiple sizes, views/downloads/likes/comments, contributor). Perfect for a local store.

## Reachability Risk
- **[Low]** The website homepage (`pixabay.com/`) is behind a Cloudflare challenge (403 `cf-mitigated: challenge`), but the **API endpoints are NOT** — `/api/` returns a clean `400 Invalid or missing API key` and `/api/docs/` returns 200. The developer API surface is deliberately open and key-gated.
- Probe-safe endpoint: `GET /api/` (returns 400 without key — expected; 200 with key).
- No 403/bot-protection on the API. No blocked-api journal entry. No reachability issues reported in community wrappers.

## API Surface (exact contract)

### Images — `GET /api/`
Params: `key`(req), `q`(≤100 chars, URL-encoded), `lang`(26 langs, default en), `id`(by-id lookup, comma list allowed), `image_type`(all/photo/illustration/vector), `orientation`(all/horizontal/vertical), `category`(20: backgrounds, fashion, nature, science, education, feelings, health, people, religion, places, animals, industry, computer, food, sports, transportation, travel, buildings, business, music), `min_width`, `min_height`, `colors`(14: grayscale, transparent, red, orange, yellow, green, turquoise, blue, lilac, pink, white, gray, black, brown — comma-separable), `editors_choice`(bool), `safesearch`(bool), `order`(popular/latest), `page`(default 1), `per_page`(3–200, default 20), `pretty`, `callback`(JSONP).
Response: `total`, `totalHits`, `hits[]`{ id, pageURL, type, tags, previewURL/Width/Height, webformatURL/Width/Height, largeImageURL, imageWidth, imageHeight, imageSize, views, downloads, likes, comments, user_id, user, userImageURL }. Full-access keys add: `fullHDURL`, `imageURL` (original), `vectorURL`.

### Videos — `GET /api/videos/`
Same params except `video_type`(all/film/animation) instead of image_type/orientation/colors.
Response: `total`, `totalHits`, `hits[]`{ id, pageURL, type, tags, duration, videos{large,medium,small,tiny}{url,width,height,size,thumbnail}, views, downloads, likes, comments, user_id, user, userImageURL }.

### Hard constraints (drive design)
- **500-result cap:** `totalHits` is capped at 500 per query and `page × per_page ≤ 500` or it errors. Harvesting more requires query-splitting (by category/color/order/type).
- **24h URL expiry:** `webformatURL` (and friends) valid 24h. Caching responses for repeat requests within 24h is a ToS obligation.
- **`webformatURL` size trick:** replace `_640` with `_180`/`_340`/`_960` to get other sizes with **no extra API call** (client-side variant URLs).
- **ToS:** must credit Pixabay; "let people search images, not automated mass-download."

## Top Workflows
1. **Search images/videos by query + filters** (the core: q + image_type/category/color/orientation/order). 90% of usage.
2. **Get a specific item by ID** (retrieve/re-resolve a known image or video).
3. **Download** a chosen image/video at a chosen size to disk (single + batch, with attribution).
4. **Browse Editor's Choice / popular / latest** within a category (discovery without a query).
5. **Build a reusable local collection** — sync search results into a local DB, then search/filter/sort offline and re-export.

## Table Stakes (every competitor has at least some of these — we match ALL)
Search images; search videos; get-by-id; filter by image_type/video_type, category, colors, orientation, lang, min_width/min_height, editors_choice, safesearch; order popular/latest; paginate (page/per_page); pick size variant (preview/web/large/fullHD/original); download single; batch/list download; parallel download workers; pretty JSON; JSONP callback; config file for API key.

## Data Layer
- **Primary entities:** `images` (photos/illustrations/vectors) and `videos`. Both ID-keyed, stable, richly attributed.
- **Sync model:** persist `hits` from any search into SQLite, deduped by `id`. Cache full API responses ≤24h to honor ToS + avoid re-spend.
- **FTS/search:** full-text over `tags` + `user`; structured filters over category/type/colors/dimensions/stats locally — no quota cost, works offline.
- **Compounding value:** likes/downloads/views snapshots over time → trend deltas; cross-query dedupe; saved collections.

## Codebase Intelligence
- Bar to beat #1: **netbrain/gopixabay** (Go, MIT, abandoned 2016, images-only, no go.mod). Richest legacy flag set: parallel `--num` download workers, `--size` (og/lg/md/sm/xs), `--response-group`, `--id` (hash/comma), YAML config. We match + add video, modernize, persist.
- Bar to beat #2: **zym9863/pixabay-mcp** (TS, ~7★, most active, only agent-native option) — but **text-only output**; structured JSON/pagination still on roadmap. We ship real structured JSON + MCP now.
- Rest of ecosystem (npm pixabayjs/pixabay-api, PyPI python-pixabay/pixabay-python) is 7–11 years stale, thin one-shot search wrappers.

## User Vision
- (none provided — user said "continue", chose official API, deferred API key to self-test later)

## Reachability / Auth Notes
- Auth: `api_key` in query param `key`. Canonical env var: `PIXABAY_API_KEY` (no universally-canonical name exists; this matches the leading MCP server's convention). Read-only API — no mutation endpoints, so no write-side smoke-test risk.

## Product Thesis
- **Name:** `pixabay-pp-cli` ("pixabay")
- **Headline:** Every Pixabay search filter the API has — plus a local SQLite index, offline FTS search, resumable bulk download that beats the 500-result cap, live rate-limit awareness, and agent-native JSON no other Pixabay tool ships today.
- **Why it should exist:** The only maintained Pixabay CLI is 10 years dead and images-only. The only agent-native tool is text-only. Nobody persists results, works around the 500-cap, surfaces rate-limit headers, or unifies image+video+by-id+download in one modern binary. This becomes the default.

## Build Priorities
1. **P0 foundation:** images + videos search clients, by-id lookup, local SQLite store for both entities, sync, offline FTS search, SQL passthrough — with `--json`/`--select`/`--compact` everywhere.
2. **P1 absorb:** the full table-stakes flag surface above (all filters, size variants, single + batch download, parallel workers, order, pagination, config/env key, rate-limit header surfacing).
3. **P2 transcend:** 500-cap-busting harvest (auto query-split), resumable/dedup bulk download with 24h-expiry handling + attribution sidecars, offline collections, trend/stat deltas, quota-aware planning — the commands only a local-store CLI can do.
