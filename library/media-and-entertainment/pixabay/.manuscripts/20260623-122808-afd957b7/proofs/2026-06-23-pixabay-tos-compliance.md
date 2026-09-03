# Pixabay API ToS Compliance Amendment

Full API access was granted with four stipulations. Each is now honored in the CLI.

| Stipulation | Implementation |
|---|---|
| **1. Credit Pixabay with a visible link when results are displayed** | `internal/cli/pixabay_attribution.go` + root.go `PersistentPostRunE` print "— Results from Pixabay (https://pixabay.com). Per Pixabay's API Terms, credit Pixabay with a visible link…" on human/TTY output of every result-displaying command (images/videos search+get, media search, similar, trends, contributors, search, sync). Silent on `--json`/`--agent`/`--quiet`/piped output so machine streams stay clean; every result also carries a per-item `pageURL` Pixabay link. Verified: fires in a pty, absent in pipes. |
| **2. Cache responses up to 24h** | Response cache TTL raised from 5 minutes to **24h** in `internal/client/client.go`. Local store framed as a 24h cache (matches the 24h URL-expiry); `pull` already re-resolves URLs older than 24h. |
| **3. User-triggered only; no background/automated requests** | Removed the `harvest` command (its `--auto-split` fired dozens of facet requests per invocation). Anti-triggers + a new "Pixabay API Compliance" section in README/SKILL forbid cron jobs, background daemons, and unattended automation. |
| **4. No AI/ML, dataset generation, or scraping** | Removed `harvest` (a bulk-corpus / dataset-generation tool). Added explicit anti-triggers and the compliance section prohibiting AI/ML training, dataset generation, and scraping. |

## Changes
- Deleted: `internal/cli/harvest.go`, `harvest_test.go`; removed `pixabayCategories` helper, root.go AddCommand wiring, and all harvest references across README/SKILL/root.go/which.go/mcp.
- Added: `internal/cli/pixabay_attribution.go` (hand-authored); root.go PostRun hook; 24h cache TTL; "Pixabay API Compliance" section in README + SKILL; 6 ToS-driven anti-triggers.
- research.json: harvest removed from novel_features/built; headline/value_prop reframed (24h cache, attribution); group "Beyond the 500-result cap" → "Search & acquisition".
- Recorded `.printing-press-patches/pixabay-tos-compliance.json`.

## Verification
- Novel features: 8 → **7** (pull, quota, media search, similar, trends, contributors, collection).
- shipcheck: **7/7 PASS**, scorecard holds at Grade A.
- go build/vet/test green; validate-narrative 10/10; attribution pty-verified; JSON output clean.
- Promoted to `$PRESS_LIBRARY/pixabay`; manuscripts synced.
