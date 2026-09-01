// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"testing"
	"time"
)

func TestClampRateNonSuperaIlTetto(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, MaxRequestsPerSecond},
		{-5, MaxRequestsPerSecond},
		{100, MaxRequestsPerSecond},
		{MaxRequestsPerSecond, MaxRequestsPerSecond},
		{0.25, 0.25},
	}
	for _, c := range cases {
		if got := ClampRate(c.in); got != c.want {
			t.Errorf("ClampRate(%v) = %v, atteso %v", c.in, got, c.want)
		}
	}
}

func TestPaceDistanziaLeChiamate(t *testing.T) {
	// Il primo giro fissa solo l'istante di partenza: l'attesa si misura
	// sui due successivi.
	Pace()
	start := time.Now()
	Pace()
	Pace()
	elapsed := time.Since(start)
	want := 2 * minInterval
	// Margine per l'imprecisione dello scheduler.
	if elapsed < want-50*time.Millisecond {
		t.Errorf("tre chiamate a Pace sono durate %s, attese almeno %s", elapsed, want)
	}
}

func TestAcquireSingleInstanceEIdempotenteNelProcesso(t *testing.T) {
	if err := AcquireSingleInstance(); err != nil {
		t.Fatalf("primo acquire fallito: %v", err)
	}
	if err := AcquireSingleInstance(); err != nil {
		t.Fatalf("secondo acquire nello stesso processo deve essere un no-op: %v", err)
	}
}
