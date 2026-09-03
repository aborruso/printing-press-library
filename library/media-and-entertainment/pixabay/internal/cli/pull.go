// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: resumable bulk download with attribution and 24h URL
// re-resolution. Hand-authored; survives `generate --force` as a whole unit.
//
// pp:data-source live

package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/client"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

// urlFreshWindow is how long Pixabay image URLs stay valid.
const urlFreshWindow = 24 * time.Hour

// maxDownloadBytes caps a single asset download (1 GiB) so a misbehaving or
// redirected URL cannot exhaust the disk. Pixabay HD videos are well under this.
const maxDownloadBytes = 1 << 30

type pullFailure struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

func newNovelPullCmd(flags *rootFlags) *cobra.Command {
	var fromCollection, query, kind, size, outDir, dbPath string
	var workers int
	var resume bool
	cmd := &cobra.Command{
		Use:   "pull [id...]",
		Short: "Download chosen size variants with parallel workers, attribution, and 24h re-resolve",
		Long: strings.TrimSpace(`
Download assets you already have in a collection or store (or by explicit IDs)
to disk. Pull re-fetches URLs that may have expired (Pixabay URLs last 24h),
runs parallel workers, writes a per-file attribution sidecar, and skips files
already downloaded with --resume. Use this to materialize a curated set; do NOT
use it to discover new assets — use 'images search' or 'media search'.`),
		Example:     "  pixabay-pp-cli pull --from-collection winter --size large --workers 8 --resume",
		Annotations: map[string]string{"mcp:read-only": "false", "pp:happy-args": "--query=nature;--workers=1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would download selected assets to", pullOutDir(outDir))
				return nil
			}
			if err := validateDataSourceStrategy(flags, "live"); err != nil {
				return err
			}
			k, err := collectionKind(kind)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			// Short-circuit verify mode before any network or filesystem IO:
			// resolving --query targets would otherwise make a live API call.
			if cliutil.IsVerifyEnv() {
				fmt.Fprintln(cmd.OutOrStdout(), "would download selected assets to", pullOutDir(outDir))
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			ids, err := resolvePullTargets(ctx, cmd, c, k, dbPath, fromCollection, query, args)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("no target IDs: pass IDs, --from-collection, or --query"))
			}
			if cliutil.IsDogfoodEnv() && len(ids) > 1 {
				ids = ids[:1]
			}

			out := pullOutDir(outDir)
			if err := os.MkdirAll(out, 0o750); err != nil {
				return fmt.Errorf("creating output dir: %w", err)
			}
			if workers < 1 {
				workers = 4
			}

			var (
				mu         sync.Mutex
				downloaded int
				skipped    int
				failures   []pullFailure
				wg         sync.WaitGroup
				sem        = make(chan struct{}, workers)
			)
			for _, id := range ids {
				wg.Add(1)
				sem <- struct{}{}
				go func(id string) {
					defer wg.Done()
					defer func() { <-sem }()
					status, ferr := downloadOne(ctx, c, dbPath, k, id, size, out, resume)
					mu.Lock()
					defer mu.Unlock()
					switch {
					case ferr != nil:
						failures = append(failures, pullFailure{ID: id, Error: ferr.Error()})
					case status == "skipped":
						skipped++
					default:
						downloaded++
					}
				}(id)
			}
			wg.Wait()

			if failures == nil {
				failures = []pullFailure{}
			}
			report := map[string]any{
				"requested":     len(ids),
				"downloaded":    downloaded,
				"skipped":       skipped,
				"failed":        len(failures),
				"out_dir":       out,
				"pull_failures": failures,
			}
			if downloaded == 0 && len(failures) == len(ids) && len(ids) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "all %d download(s) failed\n", len(ids))
			} else if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d download(s) failed\n", len(failures), len(ids))
			}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), report, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %d, skipped %d, failed %d -> %s\n", downloaded, skipped, len(failures), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromCollection, "from-collection", "", "Pull all IDs in this local collection")
	cmd.Flags().StringVar(&query, "query", "", "Search live and pull the top results")
	cmd.Flags().StringVar(&kind, "kind", "images", "Media kind: images or videos")
	cmd.Flags().StringVar(&size, "size", "large", "Size variant: preview/web/large/fullhd/original (images) or large/medium/small/tiny (videos)")
	cmd.Flags().IntVar(&workers, "workers", 4, "Parallel download workers")
	cmd.Flags().BoolVar(&resume, "resume", false, "Skip files that already exist on disk")
	cmd.Flags().StringVar(&outDir, "out", "", "Output directory (default ./pixabay-downloads)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func pullOutDir(outDir string) string {
	if strings.TrimSpace(outDir) != "" {
		return outDir
	}
	return "pixabay-downloads"
}

// resolvePullTargets builds the list of IDs to pull from explicit args, a
// collection, or a live query (whichever is provided).
func resolvePullTargets(ctx context.Context, cmd *cobra.Command, c *client.Client, kind, dbPath, fromCollection, query string, args []string) ([]string, error) {
	if ids := splitIDs(args); len(ids) > 0 {
		return ids, nil
	}
	if fromCollection != "" {
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("no local store at %s; add a collection first", dbPath)
		}
		db, err := store.OpenReadOnlyContext(ctx, dbPath)
		if err != nil {
			return nil, fmt.Errorf("opening database: %w", err)
		}
		defer db.Close()
		members, err := collectionMembers(cmd, db, fromCollection, kind)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, m := range members {
			ids = append(ids, m["id"])
		}
		return ids, nil
	}
	if query != "" {
		path := imagesPath
		if kind == "videos" {
			path = videosPath
		}
		raw, err := c.Get(ctx, path, map[string]string{"q": query, "per_page": "20"})
		if err != nil {
			return nil, err
		}
		resp, err := parsePixabayResponse(raw)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, h := range resp.Hits {
			obj, derr := decodeObj(h)
			if derr != nil {
				continue
			}
			if id := objID(obj); id != "" {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}
	return nil, nil
}

// downloadOne resolves a single ID's URL (re-fetching if stale), downloads the
// bytes, and writes an attribution sidecar. Returns "downloaded" or "skipped".
func downloadOne(ctx context.Context, c *client.Client, dbPath, kind, id, size, out string, resume bool) (string, error) {
	obj, err := loadOrResolveHit(ctx, c, dbPath, kind, id)
	if err != nil {
		return "", err
	}
	var url string
	var ok bool
	if kind == "videos" {
		url, ok = videoVariantURL(obj, size)
	} else {
		url, ok = imageVariantURL(obj, size)
	}
	if !ok || url == "" {
		return "", fmt.Errorf("no %q URL available for %s (try a different --size or request full API access)", size, id)
	}

	ext := urlExt(url, kind)
	base := safeFileBase(id)
	dest := filepath.Join(out, base+ext)
	if resume {
		if _, statErr := os.Stat(dest); statErr == nil {
			// Still (re)write the credit so attribution is never missing.
			_ = os.WriteFile(filepath.Join(out, base+".credit.txt"), []byte(attributionText(obj, kind)+"\n"), 0o600)
			return "skipped", nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", id, resp.StatusCode)
	}
	tmp := dest + ".part"
	// dest is the user-requested --out dir joined with a sanitized, filesystem-safe
	// base name (safeFileBase strips path separators and traversal characters), so
	// the variable path is constrained to the directory the user explicitly chose.
	f, err := os.Create(tmp) // #nosec G304 -- dest derives from user --out plus a sanitized base name
	if err != nil {
		return "", err
	}
	// Cap the download so a misbehaving or redirected URL cannot exhaust disk.
	// Copy one byte past the cap to detect (and reject) oversized responses.
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if cerr := f.Close(); cerr != nil {
		_ = os.Remove(tmp)
		return "", cerr
	}
	if n > maxDownloadBytes {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("download %s exceeds %d bytes; refusing", id, maxDownloadBytes)
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	_ = os.WriteFile(filepath.Join(out, base+".credit.txt"), []byte(attributionText(obj, kind)+"\n"), 0o600)
	return "downloaded", nil
}

// safeFileBase reduces an ID to a filesystem-safe base name, defending against
// path traversal if a caller passes a crafted ID. Pixabay IDs are numeric, so
// this is normally a no-op.
func safeFileBase(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "item"
	}
	return out
}

// loadOrResolveHit returns the stored hit object for an ID, re-fetching it from
// the API when it is missing or older than the 24h URL freshness window.
func loadOrResolveHit(ctx context.Context, c *client.Client, dbPath, kind, id string) (map[string]any, error) {
	if fresh, ok := loadFreshHit(ctx, dbPath, kind, id); ok {
		return fresh, nil
	}
	// Re-resolve by ID against the live API.
	path := imagesPath
	if kind == "videos" {
		path = videosPath
	}
	raw, err := c.Get(ctx, path, map[string]string{"id": id})
	if err != nil {
		return nil, fmt.Errorf("re-resolving %s: %w", id, err)
	}
	resp, err := parsePixabayResponse(raw)
	if err != nil {
		return nil, err
	}
	if len(resp.Hits) == 0 {
		return nil, fmt.Errorf("id %s not found on Pixabay", id)
	}
	obj, err := decodeObj(resp.Hits[0])
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// loadFreshHit returns the stored hit if present and synced within the 24h
// window, signaling whether a re-resolve is needed.
func loadFreshHit(ctx context.Context, dbPath, kind, id string) (map[string]any, bool) {
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		return nil, false
	}
	db, err := store.OpenReadOnlyContext(ctx, dbPath)
	if err != nil {
		return nil, false
	}
	defer db.Close()
	var data []byte
	var syncedAt sql.NullString
	err = db.DB().QueryRowContext(ctx,
		fmt.Sprintf(`SELECT data, synced_at FROM %q WHERE id = ?`, kind), id).Scan(&data, &syncedAt)
	if err != nil {
		return nil, false
	}
	// Treat as stale if we cannot parse the timestamp or it is older than 24h.
	if t, perr := parseStoreTime(syncedAt.String); perr == nil {
		if time.Since(t) > urlFreshWindow {
			return nil, false
		}
	} else {
		return nil, false
	}
	obj := map[string]any{}
	if jsonErr := json.Unmarshal(data, &obj); jsonErr != nil {
		return nil, false
	}
	return obj, true
}

func urlExt(url, kind string) string {
	clean := url
	if i := strings.IndexAny(clean, "?#"); i >= 0 {
		clean = clean[:i]
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext != "" && len(ext) <= 5 {
		return ext
	}
	if kind == "videos" {
		return ".mp4"
	}
	return ".jpg"
}

func parseStoreTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable time %q", s)
}
