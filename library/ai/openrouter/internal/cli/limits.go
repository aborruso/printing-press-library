// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelLimitsCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "limits",
		Short:       "Runway and rate-limit headroom",
		Example:     "  openrouter-pp-cli limits status --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newNovelLimitsStatusCmd(flags))
	return cmd
}
