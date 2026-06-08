# Verto Online CLI Brief

## API Identity
- Domain: Geospatial / coordinate reference system transformation (Italy)
- Provider: IGM — Istituto Geografico Militare Italiano (igmi.esercito.difesa.it)
- Endpoint: `POST https://igmi.esercito.difesa.it/porta-magna/wps/volapi` (single endpoint, JSON in/out)
- Users: GIS technicians, surveyors (geometri/topografi), civil engineers, public-administration cartographers, geospatial developers in Italy
- Data profile: tiny + stateless. One transform endpoint; the only listable data is the 20-row supported-SRS table.

## Reachability Risk
- None. Verified live 2026-06-08: `info` and `conversione` requests return HTTP 200. No auth, no bot-protection. IGM explicitly encourages third-party clients ("Si incoraggia gli utilizzatori a sviluppare e divulgare client di qualsiasi tipo").

## API Contract (ground truth from official manual + live verification)
- Two request types, discriminated by body field `richiesta`:
  - `{"richiesta":"info"}` -> `{"maxCoord":32000,"srsSupportati":[{"epsg":4265,"descrizione":"Monte Mario"},...]}` (20 systems)
  - `{"richiesta":"conversione","utente":"x","chiave":"x","inEpsg":<EPSG>,"outEpsg":<EPSG>,"coordinate":[{"e":<east/lon>,"n":<north/lat>},...]}`
    -> `{"stato":"successo","coordinate":[{"e":...,"n":...},...]}`
    or error -> `{"stato":"errore","dove":"...","messaggio":"..."}`
- `utente` and `chiave` are MANDATORY but IGNORED (placeholders, not auth). DO NOT model as auth.
- `e` = est (easting OR longitude), `n` = nord (northing OR latitude). Geographic coords ALWAYS in decimal degrees (gradi sessadecimali).
- Max 32000 coordinates per request -> batch must chunk.
- Same-datum conversions unsupported (e.g. RDN2008 2D geo -> RDN2008/TM32 is actually a projection, works; truly same SRS in==out is the edge).
- Supported SRS (epsg): 4265,3003,3004,4806 (Roma40/Monte Mario), 4230,23032,23033,23034 (ED50), 4670,3064,3065,9716 (IGM95), 3035,3034 (ETRS89 LAEA/LCC), 6706,6707,6708,6709,7794,6876 (RDN2008).

## Ground-truth acceptance fixture
- inEpsg=4265 outEpsg=6706, input (e=12.4924, n=41.8902) -> output (e=12.4921961827, n=41.8908506304)
- inEpsg=3003 outEpsg=6707, input (e=2300000, n=4640000) -> output (e=1299961.341, n=4639997.131)

## Top Workflows
1. Convert one coordinate fast from the shell: `verto convert --from 4265 --to 6706 12.4924 41.8902`
2. Batch-convert a CSV file of coordinates (lat/lon or E/N columns) -> CSV/GeoJSON out. THE killer feature; IGM's own web clients only do files via browser.
3. List / search supported reference systems offline: `verto systems`, `verto systems --search ED50`
4. Pipe coordinates from other tools (stdin JSON/CSV -> stdout), jq-composable `--json`.
5. Convert geometries inside a GeoJSON FeatureCollection between systems.

## Table Stakes (vs adjacent tools)
- The only adjacent tool is PROJ/cs2cs (and QGIS, which uses PROJ). PROJ can do generic CRS math but reproducing IGM's OFFICIAL national grid-based transforms requires the licensed IGM grigliati (NTv2) which most users do not have installed/configured.
- IGM's own surface: browser-only file converters ("client" web pages). No CLI exists.
- So table stakes for us: single + batch convert, list systems, CSV I/O. We already beat the field by simply being a scriptable CLI over the authoritative source with zero grid-file setup.

## Data Layer
- Primary (and only) syncable entity: `systems` (EPSG + descrizione), 20 rows. Cache locally for offline `systems` list/search (FTS over descrizione).
- Conversion cache (transcendence): SQLite table keyed by (inEpsg,outEpsg,e,n) -> result, so repeat conversions are instant and offline.
- No other entities. Do NOT ship dead sync/sql/reconcile/stale over an empty store.

## Product Thesis
- Name: verto (binary `verto-pp-cli`, display "Verto Online")
- Headline: Official IGM coordinate transforms between every Italian reference system (Roma40, ED50, IGM95, ETRS89, RDN2008) from the command line — single or batch, CSV/GeoJSON, zero grid-file setup, offline cache, agent-native output.
- Why it should exist: the authoritative transform lives behind a browser form; there is no CLI, no batch-from-terminal, no scriptable pipeline. IGM invites clients. This is the obvious missing tool.
- DO NOT overclaim "PROJ can't do this" — anchor on zero-setup + official source + scriptable batch + offline.

## Build Priorities
1. Hand-written volapi client (`internal/volapi`): POST envelope `{richiesta,...}`, parse `srsSupportati` and `{stato,coordinate}`, adaptive rate limiting, 32000-coord chunking, typed errors from `stato:errore`.
2. `systems` (list/search, offline from store) + sync of the SRS table.
3. `convert` (single/few coords, positional + flags, axis-order aware) with `--json/--csv` and conversion cache.
4. `batch` (CSV in -> CSV/GeoJSON out, column mapping, chunking, progress) — the killer feature.
5. `geojson` (convert FeatureCollection geometries).
6. `roundtrip` (A->B->A residual error report) + `targets` (valid conversion targets for a given EPSG).

## Out-of-grid behavior (verified live 2026-06-08)
- Sea points WITHIN the IGM grid extent convert fine (tested Tyrrhenian + open Sardinian sea to lon 6.5).
- Points OUTSIDE the grid (e.g. Aegean lon 24, Atlantic lon -5) return: {"stato":"errore","dove":"Proj","messaggio":"Coordinate to transform falls outside grid"}.
- CLI must surface this as a dedicated typed error (distinct exit code + clear message). detect/preflight can warn before sending when coords are well outside Italy's grid bounds.
- All-or-nothing chunks: a SINGLE out-of-grid coordinate fails the ENTIRE conversione request (error `dove:"Proj"` does NOT name the index). batch needs `--skip-invalid` (bisection to isolate failures + convert the rest).
