# Verto Online CLI — Absorb Manifest

Ecosystem: EMPTY. No existing CLI for IGM volapi (the "volapi" GitHub repos are for Volafile, unrelated). Only adjacent tools: PROJ/cs2cs, QGIS, ogr2ogr (generic CRS math, no official IGM grids without licensed NTv2). IGM's own surface is browser-only file/single converters and explicitly invites third-party clients.

## Absorbed (match or beat everything that exists)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Convert a single coordinate | IGM Verto Online web form | `convert --from <epsg> --to <epsg> <e> <n>` | Scriptable, --json/--csv, axis-aware, typed exit codes, offline cache |
| 2 | Convert a file of coordinates | IGM browser file clients | `batch` (CSV/GeoJSON in/out, column mapping) | Terminal, no browser, auto-chunk at 32000, dry-run |
| 3 | List supported reference systems | IGM `info` endpoint | `systems` (list/search) | Offline from SQLite, FTS over descrizione, --json |
| 4 | Transform between Italian datums (Roma40/ED50/IGM95/ETRS89/RDN2008) | PROJ/cs2cs | hand-written volapi client | Official IGM transform, zero NTv2 grid-file setup |
| 5 | Reproject GeoJSON geometries | ogr2ogr / PROJ | `geojson` | Official IGM transform, no grid files installed |

## Transcendence (only possible with our approach)
| # | Feature | Command | Score | Why Only We Can Do This |
|---|---------|---------|-------|------------------------|
| 1 | Round-trip precision report | `roundtrip --from <epsg> --to <epsg> <e> <n>` | 8 | Chains A->B->A calls + local residual arithmetic (Δe, Δn, distance); no single API call yields a precision metric |
| 2 | Valid conversion targets per EPSG | `targets <epsg>` | 7 | Local query over cached systems + encodes the same-datum/identity edge the API rejects |
| 3 | Reference-system inspection card | `inspect <epsg>` | 7 | Static-reference table (datum family, axis order e/n, units, zone, false easting/northing) joined to cached descrizione |
| 4 | Coordinate-based source EPSG detector | `detect <e> <n>` | 6 | Pure-local magnitude heuristics (decimal-deg vs Gauss-Boaga ~1.5M/2.5M easting vs UTM zone ranges); no API call |
| 5 | Offline conversion cache (replay) | folded into `convert`/`batch`; `cache --stats`/`--clear` | 6 | Write-through SQLite keyed (inEpsg,outEpsg,e,n); repeat conversions served instantly offline (CI reproducibility) |

Note: cache is a cross-cutting capability folded into convert/batch (not a dead standalone), exposing only `cache --stats`/`--clear`.
