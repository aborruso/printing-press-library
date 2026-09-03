# ListingView CLI Shipcheck

## Leg results (shipcheck umbrella)
| Leg | Result |
|-----|--------|
| verify | FAIL (cookie-auth mock limitation — see below) |
| validate-narrative | PASS |
| dogfood (structural) | PASS |
| workflow-verify | workflow-pass |
| apify-audit | PASS |
| verify-skill | PASS |
| scorecard | PASS — **82/100 Grade A** |

6/7 legs pass. Standalone `verify` (no --spec) exits 0.

## Scorecard 82/100 (Grade A)
README 10, Doctor 10, Agent-Native 10, Local Cache 10, Vision 10, Workflows 10, Insight 10, Breadth 9, Agent Workflow 9, MCP Quality 8, MCP Remote Transport 10, MCP Token-Efficiency 7, MCP Tool Design 7, Cache Freshness 5, Path Validity 10, Sync Correctness 10, Type Fidelity 4/5, Dead Code 4/5.
- **auth_protocol 2/10**: inherent to cookie-session auth — the scorecard cannot verify auth without a live browser session. Cleared by the live proof below.

## verify FAIL is the documented cookie-auth `unverified-needs-auth` state
The spec declares `auth.type: cookie` with `requires_browser_session: true`. Mock-mode `verify --spec` runs a `browser-session-proof` check that cannot pass without a live session (the runner sandboxes HOME with no cookie jar), and live-auth commands (niche/gaps/keywords/listings) can't EXEC against the mock. Per the ship threshold exception, this is a hold ONLY until `auth login --chrome` + `doctor` + a read-only browser-session proof pass against the real site. **All three are green:**
- `auth login --chrome`: imported 4 cookies for `.app.listingview.io`.
- `doctor`: "Auth: configured (browser session)", "Browser Session Proof: valid", "API: reachable", "Credentials: valid".
- **Phase 5 live dogfood: 91/91 PASS** (status:pass) with the session injected.

## Auth fix (generator bug)
`client.go` default cookie handling wrapped the whole cookie jar in a single cookie named "Cookie" (`AddCookie{Name:"Cookie"}`), which the backend 401'd. Replaced with `applyListingViewAuth` (hand-authored `internal/client/listingview_csrf.go`): Cookie header + cookie-derived `X-CSRF-Token` + optional `shopid`. RETRO CANDIDATE.

## sync fix
Generic syncer GET `list-favourite` without the required `type` discriminator → 400. Injected `type=listing` via `listingViewSyncRequiredParams` (hand-authored). sync now completes.

## Local code review (Phase 4.95)
Reviewed all 9 hand-authored files: **clean** — no SQL injection (parameterized), no nil derefs, no concurrency bugs, no credential leakage, division guarded everywhere. Ship-ready.

## Verdict: ship (cookie-auth, live-proven)
The CLI is fully functional against the real ListingView API. The single failing shipcheck leg (verify) is the inherent cookie-auth mock limitation, explicitly cleared by the live browser-session proof.
