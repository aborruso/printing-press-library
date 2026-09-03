// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored tests for the pure-logic helpers shared by the Pixabay novel
// commands.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

// execPixabay runs the CLI with the given args against a fresh root command,
// capturing stdout only (stderr hints are kept separate so machine output on
// stdout stays parseable). Used by novel-command behavioral tests.
func execPixabay(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := RootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// seedHit upserts one synthetic hit into the given temp DB's typed table.
func seedHit(t *testing.T, dbPath, kind, id, tags, user string, downloads, likes int) {
	t.Helper()
	db, err := store.OpenWithContext(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()
	hit := map[string]any{
		"id": jsonNumberID(id), "tags": tags, "user": user, "user_id": 7,
		"downloads": downloads, "likes": likes, "views": downloads * 10, "comments": 1,
		"pageURL": "https://pixabay.com/p/" + id + "/", "webformatURL": "https://cdn/x_640.jpg",
		"largeImageURL": "https://cdn/x_1280.jpg",
	}
	raw, _ := json.Marshal(hit)
	if kind == "videos" {
		if err := db.UpsertVideos(raw); err != nil {
			t.Fatalf("seed video: %v", err)
		}
		return
	}
	if err := db.UpsertImages(raw); err != nil {
		t.Fatalf("seed image: %v", err)
	}
}

// jsonNumberID returns the id as a float64 so json.Marshal emits a JSON number,
// matching the live API shape (objID must format it without exponent).
func jsonNumberID(id string) float64 {
	var f float64
	_, _ = fmt.Sscanf(id, "%g", &f)
	return f
}

func tempDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "pixabay.db")
}

func TestImageVariantURL(t *testing.T) {
	t.Parallel()
	obj := map[string]any{
		"previewURL":    "https://cdn.pixabay.com/photo/x_150.jpg",
		"webformatURL":  "https://cdn.pixabay.com/photo/x_640.jpg",
		"largeImageURL": "https://cdn.pixabay.com/photo/x_1280.jpg",
		"fullHDURL":     "https://cdn.pixabay.com/photo/x_1920.jpg",
		"imageURL":      "https://cdn.pixabay.com/photo/x_orig.jpg",
	}
	cases := []struct {
		size string
		want string
	}{
		{"preview", "https://cdn.pixabay.com/photo/x_150.jpg"},
		{"web", "https://cdn.pixabay.com/photo/x_640.jpg"},
		{"large", "https://cdn.pixabay.com/photo/x_1280.jpg"},
		{"", "https://cdn.pixabay.com/photo/x_1280.jpg"},
		{"fullhd", "https://cdn.pixabay.com/photo/x_1920.jpg"},
		{"original", "https://cdn.pixabay.com/photo/x_orig.jpg"},
		// The _640 size trick: replace the embedded marker with no extra call.
		{"180", "https://cdn.pixabay.com/photo/x_180.jpg"},
		{"340", "https://cdn.pixabay.com/photo/x_340.jpg"},
		{"960", "https://cdn.pixabay.com/photo/x_960.jpg"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.size, func(t *testing.T) {
			t.Parallel()
			got, ok := imageVariantURL(obj, tc.size)
			if !ok || got != tc.want {
				t.Fatalf("imageVariantURL(%q) = %q,%v; want %q,true", tc.size, got, ok, tc.want)
			}
		})
	}
}

func TestImageVariantURLFallback(t *testing.T) {
	t.Parallel()
	// Only webformat present: large/original/fullhd all fall back to it,
	// and numeric variants still apply the size trick.
	obj := map[string]any{"webformatURL": "https://cdn.pixabay.com/photo/y_640.jpg"}
	if got, ok := imageVariantURL(obj, "large"); !ok || got != "https://cdn.pixabay.com/photo/y_640.jpg" {
		t.Fatalf("large fallback = %q,%v", got, ok)
	}
	if got, _ := imageVariantURL(obj, "340"); got != "https://cdn.pixabay.com/photo/y_340.jpg" {
		t.Fatalf("340 trick = %q", got)
	}
	// No URLs at all -> not ok.
	if _, ok := imageVariantURL(map[string]any{}, "large"); ok {
		t.Fatalf("expected ok=false for empty object")
	}
}

func TestVideoVariantURL(t *testing.T) {
	t.Parallel()
	obj := map[string]any{
		"videos": map[string]any{
			"large":  map[string]any{"url": "L.mp4"},
			"medium": map[string]any{"url": "M.mp4"},
			"tiny":   map[string]any{"url": "T.mp4"},
		},
	}
	if got, ok := videoVariantURL(obj, "large"); !ok || got != "L.mp4" {
		t.Fatalf("large = %q,%v", got, ok)
	}
	if got, _ := videoVariantURL(obj, "tiny"); got != "T.mp4" {
		t.Fatalf("tiny = %q", got)
	}
	// 'small' is absent -> falls through to medium per the preference order.
	if got, _ := videoVariantURL(obj, "small"); got != "M.mp4" {
		t.Fatalf("small fallback = %q", got)
	}
	if _, ok := videoVariantURL(map[string]any{}, "large"); ok {
		t.Fatalf("expected ok=false when no videos object")
	}
}

func TestObjID(t *testing.T) {
	t.Parallel()
	// JSON numbers decode to float64; must not become scientific notation.
	var obj map[string]any
	if err := json.Unmarshal([]byte(`{"id":195893}`), &obj); err != nil {
		t.Fatal(err)
	}
	if got := objID(obj); got != "195893" {
		t.Fatalf("objID = %q; want 195893", got)
	}
	if got := objID(map[string]any{"id": "abc"}); got != "abc" {
		t.Fatalf("string id = %q", got)
	}
	if got := objID(map[string]any{}); got != "" {
		t.Fatalf("missing id = %q; want empty", got)
	}
}

func TestSplitIDs(t *testing.T) {
	t.Parallel()
	got := splitIDs([]string{"195893,1850181", " 42 ", "195893"})
	want := []string{"195893", "1850181", "42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitIDs = %v; want %v (comma-split + trim + dedup)", got, want)
	}
}

func TestTagOverlap(t *testing.T) {
	t.Parallel()
	a := splitTagString("winter, snow, cold")
	b := splitTagString("Snow, ice, COLD")
	shared := sortedStrings(intersectTags(a, b))
	want := []string{"cold", "snow"}
	if !reflect.DeepEqual(shared, want) {
		t.Fatalf("intersectTags = %v; want %v (case-insensitive)", shared, want)
	}
}

func TestClampPerPage(t *testing.T) {
	t.Parallel()
	cases := map[int]int{0: 3, 2: 3, 20: 20, 200: 200, 500: 200}
	for in, want := range cases {
		if got := clampPerPage(in); got != want {
			t.Fatalf("clampPerPage(%d) = %d; want %d", in, got, want)
		}
	}
}

func TestURLExt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url, kind, want string
	}{
		{"https://x/y_640.jpg", "images", ".jpg"},
		{"https://x/y.png?download=1", "images", ".png"},
		{"https://x/vid.mp4", "videos", ".mp4"},
		{"https://x/noext", "images", ".jpg"},
		{"https://x/noext", "videos", ".mp4"},
	}
	for _, tc := range cases {
		if got := urlExt(tc.url, tc.kind); got != tc.want {
			t.Fatalf("urlExt(%q,%q) = %q; want %q", tc.url, tc.kind, got, tc.want)
		}
	}
}

func TestAttributionText(t *testing.T) {
	t.Parallel()
	got := attributionText(map[string]any{"user": "Alice", "pageURL": "https://pixabay.com/p/1/"}, "images")
	if got == "" || !contains(got, "Alice") || !contains(got, "Pixabay Content License") {
		t.Fatalf("attributionText missing required parts: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
