# Verto Online CLI

**Official IGM coordinate transforms between every Italian reference system, from the command line, with zero grid-file setup.**

verto wraps the authoritative IGM Verto Online service (Roma40, ED50, IGM95, ETRS89, RDN2008) as a scriptable CLI. Convert one coordinate or batch a whole CSV/GeoJSON, list and inspect reference systems offline, detect a dataset's real EPSG, and measure round-trip precision -- all without installing licensed NTv2 grids and all with agent-native --json/--select/--csv output.

## Install

The recommended path installs both the `verto-pp-cli` binary and the `pp-verto` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install verto
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install verto --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install verto --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install verto --agent claude-code
npx -y @mvanhorn/printing-press-library install verto --agent claude-code --agent codex
```

### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/verto-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install verto --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-verto --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-verto --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install verto --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/verto-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "verto": {
      "command": "verto-pp-mcp"
    }
  }
}
```

</details>

## Authentication

No authentication. The service is free and open; the utente/chiave fields the API requires are placeholders, filled automatically.

## Quick Start

```bash
# see the 20 supported Italian reference systems (cached offline after first run)
verto-pp-cli systems

# convert one geographic coordinate Roma40 -> RDN2008
verto-pp-cli convert --from 4265 --to 6706 12.4924 41.8902

# check axis order, zone and false easting before a projected conversion
verto-pp-cli inspect 3003

# batch-convert a CSV of Gauss-Boaga points to RDN2008/TM32
verto-pp-cli batch points.csv --from 3003 --to 6707 --out converted.csv

```

## Unique Features

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

## Usage

Run `verto-pp-cli --help` for the full command reference and flag list.

## Commands

### systems

Supported Italian reference systems (EPSG + description)

- **`verto-pp-cli systems`** - List the reference systems supported by the IGM Verto Online service


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
verto-pp-cli systems

# JSON for scripting and agents
verto-pp-cli systems --json

# JSON Lines / NDJSON — one compact object per line (great for jq / streaming)
verto-pp-cli systems --jsonl

# Filter to specific fields
verto-pp-cli systems --json --select epsg,descrizione

# Dry run — show the request without sending
verto-pp-cli systems --dry-run

# Agent mode — JSON + compact + no prompts in one flag
verto-pp-cli systems --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
verto-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: ``

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **stato: errore, 'Manca l'elemento n'** — every coordinate needs both e (est/lon) and n (nord/lat); check your CSV column mapping with --e-col/--n-col.
- **Converted points land far away / in the sea** — wrong source EPSG or swapped axis. Run 'verto-pp-cli detect <e> <n>' to recover the likely system and confirm axis order with 'inspect'.
- **Batch rejected for too many coordinates** — the service caps at 32000 coordinates per request; batch auto-chunks, so re-run -- if it persists, reduce --chunk-size.
