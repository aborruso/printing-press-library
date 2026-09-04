// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelUsageCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "usage",
		Short:       "Cost governance for agent fleets",
		Example:     "  openrouter-pp-cli usage anomaly --since 24h --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newNovelUsageAnomalyCmd(flags))
	cmd.AddCommand(newNovelUsageCostByCmd(flags))
	cmd.AddCommand(newNovelUsageReconcileCmd(flags))
	return cmd
}
