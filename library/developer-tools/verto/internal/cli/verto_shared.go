// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/store"
	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
)

// getSystems returns the supported reference systems, preferring the offline
// cache. On a cache miss it fetches live and persists the result. Pass
// refresh=true to force a live fetch (used by `systems --refresh`).
func getSystems(ctx context.Context, flags *rootFlags, refresh bool) ([]volapi.System, error) {
	if !refresh {
		cached, err := store.LoadSystems()
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	c, err := flags.newClient()
	if err != nil {
		return nil, err
	}
	info, err := volapi.Info(ctx, c)
	if err != nil {
		// Offline and no cache: actionable error.
		if cached, cErr := store.LoadSystems(); cErr == nil && len(cached) > 0 {
			return cached, nil
		}
		return nil, classifyAPIError(err, flags)
	}
	_ = store.SaveSystems(info.SrsSupportati)
	return info.SrsSupportati, nil
}

// systemByEpsg returns the system with the given EPSG, if known.
func systemByEpsg(systems []volapi.System, epsg int) (volapi.System, bool) {
	for _, s := range systems {
		if s.Epsg == epsg {
			return s, true
		}
	}
	return volapi.System{}, false
}

// parseEpsg parses an EPSG code from a flag value, accepting an optional
// "EPSG:" prefix (e.g. "EPSG:3003" or "3003").
func parseEpsg(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.ToUpper(s), "EPSG:")
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid EPSG code %q: want an integer like 3003", s)
	}
	return v, nil
}

// requireEpsgFlags parses and validates the --from/--to EPSG flags against the
// known systems (offline). Unknown codes fail with the list hint.
func requireEpsgFlags(ctx context.Context, flags *rootFlags, fromStr, toStr string) (in, out int, err error) {
	if strings.TrimSpace(fromStr) == "" || strings.TrimSpace(toStr) == "" {
		return 0, 0, usageErr(fmt.Errorf("both --from and --to EPSG codes are required (e.g. --from 3003 --to 6706); run 'verto-pp-cli systems' to list them"))
	}
	in, err = parseEpsg(fromStr)
	if err != nil {
		return 0, 0, usageErr(err)
	}
	out, err = parseEpsg(toStr)
	if err != nil {
		return 0, 0, usageErr(err)
	}
	systems, sErr := getSystems(ctx, flags, false)
	if sErr == nil && len(systems) > 0 {
		if _, ok := systemByEpsg(systems, in); !ok {
			return 0, 0, usageErr(fmt.Errorf("--from %d is not a supported reference system; run 'verto-pp-cli systems'", in))
		}
		if _, ok := systemByEpsg(systems, out); !ok {
			return 0, 0, usageErr(fmt.Errorf("--to %d is not a supported reference system; run 'verto-pp-cli systems'", out))
		}
	}
	return in, out, nil
}

// vertoErr maps a volapi service error to a typed CLI error (exit 2, invalid
// input) with a coverage-specific hint for the out-of-grid case.
func vertoErr(err error) error {
	var ve *volapi.Error
	if e, ok := err.(*volapi.Error); ok {
		ve = e
	}
	if ve == nil {
		return apiErr(err)
	}
	if ve.OutsideGrid() {
		return usageErr(fmt.Errorf("%s\nhint: the IGM grid covers Italy and its surrounding seas only. "+
			"Check the source EPSG and axis order with 'verto-pp-cli detect <e> <n>' and 'verto-pp-cli inspect <epsg>'", ve.Error()))
	}
	return usageErr(ve)
}

// convertWithCache converts coords using the offline conversion cache when
// enabled (default). Cache hits avoid network calls entirely, enabling offline
// replay. Misses are converted in one batched call and written back.
func convertWithCache(ctx context.Context, flags *rootFlags, in, out int, coords []volapi.Coord) ([]volapi.Coord, error) {
	results := make([]volapi.Coord, len(coords))
	useCache := flags == nil || !flags.noCache

	var cache *store.ConvCache
	var missIdx []int
	var missCoords []volapi.Coord

	if useCache {
		var err error
		cache, err = store.LoadConvCache()
		if err != nil {
			useCache = false // corrupt cache: fall back to all-live
		}
	}

	for i, co := range coords {
		if useCache {
			if got, ok := cache.Get(in, out, co.E, co.N); ok {
				results[i] = got
				continue
			}
		}
		missIdx = append(missIdx, i)
		missCoords = append(missCoords, co)
	}

	if len(missCoords) > 0 {
		c, err := flags.newClient()
		if err != nil {
			return nil, err
		}
		converted, err := volapi.Convert(ctx, c, in, out, missCoords)
		if err != nil {
			if _, ok := err.(*volapi.Error); ok {
				return nil, vertoErr(err)
			}
			return nil, classifyAPIError(err, flags)
		}
		for j, idx := range missIdx {
			results[idx] = converted[j]
			if useCache {
				cache.Put(in, out, missCoords[j].E, missCoords[j].N, converted[j])
			}
		}
		if useCache {
			_ = cache.Save()
		}
	}
	return results, nil
}

// fmtCoord formats a coordinate value, trimming trailing zeros for readability
// while preserving full precision.
func fmtCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
