// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

// Package store is a lightweight, dependency-free JSON-file cache for the verto
// CLI. The IGM Verto Online service exposes only a tiny amount of cacheable
// data (the 20-row supported-systems table) plus the conversion results the
// user produces, so a SQLite layer would be overkill. Two files under the
// user cache dir back the two caches:
//
//	systems.json     - the supported reference systems (offline `systems`)
//	conversions.json - a write-through map of conversion results (offline replay)
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
)

// Dir returns the cache directory for verto, creating it if necessary.
func Dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = filepath.Join(os.TempDir(), "verto-cache")
	}
	dir := filepath.Join(base, "verto-pp-cli")
	// 0o700: the cache can hold a history of converted coordinates (cadastral
	// parcels, addresses); keep it owner-only on multi-user systems.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func systemsPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "systems.json"), nil
}

// ConvCachePath returns the on-disk path of the conversion cache file.
func ConvCachePath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "conversions.json"), nil
}

// LoadSystems reads the cached reference systems. A missing file returns
// (nil, nil) so callers can fall back to a live fetch.
func LoadSystems() ([]volapi.System, error) {
	p, err := systemsPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var systems []volapi.System
	if err := json.Unmarshal(raw, &systems); err != nil {
		return nil, fmt.Errorf("reading cached systems: %w", err)
	}
	return systems, nil
}

// SaveSystems writes the reference systems to the cache atomically.
func SaveSystems(systems []volapi.System) error {
	p, err := systemsPath()
	if err != nil {
		return err
	}
	return writeJSONAtomic(p, systems)
}

// ConvCache is an in-memory view of the conversion cache, keyed by
// (inEpsg,outEpsg,e,n). Load once, query/mutate, then Save once.
type ConvCache struct {
	path    string
	entries map[string][2]float64
	dirty   bool
}

func convKey(inEpsg, outEpsg int, e, n float64) string {
	// strconv with -1 precision yields the shortest round-trippable form, so
	// identical inputs always produce identical keys.
	return strconv.Itoa(inEpsg) + ":" + strconv.Itoa(outEpsg) + ":" +
		strconv.FormatFloat(e, 'f', -1, 64) + ":" +
		strconv.FormatFloat(n, 'f', -1, 64)
}

// LoadConvCache reads the conversion cache. A missing file yields an empty
// cache, not an error.
func LoadConvCache() (*ConvCache, error) {
	p, err := ConvCachePath()
	if err != nil {
		return nil, err
	}
	c := &ConvCache{path: p, entries: map[string][2]float64{}}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &c.entries); err != nil {
		return nil, fmt.Errorf("reading conversion cache: %w", err)
	}
	return c, nil
}

// Get returns the cached result for one coordinate, if present.
func (c *ConvCache) Get(inEpsg, outEpsg int, e, n float64) (volapi.Coord, bool) {
	v, ok := c.entries[convKey(inEpsg, outEpsg, e, n)]
	if !ok {
		return volapi.Coord{}, false
	}
	return volapi.Coord{E: v[0], N: v[1]}, true
}

// Put records a conversion result in the cache (in memory; call Save to persist).
func (c *ConvCache) Put(inEpsg, outEpsg int, e, n float64, out volapi.Coord) {
	c.entries[convKey(inEpsg, outEpsg, e, n)] = [2]float64{out.E, out.N}
	c.dirty = true
}

// Len returns the number of cached conversions.
func (c *ConvCache) Len() int { return len(c.entries) }

// Path returns the backing file path.
func (c *ConvCache) Path() string { return c.path }

// Save persists the cache to disk if it changed.
func (c *ConvCache) Save() error {
	if !c.dirty {
		return nil
	}
	if err := writeJSONAtomic(c.path, c.entries); err != nil {
		return err
	}
	c.dirty = false
	return nil
}

// ConvCacheStats reports the entry count and on-disk size of the conversion
// cache without loading every entry into typed form.
func ConvCacheStats() (count int, path string, sizeBytes int64, err error) {
	c, err := LoadConvCache()
	if err != nil {
		return 0, "", 0, err
	}
	count = c.Len()
	path = c.path
	if fi, statErr := os.Stat(path); statErr == nil {
		sizeBytes = fi.Size()
	}
	return count, path, sizeBytes, nil
}

// ClearConvCache removes the conversion cache file. A missing file is a no-op.
func ClearConvCache() (path string, removed bool, err error) {
	p, err := ConvCachePath()
	if err != nil {
		return "", false, err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return p, false, nil
		}
		return p, false, err
	}
	return p, true, nil
}

func writeJSONAtomic(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
