# ListingView Novel-Features Brainstorm (audit trail)

## Customer model

**1. Priya — print-on-demand niche validator.** Decides whether a niche is worth 20 mockups + 13 listing slots. Today: types term into Search Term Analyzer, eyeballs volume/competition, scrolls best-sellers one at a time. Weekly: brainstorms 5-10 niche ideas Sunday night. Frustration: 50-uses/mo free quota burns out mid-month; point-in-time estimates can't tell a rising niche from a cooling one; no batch.

**2. Marcus — catalog SEO optimizer.** 80-listing shop, quarterly tag audits. Today: opens Listing Explorer per listing, copies each of 13 tags into Tag Analyzer by hand. Weekly: swaps weakest tags on low-traffic listings. Frustration: no cross-listing view; can't see which weak tag he repeated across 40 listings; web UI strictly one-at-a-time.

**3. Dana — competitor & seasonal-trend watcher.** Tracks 3-5 rival shops + seasonal keyword watchlist. Today: re-opens Shop Analyzer weekly, hand-copies numbers to a spreadsheet to spot change. Weekly: re-pulls each rival's top listings. Frustration: web UI + watchlist show only "now" — no diff, so she misses the week a rival launches a bestseller or a keyword climbs.

**4. Sam — data-driven / agent-building seller.** Wants to script research across hundreds of terms and pipe to an LLM. Today: copy-pastes from the extension; no API exists. Frustration: extension-only, device-locked, quota-limited, no programmatic surface.

## Survivors (7, all hand-code, all >=7/10) — see manifest transcendence table

## Killed candidates
| Feature | Kill reason |
|---|---|
| catalog audit (standalone) | folded into `listings audit --shop` |
| saturation (standalone) | absorbed as winnability sub-signal of `niche` |
| trends calendar | thin until store accrues history; covered by `drift` |
| competitor teardown | pure wrapper over shops analyze + shops listings |
| keywords cluster | semantic grouping = LLM dependency; mechanical version too thin/unverifiable |
| pricing | wrapper; price breakdown already in keyword analyzer; folded into `niche` |
| tags gap (two-listing) | redundant; folded into `gaps` / `listings audit` |
