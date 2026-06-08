// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"math"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
	"github.com/spf13/cobra"
)

// pp:data-source auto
//
// roundtripRow is the residual report for one coordinate after an A->B->A trip.
type roundtripRow struct {
	InE       float64 `json:"in_e"`
	InN       float64 `json:"in_n"`
	BackE     float64 `json:"back_e"`
	BackN     float64 `json:"back_n"`
	ResidualE float64 `json:"residual_e"`
	ResidualN float64 `json:"residual_n"`
	ResidualM float64 `json:"residual_m"`
}

type roundtripReport struct {
	From       int            `json:"from_epsg"`
	To         int            `json:"to_epsg"`
	Unit       string         `json:"source_unit"`
	Rows       []roundtripRow `json:"points"`
	MaxResidM  float64        `json:"max_residual_m"`
	MeanResidM float64        `json:"mean_residual_m"`
}

func newNovelRoundtripCmd(flags *rootFlags) *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:   "roundtrip [<e> <n> ...]",
		Short: "Convert a coordinate A->B->A and report the residual error of the datum transform",
		Long: "Convert each coordinate from --from to --to and back, then report the residual\n" +
			"error (Δe, Δn, and an approximate distance in metres) introduced by the\n" +
			"round trip. Use it to certify a datum chain is lossless within tolerance\n" +
			"before publishing or depositing data.",
		Example: strings.Trim(`
  verto-pp-cli roundtrip --from 3003 --to 6707 1500000 4640000
  verto-pp-cli roundtrip --from 4265 --to 6706 12.4924 41.8902 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			in, out, err := requireEpsgFlags(ctx, flags, from, to)
			if err != nil {
				return err
			}
			coords, err := readCoordArgs(cmd, args)
			if err != nil {
				return usageErr(err)
			}
			if len(coords) == 0 {
				return cmd.Help()
			}

			forward, err := convertWithCache(ctx, flags, in, out, coords)
			if err != nil {
				return err
			}
			back, err := convertWithCache(ctx, flags, out, in, forward)
			if err != nil {
				return err
			}

			unit := "metre"
			if r, ok := refByEpsg(in); ok && r.Kind == "geographic" {
				unit = "degree"
			}

			report := roundtripReport{From: in, To: out, Unit: unit}
			var sum float64
			for i := range coords {
				de := back[i].E - coords[i].E
				dn := back[i].N - coords[i].N
				dm := residualMetres(coords[i], de, dn, unit)
				report.Rows = append(report.Rows, roundtripRow{
					InE: coords[i].E, InN: coords[i].N, BackE: back[i].E, BackN: back[i].N,
					ResidualE: math.Abs(de), ResidualN: math.Abs(dn), ResidualM: dm,
				})
				sum += dm
				if dm > report.MaxResidM {
					report.MaxResidM = dm
				}
			}
			report.MeanResidM = sum / float64(len(report.Rows))

			return flags.printJSON(cmd, report)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Source EPSG code")
	cmd.Flags().StringVar(&to, "to", "", "Target EPSG code")
	return cmd
}

// residualMetres approximates the residual distance in metres. For projected
// (metre) sources it is the Euclidean norm; for geographic (degree) sources it
// scales degrees to metres using a latitude-aware approximation.
func residualMetres(in volapi.Coord, de, dn float64, unit string) float64 {
	if unit == "degree" {
		latRad := in.N * math.Pi / 180
		mPerDegLon := 111320.0 * math.Cos(latRad)
		mPerDegLat := 110540.0
		dx := de * mPerDegLon
		dy := dn * mPerDegLat
		return math.Hypot(dx, dy)
	}
	return math.Hypot(de, dn)
}
