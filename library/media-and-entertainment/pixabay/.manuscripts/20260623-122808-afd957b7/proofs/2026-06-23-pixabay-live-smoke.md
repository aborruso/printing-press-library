# Pixabay CLI — Live Smoke Test (full API access)

Run with a real PIXABAY_API_KEY (full access) sourced from the operator's shell; the key value never appears in any artifact (verified by exact-value scan).

## Result: PASS — 61/61, level full, 0 failures
- `doctor`: Auth configured, API reachable, auth_source env:PIXABAY_API_KEY.
- Live searches (images/videos/media) return real results; `get` by id works.
- `quota` reads real X-RateLimit-* headers from the live API.
- `sync` hydrates the local store from real responses.
- `pull` downloads a real asset (curtailed to 1 under dogfood) with attribution sidecar.
- Local analytics (similar, trends, contributors, collection) operate over the synced cache.

## Probe annotations applied (made dogfood clean)
- `pp:no-error-path-probe` on images search, videos search, media search, similar, collection show — `__printing_press_invalid__` is a valid empty search / not-found lookup (exit 0 is correct), not an error.
- `pp:happy-args=--query=nature;--workers=1` on pull — supplies a real live target the synthesizer can't guess.

## Compliance behaviors confirmed live
- Attribution credit prints on human/TTY output; absent on --json/piped (verified separately via pty).
- Response cache TTL = 24h; store framed as a 24h cache.
- No harvest/dataset-generation path; usage user-triggered.

Secret scan: no API key value or fragment in any proof or library artifact.
