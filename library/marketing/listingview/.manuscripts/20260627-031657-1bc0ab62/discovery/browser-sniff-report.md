# ListingView Browser-Sniff Discovery Report

## User Goal Flow
- Goal: "Research Etsy keywords, listings, shops, and tags" (read-heavy research).
- Steps completed: logged-in dashboard load; Database listings search ("sticker") → captured `getFilteredListings` (full real request+response).
- Method: chrome-MCP capture of the user's logged-in Chrome session (extension proved unstable mid-session — repeated tab-group teardown after a heavy eval froze the renderer), supplemented by reliable JS-bundle extraction of the full endpoint catalog via shell.

## Browser-Sniff Configuration
- Backend: Claude chrome-MCP (`mcp__claude-in-chrome__*`) for the live capture + shell `curl` of public Next.js chunks for the endpoint catalog and request shapes.
- Pacing: single live capture; bundle extraction is static (no rate concern).
- Proxy pattern: YES — same-origin `/api/proxy/<upstream-path>` rewrite. App calls `app.listingview.io/api/proxy/api/integration/etsy/<op>`; server forwards to `api.listingview.io` / `backend.listingview.io`. CLI base = `app.listingview.io`.

## Endpoints Discovered (28 from bundles; 22 in CLI spec)
Method POST unless noted. Path prefix `/api/proxy/api/integration/etsy/`.
| Op | Body | Tool |
|----|------|------|
| getFilteredListings | search,sort_column,sort_order,page,limit,filter,timeframe,filters,salesInterval | Database/Listings |
| getFilteredKeywords | search,sort_column,sort_order,page,limit,filters | Search Term Analyzer |
| get-filtered-shops | search,sort_column,sort_order,page,limit,filter,timeframe,salesInterval | Database/Shops |
| getFilteredTags | search,sort_column,sort_order,page,limit,filters | Database/Tags |
| tag-analyzer (+ /analytics,/listings,/shops,/tags) | {tag} | Tag Analyzer |
| tag-extractor | {listingId} | Tag Extractor |
| tag-generator | {keyword} | Tag Generator |
| shop-analyzer/analytics,/progress | {shopName} | Shop Analyzer |
| shop-analyzer/listings | {shopName,sortBy,order,limit,page} | Shop Analyzer |
| listing-explorer/analytics | {listingId} | Listing Explorer |
| list-favourite (GET) | ?type=&limit= | Watchlist |
| add-remove-favourite | {type,id} | Watchlist |
| popular (GET), recent-history (GET) | ?type= | Discovery |
| /api/auth/me (GET) | — | Account |
Other (not in v1 spec): edit-listing, optimizer-listing, listing-optimizer*, asset-library*, files*, mockups*, stripe* (paid/management).

## Authentication Context (authenticated session used)
- Model: **cookie session (httpOnly) + `X-CSRF-Token` header (value = `csrf_token` cookie) + `shopid` header**. No Bearer token; no token in localStorage.
- Real captured request headers on getFilteredListings: `X-CSRF-Token: <uuid>`, `shopid: 65974856`, `Content-Type: application/json`. Session cookie auto-attached (httpOnly, invisible to JS).
- CLI auth plan: `auth login --chrome` imports cookies for `.app.listingview.io`; a hand-authored extension parses `csrf_token` from the imported cookie string and injects `X-CSRF-Token` (+ optional `shopid` from config) into `Config.Headers` (applied to every request by the generated client). Proven pattern: see library `kdpnichefinder/internal/cli/kdpnichefinder_csrf.go` (Laravel XSRF analog).
- Session state excluded from manuscript archiving (lives outside DISCOVERY_DIR).

## Traffic Analysis
- reachability.mode: `browser_clearance_http` (0.82) — conservative; driven by the cookie header in the capture. Direct probes of `app.listingview.io`/`api.listingview.io` showed NO WAF/Cloudflare/bot-protection, so cookie+CSRF replays over Surf/standard HTTP with no resident browser.
- auth candidates: `cookie_session_csrf` (0.95, top), `cookie` (0.80).
- Response envelope (consistent): `{"statusCode":200,"message":"...","data":{...}}` (OpenSearch-backed).

## Coverage Analysis
- Exercised: listings (real). Catalogued + spec'd: keywords, shops, tags, tag-analyzer family, shop-analyzer family, listing-explorer, watchlist, discovery, account.
- Likely missed / deferred: bulk editor, listing optimizer, asset library, AI mockups, Stripe billing (paid/management — out of v1 research scope).

## Replayability
- Printed CLI replays direct/Surf HTTP to `app.listingview.io/api/proxy/...` with imported cookies + derived CSRF header. No resident browser at runtime. Validation deferred to Phase 5 dogfood via `auth login --chrome`.
