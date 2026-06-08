// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"
)

func TestNovelInspectCommand(t *testing.T) {
	cmd := newNovelInspectCmd(&rootFlags{})
	if !strings.HasPrefix(cmd.Use, "inspect") {
		t.Errorf("Use = %q, want prefix 'inspect'", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatal("RunE is nil")
	}
	if cmd.Annotations["mcp:read-only"] != "true" {
		t.Error("inspect should be read-only")
	}
	// inspect of an unknown EPSG is a usage error, evaluated offline from the
	// static reference table (no network).
	cmd.SetArgs([]string{"999999"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	if err := cmd.Execute(); err == nil {
		t.Error("expected usage error for unknown EPSG")
	}
}
