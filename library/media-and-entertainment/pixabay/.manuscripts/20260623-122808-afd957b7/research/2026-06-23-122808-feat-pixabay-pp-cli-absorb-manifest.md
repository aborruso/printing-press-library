# Pixabay CLI Absorb Manifest

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Search images by query | gopixabay / python-pixabay / zym9863-mcp | (generated endpoint) images search | Offline FTS, --json/--select, SQL-composable |
| 2 | Search videos by query | zym9863-mcp (gopixabay lacks video) | (generated endpoint) videos search | Offline, --json, unified with images |
| 3 | Get image(s) by ID | gopixabay --id | (generated endpoint) images get | Comma-list, persisted to store |
| 4 | Get video(s) by ID | weak elsewhere | (generated endpoint) videos get | Persisted, agent-native |
| 5 | Filter by image_type (photo/illustration/vector) | all tools | (behavior in pixabay-pp-cli images search) --image-type flag | Validated enum |
| 6 | Filter by video_type (film/animation) | zym9863-mcp | (behavior in pixabay-pp-cli videos search) --video-type flag | Validated enum |
| 7 | Filter by category (20 values) | all | (behavior in pixabay-pp-cli images search) --category | Validated enum |
| 8 | Filter by colors (14, comma-list) | python-pixabay / AlexusBlack | (behavior in pixabay-pp-cli images search) --colors | Multi-select |
| 9 | Filter by orientation | gopixabay / python-pixabay | (behavior in pixabay-pp-cli images search) --orientation | Validated |
| 10 | Filter by lang (26) | all | (behavior in pixabay-pp-cli images search) --lang | Validated |
| 11 | Filter min_width/min_height | all | (behavior in pixabay-pp-cli images search) --min-width/--min-height | |
| 12 | editors_choice flag | all | (behavior in pixabay-pp-cli images search) --editors-choice | |
| 13 | safesearch flag | all | (behavior in pixabay-pp-cli images search) --safesearch | |
| 14 | order popular/latest | all | (behavior in pixabay-pp-cli images search) --order | |
| 15 | Pagination page/per_page | all | (behavior in pixabay-pp-cli images search) --page/--per-page | |
| 16 | Pretty/indent JSON output | all | (behavior in pixabay-pp-cli, global) --json | Pretty default + --compact/--select/--csv |
| 17 | Config file / env for API key | gopixabay YAML | (generated) config + PIXABAY_API_KEY env | doctor health check |
| 18 | Local SQLite store + sync of results | NOBODY | (generated) sync / search / sql | First Pixabay tool with persistence + offline FTS |

## Transcendence (only possible with our approach)

| # | Feature | Command | Buildability | Score | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|-------|------------------------|------------------|
| 1 | 500-cap-busting harvest | harvest | hand-code | 8/10 | Orchestrates many calls + dedupe-by-id past a hard 500-result API ceiling the raw endpoint cannot cross | Use this to bulk-acquire more than 500 results for a theme into your store; do NOT use it for a single quick lookup — use 'images search' instead. |
| 2 | Resumable bulk download + attribution + 24h re-resolve | pull | hand-code | 8/10 | Adds resume, re-resolve-by-id for 24h-expired URLs, and attribution sidecars on top of raw download; needs local id store | Use this to download assets already in your store/collection; do NOT use it to discover new assets — use 'harvest' or 'images search'. |
| 3 | Live rate-limit awareness + cost preview | quota | hand-code | 7/10 | Surfaces X-RateLimit-* headers no existing wrapper exposes, plus persisted last-response state | none |
| 4 | Unified image+video search | media search | hand-code | 7/10 | Parallel two-endpoint fan-out + merged ranking the API can't do in a single call | Use this when you want stills and footage together; for one medium only, use 'images search' or 'videos search'. |
| 5 | Offline "more like this" by shared tags | similar | hand-code | 6/10 | Local tag-set (Jaccard) join over SQLite — the API has no related-item endpoint | Use this for offline 'more like this' over your synced store; do NOT use it to fetch new results from the API — use 'images search'. |
| 6 | Engagement deltas over time | trends | hand-code | 6/10 | Snapshot history + diff join; the point-in-time API gives no history | none |
| 7 | Contributor ranking | contributors | hand-code | 6/10 | Local GROUP BY across image+video stores; the API has no aggregation endpoint | none |
| 8 | Named offline collections | collection | hand-code | 6/10 | Local SQLite CRUD feeding pull/credit/similar; no Pixabay tool persists collections | none |

## Scope summary
- **18 absorbed** features (most generator-emitted from the 2-endpoint spec; the full filter surface, the local store, and config/env auth).
- **8 transcendence** features, all **hand-code** (~50-150 LoC each + root.go wiring).
- No stubs. No paid dependencies. Read-only API → no mutation risk.
- Bar cleared: gopixabay (Go, 2016, images-only, no persistence) and zym9863-mcp (text-only). We ship video + persistence + working structured JSON + the 8 novel commands none of them have.
