// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package volapi

import (
	"context"
	"encoding/json"
	"testing"
)

// fakePoster simulates the IGM service. Coordinates with E >= 1000 are treated
// as out-of-grid (the whole request fails), mirroring the real all-or-nothing
// behavior, so bisection tests have a deterministic offender.
type fakePoster struct {
	calls int
}

func (f *fakePoster) PostQueryWithParams(_ context.Context, _ string, _ map[string]string, body any) (json.RawMessage, int, error) {
	f.calls++
	m := body.(map[string]any)
	if m["richiesta"] == "info" {
		return json.RawMessage(`{"maxCoord":32000,"srsSupportati":[{"epsg":4265,"descrizione":"Monte Mario"},{"epsg":6706,"descrizione":"RDN2008 2D geo"}]}`), 200, nil
	}
	coords := m["coordinate"].([]Coord)
	for _, c := range coords {
		if c.E >= 1000 {
			return json.RawMessage(`{"stato":"errore","dove":"Proj","messaggio":"Coordinate to transform falls outside grid"}`), 200, nil
		}
	}
	out := convResponse{Stato: "successo"}
	for _, c := range coords {
		out.Coordinate = append(out.Coordinate, Coord{E: c.E + 0.1, N: c.N + 0.2})
	}
	raw, _ := json.Marshal(out)
	return raw, 200, nil
}

func TestInfo(t *testing.T) {
	info, err := Info(context.Background(), &fakePoster{})
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.MaxCoord != 32000 {
		t.Errorf("MaxCoord = %d, want 32000", info.MaxCoord)
	}
	if len(info.SrsSupportati) != 2 || info.SrsSupportati[0].Epsg != 4265 {
		t.Errorf("unexpected systems: %+v", info.SrsSupportati)
	}
}

func TestConvert(t *testing.T) {
	in := []Coord{{E: 1, N: 2}, {E: 3, N: 4}}
	out, err := Convert(context.Background(), &fakePoster{}, 4265, 6706, in)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(out) != 2 || out[0].E != 1.1 || out[1].N != 4.2 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestConvertOutsideGridFailsWhole(t *testing.T) {
	in := []Coord{{E: 1, N: 2}, {E: 9999, N: 4}}
	_, err := Convert(context.Background(), &fakePoster{}, 4265, 6706, in)
	ve, ok := err.(*Error)
	if !ok {
		t.Fatalf("want *Error, got %T (%v)", err, err)
	}
	if !ve.OutsideGrid() {
		t.Errorf("OutsideGrid() = false for %q", ve.Messaggio)
	}
}

func TestConvertSkipping(t *testing.T) {
	cases := []struct {
		name      string
		coords    []Coord
		wantSkips []int
	}{
		{"all good", []Coord{{1, 2}, {3, 4}, {5, 6}}, nil},
		{"one bad middle", []Coord{{1, 2}, {9999, 0}, {5, 6}}, []int{1}},
		{"two bad", []Coord{{9999, 0}, {3, 4}, {9999, 1}}, []int{0, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, skipped, err := ConvertSkipping(context.Background(), &fakePoster{}, 4265, 6706, tc.coords)
			if err != nil {
				t.Fatalf("ConvertSkipping: %v", err)
			}
			if !equalInts(skipped, tc.wantSkips) {
				t.Errorf("skipped = %v, want %v", skipped, tc.wantSkips)
			}
			// Non-skipped entries must be converted (E+0.1).
			skipSet := map[int]bool{}
			for _, s := range skipped {
				skipSet[s] = true
			}
			for i, c := range tc.coords {
				if skipSet[i] {
					continue
				}
				if result[i].E != c.E+0.1 {
					t.Errorf("index %d not converted: got %v", i, result[i])
				}
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
