# Printing Press Retro: Pixabay

## Session Stats
- API: pixabay
- Spec source: hand-authored internal YAML (from official Pixabay API docs)
- Scorecard: 91/100 (Grade A)
- Verify pass rate: ~75% reported (leaf commands 100%; RunE-less parent groups score 2/10 — structural)
- Fix loops: 2 shipcheck-adjacent; 1 doc-error regen; 1 ToS-compliance amendment
- Manual code edits: 8 novel command files hand-authored; later 1 removed (harvest); attribution + cache + annotation hand-edits
- Features built from scratch: 8 novel commands (later 7 after ToS-driven harvest removal)

## Findings

### 1. validate-narrative does not check narrative *prose* for command references (scorer bug / coverage gap)
- **What happened:** Two phantom commands shipped into README/SKILL — `config set-key` (real command is `auth set-token`) in `auth_narrative` + a `troubleshoots` entry, and `sql` ("query with `sql`") in `value_prop`. Neither command exists on the binary.
- **Scorer correct?** No — coverage gap. `validate-narrative --strict --full-examples` passed 10/10 because it only walks `narrative.quickstart[]` and `narrative.recipes[]` command paths. The phantom commands lived in prose fields (`auth_narrative`, `value_prop`, `troubleshoots`) which the validator never inspects. They were caught only by the Phase 4.9 agentic README/SKILL audit (an LLM review — probabilistic and expensive).
- **Root cause:** scorer (`validate-narrative`) — command-reference validation scope is limited to the two structured example arrays; prose narrative fields are unchecked.
- **Cross-API check:** Universal. Every printed CLI has narrative prose (auth story, value prop, troubleshooting). Any backtick-wrapped `<cli> <command>` reference in prose can name a non-existent command and ship unchecked.
- **Frequency:** every API (the narrative prose fields are standard for all CLIs).
- **Fallback if the Printing Press doesn't fix it:** the Phase 4.9 agentic doc review *may* catch it, but it's LLM-judgment, not deterministic. Phantom commands in prose are easy authoring errors (especially auth-setup commands, which vary by auth type).
- **Worth a Printing Press fix?** Yes. `validate-narrative` already does exactly this check for quickstart/recipes; extending it to scan prose fields for `` `<cli> <words>` `` patterns and resolve them via `<binary> <words> --help` is a small, consistent expansion that turns a probabilistic catch into a deterministic one.
- **Inherent or fixable:** fixable.
- **Durable fix:** in `validate-narrative`, additionally scan `narrative.auth_narrative`, `value_prop`, `headline`, `when_to_use`, and each `troubleshoots[].fix` for backtick-fenced tokens of the form `<cli-name> <subcommand...>`, strip flags/args, and verify each resolves with `--help` (the same resolution path already used for quickstart). Report unresolved references as failures (or warnings under non-strict). Guard: only treat a backtick token as a command candidate when it starts with the CLI binary name, to avoid flagging prose that merely mentions a flag or value.
- **Test:** positive — a research.json with `auth_narrative: "...run \`<cli> config set-key <key>\`"` against a binary lacking `config` fails validate-narrative. Negative — prose mentioning `` `--json` `` or a real command passes.
- **Evidence:** This session — `validate-narrative` reported "10 narrative commands resolved" while `config set-key` and `sql` sat in README/SKILL until the Phase 4.9 agent flagged them; fix required a regen.
- **Related prior retros:** None (0 retros on file).

### 2. No re-sync of narrative-derived description surfaces after post-generation research.json edits (scorer / recurring friction)
- **What happened:** During the ToS-compliance amendment I reframed `research.json` `headline`/`value_prop` (dropping the removed `harvest` framing), then ran `dogfood`. dogfood re-synced the `novel_features` blocks but **not** the description. The stale headline ("…bulk download that beats the 500-result cap…") remained in the manifest `description` and ~10 derived surfaces (root.go `Short`/`Long`, SKILL frontmatter, `.goreleaser.yaml`, `agent_context.go`, `internal/mcp/tools.go`, `manifest.json`, `tools-manifest.json`, `agentcookie.toml`) and nearly shipped to the **public registry** (where `description` is the display field). Manual multi-surface editing missed two spots on the first pass (root.go's *truncated* `Short`/`Long`, and SKILL `when_to_use`).
- **Scorer correct?** N/A (no penalty) — this is a missing-capability / silent-drift gap, not a false penalty.
- **Root cause:** scorer (`dogfood` sync) — `dogfood` re-derives and rewrites the `novel_features`-driven blocks (README "Unique Features", SKILL "Unique Capabilities", root.go Highlights, `.printing-press.json`) from `novel_features_built`, but has no equivalent re-sync for the narrative *description* that feeds `root.go` Short/Long, SKILL `description:`, `.goreleaser` brews, `agent_context`, `mcp/tools`, and the manifest `description`. There is also no validation that manifest `description` matches the research.json-derived headline.
- **Cross-API check:** Universal for any post-generation narrative edit. First-class workflows that edit narrative after generation: **reprints** (re-derive narrative under the current binary), **compliance/scope amendments** (this session), and description fixes. The failure is silent and lands on the public registry display field.
- **Frequency:** subclass — runs that edit `research.json` narrative after the initial `generate` (reprints + amendments). High-visibility when it occurs (public registry).
- **Fallback if the Printing Press doesn't fix it:** full `generate --force` regenerates the description surfaces but clobbers hand-edits to generated files (attribution PostRun, cache TTL, annotations in this run), so the agent is pushed to hand-edit ~10 surfaces — error-prone (missed truncated Short/Long + when_to_use here).
- **Worth a Printing Press fix?** Yes. The `dogfood`-sync precedent (it already rewrites novel_features blocks from research.json) makes extending it to the description a natural, low-risk floor-raise.
- **Inherent or fixable:** fixable.
- **Durable fix (two candidates; primary first):**
  - (a) Extend `dogfood`'s existing research.json→surfaces sync to also re-derive the description-driven surfaces (root.go Short/Long, SKILL `description:`, goreleaser brews, agent_context, mcp tools description, manifest `description`) from `research.json` `narrative.headline`/`value_prop`, the same way it already syncs novel_features blocks. Handles the truncation rules centrally so agents never hand-edit truncated Short.
  - (b) Add a `dogfood`/`shipcheck` check that flags drift between the manifest `description` (and root.go Short) and the research.json-derived headline, so stale descriptions fail loudly instead of shipping.
  - Disambiguate by whether maintainers prefer auto-rewrite (a, fixes silently) or fail-loud (b, forces the agent to re-derive). (a) is the stronger floor-raise; (b) is the cheaper safety net.
- **Test:** positive — edit research.json `headline`, run dogfood, assert manifest `description` + root.go Short updated (a) or a drift failure is reported (b). Negative — no narrative edit ⇒ no change / no false drift report.
- **Evidence:** This session — stale "500-result cap" description was still in 10 surfaces after the dogfood re-sync; caught only during the publish PR body build, requiring a second round of perl edits across staged + library + working copies.
- **Related prior retros:** None.

### 3. Generated search/get endpoint-mirror commands aren't auto-annotated for the dogfood error-path probe (generator / missing scaffolding)
- **What happened:** `dogfood --live` error-path probes pass `__printing_press_invalid__` and expect a non-zero exit. Five commands failed it — `images search`, `videos search`, `media search`, `similar`, `collection show` — even though exit 0 is *correct*: for a search endpoint any string is a valid query (Pixabay returns HTTP 200 + empty `hits`), and for a local not-found lookup empty is the right answer. Fix required hand-annotating `pp:no-error-path-probe` on all five, including two **generator-emitted** files (`images_search.go`, `videos_search.go`) — which then had to be recorded as a regen-clobber patch.
- **Scorer correct?** Partially — the probe's assumption ("invalid input ⇒ non-zero exit") is wrong for read/search commands where the upstream returns 200+empty for unknown input. The `pp:no-error-path-probe` annotation is the documented escape valve, but for *generator-emitted* commands the agent must hand-edit DO-NOT-EDIT files.
- **Root cause:** generator (endpoint-mirror command emission) — for a GET endpoint-mirror command whose error-path target is a free-text param (no enum/format constraint that would itself error), the generator knows the synthesized invalid value cannot reliably produce a non-zero exit, yet emits no `pp:no-error-path-probe` annotation. (Secondary: `dogfood` could treat "exit 0 + valid-empty result" as acceptable for read endpoint commands, but it can't know upstream behavior offline.)
- **Cross-API check:** Search/list/get-by-id is one of the most common command shapes. Any GET endpoint-mirror with a free-text positional query trips this.
- **Frequency:** subclass:search-endpoint commands. Named with evidence: **pixabay** (this run — `images search`/`videos search`, free-text `q`), **nasa-images** (NASA Image and Video Library — registry entry advertises a search endpoint over a free-text query). Two named with direct evidence + the structural argument (every endpoint-mirror GET with a free-text param) — hence P3, not P2.
- **Fallback if the Printing Press doesn't fix it:** agent hand-annotates generated files each time and records a patch; easy to forget, and the patch is regen-clobber surface.
- **Worth a Printing Press fix?** Yes, modestly — the generator emits the command and knows its shape, so absorbing the annotation removes per-CLI friction on a very common command class.
- **Inherent or fixable:** fixable (with a guard).
- **Durable fix:** when emitting an endpoint-mirror GET command whose only "invalid-input" surface is a free-text param (no enum/format-validated required flag/positional that the probe's synthesized value would itself reject), set `Annotations["pp:no-error-path-probe"]="true"`. Guard: do NOT set it when the command has a required enum/format-validated input that legitimately errors on bad values, so genuine input-validation error paths still get probed.
- **Test:** positive — a generated GET search command (free-text positional) ships with `pp:no-error-path-probe` and dogfood skips its error-path probe. Negative — a command with a required enum flag still gets error-path-probed and must error on a bad enum value.
- **Evidence:** This session — 5 commands flagged "expected non-zero exit for invalid argument"; resolution required editing `images_search.go`/`videos_search.go` (generated) and recording `.printing-press-patches/pixabay-dogfood-annotations.json`.
- **Related prior retros:** None.

## Prioritized Improvements

### P2 — Medium priority
| Finding | Title | Component | Frequency | Fallback Reliability | Complexity | Guards |
|---------|-------|-----------|-----------|---------------------|------------|--------|
| F1 | validate-narrative: check prose command refs | scorer | every API | Phase 4.9 LLM review (probabilistic) | small | only flag backtick tokens starting with binary name |
| F2 | Re-sync (or drift-check) description surfaces after narrative edits | scorer | subclass: reprints + amendments | manual 10-surface edit (missed 2 here) | medium | n/a |

### P3 — Low priority
| Finding | Title | Component | Frequency | Fallback Reliability | Complexity | Guards |
|---------|-------|-----------|-----------|---------------------|------------|--------|
| F3 | Auto-annotate search-endpoint commands for error-path probe | generator | subclass: search-endpoint commands | agent hand-annotates generated files + patch | small | skip when required enum/format input would itself error |

### Skip
| Finding | Title | Why it didn't make it |
|---------|-------|------------------------|
| F4 | Response-cache TTL not spec-configurable (hardcoded 5min; Pixabay needed 24h per ToS + URL expiry) | Step B: only pixabay named with concrete evidence; others (Unsplash/Pexels) speculative. 5min is a reasonable default for most APIs. |
| F5 | `lock promote` looks for phase5 acceptance marker under a different scope hash than the run wrote it (`pixabay-pp-cli-e5157619` vs `pixabay-2ddf1c6e`) | Single occurrence, surfaced only on a *re*-promote of the same run; Step B can't name 3 APIs. Real deterministic bug — worth a targeted report if it recurs, but too narrow for a retro machine-change now. |

### Dropped at triage
| Candidate | One-liner | Drop reason |
|-----------|-----------|-------------|
| harvest "color" facet wording | Described "category/color/order" but built category/order | printed-CLI (authoring detail) |
| `config set-key` authoring slip | I wrote a non-existent command in research.json | iteration-noise (systemic finding is F1) |
| `pull` happy-args fixture | dogfood couldn't synthesize a valid `pull` invocation (needs state) | API-quirk / works-as-designed (`pp:happy-args` is the mechanism) |
| Attribution PostRun hook | Visible-credit compliance code | printed-CLI / SKILL-recipe (most APIs don't require attribution) |
| Positional-as-query handling | Probed whether non-path positional is sent as query param | not a finding — generator handled it correctly (`params["q"]=args[0]`) |

## Work Units

### WU-1: validate-narrative checks prose command references (from F1)
- **Priority:** P2
- **Component:** scorer
- **Goal:** Phantom commands referenced in narrative prose fail validate-narrative deterministically, not just via the Phase 4.9 LLM review.
- **Target:** `validate-narrative` command (`internal/narrativecheck/` and the validate-narrative entrypoint).
- **Acceptance criteria:**
  - positive: research.json whose `auth_narrative`/`value_prop`/`troubleshoots[].fix` references `` `<cli> <missing-command>` `` fails `validate-narrative --strict`.
  - negative: prose mentioning real commands or bare `--flags`/values passes.
- **Scope boundary:** Only validates backtick-fenced tokens beginning with the CLI binary name; does not parse free prose for implied commands.
- **Dependencies:** none.
- **Complexity:** small.

### WU-2: Re-sync or drift-check description surfaces from research.json narrative (from F2)
- **Priority:** P2
- **Component:** scorer
- **Goal:** After a post-generation narrative edit, the description-derived surfaces (manifest `description`, root.go Short/Long, SKILL `description:`, goreleaser brews, agent_context, mcp tools) either auto-update or fail loudly, instead of silently shipping a stale description to the public registry.
- **Target:** `dogfood` sync logic (the same code that re-derives novel_features blocks) and/or a `shipcheck` drift check.
- **Acceptance criteria:**
  - positive (option a): edit `research.json` headline, run dogfood, assert manifest `description` + root.go Short are updated (truncation handled centrally).
  - positive (option b): same edit without re-sync produces a reported drift failure.
  - negative: no narrative edit ⇒ no rewrite and no false drift report.
- **Scope boundary:** description/headline/value_prop surfaces only; does not touch novel_features sync (already handled).
- **Dependencies:** none.
- **Complexity:** medium.

### WU-3: Auto-annotate free-text search-endpoint commands for the error-path probe (from F3)
- **Priority:** P3
- **Component:** generator
- **Goal:** Generated GET endpoint-mirror commands whose invalid-input surface is a free-text param ship with `pp:no-error-path-probe`, so dogfood doesn't false-fail search/get commands and agents don't hand-edit generated files.
- **Target:** endpoint-mirror command emission in `internal/generator/`.
- **Acceptance criteria:**
  - positive: a generated GET search command (free-text positional, no enum-required input) is emitted with `pp:no-error-path-probe="true"`.
  - negative: a generated command with a required enum/format-validated input is NOT annotated and still gets error-path-probed.
- **Scope boundary:** GET endpoint-mirror commands only; does not change hand-authored novel commands or the annotation's runtime semantics.
- **Dependencies:** none.
- **Complexity:** small.

## Anti-patterns
- Editing `research.json` narrative after generation and assuming `dogfood` re-syncs everything — it only syncs the novel_features blocks, not the description.
- Relying on the Phase 4.9 agentic doc review to catch phantom command references that a mechanical `validate-narrative` extension could catch deterministically.
- Hand-annotating generated (DO-NOT-EDIT) files for predictable, structural cases (search error-path), creating regen-clobber patch surface.

## What the Printing Press Got Right
- **Query-param auth + non-path positional → query param** worked out of the box: the generator wired `q.Set("key", ...)` for `auth.in: query` and emitted `params["q"]=args[0]` for a positional not present in the path. No hand-fixes needed for either Pixabay quirk.
- **regen-merge preserved all 15 hand-authored files** across two `generate --force` passes — the hand-edit durability contract held exactly as documented.
- **The full review stack earned its keep:** code review caught a real API-key leak in the quota error path; the agentic output review seeded a store to verify ranking logic; shipcheck's dogfood+verify+scorecard gave an honest Grade-A signal. The machine's structural gates were sound; the gaps found here are coverage edges, not core failures.
