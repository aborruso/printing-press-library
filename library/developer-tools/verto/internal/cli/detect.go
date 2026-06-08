// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// pp:data-source local
// pp:novel-static-reference
//
// detect is a pure offline magnitude heuristic by design: it classifies a
// coordinate's likely source system from the ranges of its e/n values, with no
// API call or store access. It is not a reimplementation of a server endpoint.
//
// detectResult is the heuristic guess for a coordinate's source system.
type detectResult struct {
	E          float64  `json:"e"`
	N          float64  `json:"n"`
	Kind       string   `json:"kind"`       // "geographic", "projected", or "unknown"
	Confidence string   `json:"confidence"` // "low", "medium"
	Candidates []int    `json:"candidate_epsg"`
	Reason     string   `json:"reason"`
	Note       string   `json:"note"`
	InItaly    bool     `json:"plausibly_in_italy"`
	Names      []string `json:"candidate_names"`
}

func newNovelDetectCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect <e> <n>",
		Short: "Guess the likely source reference system of a coordinate from its magnitude",
		Long: "Heuristically guess which reference system a coordinate is in, from the\n" +
			"magnitude and ranges of its e (est) and n (nord) values. Useful when a dataset\n" +
			"is labeled vaguely ('UTM', 'Gauss-Boaga') and you must recover its real EPSG.\n\n" +
			"This is a HINT, not an authority: easting magnitudes overlap across systems, so\n" +
			"detect returns candidates plus a confidence. Confirm with 'verto-pp-cli inspect'.",
		Example: strings.Trim(`
  verto-pp-cli detect 12.4924 41.8902     # geographic (decimal degrees)
  verto-pp-cli detect 2300000 4640000     # projected (Gauss-Boaga belt)
  verto-pp-cli detect 514827 5034611 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) != 2 {
				return usageErr(fmt.Errorf("provide exactly two values: <e> <n>"))
			}
			if dryRunOK(flags) {
				return nil
			}
			e, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return usageErr(fmt.Errorf("invalid e value %q", args[0]))
			}
			n, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return usageErr(fmt.Errorf("invalid n value %q", args[1]))
			}

			res := classifyCoord(e, n)

			// Attach candidate names from the static table.
			for _, c := range res.Candidates {
				if r, ok := refByEpsg(c); ok {
					res.Names = append(res.Names, fmt.Sprintf("%d %s", c, r.Name))
				}
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				out := cmd.OutOrStdout()
				fmt.Fprintf(out, "input        : e=%s  n=%s\n", fmtCoord(e), fmtCoord(n))
				fmt.Fprintf(out, "kind         : %s (confidence: %s)\n", res.Kind, res.Confidence)
				fmt.Fprintf(out, "in Italy?    : %v\n", res.InItaly)
				fmt.Fprintf(out, "reason       : %s\n", res.Reason)
				if len(res.Names) > 0 {
					fmt.Fprintf(out, "candidates   :\n")
					for _, nm := range res.Names {
						fmt.Fprintf(out, "  - %s\n", nm)
					}
				}
				if res.Note != "" {
					fmt.Fprintf(out, "note         : %s\n", res.Note)
				}
				return nil
			}
			return flags.printJSON(cmd, res)
		},
	}
	return cmd
}

// classifyCoord applies grounded magnitude heuristics (calibrated against live
// IGM conversions of Rome and Milan reference points) to guess the system.
func classifyCoord(e, n float64) detectResult {
	res := detectResult{E: e, N: n, Confidence: "low"}
	ae, an := math.Abs(e), math.Abs(n)

	// Geographic: decimal degrees.
	if ae <= 180 && an <= 90 {
		res.Kind = "geographic"
		res.InItaly = e >= 5.5 && e <= 19.5 && n >= 35 && n <= 48
		res.Candidates = []int{4265, 4806, 4230, 4670, 6706}
		res.Reason = "both values are within degree range (|e|<=180, |n|<=90)"
		res.Note = "Geographic systems differ only by datum; values alone cannot tell Roma40 from ED50/IGM95/RDN2008. Confirm the datum from the dataset's metadata."
		if !res.InItaly {
			res.Note = "Coordinate is outside Italy's geographic bounds; the IGM grid will reject it. " + res.Note
		}
		return res
	}

	res.Kind = "projected"
	// Projected northing for Italian UTM/TM/Gauss-Boaga is ~3.9M–5.3M; LAEA/LCC
	// use a very different northing (~1.5M–3.5M).
	italyNorthing := an >= 3_900_000 && an <= 5_300_000
	res.InItaly = italyNorthing || (ae >= 3_800_000 && ae <= 4_700_000 && an >= 1_500_000 && an <= 3_500_000)

	switch {
	case ae >= 100_000 && ae <= 950_000 && italyNorthing:
		res.Candidates = []int{23032, 23033, 23034, 3064, 3065, 9716, 6707, 6708, 6709}
		res.Reason = "easting ~0.1–0.95M with northing ~3.9–5.3M: UTM/TM zone (32/33/34) on ED50, IGM95 or RDN2008"
		res.Note = "Easting cannot distinguish the UTM zone (each zone resets near 500 000) nor the datum. Use the dataset's stated zone/datum, or inspect a candidate."
	case ae >= 1_200_000 && ae < 2_000_000 && italyNorthing:
		res.Candidates = []int{3003}
		res.Confidence = "medium"
		res.Reason = "easting ~1.2–2.0M with Italian northing: Gauss-Boaga fuso Ovest (Monte Mario / Italy zone 1)"
	case ae >= 2_000_000 && ae < 2_700_000 && italyNorthing:
		res.Candidates = []int{3004}
		res.Confidence = "medium"
		res.Reason = "easting ~2.0–2.7M with Italian northing: Gauss-Boaga fuso Est (Monte Mario / Italy zone 2)"
	case ae >= 2_700_000 && ae < 3_200_000 && italyNorthing:
		res.Candidates = []int{6876, 3004}
		res.Reason = "easting ~2.7–3.2M: RDN2008 Zone 12 (also overlaps Gauss-Boaga fuso Est)"
	case ae >= 3_800_000 && ae <= 4_700_000 && an >= 1_500_000 && an <= 3_500_000:
		res.Candidates = []int{3035, 3034}
		res.Confidence = "medium"
		res.Reason = "easting ~3.8–4.7M with low northing ~1.5–3.5M: ETRS89 LAEA or LCC (pan-European grid)"
	case ae >= 6_500_000 && ae <= 7_200_000 && italyNorthing:
		res.Candidates = []int{7794}
		res.Confidence = "medium"
		res.Reason = "easting ~6.5–7.2M: RDN2008 Italy Zone (E-N), single national zone"
	default:
		res.Kind = "unknown"
		res.Reason = "the e/n magnitudes do not match any known Italian projected range; the coordinate may be in a non-Italian system or have swapped axes"
		res.Note = "Try swapping e and n, or check the dataset's stated CRS."
	}

	sort.Ints(res.Candidates)
	if res.Note == "" {
		res.Note = "Heuristic only — confirm with 'verto-pp-cli inspect <epsg>'."
	}
	return res
}
