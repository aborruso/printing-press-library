// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: limits status — hand-built (reprint 2026-09-01).
// pp:data-source computed

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/store"
)

// Free-variant daily quota tiers per OpenRouter's published limits: accounts
// with a lifetime purchase of at least $10 get the raised free-model daily
// quota; below that the base quota applies. Values are documented behavior as
// of 2026-09, kept here as named constants so a quota change is a one-line fix.
const (
	freeTierPurchaseThresholdUSD = 10.0
	freeTierDailyQuotaBase       = 50
	freeTierDailyQuotaRaised     = 1000
)

// newNovelLimitsStatusCmd derives current headroom from three sources at once:
// the key's cap status (/key), the account's purchase tier (/credits), and the
// local activity mirror's count of today's free-variant requests. No single
// upstream endpoint answers "how much room do I have right now".
func newNovelLimitsStatusCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "status",
		Short:       "One view of current headroom: key-cap remaining, free-tier daily quota, and today's free-model burn.",
		Example:     "  openrouter-pp-cli limits status --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "derive current cap and free-tier headroom")
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), `{"cap_remaining":0,"free_daily_quota":0,"free_requests_today":0}`)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			unwrap := func(raw []byte) (map[string]any, error) {
				var envelope struct {
					Data map[string]any `json:"data"`
				}
				out := map[string]any{}
				if json.Unmarshal(raw, &envelope) == nil && envelope.Data != nil {
					return envelope.Data, nil
				}
				if err := json.Unmarshal(raw, &out); err != nil {
					return nil, err
				}
				return out, nil
			}

			keyRaw, err := c.Get(cmd.Context(), "/key", nil)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			key, err := unwrap(keyRaw)
			if err != nil {
				return apiErr(err)
			}
			creditsRaw, err := c.Get(cmd.Context(), "/credits", nil)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			credits, err := unwrap(creditsRaw)
			if err != nil {
				return apiErr(err)
			}

			limit := asFloat(key["limit"])
			usage := asFloat(key["usage"])
			capRemaining := limit - usage
			if v, ok := key["limit_remaining"]; ok {
				capRemaining = asFloat(v)
			}
			isFreeTier := false
			if v, ok := key["is_free_tier"]; ok {
				if b, ok := v.(bool); ok {
					isFreeTier = b
				}
			}
			totalPurchased := asFloat(credits["total_credits"])
			freeDailyQuota := freeTierDailyQuotaBase
			if totalPurchased >= freeTierPurchaseThresholdUSD {
				freeDailyQuota = freeTierDailyQuotaRaised
			}

			// Today's free-variant burn from the local activity mirror. Zero rows
			// simply means no synced free-model traffic today — still an answer.
			freeRequestsToday := int64(0)
			dbPath := defaultDBPath("openrouter-pp-cli")
			if db, err := store.OpenWithContext(context.Background(), dbPath); err == nil {
				defer db.Close()
				today := time.Now().UTC().Format("2006-01-02")
				row := db.DB().QueryRowContext(cmd.Context(),
					`SELECT COALESCE(SUM(requests),0) FROM activity WHERE date = ? AND (model LIKE '%:free' OR model_permaslug LIKE '%:free')`, today)
				_ = row.Scan(&freeRequestsToday)
			}

			result := map[string]any{
				"cap_limit_usd":       limit,
				"cap_used_usd":        usage,
				"cap_remaining_usd":   capRemaining,
				"is_free_tier":        isFreeTier,
				"total_purchased_usd": totalPurchased,
				"free_daily_quota":    freeDailyQuota,
				"free_requests_today": freeRequestsToday,
				"free_quota_headroom": int64(freeDailyQuota) - freeRequestsToday,
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			if limit > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "key cap: $%.2f remaining of $%.2f\n", capRemaining, limit)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "key cap: unlimited")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "free tier: %v (lifetime purchased $%.2f → %d free-model requests/day)\n", isFreeTier, totalPurchased, freeDailyQuota)
			fmt.Fprintf(cmd.OutOrStdout(), "free burn today: %d of %d (synced view; run sync for freshness)\n", freeRequestsToday, freeDailyQuota)
			return nil
		},
	}
	return cmd
}
