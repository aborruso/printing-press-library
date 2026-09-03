// Hand-authored novel command. Revenue-weighted tag consensus across top sellers.
// pp:data-source live
package cli

import (
	"fmt"
	"sort"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/cliutil"

	"github.com/spf13/cobra"
)

type consensusTag struct {
	Tag             string  `json:"tag"`
	ListingCount    int     `json:"listing_count"`
	WeightedRevenue float64 `json:"weighted_revenue"`
	SharePct        float64 `json:"share_pct"`
}

type consensusView struct {
	Term              string         `json:"term"`
	TopSellersScanned int            `json:"top_sellers_scanned"`
	Tags              []consensusTag `json:"tags"`
	Note              string         `json:"note,omitempty"`
}

func newNovelTagsConsensusCmd(flags *rootFlags) *cobra.Command {
	var top int
	var limit int
	cmd := &cobra.Command{
		Use:   "consensus <term>",
		Short: "Find the tags that repeatedly appear across the top-selling listings for a term, weighted by each listing's revenue.",
		Long: "Rank the tags the winners actually use for a term, weighted by each top seller's revenue estimate so the consensus reflects what sells, not a flat tag count.\n\n" +
			"Term-level \"what do winners tag with.\" To audit one specific listing's tags use 'listings audit'.",
		Example: "  listingview-pp-cli tags consensus \"vinyl sticker\" --agent",
		// Any free-text term is a valid search; an unmatched term returns an
		// empty result, not an error — so there is no invalid-input error path.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would scan top sellers and rank their tags by revenue weight")
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
			scan := top
			if cliutil.IsDogfoodEnv() && scan > 3 {
				scan = 3
			}
			lData, err := callProxyPOST(ctx, c, "getFilteredListings", map[string]any{
				"search": term, "sort_column": "sales", "sort_order": "desc", "page": 1, "limit": scan,
				"filter": "top", "search_after": nil, "timeframe": "", "filters": map[string]any{}, "salesInterval": 30,
			})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			listings := listOf(lData, "listings")
			type agg struct {
				count    int
				weighted float64
			}
			tally := map[string]*agg{}
			var totalRevenue float64
			scanned := 0
			for _, l := range listings {
				id := numOf(l, "listingId")
				if id == 0 {
					continue
				}
				rev := firstNonZero(numOf(l, "moRevenue"), numOf(l, "revenue30days"), numOf(l, "revenue"), 1)
				tagData, terr := callProxyPOST(ctx, c, "tag-extractor", map[string]any{"listingId": int64(id)})
				if terr != nil {
					continue // partial failure: skip this listing, keep aggregating
				}
				scanned++
				totalRevenue += rev
				seen := map[string]bool{}
				for _, tag := range stringsOf(tagData, "tags") {
					if tag == "" || seen[tag] {
						continue
					}
					seen[tag] = true
					a := tally[tag]
					if a == nil {
						a = &agg{}
						tally[tag] = a
					}
					a.count++
					a.weighted += rev
				}
			}
			view := consensusView{Term: term, TopSellersScanned: scanned, Tags: []consensusTag{}}
			for tag, a := range tally {
				share := 0.0
				if totalRevenue > 0 {
					share = round2(a.weighted / totalRevenue * 100)
				}
				view.Tags = append(view.Tags, consensusTag{Tag: tag, ListingCount: a.count, WeightedRevenue: round2(a.weighted), SharePct: share})
			}
			sort.SliceStable(view.Tags, func(i, j int) bool {
				if view.Tags[i].WeightedRevenue != view.Tags[j].WeightedRevenue {
					return view.Tags[i].WeightedRevenue > view.Tags[j].WeightedRevenue
				}
				return view.Tags[i].ListingCount > view.Tags[j].ListingCount
			})
			if limit > 0 && len(view.Tags) > limit {
				view.Tags = view.Tags[:limit]
			}
			if scanned == 0 {
				view.Note = fmt.Sprintf("no top sellers with extractable tags found for %q", term)
			}
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().IntVar(&top, "top", 10, "How many top-selling listings to scan for tags.")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum consensus tags to return.")
	return cmd
}

func firstNonZero(vals ...float64) float64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
