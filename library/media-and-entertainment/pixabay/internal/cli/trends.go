// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: engagement deltas over time. Each run snapshots current
// stats and diffs them against a prior snapshot, so trends build up history the
// point-in-time API cannot give. Hand-authored; survives `generate --force`.
//
// pp:data-source local

package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

type trendRow struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Tags            string `json:"tags"`
	Views           int64  `json:"views"`
	Downloads       int64  `json:"downloads"`
	Likes           int64  `json:"likes"`
	Comments        int64  `json:"comments"`
	DViews          int64  `json:"delta_views"`
	DDownloads      int64  `json:"delta_downloads"`
	DLikes          int64  `json:"delta_likes"`
	DComments       int64  `json:"delta_comments"`
	Baseline        bool   `json:"baseline"`
	ComparedAgainst string `json:"compared_against,omitempty"`
}

func newNovelTrendsCmd(flags *rootFlags) *cobra.Command {
	var tags []string
	var since, kind, dbPath string
	var limit int
	var noSnapshot bool
	cmd := &cobra.Command{
		Use:   "trends",
		Short: "Report engagement deltas (views/downloads/likes) since the last snapshot",
		Long: strings.TrimSpace(`
Each run records a snapshot of current views/downloads/likes/comments and diffs
them against a prior snapshot, building up trend history the point-in-time API
cannot provide. The first run for an item is a baseline (no prior to compare).
Re-run after each 'sync' to see what is gaining.`),
		Example:     "  pixabay-pp-cli trends --tag winter --since 7d --agent",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			kinds, err := contributorKinds(kind)
			if err != nil {
				return err
			}
			var sinceDur time.Duration
			if strings.TrimSpace(since) != "" {
				sinceDur, err = cliutil.ParseDurationLoose(since)
				if err != nil {
					return usageErr(fmt.Errorf("invalid --since %q: %w", since, err))
				}
			}
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				return noMirror(cmd, flags, kinds[0])
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			var result []trendRow
			now := nowUTC(cmd)
			cutoff := now.Add(-sinceDur)
			for _, k := range kinds {
				q := fmt.Sprintf(`SELECT id, tags, views, downloads, likes, comments FROM %q`, k)
				var qargs []any
				if tagFilter := strings.TrimSpace(strings.Join(tags, " ")); tagFilter != "" {
					clauses := make([]string, 0, len(tags))
					for _, t := range tags {
						clauses = append(clauses, "tags LIKE ?")
						qargs = append(qargs, "%"+strings.ToLower(strings.TrimSpace(t))+"%")
					}
					q += " WHERE " + strings.Join(clauses, " OR ")
				}
				rows, qerr := db.DB().QueryContext(cmd.Context(), q, qargs...)
				if qerr != nil {
					return fmt.Errorf("scanning %s: %w", k, qerr)
				}
				current := readTrendCurrent(rows, k)
				for _, cur := range current {
					prior, priorAt, found := priorSnapshot(cmd.Context(), db, k, cur.ID, cutoff)
					row := cur
					if found {
						row.DViews = cur.Views - prior.Views
						row.DDownloads = cur.Downloads - prior.Downloads
						row.DLikes = cur.Likes - prior.Likes
						row.DComments = cur.Comments - prior.Comments
						row.ComparedAgainst = priorAt.Format(time.RFC3339)
					} else {
						row.Baseline = true
					}
					result = append(result, row)
					if !noSnapshot {
						recordSnapshot(cmd.Context(), db, k, cur, now)
					}
				}
			}

			sort.SliceStable(result, func(i, j int) bool {
				return result[i].DDownloads > result[j].DDownloads
			})
			if limit > 0 && len(result) > limit {
				result = result[:limit]
			}
			if result == nil {
				result = []trendRow{}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			if len(result) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No matching items in the local store. Sync first.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-10s %-10s %-10s %-10s %s\n", "ID", "ΔDOWNLOAD", "ΔLIKES", "ΔVIEWS", "STATE")
			for _, r := range result {
				state := "delta"
				if r.Baseline {
					state = "baseline"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-10s %+-10d %+-10d %+-10d %s\n", r.ID, r.DDownloads, r.DLikes, r.DViews, state)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter to items whose tags contain this value (repeatable)")
	cmd.Flags().StringVar(&since, "since", "", "Compare against the most recent snapshot at least this old (e.g. 7d, 24h)")
	cmd.Flags().StringVar(&kind, "kind", "all", "Media kind: images, videos, or all")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum items to report")
	cmd.Flags().BoolVar(&noSnapshot, "no-snapshot", false, "Do not record a new snapshot this run")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func readTrendCurrent(rows *sql.Rows, kind string) []trendRow {
	defer rows.Close()
	var out []trendRow
	for rows.Next() {
		var id string
		var tags sql.NullString
		var views, downloads, likes, comments sql.NullInt64
		if err := rows.Scan(&id, &tags, &views, &downloads, &likes, &comments); err != nil {
			continue
		}
		out = append(out, trendRow{
			ID: id, Kind: kind, Tags: tags.String,
			Views: views.Int64, Downloads: downloads.Int64,
			Likes: likes.Int64, Comments: comments.Int64,
		})
	}
	return out
}

func priorSnapshot(ctx context.Context, db *store.Store, kind, id string, cutoff time.Time) (trendRow, time.Time, bool) {
	var views, downloads, likes, comments sql.NullInt64
	var at string
	err := db.DB().QueryRowContext(ctx,
		`SELECT views, downloads, likes, comments, snapshot_at FROM pp_stat_snapshots
		 WHERE kind = ? AND item_id = ? AND snapshot_at <= ?
		 ORDER BY snapshot_at DESC LIMIT 1`,
		kind, id, cutoff.Format(time.RFC3339)).Scan(&views, &downloads, &likes, &comments, &at)
	if err != nil {
		return trendRow{}, time.Time{}, false
	}
	parsed, _ := time.Parse(time.RFC3339, at)
	return trendRow{
		Views: views.Int64, Downloads: downloads.Int64,
		Likes: likes.Int64, Comments: comments.Int64,
	}, parsed, true
}

func recordSnapshot(ctx context.Context, db *store.Store, kind string, cur trendRow, now time.Time) {
	_, _ = db.DB().ExecContext(ctx,
		`INSERT INTO pp_stat_snapshots (kind, item_id, tags, views, downloads, likes, comments, snapshot_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		kind, cur.ID, cur.Tags, cur.Views, cur.Downloads, cur.Likes, cur.Comments, now.Format(time.RFC3339))
}

// nowUTC returns the current UTC time. A dedicated helper so tests can reason
// about snapshot timestamps if needed.
func nowUTC(_ *cobra.Command) time.Time {
	return time.Now().UTC()
}
