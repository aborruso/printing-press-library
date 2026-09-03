// Hand-authored novel command. Local opportunity shortlist over researched data.
// pp:data-source local
package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

type opportunity struct {
	Kind             string  `json:"kind"`
	Key              string  `json:"key"`
	Score            float64 `json:"score"`
	Volume           float64 `json:"volume,omitempty"`
	CompetingListings float64 `json:"competing_listings,omitempty"`
	OpportunityScore float64 `json:"opportunity_score,omitempty"`
	DemandScore      float64 `json:"demand_score,omitempty"`
	CompetitionScore float64 `json:"competition_score,omitempty"`
	VelocityScore    float64 `json:"velocity_score,omitempty"`
}

type opportunitiesView struct {
	TotalResearched int           `json:"total_researched"`
	Opportunities   []opportunity `json:"opportunities"`
	Note            string        `json:"note,omitempty"`
}

func newNovelOpportunitiesCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var dbPath string
	cmd := &cobra.Command{
		Use:   "opportunities",
		Short: "Rank the best untapped plays across everything you've already researched: high demand, low competition, rising velocity.",
		Long: "Score and rank every keyword and tag you've researched by demand vs competition and velocity, with zero new API calls — turning a month of scattered research into a ranked shortlist and stretching your monthly quota.\n\n" +
			"Ranks across everything already researched. For a deep single-term verdict use 'niche'; for what's changed use 'drift'.",
		Example:     "  listingview-pp-cli opportunities --limit 10 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would rank researched keywords/tags by opportunity score")
				return nil
			}
			resolved := dbPath
			if resolved == "" {
				resolved = defaultDBPath("listingview-pp-cli")
			}
			if _, statErr := os.Stat(resolved); os.IsNotExist(statErr) {
				return emptyOppHint(cmd, flags, resolved)
			}
			db := tryOpenStore(resolved)
			if db == nil {
				return emptyOppHint(cmd, flags, resolved)
			}
			defer db.Close()
			snaps, err := latestPerKey(db.DB())
			if err != nil {
				return fmt.Errorf("reading research history: %w", err)
			}
			view := opportunitiesView{TotalResearched: len(snaps), Opportunities: []opportunity{}}
			for _, s := range snaps {
				op := opportunity{Kind: s.Kind, Key: s.Key}
				switch s.Kind {
				case "keyword":
					op.Volume = s.Metrics["volume"]
					op.CompetingListings = s.Metrics["competingListings"]
					if op.CompetingListings > 0 {
						// Map the unbounded demand/competition ratio onto a 0-100
						// score (saturating: ratio 20 -> 50, ratio 80 -> 80) so
						// keywords and tags rank on the same scale.
						ratio := op.Volume / op.CompetingListings
						op.Score = round2(100 * ratio / (ratio + 20))
					}
				case "tag":
					op.OpportunityScore = s.Metrics["opportunityScore"]
					op.DemandScore = s.Metrics["demandScore"]
					op.CompetitionScore = s.Metrics["competitionScore"]
					op.VelocityScore = s.Metrics["velocityScore"]
					// Both inputs are already 0-100; weight opportunity over
					// velocity so the blended score stays on the same 0-100 scale
					// as the keyword score above (no cross-kind ranking bias).
					op.Score = round2(0.7*op.OpportunityScore + 0.3*op.VelocityScore)
				}
				view.Opportunities = append(view.Opportunities, op)
			}
			sort.SliceStable(view.Opportunities, func(i, j int) bool {
				return view.Opportunities[i].Score > view.Opportunities[j].Score
			})
			if limit > 0 && len(view.Opportunities) > limit {
				view.Opportunities = view.Opportunities[:limit]
			}
			if view.TotalResearched == 0 {
				view.Note = "no research history yet — run 'niche' or 'tags rising' first to populate the local store"
			}
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 15, "Maximum opportunities to return.")
	cmd.Flags().StringVar(&dbPath, "db", "", "Local store path (defaults to the standard location).")
	return cmd
}

func emptyOppHint(cmd *cobra.Command, flags *rootFlags, path string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "no research history at %s\nrun research commands first, e.g.: listingview-pp-cli tags rising \"sticker\"\n", path)
	return printJSONFiltered(cmd.OutOrStdout(), opportunitiesView{Opportunities: []opportunity{}, Note: "no research history yet"}, flags)
}
