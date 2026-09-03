# Phase 4.95 Local Code Review — findings

Reviewer: pr-review-toolkit:code-reviewer (in-scope: 9 hand-authored novel files only).
go build/vet clean; SQL %q injection vectors all safe (kind whitelisted via collectionKind/contributorKinds); pull goroutines mutex-guarded; media_search disjoint-index writes race-free; safeFileBase applied at all filename sites; Jaccard math correct.

## Autofixed in-place (3 findings, 1 round)
1. [error] quota.go credential leak — transport error (*url.Error) embedded key=<secret> in URL and classifyAPIError does not mask. FIXED: added scrubSecret() to redact the key before classify.
2. [warning] pull.go unbounded download — io.Copy had no size cap. FIXED: io.LimitReader cap at 1 GiB (maxDownloadBytes) with oversize rejection.
3. [warning] pull.go --query fetched live in verify mode — resolvePullTargets ran before IsVerifyEnv. FIXED: moved IsVerifyEnv short-circuit above target resolution.

## Clean files
pixabay_shared.go, media.go, similar.go, trends.go, contributors.go, media_search.go, harvest.go — no findings.

Convergence: all in-scope findings cleared in round 1.
