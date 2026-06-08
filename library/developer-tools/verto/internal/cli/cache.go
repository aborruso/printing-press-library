// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/store"
	"github.com/spf13/cobra"
)

// pp:data-source local
//
// newNovelCacheCmd manages the on-disk JSON conversion cache; it reads and
// clears local store state only, never the API.
func newNovelCacheCmd(flags *rootFlags) *cobra.Command {
	var clear bool
	var stats bool // default action; flag exists for explicitness/discoverability

	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clear the local conversion cache (offline, reproducible replay)",
		Long: "Every convert/batch writes its results to a local write-through cache keyed by\n" +
			"(inEpsg, outEpsg, e, n). Cached conversions are served instantly and offline, so\n" +
			"re-running the same job in CI never hits the live service. This command reports\n" +
			"cache statistics (default) or clears the cache (--clear).",
		Example: strings.Trim(`
  verto-pp-cli cache              # show entry count, path and size
  verto-pp-cli cache --json
  verto-pp-cli cache --clear      # delete the cache file
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if clear {
				path, removed, err := store.ClearConvCache()
				if err != nil {
					return apiErr(err)
				}
				result := map[string]any{"cleared": removed, "path": path}
				if wantsHumanTable(cmd.OutOrStdout(), flags) {
					if removed {
						fmt.Fprintf(cmd.OutOrStdout(), "Cleared conversion cache: %s\n", path)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "Conversion cache was already empty: %s\n", path)
					}
					return nil
				}
				return flags.printJSON(cmd, result)
			}

			count, path, size, err := store.ConvCacheStats()
			if err != nil {
				return apiErr(err)
			}
			result := map[string]any{"entries": count, "path": path, "size_bytes": size}
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				fmt.Fprintf(cmd.OutOrStdout(), "Conversion cache\n")
				fmt.Fprintf(cmd.OutOrStdout(), "  entries   : %d\n", count)
				fmt.Fprintf(cmd.OutOrStdout(), "  size      : %d bytes\n", size)
				fmt.Fprintf(cmd.OutOrStdout(), "  path      : %s\n", path)
				return nil
			}
			return flags.printJSON(cmd, result)
		},
	}
	cmd.Flags().BoolVar(&clear, "clear", false, "Delete the conversion cache file")
	cmd.Flags().BoolVar(&stats, "stats", false, "Show cache statistics (the default action)")
	return cmd
}
