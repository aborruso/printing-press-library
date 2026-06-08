// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

// Package volapi is a thin typed client over the generated HTTP client for the
// IGM Verto Online service (https://igmi.esercito.difesa.it/porta-magna/wps/volapi).
//
// The service is a single POST endpoint discriminated by the body field
// "richiesta": "info" lists the supported reference systems, "conversione"
// transforms an array of {e,n} coordinates between two EPSG codes. The
// "utente"/"chiave" fields are mandatory but ignored by the service (they are
// NOT authentication); we send fixed placeholders.
package volapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Path is the single endpoint path (relative to the configured base URL).
const Path = "/volapi"

// MaxCoord is the maximum number of coordinates the service accepts per
// conversione request (reported by the info endpoint).
const MaxCoord = 32000

// Poster is the subset of the generated client used by volapi. Both info and
// conversione are read-only (no remote state mutates), so they ride the
// read-only POST helper that stays live under PRINTING_PRESS_VERIFY=1.
type Poster interface {
	PostQueryWithParams(ctx context.Context, path string, params map[string]string, body any) (json.RawMessage, int, error)
}

// System is one supported reference system (EPSG code + IGM description).
type System struct {
	Epsg        int    `json:"epsg"`
	Descrizione string `json:"descrizione"`
}

// InfoResponse is the reply to a {"richiesta":"info"} request.
type InfoResponse struct {
	MaxCoord      int      `json:"maxCoord"`
	SrsSupportati []System `json:"srsSupportati"`
}

// Coord is a single coordinate. "e" is est (easting OR longitude) and "n" is
// nord (northing OR latitude). Geographic coordinates are ALWAYS decimal
// degrees (gradi sessadecimali).
type Coord struct {
	E float64 `json:"e"`
	N float64 `json:"n"`
}

// Error is a service-level error carried in an HTTP 200 body as
// {"stato":"errore","dove":...,"messaggio":...}. The most common case is an
// out-of-grid coordinate (dove="Proj").
type Error struct {
	Dove      string `json:"dove"`
	Messaggio string `json:"messaggio"`
}

func (e *Error) Error() string {
	if e.Dove != "" {
		return fmt.Sprintf("verto service error (%s): %s", e.Dove, e.Messaggio)
	}
	return fmt.Sprintf("verto service error: %s", e.Messaggio)
}

// OutsideGrid reports whether this error is the "coordinate falls outside the
// IGM grid coverage" case. The canonical signal is dove=="Proj" (the underlying
// PROJ transform rejecting an out-of-coverage coordinate); the message is also
// matched in English and Italian as a fallback so a localized messaggio still
// triggers the coverage hint.
func (e *Error) OutsideGrid() bool {
	if strings.EqualFold(strings.TrimSpace(e.Dove), "Proj") {
		return true
	}
	m := strings.ToLower(e.Messaggio)
	return strings.Contains(m, "outside grid") || strings.Contains(m, "griglia")
}

type convResponse struct {
	Stato      string  `json:"stato"`
	Dove       string  `json:"dove"`
	Messaggio  string  `json:"messaggio"`
	Coordinate []Coord `json:"coordinate"`
}

// Info fetches the list of supported reference systems and the per-request
// coordinate cap.
func Info(ctx context.Context, c Poster) (*InfoResponse, error) {
	body := map[string]any{"richiesta": "info"}
	raw, _, err := c.PostQueryWithParams(ctx, Path, nil, body)
	if err != nil {
		return nil, err
	}
	var out InfoResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parsing info response: %w", err)
	}
	return &out, nil
}

// convertChunk converts a single chunk (<= MaxCoord coordinates) in one
// request. A service-level {"stato":"errore"} body is returned as *Error.
func convertChunk(ctx context.Context, c Poster, inEpsg, outEpsg int, coords []Coord) ([]Coord, error) {
	body := map[string]any{
		"richiesta":  "conversione",
		"utente":     "verto-pp-cli",
		"chiave":     "verto-pp-cli",
		"inEpsg":     inEpsg,
		"outEpsg":    outEpsg,
		"coordinate": coords,
	}
	raw, _, err := c.PostQueryWithParams(ctx, Path, nil, body)
	if err != nil {
		return nil, err
	}
	var out convResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parsing conversion response: %w", err)
	}
	if out.Stato != "successo" {
		return nil, &Error{Dove: out.Dove, Messaggio: out.Messaggio}
	}
	if len(out.Coordinate) != len(coords) {
		return nil, fmt.Errorf("service returned %d coordinates for %d input", len(out.Coordinate), len(coords))
	}
	return out.Coordinate, nil
}

// Convert transforms coords from inEpsg to outEpsg, automatically splitting the
// input into chunks of at most MaxCoord. The result preserves input order. A
// single out-of-grid coordinate fails its entire chunk (the service is
// all-or-nothing per request); use ConvertSkipping to isolate and skip
// offenders instead.
func Convert(ctx context.Context, c Poster, inEpsg, outEpsg int, coords []Coord) ([]Coord, error) {
	if len(coords) == 0 {
		return nil, nil
	}
	out := make([]Coord, 0, len(coords))
	for start := 0; start < len(coords); start += MaxCoord {
		end := start + MaxCoord
		if end > len(coords) {
			end = len(coords)
		}
		got, err := convertChunk(ctx, c, inEpsg, outEpsg, coords[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}

// ConvertSkipping converts coords, isolating and skipping any coordinate the
// service rejects (e.g. out-of-grid) by bisecting failing chunks. It returns a
// result slice the same length as coords: successfully converted coordinates
// carry their result, skipped ones are reported via the returned index set
// (and left as the zero Coord in the result). Non-service errors (network,
// context) abort and are returned as err.
func ConvertSkipping(ctx context.Context, c Poster, inEpsg, outEpsg int, coords []Coord) (result []Coord, skipped []int, err error) {
	result = make([]Coord, len(coords))
	skipped, err = convertRange(ctx, c, inEpsg, outEpsg, coords, 0, result)
	return result, skipped, err
}

// convertRange converts coords[offset:offset+len(sub)] where sub is the slice
// to process, writing results into result at absolute indices. On a
// service-level error it bisects; a single failing coordinate is recorded as
// skipped.
func convertRange(ctx context.Context, c Poster, inEpsg, outEpsg int, sub []Coord, offset int, result []Coord) ([]int, error) {
	if len(sub) == 0 {
		return nil, nil
	}
	// Process in MaxCoord windows so a huge input still chunks correctly.
	if len(sub) > MaxCoord {
		mid := len(sub) / 2
		left, err := convertRange(ctx, c, inEpsg, outEpsg, sub[:mid], offset, result)
		if err != nil {
			return left, err
		}
		right, err := convertRange(ctx, c, inEpsg, outEpsg, sub[mid:], offset+mid, result)
		return append(left, right...), err
	}
	got, err := convertChunk(ctx, c, inEpsg, outEpsg, sub)
	if err == nil {
		copy(result[offset:offset+len(got)], got)
		return nil, nil
	}
	var sErr *Error
	if !isServiceError(err, &sErr) {
		return nil, err // network/context error: abort
	}
	if len(sub) == 1 {
		return []int{offset}, nil // isolated the offender
	}
	mid := len(sub) / 2
	left, err := convertRange(ctx, c, inEpsg, outEpsg, sub[:mid], offset, result)
	if err != nil {
		return left, err
	}
	right, err := convertRange(ctx, c, inEpsg, outEpsg, sub[mid:], offset+mid, result)
	return append(left, right...), err
}

func isServiceError(err error, target **Error) bool {
	if e, ok := err.(*Error); ok {
		*target = e
		return true
	}
	return false
}
