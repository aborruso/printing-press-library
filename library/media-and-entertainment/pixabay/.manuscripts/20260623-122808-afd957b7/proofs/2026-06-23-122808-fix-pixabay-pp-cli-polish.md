# Pixabay CLI Polish

## Result: ship (further_polish_recommended: no)
| Metric | Before | After |
|--------|--------|-------|
| Scorecard | 91/100 | 91/100 |
| Verify | 75% | 75% (leaf commands 100%; 2/10 entries are RunE-less parent groups) |
| Dogfood | PASS | PASS (1 non-blocking dead generated-helper advisory) |
| Go vet | 0 | 0 |
| Gosec (hand-authored) | 8 | 0 |
| Tools-audit | 0 pending | 0 pending |
| PII-audit | 0 pending | 0 pending |

## Fixes applied
- Resolved all 8 gosec findings in hand-authored internal/cli/pull.go:
  - G301: output dir perms 0o750
  - G306 (x2): credit-sidecar WriteFile perms 0o600
  - G304: narrow #nosec with rationale on download temp-file create (path = user --out + sanitized base)
  - G104 (x4): explicit handling of f.Close()/os.Remove() cleanup errors

## Skipped (structural / generator-owned retro candidates)
- verify 2/10 on images/media/videos/profile/workflow: RunE-less Cobra parent groups (execute:false correct); every leaf passes.
- dead replacePathParam in generator-emitted helpers.go (DO NOT EDIT) — generator retro candidate.
- 26 remaining gosec findings all in generator-emitted files — retro candidates.
- MCP Token Efficiency 7/10, Breadth 7/10: 2-endpoint API; thresholds calibrated for large APIs.
- Cache Freshness 5/10: intentional (rate-limited API); freshness tracked via synced_at + sync hints.

Phase 3 gate: 8/8 transcendence rows planned/built; not a sub-60 reprint — no forced hold.
