// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"testing"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
)

func TestSystemsRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	want := []volapi.System{{Epsg: 4265, Descrizione: "Monte Mario"}, {Epsg: 6706, Descrizione: "RDN2008 2D geo"}}
	if err := SaveSystems(want); err != nil {
		t.Fatalf("SaveSystems: %v", err)
	}
	got, err := LoadSystems()
	if err != nil {
		t.Fatalf("LoadSystems: %v", err)
	}
	if len(got) != 2 || got[0].Epsg != 4265 || got[1].Descrizione != "RDN2008 2D geo" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestLoadSystemsMissing(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	got, err := LoadSystems()
	if err != nil {
		t.Fatalf("LoadSystems on missing: %v", err)
	}
	if got != nil {
		t.Errorf("want nil for missing cache, got %+v", got)
	}
}

func TestConvCachePutGetSave(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c, err := LoadConvCache()
	if err != nil {
		t.Fatalf("LoadConvCache: %v", err)
	}
	if _, ok := c.Get(3003, 6707, 1.0, 2.0); ok {
		t.Fatal("unexpected hit on empty cache")
	}
	c.Put(3003, 6707, 1.0, 2.0, volapi.Coord{E: 10.5, N: 20.5})
	got, ok := c.Get(3003, 6707, 1.0, 2.0)
	if !ok || got.E != 10.5 || got.N != 20.5 {
		t.Errorf("Get after Put = %+v ok=%v", got, ok)
	}
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1", c.Len())
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload from disk and confirm persistence.
	c2, err := LoadConvCache()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got, ok := c2.Get(3003, 6707, 1.0, 2.0); !ok || got.E != 10.5 {
		t.Errorf("persisted Get = %+v ok=%v", got, ok)
	}

	// Different key must miss.
	if _, ok := c2.Get(3003, 6707, 1.0, 2.5); ok {
		t.Error("distinct-coordinate key should miss")
	}
}

func TestConvCacheStatsAndClear(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c, _ := LoadConvCache()
	c.Put(1, 2, 3, 4, volapi.Coord{E: 5, N: 6})
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	count, _, size, err := ConvCacheStats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 || size <= 0 {
		t.Errorf("Stats count=%d size=%d", count, size)
	}
	_, removed, err := ClearConvCache()
	if err != nil || !removed {
		t.Fatalf("Clear: removed=%v err=%v", removed, err)
	}
	count, _, _, _ = ConvCacheStats()
	if count != 0 {
		t.Errorf("count after clear = %d, want 0", count)
	}
}
