// Hand-authored novel command. Revenue-weighted shop tag gaps.
// pp:data-source live
package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/client"
	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/cliutil"

	"github.com/spf13/cobra"
)

type gapTag struct {
	Tag             string  `json:"tag"`
	WeightedRevenue float64 `json:"weighted_revenue"`
	ListingCount    int     `json:"listing_count"`
}

type gapsView struct {
	MyShop                    string   `json:"my_shop"`
	Competitor                string   `json:"competitor"`
	MyListingsScanned         int      `json:"my_listings_scanned"`
	CompetitorListingsScanned int      `json:"competitor_listings_scanned"`
	GapTags                   []gapTag `json:"gap_tags"`
	Note                      string   `json:"note,omitempty"`
}

func newNovelGapsCmd(flags *rootFlags) *cobra.Command {
	var perShop int
	var limit int
	cmd := &cobra.Command{
		Use:   "gaps <my-shop> <competitor>",
		Short: "Compare your shop's tags against a competitor's, ranked by the revenue those tags drive for the competitor.",
		Long: "Find the tags a competitor sells on that your shop does not cover, ranked by the revenue those tags drive for them — so you prioritize the gaps that actually convert.\n\n" +
			"Two-shop comparison. For tag quality on a single listing use 'listings audit'; for term-level winning tags use 'tags consensus'.",
		Example: "  listingview-pp-cli gaps MyShop RivalShop --agent",
		// Any free-text shop names are valid; unknown shops return empty, not an error.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would diff two shops' tags ranked by competitor revenue")
				return nil
			}
			if len(args) < 2 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("both <my-shop> and <competitor> are required"))
			}
			myShop, competitor := args[0], args[1]
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			scan := perShop
			if cliutil.IsDogfoodEnv() && scan > 2 {
				scan = 2
			}
			mySet, myScanned, err := shopTagSet(ctx, c, myShop, scan)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			theirTags, theirScanned, err := shopTagRevenue(ctx, c, competitor, scan)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			view := gapsView{MyShop: myShop, Competitor: competitor, MyListingsScanned: myScanned, CompetitorListingsScanned: theirScanned, GapTags: []gapTag{}}
			for tag, gt := range theirTags {
				if mySet[tag] {
					continue
				}
				view.GapTags = append(view.GapTags, gapTag{Tag: tag, WeightedRevenue: round2(gt.weighted), ListingCount: gt.count})
			}
			sort.SliceStable(view.GapTags, func(i, j int) bool {
				return view.GapTags[i].WeightedRevenue > view.GapTags[j].WeightedRevenue
			})
			if limit > 0 && len(view.GapTags) > limit {
				view.GapTags = view.GapTags[:limit]
			}
			if theirScanned == 0 {
				view.Note = fmt.Sprintf("no listings found for competitor %q (check the exact shop name)", competitor)
			} else if len(view.GapTags) == 0 {
				view.Note = "no tag gaps — your shop already covers the competitor's tags in the scanned listings"
			}
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().IntVar(&perShop, "per-shop", 8, "Listings to scan per shop (each listing is one tag-extract call).")
	cmd.Flags().IntVar(&limit, "limit", 30, "Maximum gap tags to return.")
	return cmd
}

type tagRev struct {
	count    int
	weighted float64
}

// shopTagSet returns the set of tags used across a shop's top listings.
func shopTagSet(ctx context.Context, c *client.Client, shop string, n int) (map[string]bool, int, error) {
	data, err := callProxyPOST(ctx, c, "shop-analyzer/listings", map[string]any{
		"shopName": shop, "sortBy": "sales", "order": "desc", "limit": n, "page": 1,
	})
	if err != nil {
		return nil, 0, err
	}
	set := map[string]bool{}
	scanned := 0
	for _, l := range listOf(data, "listings") {
		id := numOf(l, "listingId")
		if id == 0 {
			continue
		}
		tagData, terr := callProxyPOST(ctx, c, "tag-extractor", map[string]any{"listingId": int64(id)})
		if terr != nil {
			continue
		}
		scanned++
		for _, t := range stringsOf(tagData, "tags") {
			if t != "" {
				set[t] = true
			}
		}
	}
	return set, scanned, nil
}

// shopTagRevenue returns each tag's revenue-weighted score across a shop's top listings.
func shopTagRevenue(ctx context.Context, c *client.Client, shop string, n int) (map[string]*tagRev, int, error) {
	data, err := callProxyPOST(ctx, c, "shop-analyzer/listings", map[string]any{
		"shopName": shop, "sortBy": "sales", "order": "desc", "limit": n, "page": 1,
	})
	if err != nil {
		return nil, 0, err
	}
	out := map[string]*tagRev{}
	scanned := 0
	for _, l := range listOf(data, "listings") {
		id := numOf(l, "listingId")
		if id == 0 {
			continue
		}
		rev := firstNonZero(numOf(l, "moRevenue"), numOf(l, "revenue30days"), numOf(l, "revenue"), 1)
		tagData, terr := callProxyPOST(ctx, c, "tag-extractor", map[string]any{"listingId": int64(id)})
		if terr != nil {
			continue
		}
		scanned++
		seen := map[string]bool{}
		for _, t := range stringsOf(tagData, "tags") {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			tr := out[t]
			if tr == nil {
				tr = &tagRev{}
				out[t] = tr
			}
			tr.count++
			tr.weighted += rev
		}
	}
	return out, scanned, nil
}
