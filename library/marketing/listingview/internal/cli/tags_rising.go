// Hand-authored novel command. Velocity-ranked rising tags.
// pp:data-source live
package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

type risingTag struct {
	Tag             string  `json:"tag"`
	VelocityScore   float64 `json:"velocity_score"`
	DemandScore     float64 `json:"demand_score"`
	CompetitionScore float64 `json:"competition_score"`
	OpportunityScore float64 `json:"opportunity_score"`
	CompetingListings float64 `json:"competing_listings"`
	AvgRevenue      float64 `json:"avg_revenue"`
}

type risingView struct {
	Term                string      `json:"term"`
	MinCompetitionScore float64     `json:"min_competition_score"`
	Count               int         `json:"count"`
	Tags                []risingTag `json:"tags"`
	Note                string      `json:"note,omitempty"`
}

func newNovelTagsRisingCmd(flags *rootFlags) *cobra.Command {
	var minCompetitionScore int
	var limit int
	var dbPath string
	cmd := &cobra.Command{
		Use:   "rising <term>",
		Short: "Rank tags for a niche by ListingView's velocity score (accelerating demand) while competition is still low.",
		Long: "Surface early-mover tags: those whose demand is accelerating (high velocity) while competition is still low.\n\n" +
			"ListingView's velocity score is exclusive to its data — neither eRank nor EverBee exposes it. ListingView's competition score runs high-is-uncrowded (a heavily-used tag like \"sticker\" scores low); --min-competition-score keeps only tags with that much headroom.",
		Example: "  listingview-pp-cli tags rising \"sticker\" --min-competition-score 50 --agent",
		// Any free-text term is valid; an unmatched term returns empty, not an error.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would rank tags by velocity score filtered by competition")
				return nil
			}
			if len(args) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a search term is required"))
			}
			term := args[0]
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			// Fetch by opportunity (demand-vs-competition balance) rather than raw
			// velocity: the top tags by velocity alone are saturated (competition
			// maxed), which the local competition filter would discard. Fetching a
			// broad opportunity-ranked set and re-ranking by velocity locally
			// surfaces tags that are both rising AND still uncrowded.
			data, err := callProxyPOST(ctx, c, "getFilteredTags", map[string]any{
				"search": term, "sort_column": "opportunityScore", "sort_order": "desc",
				"page": 1, "limit": 200, "filters": map[string]any{},
			})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var db = tryOpenStore(dbPath)
			if db != nil {
				defer db.Close()
			}
			view := risingView{Term: term, MinCompetitionScore: float64(minCompetitionScore), Tags: []risingTag{}}
			for _, t := range listOf(data, "tags") {
				comp := numOf(t, "competitionScore")
				// competitionScore is high-is-uncrowded; keep only tags with enough headroom.
				if minCompetitionScore > 0 && comp < float64(minCompetitionScore) {
					continue
				}
				rt := risingTag{
					Tag:              strOf(t, "tag"),
					VelocityScore:    numOf(t, "velocityScore"),
					DemandScore:      numOf(t, "demandScore"),
					CompetitionScore: comp,
					OpportunityScore: numOf(t, "opportunityScore"),
					CompetingListings: numOf(t, "competingListings"),
					AvgRevenue:       numOf(t, "avgRevenue"),
				}
				view.Tags = append(view.Tags, rt)
				if db != nil {
					saveSnapshot(db.DB(), "tag", rt.Tag, map[string]float64{
						"velocityScore": rt.VelocityScore, "demandScore": rt.DemandScore,
						"competitionScore": rt.CompetitionScore, "opportunityScore": rt.OpportunityScore,
						"avgRevenue": rt.AvgRevenue,
					})
				}
			}
			sort.SliceStable(view.Tags, func(i, j int) bool {
				return view.Tags[i].VelocityScore > view.Tags[j].VelocityScore
			})
			if limit > 0 && len(view.Tags) > limit {
				view.Tags = view.Tags[:limit]
			}
			view.Count = len(view.Tags)
			if view.Count == 0 {
				view.Note = fmt.Sprintf("no rising tags for %q with competition headroom >= %d; lower --min-competition-score or try a broader term", term, minCompetitionScore)
			}
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().IntVar(&minCompetitionScore, "min-competition-score", 40, "Only include tags with at least this much competition headroom (higher = less crowded; 0 = no filter).")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum rising tags to return.")
	cmd.Flags().StringVar(&dbPath, "db", "", "Local store path for snapshot history (defaults to the standard location).")
	return cmd
}
