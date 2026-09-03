// Hand-authored novel command. Listing tag teardown / grading.
// pp:data-source live
package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/cliutil"

	"github.com/spf13/cobra"
)

type gradedTag struct {
	Tag              string  `json:"tag"`
	OpportunityScore float64 `json:"opportunity_score"`
	DemandScore      float64 `json:"demand_score"`
	CompetitionScore float64 `json:"competition_score"`
	VelocityScore    float64 `json:"velocity_score"`
	Grade            string  `json:"grade"`
}

type auditView struct {
	ListingID  int64       `json:"listing_id"`
	TagCount   int         `json:"tag_count"`
	GradedTags []gradedTag `json:"graded_tags"`
	WeakTags   []string    `json:"weak_tags"`
	StrongTags []string    `json:"strong_tags"`
	Note       string      `json:"note,omitempty"`
}

func newNovelListingsAuditCmd(flags *rootFlags) *cobra.Command {
	var maxTags int
	cmd := &cobra.Command{
		Use:   "audit <listing-id>",
		Short: "Audit a listing's tags: extract the tags it uses and grade each by ListingView's opportunity, demand, competition, and velocity scores.",
		Long: "Extract the tags a listing uses and grade each by ListingView's opportunity, demand, competition, and velocity scores, flagging dead-weight tags worth swapping.\n\n" +
			"Works on any listing (yours or a rival's). For the consensus tag set across a term's top sellers use 'tags consensus'; for missing tags vs a competitor shop use 'gaps'.",
		Example:     "  listingview-pp-cli listings audit 1581100221 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would extract a listing's tags and grade each by opportunity/demand/competition")
				return nil
			}
			if len(args) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a listing id is required"))
			}
			id, perr := strconv.ParseInt(args[0], 10, 64)
			if perr != nil {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("listing id must be a number: %q", args[0]))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			extractData, err := callProxyPOST(ctx, c, "tag-extractor", map[string]any{"listingId": id})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			tags := stringsOf(extractData, "tags")
			view := auditView{ListingID: id, TagCount: len(tags), GradedTags: []gradedTag{}, WeakTags: []string{}, StrongTags: []string{}}
			if len(tags) == 0 {
				view.Note = "no extractable tags found for this listing"
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			cap := maxTags
			if cliutil.IsDogfoodEnv() && cap > 3 {
				cap = 3
			}
			for i, tag := range tags {
				if cap > 0 && i >= cap {
					break
				}
				aData, aerr := callProxyPOST(ctx, c, "tag-analyzer", map[string]any{"tag": tag})
				if aerr != nil {
					continue
				}
				gt := gradedTag{
					Tag:              tag,
					OpportunityScore: numOf(aData, "opportunityScore"),
					DemandScore:      numOf(aData, "demandScore"),
					CompetitionScore: numOf(aData, "competitionScore"),
					VelocityScore:    numOf(aData, "velocityScore"),
				}
				gt.Grade = gradeTag(gt)
				view.GradedTags = append(view.GradedTags, gt)
				switch gt.Grade {
				case "weak":
					view.WeakTags = append(view.WeakTags, tag)
				case "strong":
					view.StrongTags = append(view.StrongTags, tag)
				}
			}
			sort.SliceStable(view.GradedTags, func(i, j int) bool {
				return view.GradedTags[i].OpportunityScore > view.GradedTags[j].OpportunityScore
			})
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().IntVar(&maxTags, "max-tags", 15, "Maximum tags to grade (each grade is one API call).")
	return cmd
}

func gradeTag(g gradedTag) string {
	if g.OpportunityScore == 0 && g.DemandScore == 0 && g.CompetitionScore == 0 {
		return "unknown"
	}
	switch {
	case g.OpportunityScore >= 60:
		return "strong"
	case g.OpportunityScore < 40:
		return "weak"
	default:
		return "ok"
	}
}
