// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: rank Pixabay contributors by engagement across the local
// store. Hand-authored; survives `generate --force` as a whole unit.
//
// pp:data-source local

package cli

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

type contributorAgg struct {
	User      string `json:"user"`
	UserID    string `json:"user_id"`
	Assets    int    `json:"assets"`
	Downloads int64  `json:"downloads"`
	Likes     int64  `json:"likes"`
	Views     int64  `json:"views"`
	Comments  int64  `json:"comments"`
	// AvgMetric and RankedBy describe the chosen ranking metric.
	AvgMetric float64 `json:"avg_metric"`
	RankedBy  string  `json:"ranked_by"`
}

func (c contributorAgg) total(metric string) int64 {
	switch metric {
	case "likes":
		return c.Likes
	case "views":
		return c.Views
	case "comments":
		return c.Comments
	default:
		return c.Downloads
	}
}

func newNovelContributorsCmd(flags *rootFlags) *cobra.Command {
	var by, kind, dbPath string
	var minAssets, limit int
	cmd := &cobra.Command{
		Use:   "contributors",
		Short: "Rank contributors by total/average engagement across your synced store",
		Long: strings.TrimSpace(`
Aggregate synced image and video hits by contributor and rank them by total or
average downloads, likes, views, or comments. The Pixabay API has no aggregation
endpoint, so this is computed locally over whatever you have synced.
Use it to find the strongest contributors for a theme.`),
		Example:     "  pixabay-pp-cli contributors --by downloads --min-assets 3 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			metric, err := contributorMetric(by)
			if err != nil {
				return err
			}
			kinds, err := contributorKinds(kind)
			if err != nil {
				return err
			}
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				return noMirror(cmd, flags, kinds[0])
			}
			db, err := store.OpenReadOnlyContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			byUser := map[string]*contributorAgg{}
			for _, k := range kinds {
				rows, qerr := db.DB().QueryContext(cmd.Context(),
					fmt.Sprintf(`SELECT user, user_id, downloads, likes, views, comments FROM %q`, k))
				if qerr != nil {
					return fmt.Errorf("scanning %s: %w", k, qerr)
				}
				func() {
					defer rows.Close()
					for rows.Next() {
						var user, userID sql.NullString
						var downloads, likes, views, comments sql.NullInt64
						if err := rows.Scan(&user, &userID, &downloads, &likes, &views, &comments); err != nil {
							continue
						}
						key := userID.String + "|" + user.String
						a := byUser[key]
						if a == nil {
							a = &contributorAgg{User: user.String, UserID: userID.String, RankedBy: metric}
							byUser[key] = a
						}
						a.Assets++
						a.Downloads += downloads.Int64
						a.Likes += likes.Int64
						a.Views += views.Int64
						a.Comments += comments.Int64
					}
				}()
			}

			result := make([]contributorAgg, 0, len(byUser))
			for _, a := range byUser {
				if a.Assets < minAssets {
					continue
				}
				if a.Assets > 0 {
					a.AvgMetric = round3(float64(a.total(metric)) / float64(a.Assets))
				}
				result = append(result, *a)
			}
			sort.SliceStable(result, func(i, j int) bool {
				ti, tj := result[i].total(metric), result[j].total(metric)
				if ti != tj {
					return ti > tj
				}
				return result[i].Assets > result[j].Assets
			})
			if limit > 0 && len(result) > limit {
				result = result[:limit]
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			if len(result) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No contributors found. Sync some results first (run a search to populate the cache).")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-7s %-10s %-10s\n", "USER", "ASSETS", strings.ToUpper(metric), "AVG")
			for _, r := range result {
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-7d %-10d %-10.1f\n", truncate(r.User, 24), r.Assets, r.total(metric), r.AvgMetric)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&by, "by", "downloads", "Rank by: downloads, likes, views, or comments")
	cmd.Flags().StringVar(&kind, "kind", "all", "Media kind: images, videos, or all")
	cmd.Flags().IntVar(&minAssets, "min-assets", 1, "Only contributors with at least this many synced assets")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum contributors to return")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func contributorMetric(by string) (string, error) {
	switch strings.ToLower(by) {
	case "", "downloads":
		return "downloads", nil
	case "likes":
		return "likes", nil
	case "views":
		return "views", nil
	case "comments":
		return "comments", nil
	default:
		return "", usageErr(fmt.Errorf("--by must be downloads, likes, views, or comments, got %q", by))
	}
}

func contributorKinds(kind string) ([]string, error) {
	switch strings.ToLower(kind) {
	case "", "all":
		return []string{"images", "videos"}, nil
	case "images", "image":
		return []string{"images"}, nil
	case "videos", "video":
		return []string{"videos"}, nil
	default:
		return nil, usageErr(fmt.Errorf("--kind must be images, videos, or all, got %q", kind))
	}
}
