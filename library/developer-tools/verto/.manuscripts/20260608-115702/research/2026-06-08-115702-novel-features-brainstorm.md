# Novel-features brainstorm (verto) — audit trail

(Full subagent output — Customer model, candidates, survivors, kills.)

## Customer model
- Geom. Laura R. — surveyor/topografo (Bologna): Roma40/Gauss-Boaga 3003 -> RDN2008 cadastral, needs official IGM numbers not QGIS generic PROJ.
- Ing. Marco T. — civil engineer (Rome): mixed legacy ED50/Gauss-Boaga, guesses 3003 vs 3004 / 23032 vs 23033; false-easting & zone errors corrupt datasets.
- Dev. Sara C. — geospatial dev (civic-tech ETL): nightly normalize to RDN2008, needs jq-composable JSON, typed exit codes, offline replay in CI.
- Dott.ssa Elena V. — PA cartographer (regional GIS): certify lossless datum chain, wants a deterministic residual-error metric for QA reports.

## Survivors (>=5/10)
1. roundtrip (8) — A->B->A residual error report (de,dn,distance). Personas: Elena, Laura.
2. targets (7) — valid conversion targets per EPSG, encodes same-datum/identity edge. Personas: Marco, Sara.
3. inspect (7) — system card: datum family, axis order e/n, units, zone, false easting/northing (static-reference over 20 EPSG). Personas: Marco, Laura.
4. detect (6) — guess source EPSG from coordinate magnitude (pure-local heuristic, low-confidence). Personas: Marco, Sara.
5. cache (6) — offline write-through conversion cache keyed (inEpsg,outEpsg,e,n); folded into convert/batch + `cache --stats/--clear`. Persona: Sara.

## Killed
- stdin pipe mode (folded into convert/batch), sql (empty store), reconcile (no 2nd entity), compare (wrapper over inspect), verify --fixture (overlaps roundtrip), grid-status (help text), watch (scope creep), zones (overlaps inspect/targets).
