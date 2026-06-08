// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// pp:data-source auto
//
// targetRow is one valid conversion destination for a source EPSG.
type targetRow struct {
	Epsg        int    `json:"epsg"`
	Description string `json:"description"`
	DatumFamily string `json:"datum_family"`
	Kind        string `json:"kind"`
}

func newNovelTargetsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "targets <epsg>",
		Short: "List the reference systems you can convert a given EPSG into",
		Long: "List the valid conversion destinations for a source reference system. The IGM\n" +
			"service rejects conversions between systems that share the same datum (e.g.\n" +
			"Monte Mario geographic -> Monte Mario / Gauss-Boaga), so this command returns\n" +
			"only the systems in a different datum family.",
		Example: strings.Trim(`
  verto-pp-cli targets 3003
  verto-pp-cli targets 6706 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) != 1 {
				return usageErr(fmt.Errorf("provide exactly one source EPSG"))
			}
			if dryRunOK(flags) {
				return nil
			}
			epsg, err := parseEpsg(args[0])
			if err != nil {
				return usageErr(err)
			}
			srcFamily := datumFamily(epsg)
			if srcFamily == "" {
				return usageErr(fmt.Errorf("EPSG %d is not one of the IGM-supported systems; run 'verto-pp-cli systems'", epsg))
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			systems, _ := getSystems(ctx, flags, false) // live descriptions optional

			rows := make([]targetRow, 0, len(refTable))
			for code, ref := range refTable {
				if ref.Family == srcFamily {
					continue // same datum: rejected by the service
				}
				desc := ref.Name
				for _, s := range systems {
					if s.Epsg == code {
						desc = strings.TrimSpace(s.Descrizione)
						break
					}
				}
				rows = append(rows, targetRow{Epsg: code, Description: desc, DatumFamily: ref.Family, Kind: ref.Kind})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].Epsg < rows[j].Epsg })

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				items := make([]map[string]any, len(rows))
				for i, r := range rows {
					items[i] = map[string]any{"epsg": r.Epsg, "description": r.Description, "datum_family": r.DatumFamily, "kind": r.Kind}
				}
				return printAutoTable(cmd.OutOrStdout(), items)
			}
			return flags.printJSON(cmd, rows)
		},
	}
	return cmd
}
