// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: offline "more like this" via local tag-set overlap (Jaccard).
// Hand-authored; survives `generate --force` as a whole unit.
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

func newNovelSimilarCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var kind, dbPath string
	cmd := &cobra.Command{
		Use:   "similar <id>",
		Short: "Find synced hits sharing the most tags with a given ID (local Jaccard overlap)",
		Long: strings.TrimSpace(`
Rank synced hits by how many tags they share with a target ID, using a local
tag-set overlap (Jaccard) score. Use this for offline 'more like this' over your
synced store; do NOT use it to fetch new results from the API — use
'images search' for that. Sync results first so there is cached data to
compare against.`),
		Example:     "  pixabay-pp-cli similar 195893 --limit 20 --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("similar requires a target ID"))
			}
			k, err := collectionKind(kind)
			if err != nil {
				return err
			}
			targetID := strings.TrimSpace(args[0])
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				return noMirror(cmd, flags, k)
			}
			db, err := store.OpenReadOnlyContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			if !hintIfUnsynced(cmd, db, k) {
				hintIfStale(cmd, db, k, flags.maxAge)
			}

			// Fetch target tags.
			var targetTags sql.NullString
			err = db.DB().QueryRowContext(cmd.Context(),
				fmt.Sprintf(`SELECT tags FROM %q WHERE id = ?`, k), targetID).Scan(&targetTags)
			if err == sql.ErrNoRows {
				fmt.Fprintf(cmd.ErrOrStderr(), "ID %s not found in local %s store; sync it first\n", targetID, k)
				return emptyJSONArray(cmd, flags)
			}
			if err != nil {
				return fmt.Errorf("looking up target: %w", err)
			}
			target := splitTagString(targetTags.String)
			if len(target) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "target %s has no tags to compare\n", targetID)
				return emptyJSONArray(cmd, flags)
			}

			rows, err := db.DB().QueryContext(cmd.Context(),
				fmt.Sprintf(`SELECT id, tags, page_url, user FROM %q WHERE id != ?`, k), targetID)
			if err != nil {
				return fmt.Errorf("scanning store: %w", err)
			}
			defer rows.Close()

			type simRow struct {
				ID         string   `json:"id"`
				Kind       string   `json:"kind"`
				SharedTags []string `json:"shared_tags"`
				SharedN    int      `json:"shared"`
				Jaccard    float64  `json:"jaccard"`
				PageURL    string   `json:"page_url"`
				User       string   `json:"user"`
			}
			var matches []simRow
			for rows.Next() {
				var id string
				var tags, pageURL, user sql.NullString
				if err := rows.Scan(&id, &tags, &pageURL, &user); err != nil {
					continue
				}
				other := splitTagString(tags.String)
				if len(other) == 0 {
					continue
				}
				shared := intersectTags(target, other)
				if len(shared) == 0 {
					continue
				}
				union := len(target) + len(other) - len(shared)
				jac := 0.0
				if union > 0 {
					jac = float64(len(shared)) / float64(union)
				}
				matches = append(matches, simRow{
					ID:         id,
					Kind:       k,
					SharedTags: sortedStrings(shared),
					SharedN:    len(shared),
					Jaccard:    round3(jac),
					PageURL:    pageURL.String,
					User:       user.String,
				})
			}
			sort.SliceStable(matches, func(i, j int) bool {
				if matches[i].SharedN != matches[j].SharedN {
					return matches[i].SharedN > matches[j].SharedN
				}
				return matches[i].Jaccard > matches[j].Jaccard
			})
			if limit > 0 && len(matches) > limit {
				matches = matches[:limit]
			}
			if matches == nil {
				matches = []simRow{}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), matches, flags)
			}
			if len(matches) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No similar items found in the local store.")
				return nil
			}
			for _, m := range matches {
				fmt.Fprintf(cmd.OutOrStdout(), "%-10s shared=%-2d jaccard=%.3f  %s\n", m.ID, m.SharedN, m.Jaccard, strings.Join(m.SharedTags, ", "))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum similar items to return")
	cmd.Flags().StringVar(&kind, "kind", "images", "Media kind: images or videos")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func splitTagString(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range strings.Split(s, ",") {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			out[t] = struct{}{}
		}
	}
	return out
}

func intersectTags(a, b map[string]struct{}) []string {
	var out []string
	for t := range a {
		if _, ok := b[t]; ok {
			out = append(out, t)
		}
	}
	return out
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

// noMirror prints the standard missing-local-mirror hint and an empty machine
// result.
func noMirror(cmd *cobra.Command, flags *rootFlags, resource string) error {
	dbPath := defaultDBPath(pixabayCLIName)
	fmt.Fprintf(cmd.ErrOrStderr(), "no local mirror at %s\nrun: pixabay-pp-cli sync --resources %s --db %s\n", dbPath, resource, dbPath)
	return emptyJSONArray(cmd, flags)
}

func emptyJSONArray(cmd *cobra.Command, flags *rootFlags) error {
	if flags.asJSON || flags.agent || !isTerminal(cmd.OutOrStdout()) {
		fmt.Fprintln(cmd.OutOrStdout(), "[]")
	}
	return nil
}
