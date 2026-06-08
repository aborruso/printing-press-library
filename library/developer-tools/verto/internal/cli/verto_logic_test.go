// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintJSONL(t *testing.T) {
	var buf bytes.Buffer
	// Array: one compact object per line, key order preserved.
	if err := printJSONL(&buf, json.RawMessage(`[{"a":1,"b":2},{"c":3}]`)); err != nil {
		t.Fatalf("printJSONL array: %v", err)
	}
	if got, want := buf.String(), "{\"a\":1,\"b\":2}\n{\"c\":3}\n"; got != want {
		t.Errorf("array: got %q, want %q", got, want)
	}
	// Single object: one line.
	buf.Reset()
	if err := printJSONL(&buf, json.RawMessage(`{ "x": 1 }`)); err != nil {
		t.Fatalf("printJSONL object: %v", err)
	}
	if got, want := buf.String(), "{\"x\":1}\n"; got != want {
		t.Errorf("object: got %q, want %q", got, want)
	}
}

func TestClassifyCoord(t *testing.T) {
	cases := []struct {
		name     string
		e, n     float64
		wantKind string
		wantCand int // an EPSG that must appear in candidates (0 = skip check)
		inItaly  bool
	}{
		{"rome geographic", 12.4924, 41.8902, "geographic", 4265, true},
		{"out-of-italy geographic", 24.0, 37.0, "geographic", 6706, false},
		{"gauss-boaga est", 2300000, 4640000, "projected", 3004, true},
		{"utm zone", 514827, 5034611, "projected", 23032, true},
		{"laea low northing", 4528557, 2090943, "projected", 3035, true},
		{"italy zone EN", 7040787, 4632671, "projected", 7794, true},
		{"nonsense", -999, -999, "unknown", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := classifyCoord(tc.e, tc.n)
			if r.Kind != tc.wantKind {
				t.Errorf("kind = %q, want %q", r.Kind, tc.wantKind)
			}
			if r.InItaly != tc.inItaly {
				t.Errorf("inItaly = %v, want %v", r.InItaly, tc.inItaly)
			}
			if tc.wantCand != 0 && !containsInt(r.Candidates, tc.wantCand) {
				t.Errorf("candidates %v missing %d", r.Candidates, tc.wantCand)
			}
		})
	}
}

func TestDatumFamilyPartition(t *testing.T) {
	// Same-family pairs the service rejects; cross-family pairs it allows.
	same := [][2]int{{4265, 4806}, {4265, 3003}, {4230, 23032}, {4670, 3035}, {6706, 6707}, {6706, 7794}}
	for _, p := range same {
		if datumFamily(p[0]) != datumFamily(p[1]) {
			t.Errorf("EPSG %d and %d should share a datum family", p[0], p[1])
		}
	}
	cross := [][2]int{{4265, 4230}, {4670, 6706}, {4230, 6706}, {4265, 4670}}
	for _, p := range cross {
		if datumFamily(p[0]) == datumFamily(p[1]) {
			t.Errorf("EPSG %d and %d should be in different families", p[0], p[1])
		}
	}
}

func TestParseEpsg(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"3003", 3003, false},
		{"EPSG:6706", 6706, false},
		{"epsg:4265", 4265, false},
		{" 23032 ", 23032, false},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tc := range cases {
		got, err := parseEpsg(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseEpsg(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("parseEpsg(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestResolveColumn(t *testing.T) {
	header := []string{"id", "Est", "Nord", "name"}
	eAliases := []string{"e", "est", "easting", "lon", "x"}
	nAliases := []string{"n", "nord", "lat", "y"}

	if idx, err := resolveColumn(header, "", eAliases); err != nil || idx != 1 {
		t.Errorf("auto e-col = %d err=%v, want 1", idx, err)
	}
	if idx, err := resolveColumn(header, "", nAliases); err != nil || idx != 2 {
		t.Errorf("auto n-col = %d err=%v, want 2", idx, err)
	}
	if idx, err := resolveColumn(header, "name", eAliases); err != nil || idx != 3 {
		t.Errorf("explicit col = %d err=%v, want 3", idx, err)
	}
	if _, err := resolveColumn(header, "missing", eAliases); err == nil {
		t.Error("expected error for missing explicit column")
	}
	if _, err := resolveColumn([]string{"id", "name"}, "", eAliases); err == nil {
		t.Error("expected error when no alias matches")
	}
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
