# Pixabay novel-features brainstorm (subagent audit trail)

## Customer model

**Maya — Frontend developer building a CMS-integrated media picker.** Wires Pixabay search into a Vue admin so editors drop royalty-free hero images into articles.
- Today: hand-rolls `fetch` calls, re-reads docs for the 14 colors / 20 categories, copy-pastes `webformatURL` munging.
- Weekly ritual: ~5 candidate photos per article across a dozen articles, eyeballed in a browser tab.
- Frustration: URLs silently 404 after 24h in staging; hits the 500-result error bumping page+per_page; no rate-limit visibility before a batch dies mid-run.

**Devon — Brand designer assembling a themed asset library.** Curates "winter / cozy / muted" collections of photos AND short video loops for campaign decks.
- Today: re-runs the same query on the website, downloads to a Downloads folder with cryptic filenames, loses track of contributor/license.
- Weekly ritual: 2-3 mood collections/week, ~40 assets mixing stills + footage, needs attribution at delivery.
- Frustration: no way to search photos and videos together; website caps discovery at 500; attribution is a manual scavenger hunt.

**Priya — AI agent / automation engineer wiring Pixabay into a content pipeline.**
- Today: uses the one text-only MCP server, regex-scrapes its prose back into JSON.
- Weekly ritual: thousands of automated lookups; needs deterministic JSON, stable IDs, reproducible local state.
- Frustration: text-only output unparseable; no local cache means every re-run burns quota; no quota visibility = nondeterministic batch failures.

**Sam — Data-minded content strategist tracking what performs.**
- Today: manually notes like/download counts in a spreadsheet.
- Weekly ritual: re-runs saved topic searches weekly, compares engagement to last week.
- Frustration: API gives only point-in-time snapshots; no history, no deltas, no contributor/tag ranking.

## Candidates (pre-cut)
(C1 harvest · C2 pull · C3 quota · C4 sizes · C5 credit · C6 collection · C7 trends · C8 contributors · C9 browse · C10 similar · C11 dupes · C12 media · C13 expired · C14 palette — full reasoning in run log)

## Survivors and kills

### Survivors

| # | Feature | Command | Score | Buildability | Persona | Buildability proof | Long Description |
|---|---------|---------|-------|--------------|---------|--------------------|------------------|
| 1 | 500-cap-busting harvest | `harvest "winter" --type images --auto-split --max 2000` | 8/10 | hand-code | Devon/Priya | Loops search clients across category/color/order facets, dedupes by id into SQLite | Use this to bulk-acquire >500 results for a theme into your store; do NOT use it for a single quick lookup — use 'images search' instead. |
| 2 | Resumable bulk download + attribution + 24h re-resolve | `pull --from-collection winter --size large --workers 8 --resume` | 8/10 | hand-code | Devon/Maya | Reads ids from a collection, re-fetches expired-URL ids by id, parallel workers, attribution sidecars, skips done ids | Use this to download assets already in your store/collection; do NOT use it to discover new assets — use 'harvest' or 'images search'. |
| 3 | Live rate-limit awareness + cost preview | `quota` | 7/10 | hand-code | Maya/Priya | Reads persisted X-RateLimit-* headers from last response, projects request cost of a planned harvest/pull | none |
| 4 | Unified image+video search | `media search "drone coastline" --limit 40` | 7/10 | hand-code | Devon/Priya | Fans one query across /api/ and /api/videos/ in parallel, merges into one persisted result set | Use this when you want stills and footage together; for one medium only, use 'images search' or 'videos search'. |
| 5 | Offline "more like this" by shared tags | `similar <id> --limit 20` | 6/10 | hand-code | Maya/Devon | Tag-set overlap (Jaccard) between target id and all synced hits, ranks by shared-tag count | Use this for offline 'more like this' over your synced store; do NOT use it to fetch new results from the API — use 'images search'. |
| 6 | Engagement deltas over time | `trends --tag "winter" --since last-week` | 6/10 | hand-code | Sam | Stores a stats snapshot per sync, diffs views/downloads/likes/comments per tag/id across runs | none |
| 7 | Contributor ranking | `contributors --by downloads --min-assets 3` | 6/10 | hand-code | Sam/Devon | GROUP BY user_id over synced image+video hits, ranks by total/avg engagement | none |
| 8 | Named offline collections | `collection add winter <ids>` / `collection list` | 6/10 | hand-code | Devon/Sam | CRUD over a local collection table keyed to synced hit ids; feeds pull/credit/similar | none |

### Killed candidates

| Feature | Kill reason | Closest-surviving-sibling |
|---------|-------------|---------------------------|
| C4 `sizes` | One-shot string transform, no weekly ritual; fold `_640→_180/340/960` swap into `pull --size` and search `--select`. | `pull` |
| C5 `credit` | Absorbed into `pull`'s attribution sidecars; standalone duplicates the same local walk. | `pull` |
| C9 `browse` | Thin wrapper over absorbed `images search` with `q` omitted — no transcendence. | `harvest` |
| C11 `dupes` | Diagnostic-only, no recurring action; dedupe-by-id already a property of harvest/sync. | `trends` |
| C13 `expired` | Absorbed into `pull`'s pre-download re-resolve + `sync --since`; standalone redundant. | `pull` |
| C14 `palette` | `colors` is an absorbed server facet; locally collapses to a `WHERE` covered by `sql`/`--select`. | `similar` |
