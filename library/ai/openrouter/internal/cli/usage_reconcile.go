// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: usage reconcile — hand-built (reprint 2026-09-01).
// pp:data-source computed

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/store"
)

// reconcileToleranceUSD is the per-day disagreement below which upstream and
// the local mirror are considered in agreement (float accumulation noise).
const reconcileToleranceUSD = 0.000001

// newNovelUsageReconcileCmd diffs the local activity mirror's daily totals
// against a live /activity fetch. Disagreement means missed syncs or spend
// recorded against other keys/apps — either way, local analyses built on the
// mirror are not trustworthy until this is clean.
func newNovelUsageReconcileCmd(flags *rootFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:         "reconcile",
		Short:       "Verify the local usage mirror against upstream daily totals and flag days that disagree.",
		Example:     "  openrouter-pp-cli usage reconcile --since 7d --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "reconcile the local usage mirror against upstream activity")
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), `{"days":0,"mismatches":0,"verdict":"n/a"}`)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			raw, err := c.Get(cmd.Context(), "/activity", nil)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			var envelope struct {
				Data []map[string]any `json:"data"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				return apiErr(fmt.Errorf("parse /activity response: %w", err))
			}

			sinceT, err := parseSinceDuration(since)
			if err != nil {
				return usageErr(err)
			}
			cutoff := sinceT.UTC().Format("2006-01-02")
			upstream := map[string]float64{}
			upstreamReq := map[string]int64{}
			for _, row := range envelope.Data {
				date := asString(row["date"])
				if len(date) >= 10 {
					date = date[:10]
				}
				if date == "" || date < cutoff {
					continue
				}
				upstream[date] += asFloat(row["usage"])
				upstreamReq[date] += int64(asFloat(row["requests"]))
			}

			local := map[string]float64{}
			localReq := map[string]int64{}
			dbPath := defaultDBPath("openrouter-pp-cli")
			db, err := store.OpenWithContext(context.Background(), dbPath)
			if err != nil {
				return configErr(fmt.Errorf("open local store: %w", err))
			}
			defer db.Close()
			rows, err := db.DB().QueryContext(cmd.Context(),
				`SELECT substr(date,1,10), COALESCE(SUM(usage),0), COALESCE(SUM(requests),0) FROM activity WHERE substr(date,1,10) >= ? GROUP BY substr(date,1,10)`, cutoff)
			if err != nil {
				return configErr(fmt.Errorf("query local activity: %w", err))
			}
			defer rows.Close()
			for rows.Next() {
				var d string
				var u float64
				var r int64
				if err := rows.Scan(&d, &u, &r); err != nil {
					return configErr(err)
				}
				local[d] = u
				localReq[d] = r
			}
			if err := rows.Err(); err != nil {
				return configErr(err)
			}

			allDates := map[string]bool{}
			for d := range upstream {
				allDates[d] = true
			}
			for d := range local {
				allDates[d] = true
			}
			dates := make([]string, 0, len(allDates))
			for d := range allDates {
				dates = append(dates, d)
			}
			sort.Strings(dates)

			type dayDiff struct {
				Date         string  `json:"date"`
				UpstreamUSD  float64 `json:"upstream_usd"`
				LocalUSD     float64 `json:"local_usd"`
				DeltaUSD     float64 `json:"delta_usd"`
				UpstreamReqs int64   `json:"upstream_requests"`
				LocalReqs    int64   `json:"local_requests"`
				Match        bool    `json:"match"`
			}
			diffs := make([]dayDiff, 0, len(dates))
			mismatches := 0
			for _, d := range dates {
				delta := upstream[d] - local[d]
				match := math.Abs(delta) <= reconcileToleranceUSD && upstreamReq[d] == localReq[d]
				if !match {
					mismatches++
				}
				diffs = append(diffs, dayDiff{
					Date: d, UpstreamUSD: upstream[d], LocalUSD: local[d], DeltaUSD: delta,
					UpstreamReqs: upstreamReq[d], LocalReqs: localReq[d], Match: match,
				})
			}
			verdict := "clean"
			if mismatches > 0 {
				verdict = "drift"
			}
			if len(dates) == 0 {
				verdict = "no-data"
			}

			result := map[string]any{
				"since":      since,
				"days":       len(dates),
				"mismatches": mismatches,
				"verdict":    verdict,
				"daily":      diffs,
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			for _, dd := range diffs {
				mark := "ok"
				if !dd.Match {
					mark = "MISMATCH"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s  upstream=$%.6f local=$%.6f delta=$%.6f reqs=%d/%d  %s\n",
					dd.Date, dd.UpstreamUSD, dd.LocalUSD, dd.DeltaUSD, dd.UpstreamReqs, dd.LocalReqs, mark)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "verdict: %s (%d/%d days mismatched)\n", verdict, mismatches, len(dates))
			if verdict == "drift" {
				fmt.Fprintln(cmd.OutOrStdout(), "hint: run 'openrouter-pp-cli sync --resources activity' then re-check; residual drift means spend from other keys/apps")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "7d", "Window to reconcile (e.g. 7d, 24h)")
	return cmd
}
