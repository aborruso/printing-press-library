// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"
)

func TestNovelDetectCommand(t *testing.T) {
	cmd := newNovelDetectCmd(&rootFlags{})
	if !strings.HasPrefix(cmd.Use, "detect") {
		t.Errorf("Use = %q, want prefix 'detect'", cmd.Use)
	}
	if cmd.RunE == nil {
		t.Fatal("RunE is nil")
	}
	if cmd.Annotations["mcp:read-only"] != "true" {
		t.Error("detect should be read-only")
	}
	// Wrong arg count is a usage error, not a panic or a network call.
	cmd.SetArgs([]string{"1"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	if err := cmd.Execute(); err == nil {
		t.Error("expected usage error for single positional arg")
	}
}
