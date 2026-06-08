// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
)

func newBatchCmd(flags *rootFlags) *cobra.Command {
	var from, to, eCol, nCol, outPath, format, rejectsPath string
	var chunkSize int
	var skipInvalid bool

	cmd := &cobra.Command{
		Use:   "batch <file.csv>",
		Short: "Convert a CSV of coordinates to CSV or GeoJSON (the killer batch feature)",
		Long: "Convert a whole CSV file of coordinates between two reference systems. Maps the\n" +
			"e/n columns (by header name; common aliases like lon/lat/est/nord/x/y are\n" +
			"auto-detected), auto-chunks at the 32000-coordinate service cap, caches results\n" +
			"for offline replay, and writes CSV (original columns + e_out/n_out) or GeoJSON.\n\n" +
			"A single out-of-grid coordinate fails its entire request; use --skip-invalid to\n" +
			"bisect and skip the offending rows while converting the rest.",
		Example: strings.Trim(`
  verto-pp-cli batch points.csv --from 3003 --to 6706 --out rdn2008.csv
  verto-pp-cli batch catasto.csv --from 3003 --to 6707 --e-col est --n-col nord
  verto-pp-cli batch mixed.csv --from 4265 --to 6706 --skip-invalid --rejects bad.csv
  verto-pp-cli batch points.csv --from 4230 --to 6706 --format geojson --out out.geojson
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		Args:        cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintf(cmd.ErrOrStderr(), "would convert coordinates from %s (from=%s to=%s, format=%s)\n", args[0], from, to, format)
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			in, out, err := requireEpsgFlags(ctx, flags, from, to)
			if err != nil {
				return err
			}
			if format != "csv" && format != "geojson" {
				return usageErr(fmt.Errorf("--format must be 'csv' or 'geojson', got %q", format))
			}
			if chunkSize <= 0 || chunkSize > volapi.MaxCoord {
				chunkSize = volapi.MaxCoord
			}

			header, records, err := readCSVFile(args[0])
			if err != nil {
				return usageErr(err)
			}
			eIdx, err := resolveColumn(header, eCol, []string{"e", "est", "easting", "lon", "long", "longitude", "x"})
			if err != nil {
				return usageErr(fmt.Errorf("east/longitude column: %w", err))
			}
			nIdx, err := resolveColumn(header, nCol, []string{"n", "nord", "northing", "lat", "latitude", "y"})
			if err != nil {
				return usageErr(fmt.Errorf("north/latitude column: %w", err))
			}

			coords := make([]volapi.Coord, len(records))
			for i, rec := range records {
				if eIdx >= len(rec) || nIdx >= len(rec) {
					return usageErr(fmt.Errorf("row %d: only %d field(s), but the e/n columns need at least %d", i+1, len(rec), max(eIdx, nIdx)+1))
				}
				e, perr := strconv.ParseFloat(strings.TrimSpace(rec[eIdx]), 64)
				if perr != nil {
					return usageErr(fmt.Errorf("row %d: invalid e value %q", i+1, rec[eIdx]))
				}
				n, perr := strconv.ParseFloat(strings.TrimSpace(rec[nIdx]), 64)
				if perr != nil {
					return usageErr(fmt.Errorf("row %d: invalid n value %q", i+1, rec[nIdx]))
				}
				coords[i] = volapi.Coord{E: e, N: n}
			}

			converted, skipped, err := batchConvert(ctx, flags, in, out, coords, skipInvalid)
			if err != nil {
				return err
			}

			skippedSet := map[int]bool{}
			for _, s := range skipped {
				skippedSet[s] = true
			}

			// --json (also set by --agent) or --jsonl emits the converted rows as
			// structured output to stdout for agents/pipelines, ignoring
			// --out/--format. printJSON routes through printOutputWithFlags, so
			// --jsonl produces one object per line.
			if flags.asJSON || flags.jsonl {
				rows := batchJSONRows(header, records, converted, skippedSet)
				if err := flags.printJSON(cmd, rows); err != nil {
					return err
				}
			} else {
				// Open output sink (file when --out, else stdout).
				w := cmd.OutOrStdout()
				if outPath != "" {
					f, ferr := os.Create(outPath)
					if ferr != nil {
						return apiErr(ferr)
					}
					defer f.Close()
					w = f
				}
				if format == "geojson" {
					if err := writeGeoJSON(w, header, records, converted, skippedSet, out); err != nil {
						return err
					}
				} else {
					if err := writeBatchCSV(w, header, records, converted, skippedSet); err != nil {
						return err
					}
				}
			}

			if len(skipped) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "skipped %d coordinate(s) outside the IGM grid (rows: %s)\n", len(skipped), rowList(skipped))
				if rejectsPath != "" {
					if err := writeRejects(rejectsPath, header, records, skipped); err != nil {
						return apiErr(err)
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "wrote rejected rows to %s\n", rejectsPath)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Source EPSG code")
	cmd.Flags().StringVar(&to, "to", "", "Target EPSG code")
	cmd.Flags().StringVar(&eCol, "e-col", "", "Name of the east/longitude column (auto-detected if empty)")
	cmd.Flags().StringVar(&nCol, "n-col", "", "Name of the north/latitude column (auto-detected if empty)")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file (default stdout)")
	cmd.Flags().StringVar(&format, "format", "csv", "Output format: csv or geojson")
	cmd.Flags().BoolVar(&skipInvalid, "skip-invalid", false, "Bisect and skip coordinates the service rejects (e.g. out-of-grid)")
	cmd.Flags().StringVar(&rejectsPath, "rejects", "", "Write skipped rows to this CSV (requires --skip-invalid)")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", volapi.MaxCoord, "Max coordinates per request (<= 32000)")
	return cmd
}

// batchConvert converts coords, optionally skipping service-rejected ones via
// bisection. The non-skip path uses the offline cache; the skip path goes
// straight to the service so it can isolate failures.
func batchConvert(ctx context.Context, flags *rootFlags, in, out int, coords []volapi.Coord, skipInvalid bool) (converted []volapi.Coord, skipped []int, err error) {
	if !skipInvalid {
		converted, err = convertWithCache(ctx, flags, in, out, coords)
		return converted, nil, err
	}
	c, err := flags.newClient()
	if err != nil {
		return nil, nil, err
	}
	converted, skipped, err = volapi.ConvertSkipping(ctx, c, in, out, coords)
	if err != nil {
		return nil, nil, classifyAPIError(err, flags)
	}
	return converted, skipped, nil
}

func readCSVFile(path string) (header []string, records [][]string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("empty CSV file")
	}
	return all[0], all[1:], nil
}

// resolveColumn finds the index of a column. An explicit name (preferred) is
// matched case-insensitively; otherwise the first matching alias wins.
func resolveColumn(header []string, explicit string, aliases []string) (int, error) {
	find := func(name string) int {
		for i, h := range header {
			if strings.EqualFold(strings.TrimSpace(h), name) {
				return i
			}
		}
		return -1
	}
	if strings.TrimSpace(explicit) != "" {
		if idx := find(explicit); idx >= 0 {
			return idx, nil
		}
		return -1, fmt.Errorf("column %q not found; available: %s", explicit, strings.Join(header, ", "))
	}
	for _, a := range aliases {
		if idx := find(a); idx >= 0 {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("could not auto-detect a column (tried %s); pass --e-col/--n-col. Available: %s",
		strings.Join(aliases, ", "), strings.Join(header, ", "))
}

// batchJSONRows builds one object per converted row: the original columns plus
// e_out/n_out as numbers. Skipped rows are omitted.
func batchJSONRows(header []string, records [][]string, converted []volapi.Coord, skipped map[int]bool) []map[string]any {
	rows := make([]map[string]any, 0, len(records))
	for i, rec := range records {
		if skipped[i] {
			continue
		}
		row := map[string]any{}
		for j, h := range header {
			if j < len(rec) {
				row[h] = rec[j]
			}
		}
		row["e_out"] = converted[i].E
		row["n_out"] = converted[i].N
		rows = append(rows, row)
	}
	return rows
}

func writeBatchCSV(w io.Writer, header []string, records [][]string, converted []volapi.Coord, skipped map[int]bool) error {
	cw := csv.NewWriter(w)
	out := append(append([]string{}, header...), "e_out", "n_out")
	if err := cw.Write(out); err != nil {
		return err
	}
	for i, rec := range records {
		if skipped[i] {
			continue
		}
		row := append(append([]string{}, rec...), fmtCoord(converted[i].E), fmtCoord(converted[i].N))
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeGeoJSON(w io.Writer, header []string, records [][]string, converted []volapi.Coord, skipped map[int]bool, outEpsg int) error {
	type geometry struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	}
	type feature struct {
		Type       string         `json:"type"`
		Geometry   geometry       `json:"geometry"`
		Properties map[string]any `json:"properties"`
	}
	type fc struct {
		Type     string         `json:"type"`
		CRS      map[string]any `json:"crs"`
		Features []feature      `json:"features"`
	}
	collection := fc{
		Type: "FeatureCollection",
		CRS: map[string]any{
			"type":       "name",
			"properties": map[string]any{"name": fmt.Sprintf("urn:ogc:def:crs:EPSG::%d", outEpsg)},
		},
	}
	for i, rec := range records {
		if skipped[i] {
			continue
		}
		props := map[string]any{}
		for j, h := range header {
			if j < len(rec) {
				props[h] = rec[j]
			}
		}
		collection.Features = append(collection.Features, feature{
			Type:       "Feature",
			Geometry:   geometry{Type: "Point", Coordinates: []float64{converted[i].E, converted[i].N}},
			Properties: props,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(collection)
}

func writeRejects(path string, header []string, records [][]string, skipped []int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, i := range skipped {
		if err := cw.Write(records[i]); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func rowList(skipped []int) string {
	parts := make([]string, 0, len(skipped))
	for i, s := range skipped {
		if i >= 10 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, strconv.Itoa(s+1))
	}
	return strings.Join(parts, ", ")
}
