// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command: models churn — hand-built (reprint 2026-09-01).
// pp:data-source local

package cli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/ai/openrouter/internal/store"
)

// churnSnapshotRetentionDays bounds how much catalog history the local store
// keeps; snapshots older than this are pruned on each invocation.
const churnSnapshotRetentionDays = 90

type churnModelRow struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PromptPrice     string `json:"prompt_price"`
	CompletionPrice string `json:"completion_price"`
}

// newNovelModelsChurnCmd diffs the current synced catalog against the most
// recent snapshot at or older than the --since window. Each invocation stores
// a fresh snapshot, so history accumulates with use. No upstream history
// endpoint exists; this diff is only possible over sync-kept local state.
func newNovelModelsChurnCmd(flags *rootFlags) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:         "churn",
		Short:       "See what changed in the model catalog between syncs: additions, removals, and repricings with deltas.",
		Example:     "  openrouter-pp-cli models churn --since 7d --agent",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "diff the model catalog against the stored baseline snapshot")
			}
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), `{"added":0,"removed":0,"repriced":0,"baseline":"n/a"}`)
				return nil
			}
			dbPath := defaultDBPath("openrouter-pp-cli")
			db, err := store.OpenWithContext(context.Background(), dbPath)
			if err != nil {
				return configErr(fmt.Errorf("open local store: %w", err))
			}
			defer db.Close()
			ctx := cmd.Context()

			readCurrent := func() (map[string]churnModelRow, error) {
				rows, err := db.DB().QueryContext(ctx,
					`SELECT id, COALESCE(name,''), COALESCE(json_extract(data,'$.pricing.prompt'),''), COALESCE(json_extract(data,'$.pricing.completion'),'') FROM models`)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				out := map[string]churnModelRow{}
				for rows.Next() {
					var r churnModelRow
					if err := rows.Scan(&r.ID, &r.Name, &r.PromptPrice, &r.CompletionPrice); err != nil {
						return nil, err
					}
					out[r.ID] = r
				}
				return out, rows.Err()
			}
			current, err := readCurrent()
			if err != nil {
				return configErr(fmt.Errorf("read models table: %w", err))
			}
			if len(current) == 0 {
				return notFoundErr(fmt.Errorf("models table is empty — run 'openrouter-pp-cli sync --resources models' first"))
			}

			// Baseline: most recent snapshot at or older than the window start;
			// fall back to the oldest available snapshot.
			now := time.Now().UTC()
			sinceT, err := parseSinceDuration(since)
			if err != nil {
				return usageErr(err)
			}
			windowStart := sinceT.UTC().Format(time.RFC3339)
			var baselineAt string
			row := db.DB().QueryRowContext(ctx,
				`SELECT COALESCE(MAX(snap_at),'') FROM models_catalog_snapshots WHERE snap_at <= ?`, windowStart)
			_ = row.Scan(&baselineAt)
			if baselineAt == "" {
				row = db.DB().QueryRowContext(ctx,
					`SELECT COALESCE(MIN(snap_at),'') FROM models_catalog_snapshots`)
				_ = row.Scan(&baselineAt)
			}

			baseline := map[string]churnModelRow{}
			if baselineAt != "" {
				rows, err := db.DB().QueryContext(ctx,
					`SELECT model_id, COALESCE(name,''), COALESCE(prompt_price,''), COALESCE(completion_price,'') FROM models_catalog_snapshots WHERE snap_at = ?`, baselineAt)
				if err != nil {
					return configErr(err)
				}
				defer rows.Close()
				for rows.Next() {
					var r churnModelRow
					if err := rows.Scan(&r.ID, &r.Name, &r.PromptPrice, &r.CompletionPrice); err != nil {
						return configErr(err)
					}
					baseline[r.ID] = r
				}
				if err := rows.Err(); err != nil {
					return configErr(err)
				}
			}

			// Store the fresh snapshot + prune old history.
			snapAt := now.Format(time.RFC3339)
			tx, err := db.DB().BeginTx(ctx, nil)
			if err != nil {
				return configErr(err)
			}
			for _, r := range current {
				if _, err := tx.ExecContext(ctx,
					`INSERT OR REPLACE INTO models_catalog_snapshots (snap_at, model_id, name, prompt_price, completion_price) VALUES (?, ?, ?, ?, ?)`,
					snapAt, r.ID, r.Name, r.PromptPrice, r.CompletionPrice); err != nil {
					_ = tx.Rollback()
					return configErr(fmt.Errorf("store catalog snapshot: %w", err))
				}
			}
			pruneBefore := now.AddDate(0, 0, -churnSnapshotRetentionDays).Format(time.RFC3339)
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM models_catalog_snapshots WHERE snap_at < ?`, pruneBefore); err != nil {
				_ = tx.Rollback()
				return configErr(err)
			}
			if err := tx.Commit(); err != nil {
				return configErr(err)
			}

			if len(baseline) == 0 {
				result := map[string]any{
					"baseline": "none", "snapshot_stored": snapAt,
					"added": 0, "removed": 0, "repriced": 0,
					"note": "first snapshot stored; re-run after the next sync to see churn",
				}
				if flags.asJSON || flags.agent {
					return printJSONFiltered(cmd.OutOrStdout(), result, flags)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "no baseline yet — stored first snapshot (%d models) at %s; re-run after the next sync\n", len(current), snapAt)
				return nil
			}

			type repricing struct {
				ID            string `json:"id"`
				OldPrompt     string `json:"old_prompt"`
				NewPrompt     string `json:"new_prompt"`
				OldCompletion string `json:"old_completion"`
				NewCompletion string `json:"new_completion"`
			}
			var added, removed []string
			var repriced []repricing
			for id := range current {
				if _, ok := baseline[id]; !ok {
					added = append(added, id)
				}
			}
			for id, b := range baseline {
				cur, ok := current[id]
				if !ok {
					removed = append(removed, id)
					continue
				}
				if b.PromptPrice != cur.PromptPrice || b.CompletionPrice != cur.CompletionPrice {
					repriced = append(repriced, repricing{
						ID: id, OldPrompt: b.PromptPrice, NewPrompt: cur.PromptPrice,
						OldCompletion: b.CompletionPrice, NewCompletion: cur.CompletionPrice,
					})
				}
			}
			sort.Strings(added)
			sort.Strings(removed)
			sort.Slice(repriced, func(i, j int) bool { return repriced[i].ID < repriced[j].ID })

			result := map[string]any{
				"baseline":        baselineAt,
				"snapshot_stored": snapAt,
				"added":           added,
				"removed":         removed,
				"repriced":        repriced,
				"added_count":     len(added),
				"removed_count":   len(removed),
				"repriced_count":  len(repriced),
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "baseline %s → now: +%d added, -%d removed, %d repriced\n", baselineAt, len(added), len(removed), len(repriced))
			for _, id := range added {
				fmt.Fprintf(cmd.OutOrStdout(), "  + %s\n", id)
			}
			for _, id := range removed {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", id)
			}
			for _, r := range repriced {
				fmt.Fprintf(cmd.OutOrStdout(), "  Δ %s prompt %s→%s completion %s→%s\n", r.ID, r.OldPrompt, r.NewPrompt, r.OldCompletion, r.NewCompletion)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "7d", "Diff against the most recent snapshot at least this old (e.g. 7d)")
	return cmd
}
