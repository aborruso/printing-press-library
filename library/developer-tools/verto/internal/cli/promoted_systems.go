// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"sort"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
	"github.com/spf13/cobra"
)

func newSystemsPromotedCmd(flags *rootFlags) *cobra.Command {
	var search string
	var refresh bool

	cmd := &cobra.Command{
		Use:   "systems",
		Short: "List the reference systems supported by the IGM Verto Online service",
		Long: "List the 20 Italian reference systems the IGM Verto Online service supports\n" +
			"(Roma40/Monte Mario, ED50, IGM95, ETRS89, RDN2008), with their EPSG codes. The\n" +
			"list is cached locally after the first run, so subsequent calls work offline.\n" +
			"Filter with --search; force a live refresh with --refresh.",
		Example: strings.Trim(`
  verto-pp-cli systems
  verto-pp-cli systems --search ED50
  verto-pp-cli systems --json --select epsg,descrizione
`, "\n"),
		Annotations: map[string]string{"pp:endpoint": "systems.list", "pp:method": "POST", "pp:path": "/volapi", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			systems, err := getSystems(ctx, flags, refresh)
			if err != nil {
				return err
			}

			if q := strings.TrimSpace(search); q != "" {
				lq := strings.ToLower(q)
				filtered := systems[:0:0]
				for _, s := range systems {
					if strings.Contains(strings.ToLower(s.Descrizione), lq) || strings.Contains(strconv.Itoa(s.Epsg), q) {
						filtered = append(filtered, s)
					}
				}
				systems = filtered
			}
			sort.Slice(systems, func(i, j int) bool { return systems[i].Epsg < systems[j].Epsg })

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				items := make([]map[string]any, len(systems))
				for i, s := range systems {
					items[i] = map[string]any{"epsg": s.Epsg, "descrizione": strings.TrimSpace(s.Descrizione)}
				}
				return printAutoTable(cmd.OutOrStdout(), items)
			}
			// Normalize descriptions (the API embeds a leading space on some).
			normalized := make([]volapi.System, len(systems))
			for i, s := range systems {
				normalized[i] = volapi.System{Epsg: s.Epsg, Descrizione: strings.TrimSpace(s.Descrizione)}
			}
			return flags.printJSON(cmd, normalized)
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "Filter systems by EPSG or description substring")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force a live refresh of the cached systems list")

	return cmd
}
