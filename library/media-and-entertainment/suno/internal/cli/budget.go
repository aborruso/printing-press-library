// Copyright 2026 mvanhorn. Licensed under Apache-2.0. See LICENSE.
// PATCH: Add local budget caps and spend summary.

package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

type budgetSetting struct {
	DailyCredits   int    `json:"daily_credits,omitempty"`
	MonthlyCredits int    `json:"monthly_credits,omitempty"`
	UpdatedAt      string `json:"updated_at"`
}

func newBudgetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "budget", Short: "Show and configure local generation credit caps"}
	cmd.AddCommand(newBudgetShowCmd(flags))
	cmd.AddCommand(newBudgetSetCmd(flags))
	cmd.AddCommand(newBudgetClearCmd(flags))
	return cmd
}

func newBudgetShowCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "show",
		Short:       "Show budget caps and month-to-date spend",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openDefaultStore(cmd.Context())
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer s.Close()
			var setting budgetSetting
			raw, err := s.Get("budget_setting", "current")
			if err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("reading budget setting: %w", err)
			}
			if raw != nil {
				_ = json.Unmarshal(raw, &setting)
			}
			now := time.Now()
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			spend, err := estimatedSpendSince(cmd.Context(), s, monthStart)
			if err != nil {
				return err
			}
			daily, err := estimatedSpendSince(cmd.Context(), s, dayStart)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"daily_credits":          setting.DailyCredits,
				"monthly_credits":        setting.MonthlyCredits,
				"today_spend":            daily,
				"month_to_date_spend":    spend,
				"month_to_date_start_at": monthStart.UTC().Format(time.RFC3339),
				"updated_at":             setting.UpdatedAt,
			}, flags)
		},
	}
}

func newBudgetSetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "set", Short: "Set budget caps"}
	cmd.AddCommand(newBudgetSetKindCmd(flags, "daily"))
	cmd.AddCommand(newBudgetSetKindCmd(flags, "monthly"))
	return cmd
}

func newBudgetSetKindCmd(flags *rootFlags, kind string) *cobra.Command {
	return &cobra.Command{
		Use:   kind + " <N>",
		Short: "Set " + kind + " credit cap",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			n, err := strconv.Atoi(args[0])
			if err != nil || n < 0 {
				return usageErr(fmt.Errorf("%s budget must be a non-negative integer", kind))
			}
			if dryRunOK(flags) {
				return nil
			}
			s, err := openDefaultStore(cmd.Context())
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer s.Close()
			var setting budgetSetting
			if raw, err := s.Get("budget_setting", "current"); err == nil {
				_ = json.Unmarshal(raw, &setting)
			}
			if kind == "daily" {
				setting.DailyCredits = n
			} else {
				setting.MonthlyCredits = n
			}
			setting.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := s.Upsert("budget_setting", "current", mustJSON(setting)); err != nil {
				return fmt.Errorf("saving budget setting: %w", err)
			}
			return printJSONFiltered(cmd.OutOrStdout(), setting, flags)
		},
	}
}

func newBudgetClearCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "clear",
		Short:   "Remove the daily and monthly credit caps so generate runs without a spend gate",
		Example: "  suno-pp-cli budget clear",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			s, err := openDefaultStore(cmd.Context())
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer s.Close()
			_, err = s.DB().ExecContext(cmd.Context(), `DELETE FROM resources WHERE resource_type='budget_setting' AND id='current'`)
			if err != nil {
				return fmt.Errorf("clearing budget setting: %w", err)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"cleared": true}, flags)
		},
	}
}

func estimatedSpendSince(ctx context.Context, s interface{ DB() *sql.DB }, since time.Time) (int, error) {
	rows, err := s.DB().QueryContext(ctx, `SELECT data FROM resources WHERE resource_type IN ('clip','clips')`)
	if err != nil {
		return 0, fmt.Errorf("querying clip spend: %w", err)
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, fmt.Errorf("scanning clip spend: %w", err)
		}
		obj := unmarshalObject(json.RawMessage(raw))
		if t := clipCreatedAt(obj); !t.IsZero() && !t.Before(since) {
			total += 10
		}
	}
	return total, rows.Err()
}
