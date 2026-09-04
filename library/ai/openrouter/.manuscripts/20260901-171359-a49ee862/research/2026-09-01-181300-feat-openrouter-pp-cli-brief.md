# OpenRouter CLI Brief (reprint, redo-research 2026-09-01)

## API Identity
- Domain: unified LLM gateway — 400+ models, multi-provider routing, credits/billing, usage analytics
- Users: agent-fleet operators, LLM cost owners, model-selection engineers (the operator's estate is the reference persona: nightly cron lineages drafting via OpenRouter)
- Data profile: model catalog (static-ish, rich pricing/caps metadata), generations (append-only usage ledger), credits (counters), provider/endpoint status (volatile)

## Reachability Risk
- None: official spec at https://openrouter.ai/openapi.json (77 paths, OpenAPI 3.1); bearer auth; probe 200 on spec fetch. Prior print validated live (96/102 dogfood).

## Top Workflows (power-user)
1. Model selection: query catalog by capability/price/context (offline DSL → SQL)
2. Cost forensics: per-generation cost/latency vs cheapest-provider delta; spend by agent/cron tag
3. Budget governance: pre-flight cap checks (typed exit codes) for cron lineages; anomaly alarms
4. Provider health: degraded-endpoint watch, failover maps for router pre-emption
5. Account ops: credits, key limits/ETA, key management (admin API)

## Table Stakes (absorbed landscape)
- Fuzzy model search + model info (jwill9999, all MCP chat servers)
- Key management via admin API (maxxie114)
- Chat/completions passthrough (every competitor — covered by generated endpoint surface, NOT hand-built playground UX)
- Credits/limits display (official SDKs)

## Pain Points (2026 research)
- Per-provider/per-model rate limits opaque; 429s trip hardest on free models (20/min, 50/day; 1000/day post-$10) → limit-status/eligibility introspection
- 402/429 handling burden → typed exit codes + retry guidance in errors
- Agentic workloads → surprise costs (the transcendence layer's whole thesis)

## Data Layer
- Primary entities: models (+endpoints), generations, credits snapshots, providers
- Sync cursor: generations by created_at; models full-refresh
- FTS: model ids/names/descriptions

## User Vision
- Estate use: model catalog search/inspect, credits/usage queries, generation stats for drafting-lineage work; MCP tool surface (HTTP transport REQUIRED — prior machine silently dropped it, upstream #825)

## Prior Print (Pass 2(d) input)
- 8 novel features (all validated live May-Aug 2026): cost-by-cron, models query DSL, providers degraded watch, generation forensics, cost regression alarm, weekly-cap ETA, per-cron budget contract, endpoint failover map
- 7 patches watch-list: Go≥1.26.5 floor; manifest printer_name guard; mcp.transport must EMIT http (verify!); models_query 13-test suite must survive; spec example-token sanitization pre-publish (#829); transcendence = the product

## Spec Delta vs May
- NEW surfaces: guardrails (+assignments), workspaces (+budgets/members), presets, analytics/meta+query, benchmarks, activity, embeddings, images/videos/audio, BYOK, SCIM, observability destinations
- Estate-relevant new: activity, analytics, benchmarks, workspaces/budgets (org-level budget primitives — compare vs our per-cron contracts)

## Product Thesis
- Name: openrouter-pp-cli (reprint)
- Why: the ONLY ops/introspection CLI for OpenRouter — everything else wraps chat. For agent fleets, the governance layer (budgets, anomalies, failover) is the product; chat passthrough is free from the spec.

## Build Priorities
1. Preserve+revalidate the 8 transcendence commands on the 4.31.4 machine
2. Absorb the NEW spec surfaces (activity/analytics/benchmarks at minimum as generated endpoints)
3. MCP: Cloudflare pattern (77 paths > 50) + [stdio,http] transport VERIFIED EMITTED
4. Canonical OPENROUTER_API_KEY auth (+ management-key nuance for /keys admin surface)

## Reachability Gate
- Decision: PASS
- Probe: GET https://openrouter.ai/api/v1/models/count → 200, body {"data":{"count":419}} (2026-09-01)
