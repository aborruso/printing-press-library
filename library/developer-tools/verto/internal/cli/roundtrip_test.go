// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"math"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
)

func TestNovelRoundtripCommand(t *testing.T) {
	cmd := newNovelRoundtripCmd(&rootFlags{})
	if !strings.HasPrefix(cmd.Use, "roundtrip") {
		t.Errorf("Use = %q, want prefix 'roundtrip'", cmd.Use)
	}
	if cmd.Flags().Lookup("from") == nil || cmd.Flags().Lookup("to") == nil {
		t.Error("roundtrip must expose --from and --to")
	}
}

func TestResidualMetres(t *testing.T) {
	// Projected: Euclidean norm in metres.
	if got := residualMetres(volapi.Coord{E: 1000, N: 2000}, 3, 4, "metre"); math.Abs(got-5) > 1e-9 {
		t.Errorf("projected residual = %v, want 5", got)
	}
	// Geographic: a degree of latitude is ~110 km, so a tiny degree delta maps
	// to a small but non-zero metre distance.
	got := residualMetres(volapi.Coord{E: 12, N: 42}, 0, 1e-5, "degree")
	if got <= 0 || got > 2 {
		t.Errorf("geographic residual = %v, want ~1.1 m", got)
	}
}
