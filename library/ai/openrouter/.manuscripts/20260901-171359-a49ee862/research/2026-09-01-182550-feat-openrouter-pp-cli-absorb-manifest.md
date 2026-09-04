# Absorb Manifest — openrouter-pp-cli (reprint 20260901-171359-a49ee862)

Reprint of run 20260509-165428 (press 4.2.0 → 4.31.4). Spec: official
https://openrouter.ai/openapi.json — OpenAPI 3.1, 77 paths, bearer auth
(inference key + separate management key for /keys). Prior CLI validated
live May–Aug 2026; 7 post-publish patches recorded (watch-list in brief,
headline: #825 mcp.transport [stdio,http] must be VERIFIED emitted; #829
spec example-token sanitization before publish).

## Absorbed features (covered by generation or framework)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | Fuzzy model search | jwill9999/openrouter-cli | framework `search "term" --type models` | offline FTS over synced catalog, --json |
| 2 | Model detail/info | MCP chat servers | (generated) models get /models/{author}/{slug} | typed, --select |
| 3 | Per-model provider endpoints | th3nolo MCP | (generated) models endpoints | feeds `endpoints failover` overlay |
| 4 | Key admin CRUD | maxxie114/openrouter-cli | (generated) keys list/create/delete | management-key auth documented |
| 5 | Credits balance | official SDKs | (generated) credits get | snapshotted into local store (runway input) |
| 6 | Key limit/status | official SDKs | (generated) key get | feeds `key eta` + `limits status` |
| 7 | Chat/completions passthrough | every competitor | (generated) chat completions | agent-native output, not a playground |
| 8 | Activity/analytics (new spec surface) | none (spec-only) | (generated) activity get, analytics query | joined by `usage reconcile` |
| 9 | Providers list | physics91 MCP | (generated) providers list | feeds `providers degraded` snapshot diff |

## Compound use cases

- Monday cost review: `usage cost-by --since 7d` → `usage anomaly` → `generation explain <id>` (attribution → alarm → forensics chain).
- Pre-dispatch gate for cron lineages: `budget check <cron>` (exit 0/8) + `limits status` + `providers degraded` before firing a drafting run.
- Friday reconciliation: `usage reconcile --since 7d` (mirror vs upstream trust root) → `credits runway` (402 leading indicator).
- Model shortlist ritual: `models query` DSL over synced catalog → `models churn --since 7d` (repricing watch) → `endpoints failover <model>`.

## Transcendence (only possible with our approach)

| # | Feature | Command | Buildability | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|-------------------------|------------------|
| 1 | Models query DSL (prior, kept) | models query | hand-code | Structured DSL→SQL over local FTS catalog; /models has no query params; offline, ~200 tokens vs 425KB | Use this command for structured capability/price/context filters over the synced catalog. Do NOT use it for fuzzy name lookup; use 'search "term" --type models' instead. |
| 2 | Cost-by-cron rollup (prior, kept) | usage cost-by | hand-code | Joins local generations with caller tags; upstream /activity groups by model+provider only | Use this command to attribute spend to crons/agents over a window. Do NOT use it to verify mirror-vs-upstream totals; use 'usage reconcile' instead. |
| 3 | Cost regression alarm (prior, kept) | usage anomaly | hand-code | Deterministic z-score over the local ledger; upstream's only alarm is absolute credits-low | Use this command to flag per-model cost spikes vs the trailing baseline. Do NOT use it to project credit depletion; use 'credits runway' instead. |
| 4 | Per-cron budget contract (prior, kept) | budget | hand-code | Local contract table + tagged generations + typed exit codes; upstream workspaces/budgets is org-level, never per-cron | Use this command to set and enforce per-cron caps pre-flight (exit 0 under, 8 over). Do NOT use it to project account-level depletion; use 'credits runway' or 'key eta' instead. |
| 5 | Providers degraded watch (prior, kept) | providers degraded | hand-code | Stateful set-diff vs local snapshot turns reactive 429s into a leading indicator | Use this command for the currently-degraded provider/model set. Do NOT use it to rank providers for a single model; use 'endpoints failover' instead. |
| 6 | Weekly-cap ETA (prior, kept) | key eta | hand-code | /key snapshot + local trailing 7-day burn → cap-trip date upstream never projects | Use this command to project when the key's spend cap trips. Do NOT use it for current free-tier daily headroom; use 'limits status' instead. |
| 7 | Endpoint failover map (prior, kept) | endpoints failover | hand-code | Ranks providers per model by status+price+observed local p50 | Use this command to rank providers serving one model for dispatch. Do NOT use it for the global degraded set; use 'providers degraded' instead. |
| 8 | Generation cost forensics (prior, kept) | generation explain | hand-code | /generation joined to local pricing for a cheapest-provider delta no raw dump shows | Use this command for one generation's cost anatomy (tokens, latency, provider delta). Do NOT use it for aggregate attribution; use 'usage cost-by' instead. |
| 9 | Free-tier and cap headroom (new) | limits status | hand-code | Derives free-tier quota tier + today's :free burn from /key + /credits + local ledger | Use this command for current headroom: free-tier daily quota and key-cap remaining. Do NOT use it to project when the cap trips; use 'key eta' instead. |
| 10 | Credits runway (new) | credits runway | hand-code | Projects days-to-zero from the local credits-snapshot series; /credits is a point value — the 402 leading indicator | Use this command to project when prepaid credits hit zero (402 risk). Do NOT use it to project the key cap trip (429 risk); use 'key eta' instead. |
| 11 | Ledger reconcile (new) | usage reconcile | hand-code | Diffs local mirror totals against the new /activity surface — the trust root for every other usage command | Use this command to verify local mirror totals against upstream activity. Do NOT use it to attribute spend to crons; use 'usage cost-by' instead. |
| 12 | Catalog churn watch (new) | models churn | hand-code | Temporal catalog diff (added/removed/repriced) from sync-kept history; no upstream history endpoint exists | Use this command to see catalog changes between syncs. Do NOT use it to filter the current catalog; use 'models query' instead. |

Scores (subagent rubric, 4-dim /10): rows 1-6 = 10; rows 7-9 = 9; row 10 = 9; row 11 = 8; row 12 = 7. All ≥ 5/10.

## Reprint reconciliation (Pass 2(d))

All 8 prior novel features KEPT (each re-scored 9-10/10 against current
personas; prior --llm flag maps to global --agent). prior-drop: none.
prior-reframe: none. P7 budget novelty re-verified against the NEW
workspaces/budgets spec surface: upstream budgets are org/member-level,
per-cron granularity remains local-only.

New-candidate cut: 8 → 4. Killed: workspace-budget drift (unconfirmed
management auth, org-level compare folded into P7's rationale), ledger
counterfactual (episodic use), benchmark-price shortlist (unverified
schema), daily usage report (thin wrapper over generated /activity).

## Build requirements (not features)

- MCP transport [stdio,http]: VERIFY emitted under 4.31.4 (patch #825 watch).
- Typed 402/429 exit codes + retry guidance in error envelope.
- Sync keeps catalog history (for `models churn`) + credits snapshots (for `credits runway`).
- Caller-tag log for `usage cost-by`/`budget` (local, free, no auth).
- Preserve models_query 13-test suite through regen-merge.
- Sanitize spec example bearer tokens before publish (patch #829).
