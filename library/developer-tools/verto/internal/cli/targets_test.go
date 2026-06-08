// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"
)

func TestNovelTargetsCommand(t *testing.T) {
	cmd := newNovelTargetsCmd(&rootFlags{})
	if !strings.HasPrefix(cmd.Use, "targets") {
		t.Errorf("Use = %q, want prefix 'targets'", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatal("RunE is nil")
	}
	if cmd.Annotations["mcp:read-only"] != "true" {
		t.Error("targets should be read-only")
	}
	// An unknown source EPSG fails offline (datumFamily lookup), no network.
	cmd.SetArgs([]string{"999999"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	if err := cmd.Execute(); err == nil {
		t.Error("expected usage error for unknown source EPSG")
	}
}
