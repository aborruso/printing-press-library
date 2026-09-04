// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: credits runway — hand-built (reprint 2026-09-01).
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

// newNovelCreditsRunwayCmd projects days-to-zero for prepaid credits from the
// locally accumulated credits-snapshot series. Each invocation appends the
// current upstream balance to credits_snapshots, so the projection sharpens
// with use; /credits alone is a point value and cannot answer "when do we hit
// 402".
func newNovelCreditsRunwayCmd(flags *rootFlags) *cobra.Command {
	var windowDays int
	cmd := &cobra.Command{
		Use:         "runway",
		Short:       "Project days-to-zero for prepaid credits at the trailing burn rate — the 402 leading indicator.",
		Example:     "  openrouter-pp-cli credits runway --agent",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "project credits runway from the local snapshot series")
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), `{"balance":0,"burn_per_day":0,"days_to_zero":"n/a"}`)
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get(cmd.Context(), "/credits", nil)
			if err != nil {
				return classifyAPIError(cmd.OutOrStdout(), err, flags)
			}
			var envelope struct {
				Data map[string]any `json:"data"`
			}
			credits := map[string]any{}
			if json.Unmarshal(data, &envelope) == nil && envelope.Data != nil {
				credits = envelope.Data
			} else if err := json.Unmarshal(data, &credits); err != nil {
				return apiErr(err)
			}
			totalCredits := asFloat(credits["total_credits"])
			totalUsage := asFloat(credits["total_usage"])
			balance := totalCredits - totalUsage

			dbPath := defaultDBPath("openrouter-pp-cli")
			db, err := store.OpenWithContext(context.Background(), dbPath)
			if err != nil {
				return configErr(fmt.Errorf("open local store: %w", err))
			}
			defer db.Close()

			now := time.Now().UTC()
			if _, err := db.DB().ExecContext(cmd.Context(),
				`INSERT OR REPLACE INTO credits_snapshots (taken_at, total_credits, total_usage) VALUES (?, ?, ?)`,
				now.Format(time.RFC3339), totalCredits, totalUsage); err != nil {
				return configErr(fmt.Errorf("record credits snapshot: %w", err))
			}

			// Trailing burn: oldest snapshot inside the window vs now.
			since := now.AddDate(0, 0, -windowDays).Format(time.RFC3339)
			var oldestAt string
			var oldestUsage float64
			burnPerDay := 0.0
			spanDays := 0.0
			row := db.DB().QueryRowContext(cmd.Context(),
				`SELECT taken_at, total_usage FROM credits_snapshots WHERE taken_at >= ? ORDER BY taken_at ASC LIMIT 1`, since)
			if err := row.Scan(&oldestAt, &oldestUsage); err == nil {
				if t, perr := time.Parse(time.RFC3339, oldestAt); perr == nil {
					spanDays = now.Sub(t).Hours() / 24.0
					if spanDays > 0.01 {
						burnPerDay = (totalUsage - oldestUsage) / spanDays
					}
				}
			}

			daysToZero := "n/a"
			zeroAt := "n/a"
			if burnPerDay > 0 && balance > 0 {
				d := balance / burnPerDay
				daysToZero = fmt.Sprintf("%.1f", d)
				zeroAt = now.Add(time.Duration(d * 24 * float64(time.Hour))).Format(time.RFC3339)
			}

			result := map[string]any{
				"balance_usd":        balance,
				"total_credits":      totalCredits,
				"total_usage":        totalUsage,
				"burn_usd_per_day":   burnPerDay,
				"window_days":        windowDays,
				"observed_span_days": spanDays,
				"days_to_zero":       daysToZero,
				"projected_zero_at":  zeroAt,
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "balance: $%.2f (purchased $%.2f, used $%.2f)\n", balance, totalCredits, totalUsage)
			if burnPerDay > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "burn: $%.2f/day over %.1fd observed\n", burnPerDay, spanDays)
				fmt.Fprintf(cmd.OutOrStdout(), "runway: %s days (zero ~ %s)\n", daysToZero, zeroAt)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "runway: n/a — need at least two snapshots with usage growth; re-run after some traffic")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&windowDays, "window", 14, "Trailing window in days for the burn-rate estimate")
	return cmd
}
