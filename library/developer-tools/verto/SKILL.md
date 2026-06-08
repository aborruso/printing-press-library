---
name: pp-verto
description: "Official IGM coordinate transforms between every Italian reference system, from the command line Trigger phrases: `converti coordinate da Roma40 a RDN2008`, `trasforma coordinate IGM`, `da Gauss-Boaga a ETRS89`, `converti un CSV di coordinate`, `che EPSG hanno queste coordinate`, `usa verto`, `run verto`."
author: "aborruso"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - verto-pp-cli
---

# Verto Online — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `verto-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install verto --cli-only
   ```
2. Verify: `verto-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

verto wraps the authoritative IGM Verto Online service (Roma40, ED50, IGM95, ETRS89, RDN2008) as a scriptable CLI. Convert one coordinate or batch a whole CSV/GeoJSON, list and inspect reference systems offline, detect a dataset's real EPSG, and measure round-trip precision -- all without installing licensed NTv2 grids and all with agent-native --json/--select/--csv output.

## When to Use This CLI

Use verto whenever you need the authoritative IGM transform between Italian datums from a script, pipeline, or agent -- single points, CSV/GeoJSON batches, CI jobs, or QA checks -- instead of the browser-only IGM web form or generic PROJ math that lacks the official grids.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Precision and trust
- **`roundtrip`** — Convert a coordinate from A to B and back, and report the residual error introduced by the datum transform.

  _Reach for this when you must certify a datum chain is lossless before publishing or depositing data._

  ```bash
  verto-pp-cli roundtrip --from 3003 --to 6707 1500000 4640000 --json
  ```
- **`cache`** — Inspect or clear the local write-through cache of conversion results for offline, reproducible replay.

  _Reach for this in CI or offline pipelines where re-running the same conversions must not hit the live service._

  ```bash
  verto-pp-cli cache --stats --json
  ```

### Reference-system intelligence
- **`targets`** — List which Italian reference systems are valid destinations for a given source EPSG.

  _Reach for this before converting to avoid picking an unsupported or meaningless destination datum._

  ```bash
  verto-pp-cli targets 3003 --json
  ```
- **`inspect`** — Show datum family, axis order (e/n), units, UTM/Gauss-Boaga zone, and false easting/northing for an EPSG.

  _Reach for this to disambiguate look-alike systems (3003 vs 3004, 23032 vs 23033) before a conversion goes wrong._

  ```bash
  verto-pp-cli inspect 3004 --json
  ```
- **`detect`** — Guess the plausible source reference system of a coordinate from its magnitude and axis ranges.

  _Reach for this when a dataset is labeled vaguely ('UTM', 'Gauss-Boaga') and you must recover its real EPSG._

  ```bash
  verto-pp-cli detect 2300000 4640000 --json
  ```

## Command Reference

**systems** — Supported Italian reference systems (EPSG + description)

- `verto-pp-cli systems` — List the reference systems supported by the IGM Verto Online service


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
verto-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes

### Convert a CSV of cadastral points to RDN2008

```bash
verto-pp-cli batch catasto.csv --from 3003 --to 6706 --e-col est --n-col nord --out rdn2008.csv
```

Maps the est/nord columns, auto-chunks at 32000, writes a converted CSV.

### Pipe a deeply nested GeoJSON conversion into jq

```bash
verto-pp-cli geojson aree.geojson --from 4230 --to 6706 --agent --select features.geometry.coordinates
```

Reprojects FeatureCollection geometries and narrows the response to just the coordinate arrays for downstream tooling.

### QA a datum chain before publishing open data

```bash
verto-pp-cli roundtrip --from 3003 --to 6707 1500000 4640000 --json
```

Reports the residual error of A->B->A so you can certify the transform is within tolerance.

### Recover the real EPSG of a vaguely-labeled file

```bash
verto-pp-cli detect 2300000 4640000
```

Heuristically identifies the likely source system from coordinate magnitude when the dataset just says 'Gauss-Boaga'.

## Auth Setup

No authentication. The service is free and open; the utente/chiave fields the API requires are placeholders, filled automatically.

Run `verto-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  verto-pp-cli systems --agent --select epsg,descrizione
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
verto-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
verto-pp-cli feedback --stdin < notes.txt
verto-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.local/share/verto-pp-cli/feedback.jsonl`. They are never POSTed unless `VERTO_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `VERTO_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
verto-pp-cli profile save briefing --json
verto-pp-cli --profile briefing systems
verto-pp-cli profile list --json
verto-pp-cli profile show briefing
verto-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `verto-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add verto-pp-mcp -- verto-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which verto-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   verto-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `verto-pp-cli <command> --help`.
