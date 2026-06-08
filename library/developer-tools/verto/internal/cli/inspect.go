// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// pp:data-source auto
//
// inspectCard is the resolved metadata for one EPSG, joining the static
// reference table with the live IGM description.
type inspectCard struct {
	Epsg         int    `json:"epsg"`
	Description  string `json:"description"`
	Name         string `json:"name"`
	DatumFamily  string `json:"datum_family"`
	Kind         string `json:"kind"`
	Unit         string `json:"unit"`
	Zone         string `json:"zone,omitempty"`
	FalseEasting string `json:"false_easting,omitempty"`
	AxisOrder    string `json:"axis_order"`
	Notes        string `json:"notes,omitempty"`
}

func newNovelInspectCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <epsg> [<epsg> ...]",
		Short: "Show datum family, axis order, units, zone and false easting for a reference system",
		Long: "Inspect one or more reference systems by EPSG code. Disambiguates look-alike\n" +
			"systems (3003 vs 3004, 23032 vs 23033) before a conversion goes wrong by\n" +
			"showing the datum family (which controls valid conversion targets), the axis\n" +
			"order (e=est/lon, n=nord/lat), units, zone and false easting.",
		Example: strings.Trim(`
  verto-pp-cli inspect 3003
  verto-pp-cli inspect 3003 3004 --json
  verto-pp-cli inspect 6707 --json --select epsg,datum_family,false_easting
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			// Live descriptions are a nice-to-have; an offline inspect still
			// works entirely from the static reference table.
			systems, _ := getSystems(ctx, flags, false)

			cards := make([]inspectCard, 0, len(args))
			for _, a := range args {
				epsg, err := parseEpsg(a)
				if err != nil {
					return usageErr(err)
				}
				ref, ok := refByEpsg(epsg)
				if !ok {
					return usageErr(fmt.Errorf("EPSG %d is not one of the IGM-supported systems; run 'verto-pp-cli systems'", epsg))
				}
				desc := ref.Name
				for _, s := range systems {
					if s.Epsg == epsg {
						desc = strings.TrimSpace(s.Descrizione)
						break
					}
				}
				axis := "e=est/longitude, n=nord/latitude (decimal degrees)"
				if ref.Kind == "projected" {
					axis = "e=easting (m), n=northing (m)"
				}
				cards = append(cards, inspectCard{
					Epsg: epsg, Description: desc, Name: ref.Name, DatumFamily: ref.Family,
					Kind: ref.Kind, Unit: ref.Unit, Zone: ref.Zone, FalseEasting: ref.FalseEasting,
					AxisOrder: axis, Notes: ref.Notes,
				})
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				out := cmd.OutOrStdout()
				for i, c := range cards {
					if i > 0 {
						fmt.Fprintln(out)
					}
					fmt.Fprintf(out, "EPSG %d — %s\n", c.Epsg, c.Description)
					fmt.Fprintf(out, "  datum family : %s\n", c.DatumFamily)
					fmt.Fprintf(out, "  kind         : %s (%s)\n", c.Kind, c.Unit)
					if c.Zone != "" {
						fmt.Fprintf(out, "  zone         : %s\n", c.Zone)
					}
					if c.FalseEasting != "" {
						fmt.Fprintf(out, "  false easting: %s\n", c.FalseEasting)
					}
					fmt.Fprintf(out, "  axis order   : %s\n", c.AxisOrder)
					if c.Notes != "" {
						fmt.Fprintf(out, "  notes        : %s\n", c.Notes)
					}
				}
				return nil
			}
			if len(cards) == 1 {
				return flags.printJSON(cmd, cards[0])
			}
			return flags.printJSON(cmd, cards)
		},
	}
	return cmd
}
