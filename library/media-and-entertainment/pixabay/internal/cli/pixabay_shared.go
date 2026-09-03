// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Hand-authored shared helpers for Pixabay novel commands (pull, quota,
// media, similar, trends, contributors, collection). Kept in its own file so it
// survives `generate --force` as a whole hand-authored unit.

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

// pixabayCLIName is the binary/library name used for the default DB path.
const pixabayCLIName = "pixabay-pp-cli"

// imagesPath and videosPath are the two upstream endpoints.
const (
	imagesPath = "/api/"
	videosPath = "/api/videos/"
)

// pixabayResponse is the common Pixabay search envelope for both endpoints.
type pixabayResponse struct {
	Total     int               `json:"total"`
	TotalHits int               `json:"totalHits"`
	Hits      []json.RawMessage `json:"hits"`
}

// parsePixabayResponse decodes the {total,totalHits,hits} envelope returned by
// both /api/ and /api/videos/.
func parsePixabayResponse(raw json.RawMessage) (pixabayResponse, error) {
	var r pixabayResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("parsing Pixabay response: %w", err)
	}
	return r, nil
}

// decodeObj decodes a single hit into a generic map.
func decodeObj(raw json.RawMessage) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// objID returns the numeric Pixabay id of a hit as a decimal string. JSON
// numbers decode to float64, so format without scientific notation.
func objID(obj map[string]any) string {
	switch v := obj["id"].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	case string:
		return v
	default:
		return ""
	}
}

// objStr returns a string field, empty when absent or non-string.
func objStr(obj map[string]any, key string) string {
	if s, ok := obj[key].(string); ok {
		return s
	}
	return ""
}

// objInt returns an integer field, 0 when absent or non-numeric.
func objInt(obj map[string]any, key string) int {
	switch v := obj[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case int:
		return v
	default:
		return 0
	}
}

// tagSet splits a Pixabay "tags" string ("a, b, c") into a normalized set.
func tagSet(obj map[string]any) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range strings.Split(objStr(obj, "tags"), ",") {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			out[t] = struct{}{}
		}
	}
	return out
}

// imageVariantURL resolves a requested size keyword to a concrete image URL.
// It exploits Pixabay's documented webformatURL size trick (replace _640 with
// _180/_340/_960) for the numeric variants, costing no extra API call.
func imageVariantURL(obj map[string]any, size string) (string, bool) {
	web := objStr(obj, "webformatURL")
	switch strings.ToLower(size) {
	case "preview", "thumb", "xs":
		return firstNonEmpty(objStr(obj, "previewURL"), web), objStr(obj, "previewURL") != "" || web != ""
	case "web", "webformat", "640", "md":
		return web, web != ""
	case "180", "340", "960":
		if web == "" {
			return "", false
		}
		return strings.Replace(web, "_640", "_"+size, 1), true
	case "large", "lg", "":
		u := firstNonEmpty(objStr(obj, "largeImageURL"), web)
		return u, u != ""
	case "fullhd", "hd":
		u := firstNonEmpty(objStr(obj, "fullHDURL"), objStr(obj, "largeImageURL"), web)
		return u, u != ""
	case "original", "image", "og":
		u := firstNonEmpty(objStr(obj, "imageURL"), objStr(obj, "vectorURL"), objStr(obj, "largeImageURL"), web)
		return u, u != ""
	default:
		u := firstNonEmpty(objStr(obj, "largeImageURL"), web)
		return u, u != ""
	}
}

// videoVariantURL resolves a video rendition URL from the nested "videos"
// object (large/medium/small/tiny).
func videoVariantURL(obj map[string]any, size string) (string, bool) {
	vids, ok := obj["videos"].(map[string]any)
	if !ok {
		return "", false
	}
	order := []string{"large", "medium", "small", "tiny"}
	switch strings.ToLower(size) {
	case "large", "lg", "fullhd", "hd", "original", "":
		order = []string{"large", "medium", "small", "tiny"}
	case "medium", "md", "web":
		order = []string{"medium", "large", "small", "tiny"}
	case "small", "sm":
		order = []string{"small", "medium", "tiny", "large"}
	case "tiny", "xs", "preview", "thumb":
		order = []string{"tiny", "small", "medium", "large"}
	}
	for _, k := range order {
		if rend, ok := vids[k].(map[string]any); ok {
			if u := objStr(rend, "url"); u != "" {
				return u, true
			}
		}
	}
	return "", false
}

// attributionText builds a ready-to-paste credit line honoring the Pixabay
// Content License attribution recommendation.
func attributionText(obj map[string]any, kind string) string {
	user := objStr(obj, "user")
	page := objStr(obj, "pageURL")
	noun := "Image"
	if kind == "videos" {
		noun = "Video"
	}
	if user == "" {
		user = "a Pixabay contributor"
	}
	return fmt.Sprintf("%s by %s from Pixabay (%s) — Pixabay Content License: https://pixabay.com/service/license-summary/", noun, user, page)
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Novel-feature tables (pp_collections, pp_stat_snapshots) are created in the
// canonical migration location, internal/store/extras.go migrateExtras, which
// runs on every write-capable store open. Commands no longer create them
// ad-hoc; read-only paths tolerate their absence via isNoSuchTableErr.

// isNoSuchTableErr reports whether err is a SQLite "no such table" error, which
// can happen on a read-only open of a store that has never been write-opened.
func isNoSuchTableErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

// persistHits upserts a slice of raw hits into the correct typed store table.
func persistHits(db *store.Store, kind string, hits []json.RawMessage) (int, error) {
	added := 0
	for _, h := range hits {
		var err error
		if kind == "videos" {
			err = db.UpsertVideos(h)
		} else {
			err = db.UpsertImages(h)
		}
		if err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// sortedStrings returns a sorted copy for deterministic output.
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
