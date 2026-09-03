// Hand-authored novel command. Snapshot drift diff over local research history.
// pp:data-source local
package cli

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/cliutil"

	"github.com/spf13/cobra"
)

type driftChange struct {
	Kind      string  `json:"kind"`
	Key       string  `json:"key"`
	Metric    string  `json:"metric"`
	From      float64 `json:"from"`
	To        float64 `json:"to"`
	Delta     float64 `json:"delta"`
	PctChange float64 `json:"pct_change"`
	AgoHours  float64 `json:"prev_snapshot_age_hours"`
}

type driftView struct {
	Window      string        `json:"window,omitempty"`
	KeysTracked int           `json:"keys_tracked"`
	Changes     []driftChange `json:"changes"`
	Note        string        `json:"note,omitempty"`
}

func newNovelDriftCmd(flags *rootFlags) *cobra.Command {
	var since string
	var minPct float64
	var dbPath string
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Diff your saved keyword/tag research snapshots: volume changes, competition shifts, and score movement over time.",
		Long: "Compare the two most recent snapshots for every term/tag you've researched and report what moved — search-volume changes, competition shifts, opportunity/velocity score movement.\n\n" +
			"Snapshots accumulate every time you run research commands like 'niche' and 'tags rising'. Detects change across cached entities. For a one-shot verdict on a new term use 'niche'; for the best static opportunities use 'opportunities'.",
		Example:     "  listingview-pp-cli drift --since 7d --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would diff the two most recent snapshots per researched term/tag")
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			_ = ctx
			resolved := dbPath
			if resolved == "" {
				resolved = defaultDBPath("listingview-pp-cli")
			}
			if _, statErr := os.Stat(resolved); os.IsNotExist(statErr) {
				return emptyDriftHint(cmd, flags, resolved)
			}
			var window time.Duration
			if since != "" {
				d, perr := cliutil.ParseDurationLoose(since)
				if perr != nil {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("invalid --since %q: %w", since, perr))
				}
				window = d
			}
			db := tryOpenStore(resolved)
			if db == nil {
				return emptyDriftHint(cmd, flags, resolved)
			}
			defer db.Close()
			keys, err := distinctKeys(db.DB(), 2)
			if err != nil {
				return fmt.Errorf("reading snapshot history: %w", err)
			}
			view := driftView{KeysTracked: len(keys), Changes: []driftChange{}}
			if since != "" {
				view.Window = since
			}
			now := time.Now()
			for _, k := range keys {
				hist, herr := historyForKey(db.DB(), k.Kind, k.Key)
				if herr != nil || len(hist) < 2 {
					continue
				}
				// Compare the latest snapshot against the most recent one taken at
				// a different time (same-second snapshots carry identical values).
				latest := hist[0]
				var prev snapRecord
				foundPrev := false
				for _, h := range hist[1:] {
					if h.FetchedAt < latest.FetchedAt {
						prev = h
						foundPrev = true
						break
					}
				}
				if !foundPrev {
					continue
				}
				if window > 0 && now.Sub(time.Unix(latest.FetchedAt, 0)) > window {
					continue // latest snapshot older than the window
				}
				ageHours := round2(time.Unix(latest.FetchedAt, 0).Sub(time.Unix(prev.FetchedAt, 0)).Hours())
				for metric, to := range latest.Metrics {
					from, ok := prev.Metrics[metric]
					if !ok || from == to {
						continue
					}
					delta := to - from
					pct := 0.0
					if from != 0 {
						pct = round2(delta / from * 100)
					}
					if minPct > 0 && (pct < minPct && pct > -minPct) {
						continue
					}
					view.Changes = append(view.Changes, driftChange{
						Kind: k.Kind, Key: k.Key, Metric: metric,
						From: round2(from), To: round2(to), Delta: round2(delta), PctChange: pct, AgoHours: ageHours,
					})
				}
			}
			sort.SliceStable(view.Changes, func(i, j int) bool {
				return abs(view.Changes[i].PctChange) > abs(view.Changes[j].PctChange)
			})
			if len(view.Changes) == 0 {
				view.Note = "no drift detected yet — re-run research commands (e.g. 'niche', 'tags rising') over time to build snapshot history"
			}
			return printJSONFiltered(cmd.OutOrStdout(), view, flags)
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "Only report keys whose latest snapshot is within this window (e.g. 7d, 24h, 1w).")
	cmd.Flags().Float64Var(&minPct, "min-pct", 0, "Only report changes whose magnitude is at least this percent.")
	cmd.Flags().StringVar(&dbPath, "db", "", "Local store path (defaults to the standard location).")
	return cmd
}

func emptyDriftHint(cmd *cobra.Command, flags *rootFlags, path string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "no research history at %s\nrun research commands first, e.g.: listingview-pp-cli niche \"sticker\"\n", path)
	return printJSONFiltered(cmd.OutOrStdout(), driftView{Changes: []driftChange{}, Note: "no research history yet"}, flags)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
