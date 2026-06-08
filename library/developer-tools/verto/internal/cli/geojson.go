// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
	"github.com/spf13/cobra"
)

func newGeojsonCmd(flags *rootFlags) *cobra.Command {
	var from, to, outPath string

	cmd := &cobra.Command{
		Use:   "geojson <file.geojson>",
		Short: "Reproject the geometries of a GeoJSON file between reference systems",
		Long: "Reproject every geometry coordinate in a GeoJSON file (Feature, FeatureCollection,\n" +
			"or a bare geometry) from --from to --to using the official IGM transform, and\n" +
			"write the converted GeoJSON. GeoJSON positions are [e, n] (lon, lat for geographic\n" +
			"systems), which matches the service's axis order directly. A CRS member naming the\n" +
			"target EPSG is added to the output.",
		Example: strings.Trim(`
  verto-pp-cli geojson aree.geojson --from 4230 --to 6706 --out out.geojson
  cat in.geojson | verto-pp-cli geojson - --from 3003 --to 6706 > out.geojson
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		Args:        cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintf(cmd.ErrOrStderr(), "would reproject %s (from=%s to=%s)\n", args[0], from, to)
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			in, out, err := requireEpsgFlags(ctx, flags, from, to)
			if err != nil {
				return err
			}

			raw, err := readInput(cmd, args[0])
			if err != nil {
				return usageErr(err)
			}
			var doc any
			if err := json.Unmarshal(raw, &doc); err != nil {
				return usageErr(fmt.Errorf("invalid GeoJSON: %w", err))
			}

			var coords []volapi.Coord
			var setters []func(volapi.Coord)
			collectPositions(doc, &coords, &setters)
			if len(coords) == 0 {
				return usageErr(fmt.Errorf("no coordinates found in the GeoJSON document"))
			}

			converted, err := convertWithCache(ctx, flags, in, out, coords)
			if err != nil {
				return err
			}
			for i, set := range setters {
				set(converted[i])
			}
			doc = setGeoJSONCRS(doc, out)

			// The output is already GeoJSON; --json forces it to stdout for
			// pipelines instead of an --out file.
			w := cmd.OutOrStdout()
			if outPath != "" && !flags.asJSON {
				f, ferr := os.Create(outPath)
				if ferr != nil {
					return apiErr(ferr)
				}
				defer f.Close()
				w = f
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(doc)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Source EPSG code")
	cmd.Flags().StringVar(&to, "to", "", "Target EPSG code")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file (default stdout)")
	return cmd
}

// readInput reads a file, or stdin when path is "-".
func readInput(cmd *cobra.Command, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}

// collectPositions walks a decoded GeoJSON value and records every innermost
// [x, y, ...] position along with a setter that writes the converted x/y back.
func collectPositions(v any, coords *[]volapi.Coord, setters *[]func(volapi.Coord)) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == "coordinates" {
				walkCoordinates(child, coords, setters)
				continue
			}
			collectPositions(child, coords, setters)
		}
	case []any:
		for _, child := range t {
			collectPositions(child, coords, setters)
		}
	}
}

// walkCoordinates descends a coordinates value (arbitrarily nested arrays) until
// it reaches innermost positions ([num, num, ...]).
func walkCoordinates(v any, coords *[]volapi.Coord, setters *[]func(volapi.Coord)) {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return
	}
	// A position is a flat array whose first two elements are numbers.
	if len(arr) >= 2 {
		if e, eok := arr[0].(float64); eok {
			if n, nok := arr[1].(float64); nok {
				*coords = append(*coords, volapi.Coord{E: e, N: n})
				pos := arr // capture the slice; elements are mutated in place
				*setters = append(*setters, func(c volapi.Coord) {
					pos[0] = c.E
					pos[1] = c.N
				})
				return
			}
		}
	}
	for _, child := range arr {
		walkCoordinates(child, coords, setters)
	}
}

// setGeoJSONCRS adds/updates a top-level CRS member naming the target EPSG.
func setGeoJSONCRS(v any, epsg int) any {
	if m, ok := v.(map[string]any); ok {
		m["crs"] = map[string]any{
			"type":       "name",
			"properties": map[string]any{"name": fmt.Sprintf("urn:ogc:def:crs:EPSG::%d", epsg)},
		}
	}
	return v
}
