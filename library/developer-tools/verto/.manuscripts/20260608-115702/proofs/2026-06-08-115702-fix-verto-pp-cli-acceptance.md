# Acceptance Report: verto

Level: Full Dogfood (live, no-auth API)
Tests: 40/44 passed; 4 BLOCKED_FIXTURE (not failures)

## BLOCKED_FIXTURE (machine limitation, not CLI defects)
- batch (happy_path, json_fidelity): example invokes `batch points.csv ...`; the live runner's sandbox cwd has no `points.csv` to provision -> exit 2 (file not found). Behavior verified manually live: CSV->CSV/GeoJSON, column auto-detect, 32000 chunking, --skip-invalid bisection + --rejects all correct.
- geojson (happy_path, json_fidelity): example invokes `geojson aree.geojson ...`; same missing-fixture cause. Behavior verified manually live: Point + LineString reprojection + CRS member correct.
- Machine gap for retro: the live dogfood matrix derives file-arg happy-paths from command examples but does not provision input files, so any command taking a required input file path fails in the sandbox.

## Verified correct (manual live)
- convert ground-truth fixtures EXACT (4265->6706 and 3003->6707).
- systems (offline cache + --search), inspect, targets (datum-family-aware), detect (heuristic), roundtrip (residual; back-conversion exact), cache (--stats/--clear).
- Out-of-grid -> typed usage error with coverage hint; batch --skip-invalid isolates offenders.

## Redaction
No PII in live responses (coordinate data only).

Gate: PASS for the runnable matrix (40/40 non-fixture tests); 4 blocked by un-provisionable file fixtures.
