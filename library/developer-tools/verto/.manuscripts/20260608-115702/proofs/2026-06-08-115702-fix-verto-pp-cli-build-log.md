# verto-pp-cli — Build Log

## Generator scaffold
- Minimal internal YAML spec (one `systems` resource, auth: none). Generator emitted root, cliutil, output helpers, MCP cobratree mirror, doctor, and STUBS for the 5 novel commands (from research.json).
- No store generated (single POST endpoint) -> hand-built JSON-file store.

## Hand-built
- internal/volapi: typed client over generated client.Post* (Info, Convert w/ 32000 chunking, ConvertSkipping bisection, typed *Error incl. OutsideGrid()).
- internal/store: dependency-free JSON-file cache (systems offline + conversion write-through cache).
- internal/cli/verto_shared.go: getSystems(cache-first), parseEpsg, requireEpsgFlags, convertWithCache, vertoErr mapping.
- internal/cli/refdata.go: curated static reference table for all 20 EPSGs (datum family, kind, unit, zone, false easting) — pp:novel-static-reference.

## Commands
Absorbed: convert (single/stdin), batch (CSV/GeoJSON, column auto-detect, chunking, --skip-invalid bisection + --rejects), geojson (recursive geometry reprojection + CRS), systems (offline cache, --search/--refresh).
Transcendence: roundtrip (A->B->A residual, metres), targets (datum-family-aware valid destinations), inspect (system card), detect (heuristic source EPSG), cache (--stats/--clear offline replay).

## Verified live against the API
- Ground-truth fixtures exact: 4265->6706 (12.4924,41.8902)->(12.4921961827,41.8908506304); 3003->6707 (2300000,4640000)->(1299961.341,4639997.131).
- Datum-family partition confirmed (same-family rejected, cross allowed).
- Out-of-grid -> typed usage error; batch --skip-invalid isolates offenders.
- roundtrip back-conversion exact (projected residual 0; geographic ~1.1e-5 m).

## Deferred / notes
- detect is an explicit low-confidence heuristic (easting magnitudes overlap); always recommends `inspect` to confirm.
- geojson uses all-or-nothing convert (no --skip-invalid); acceptable for typical in-grid data.
- `utente`/`chiave` hardcoded placeholders (NOT auth, per the manual).
