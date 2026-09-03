// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: unified image+video search. Fans one query across both
// endpoints in parallel, merges into one ranked result set, and write-through
// caches into the local store. Hand-authored; survives `generate --force`.
//
// pp:data-source live

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

type mediaFetchFailure struct {
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

type mediaFetchOut struct {
	kind  string
	hits  []json.RawMessage
	total int
	err   error
}

func newNovelMediaSearchCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var category, lang, order string
	var safesearch bool
	var dbPath string
	cmd := &cobra.Command{
		Use:   "search <q>",
		Short: "Search images and videos together, merged into one ranked result set",
		Long: strings.TrimSpace(`
Fan one query across both the image and video endpoints in parallel and merge
stills and footage into a single result set ranked by downloads. Use this when
you want stills and footage together; for one medium only, use 'images search'
or 'videos search'. Results are written through to the local store.`),
		Example:     "  pixabay-pp-cli media search \"drone coastline\" --limit 40 --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if err := validateDataSourceStrategy(flags, "live"); err != nil {
				return err
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("media search requires a query"))
			}
			q := args[0]
			perPage := clampPerPage(limit)
			if cliutil.IsDogfoodEnv() && perPage > 5 {
				perPage = 5
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			baseParams := map[string]string{"q": q, "per_page": strconv.Itoa(perPage)}
			if category != "" {
				baseParams["category"] = category
			}
			if lang != "" {
				baseParams["lang"] = lang
			}
			if order != "" {
				baseParams["order"] = order
			}
			if safesearch {
				baseParams["safesearch"] = "true"
			}

			endpoints := []struct {
				kind string
				path string
			}{{"image", imagesPath}, {"video", videosPath}}
			results := make([]mediaFetchOut, len(endpoints))
			var wg sync.WaitGroup
			for i, ep := range endpoints {
				wg.Add(1)
				go func(i int, kind, path string) {
					defer wg.Done()
					params := map[string]string{}
					for k, v := range baseParams {
						params[k] = v
					}
					raw, gerr := c.Get(ctx, path, params)
					if gerr != nil {
						results[i] = mediaFetchOut{kind: kind, err: gerr}
						return
					}
					resp, perr := parsePixabayResponse(raw)
					if perr != nil {
						results[i] = mediaFetchOut{kind: kind, err: perr}
						return
					}
					results[i] = mediaFetchOut{kind: kind, hits: resp.Hits, total: resp.TotalHits}
				}(i, ep.kind, ep.path)
			}
			wg.Wait()

			merged := make([]json.RawMessage, 0)
			failures := make([]mediaFetchFailure, 0)
			var imagesTotal, videosTotal int
			for _, r := range results {
				if r.err != nil {
					failures = append(failures, mediaFetchFailure{Kind: r.kind, Error: r.err.Error()})
					continue
				}
				if r.kind == "image" {
					imagesTotal = r.total
				} else {
					videosTotal = r.total
				}
				for _, h := range r.hits {
					merged = append(merged, injectMediaKind(h, r.kind))
				}
			}

			if len(failures) == len(endpoints) {
				return classifyAPIError(fmt.Errorf("all media fetches failed: %s", failures[0].Error), flags)
			}
			if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d media fetches failed; results are partial\n", len(failures), len(endpoints))
			}

			sortByDownloadsDesc(merged)
			if limit > 0 && len(merged) > limit {
				merged = merged[:limit]
			}

			persistMediaResults(cmd.Context(), dbPath, results)

			out := map[string]any{
				"query":          q,
				"images_total":   imagesTotal,
				"videos_total":   videosTotal,
				"returned":       len(merged),
				"results":        merged,
				"fetch_failures": failures,
			}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d result(s) (%d image hits, %d video hits available)\n", len(merged), imagesTotal, videosTotal)
			for _, h := range merged {
				obj, _ := decodeObj(h)
				fmt.Fprintf(cmd.OutOrStdout(), "%-6s %-10s %-8d %s\n", objStr(obj, "media_kind"), objID(obj), objInt(obj, "downloads"), objStr(obj, "tags"))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum merged results to return")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	cmd.Flags().StringVar(&lang, "lang", "", "Language code to search in")
	cmd.Flags().StringVar(&order, "order", "", "Order results: popular or latest")
	cmd.Flags().BoolVar(&safesearch, "safesearch", false, "Only results suitable for all ages")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path for write-through cache")
	return cmd
}

func clampPerPage(n int) int {
	if n < 3 {
		return 3
	}
	if n > 200 {
		return 200
	}
	return n
}

func injectMediaKind(raw json.RawMessage, kind string) json.RawMessage {
	obj, err := decodeObj(raw)
	if err != nil {
		return raw
	}
	obj["media_kind"] = kind
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

func sortByDownloadsDesc(items []json.RawMessage) {
	sort.SliceStable(items, func(i, j int) bool {
		oi, _ := decodeObj(items[i])
		oj, _ := decodeObj(items[j])
		return objInt(oi, "downloads") > objInt(oj, "downloads")
	})
}

func persistMediaResults(ctx context.Context, dbPath string, results []mediaFetchOut) {
	if dbPath == "" {
		dbPath = defaultDBPath(pixabayCLIName)
	}
	db, err := store.OpenWithContext(ctx, dbPath)
	if err != nil {
		return
	}
	defer db.Close()
	for _, r := range results {
		if r.err != nil {
			continue
		}
		if r.kind == "image" {
			_, _ = persistHits(db, "images", r.hits)
		} else {
			_, _ = persistHits(db, "videos", r.hits)
		}
	}
}
