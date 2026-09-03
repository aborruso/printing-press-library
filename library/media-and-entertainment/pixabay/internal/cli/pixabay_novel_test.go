// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored behavioral tests for the Pixabay novel commands. These replace
// the generated t.Skip placeholders with real assertions against output content.

package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectionRoundTrip(t *testing.T) {
	db := tempDB(t)
	if out, err := execPixabay(t, "collection", "add", "winter", "195893,1850181", "--db", db, "--json"); err != nil {
		t.Fatalf("add: %v (%s)", err, out)
	}
	out, err := execPixabay(t, "collection", "show", "winter", "--db", db, "--json")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	var members []map[string]string
	if err := json.Unmarshal([]byte(out), &members); err != nil {
		t.Fatalf("show output not a JSON array: %v (%s)", err, out)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d: %s", len(members), out)
	}
	if _, err := execPixabay(t, "collection", "remove", "winter", "195893", "--db", db, "--json"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	out, _ = execPixabay(t, "collection", "show", "winter", "--db", db, "--json")
	_ = json.Unmarshal([]byte(out), &members)
	if len(members) != 1 || members[0]["id"] != "1850181" {
		t.Fatalf("after remove expected only 1850181, got %s", out)
	}
}

func TestSimilarRanksBySharedTags(t *testing.T) {
	db := tempDB(t)
	seedHit(t, db, "images", "100", "winter, snow, cold", "Ann", 50, 5)
	seedHit(t, db, "images", "200", "snow, ice, cold", "Bob", 30, 3)    // 2 shared
	seedHit(t, db, "images", "300", "summer, beach, sun", "Cat", 90, 9) // 0 shared
	out, err := execPixabay(t, "similar", "100", "--db", db, "--json")
	if err != nil {
		t.Fatalf("similar: %v (%s)", err, out)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 similar (200), got %d: %s", len(rows), out)
	}
	if rows[0]["id"] != "200" {
		t.Fatalf("expected match 200, got %v", rows[0]["id"])
	}
}

func TestContributorsRanking(t *testing.T) {
	db := tempDB(t)
	seedHit(t, db, "images", "1", "a", "Ann", 100, 1)
	seedHit(t, db, "images", "2", "b", "Ann", 200, 1)
	seedHit(t, db, "images", "3", "c", "Bob", 50, 1)
	out, err := execPixabay(t, "contributors", "--by", "downloads", "--db", db, "--json")
	if err != nil {
		t.Fatalf("contributors: %v (%s)", err, out)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, out)
	}
	if len(rows) < 1 || rows[0]["user"] != "Ann" {
		t.Fatalf("expected Ann ranked first (300 downloads), got %s", out)
	}
	if dl, _ := rows[0]["downloads"].(float64); dl != 300 {
		t.Fatalf("expected Ann downloads=300, got %v", rows[0]["downloads"])
	}
}

func TestTrendsBaselineThenDelta(t *testing.T) {
	db := tempDB(t)
	seedHit(t, db, "images", "1", "winter", "Ann", 100, 1)
	// First run: baseline (no prior snapshot).
	out, err := execPixabay(t, "trends", "--db", db, "--json")
	if err != nil {
		t.Fatalf("trends run 1: %v (%s)", err, out)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, out)
	}
	if len(rows) != 1 || rows[0]["baseline"] != true {
		t.Fatalf("first run should be baseline, got %s", out)
	}
	// Bump downloads, re-run: should now report a positive delta vs the snapshot.
	seedHit(t, db, "images", "1", "winter", "Ann", 150, 1)
	out, _ = execPixabay(t, "trends", "--db", db, "--json")
	_ = json.Unmarshal([]byte(out), &rows)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %s", out)
	}
	if d, _ := rows[0]["delta_downloads"].(float64); d != 50 {
		t.Fatalf("expected delta_downloads=50, got %v (%s)", rows[0]["delta_downloads"], out)
	}
}

func TestNetworkCommandsDryRunNoNetwork(t *testing.T) {
	cases := [][]string{
		{"pull", "--from-collection", "x", "--dry-run"},
		{"media", "search", "cats", "--dry-run"},
		{"quota", "--dry-run"},
	}
	for _, args := range cases {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if out, err := execPixabay(t, args...); err != nil {
				t.Fatalf("%v: dry-run should not error, got %v (%s)", args, err, out)
			}
		})
	}
}

func TestQuotaVerifyEnvNoNetwork(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "1")
	out, err := execPixabay(t, "quota", "--json")
	if err != nil {
		t.Fatalf("quota verify-env: %v (%s)", err, out)
	}
	if !strings.Contains(out, "verify mode") {
		t.Fatalf("expected verify-mode note, got %s", out)
	}
}
