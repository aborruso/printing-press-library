// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/developer-tools/verto/internal/volapi"
	"github.com/spf13/cobra"
)

// convRow is one converted coordinate, carrying both input and output so the
// correlation survives JSON/CSV piping.
type convRow struct {
	InE  float64 `json:"in_e"`
	InN  float64 `json:"in_n"`
	OutE float64 `json:"out_e"`
	OutN float64 `json:"out_n"`
}

func newConvertCmd(flags *rootFlags) *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:   "convert [<e> <n> ...]",
		Short: "Convert one or more coordinates between Italian reference systems",
		Long: "Convert coordinates between two Italian reference systems using the official IGM\n" +
			"Verto Online transform. Pass coordinate pairs as positional arguments (e first,\n" +
			"n second: e=est/longitude, n=nord/latitude), or pipe 'e,n' lines on stdin.\n" +
			"Geographic coordinates are always decimal degrees.",
		Example: strings.Trim(`
  # Roma40 geographic -> RDN2008 geographic (lon lat)
  verto-pp-cli convert --from 4265 --to 6706 12.4924 41.8902

  # Gauss-Boaga fuso Ovest -> RDN2008/TM32 (easting northing), JSON out
  verto-pp-cli convert --from 3003 --to 6707 1500000 4640000 --json

  # Many coordinates from stdin, CSV out
  printf '12.49,41.89\n13.0,42.0\n' | verto-pp-cli convert --from 4265 --to 6706 --csv
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			in, out, err := requireEpsgFlags(ctx, flags, from, to)
			if err != nil {
				return err
			}

			coords, err := readCoordArgs(cmd, args)
			if err != nil {
				return usageErr(err)
			}
			if len(coords) == 0 {
				return cmd.Help()
			}

			converted, err := convertWithCache(ctx, flags, in, out, coords)
			if err != nil {
				return err
			}

			rows := make([]convRow, len(coords))
			for i := range coords {
				rows[i] = convRow{InE: coords[i].E, InN: coords[i].N, OutE: converted[i].E, OutN: converted[i].N}
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				items := make([]map[string]any, len(rows))
				for i, r := range rows {
					items[i] = map[string]any{
						"in_e": fmtCoord(r.InE), "in_n": fmtCoord(r.InN),
						"out_e": fmtCoord(r.OutE), "out_n": fmtCoord(r.OutN),
					}
				}
				return printAutoTable(cmd.OutOrStdout(), items)
			}
			return flags.printJSON(cmd, rows)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Source EPSG code (e.g. 3003)")
	cmd.Flags().StringVar(&to, "to", "", "Target EPSG code (e.g. 6706)")
	return cmd
}

// readCoordArgs reads coordinate pairs from positional args, or from stdin when
// no positionals were given and stdin is piped.
func readCoordArgs(cmd *cobra.Command, args []string) ([]volapi.Coord, error) {
	if len(args) > 0 {
		if len(args)%2 != 0 {
			return nil, fmt.Errorf("coordinates must come in pairs of <e> <n>; got %d values", len(args))
		}
		coords := make([]volapi.Coord, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			e, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid e (est/longitude) value %q", args[i])
			}
			n, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid n (nord/latitude) value %q", args[i+1])
			}
			coords = append(coords, volapi.Coord{E: e, N: n})
		}
		return coords, nil
	}

	// No positional args: read 'e,n' (or 'e n', tab-separated) lines from stdin.
	in := cmd.InOrStdin()
	var coords []volapi.Coord
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		co, err := parseCoordLine(raw)
		if err != nil {
			return nil, fmt.Errorf("stdin line %d: %w", line, err)
		}
		coords = append(coords, co)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return coords, nil
}

func parseCoordLine(raw string) (volapi.Coord, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t'
	})
	if len(fields) != 2 {
		return volapi.Coord{}, fmt.Errorf("want exactly two values 'e,n', got %q", raw)
	}
	e, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
	if err != nil {
		return volapi.Coord{}, fmt.Errorf("invalid e value %q", fields[0])
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
	if err != nil {
		return volapi.Coord{}, fmt.Errorf("invalid n value %q", fields[1])
	}
	return volapi.Coord{E: e, N: n}, nil
}
