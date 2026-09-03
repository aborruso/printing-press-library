# Pixabay CLI

**Every Pixabay search filter the API has — plus a 24-hour local cache, offline search, resumable downloads with built-in attribution, live rate-limit awareness, and agent-native JSON no other Pixabay tool ships.**

The only maintained Pixabay CLI is ten years dead and images-only; the only agent-native tool emits text you have to scrape. This one unifies image and video search, caches results in a local SQLite store for up to 24 hours (per Pixabay's API Terms) so you can search offline with `search`, credits Pixabay with a visible link on every result, and adds commands the raw API can't do: `media search` merges photos and video, `pull` does resumable downloads with attribution and 24h URL re-resolution, and `quota` surfaces the rate-limit headers nobody else exposes so you can pace within limits.

## Install

The recommended path installs both the `pixabay-pp-cli` binary and the `pp-pixabay` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install pixabay
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install pixabay --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install pixabay --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install pixabay --agent claude-code
npx -y @mvanhorn/printing-press-library install pixabay --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.4 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/cmd/pixabay-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/pixabay-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install pixabay --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-pixabay --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-pixabay --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install pixabay --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/pixabay-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `PIXABAY_API_KEY` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/cmd/pixabay-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "pixabay": {
      "command": "pixabay-pp-mcp",
      "env": {
        "PIXABAY_API_KEY": "<your-key>"
      }
    }
  }
}
```

</details>

## Authentication

Pixabay uses a single free API key passed as a query parameter. Get one from your account at pixabay.com/api/docs (when logged in), then set PIXABAY_API_KEY in your environment or run `pixabay-pp-cli auth set-token <key>`. The API is read-only — search only, no writes.

## Pixabay API Compliance

This CLI is built to honor the [Pixabay API Terms of Service](https://pixabay.com/api/docs/). When you use it, you agree to:

1. **Credit Pixabay with a visible link** whenever you display search results. The CLI prints a Pixabay credit line on human output and returns a `pageURL` (Pixabay link) with every result — surface it wherever results appear.
2. **Cache, don't archive.** API responses are cached locally for up to 24 hours to avoid repeated identical queries, then expire. The local SQLite store is a 24-hour cache, not a permanent dataset.
3. **Keep usage user-triggered.** Every invocation must correspond to a real user request. Do not run this CLI from cron jobs, background daemons, or unattended automation.
4. **No AI/ML or scraping.** Pixabay content may not be used for AI or machine-learning training, dataset generation, or automated scraping.

These obligations are why this CLI ships without a bulk-corpus "harvest" command and treats its local store as a 24-hour cache.

## Quick Start

```bash
# Confirm the binary and config wiring before you need a key.
pixabay-pp-cli doctor --dry-run

# The core workflow: search photos by query with filters.
pixabay-pp-cli images search "yellow flowers" --image-type photo --per-page 5

# Search photos and videos together, structured JSON for an agent.
pixabay-pp-cli media search "drone coastline" --limit 20 --agent

# Persist results into the local SQLite store for offline search.
pixabay-pp-cli sync --resources images --param q=mountains --param image_type=photo

# Offline full-text search over what you've synced — zero API quota.
pixabay-pp-cli search "mountain" --type images

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Search & acquisition
- **`pull`** — Download chosen size variants for a collection with parallel workers, re-resolving 24h-expired URLs by id and writing per-file attribution sidecars; re-runs skip what's already on disk.

  _Use this to materialize an already-curated collection to disk safely; it survives interruptions and expired URLs that break naive downloaders._

  ```bash
  pixabay-pp-cli pull --from-collection winter --size large --workers 8 --resume
  ```
- **`media search`** — Fan one query across both the image and video endpoints in parallel and merge stills and footage into a single ranked, persisted result set.

  _Use this when a task needs both photos and video for a theme and you don't want to run two searches and merge by hand._

  ```bash
  pixabay-pp-cli media search "drone coastline" --limit 40 --agent
  ```

### Agent-native plumbing
- **`quota`** — Surface the X-RateLimit-Limit/Remaining/Reset headers from your last call (which no other Pixabay tool exposes) and project how many requests a planned batch of requests will cost before it throttles.

  _Check this before a big batch so an agent can pace itself instead of dying mid-run on an HTTP 429._

  ```bash
  pixabay-pp-cli quota --agent
  ```

### Local state that compounds
- **`similar`** — Find synced hits that share the most tags with a given id using a local tag-set overlap (Jaccard) score — no API endpoint for this exists.

  _Reach for this to expand a curated set with visually-related assets you already pulled, with zero extra API quota._

  ```bash
  pixabay-pp-cli similar 195893 --limit 20 --agent
  ```
- **`trends`** — Snapshot views/downloads/likes/comments on each sync and report the deltas per tag or id between sync runs.

  _Use this to see which themes are gaining downloads/likes week over week — impossible from a single API response._

  ```bash
  pixabay-pp-cli trends --tag winter --since 7d --agent
  ```
- **`contributors`** — Aggregate synced image and video hits by contributor and rank them by total or average downloads, likes, or views across your store.

  _Use this to find the strongest contributors for a theme so you can source consistently from people whose work performs._

  ```bash
  pixabay-pp-cli contributors --by downloads --min-assets 3 --agent
  ```
- **`collection`** — Group synced hits into named local collections that feed downstream pull, similar, and trends — persistence no Pixabay tool offers.

  _Use this as the backbone for curation — build a set once, then download, expand, or track it later without re-searching._

  ```bash
  pixabay-pp-cli collection add winter 195893,1850181
  ```

## Recipes


### Find Editor's Choice nature photos, agent-ready

```bash
pixabay-pp-cli images search "forest" --image-type photo --category nature --editors-choice --order popular --agent --select hits.id,hits.tags,hits.largeImageURL,hits.likes
```

Narrows to award-winning nature photos and returns only the fields an agent needs, keeping payloads small.

### Search videos and pull the smallest rendition URL

```bash
pixabay-pp-cli videos search "ocean waves" --video-type film --per-page 10 --agent --select hits.id,hits.duration,hits.videos.tiny.url,hits.videos.large.url
```

Video hits nest renditions deeply (videos.tiny.url, videos.large.url); --select with dotted paths extracts just the URLs you want.

### Download a curated collection with attribution

```bash
pixabay-pp-cli pull --from-collection campaign --size large --workers 6
```

After adding IDs with 'collection add campaign ...', materialize the set to disk with parallel workers and per-file credit sidecars.

### Check quota before a batch

```bash
pixabay-pp-cli quota --agent
```

Reads the persisted rate-limit headers so an automated job can pace itself instead of failing on a 429.

## Usage

Run `pixabay-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `PIXABAY_CONFIG_DIR`, `PIXABAY_DATA_DIR`, `PIXABAY_STATE_DIR`, or `PIXABAY_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `PIXABAY_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export PIXABAY_HOME=/srv/pixabay
pixabay-pp-cli doctor
```

Under `PIXABAY_HOME=/srv/pixabay`, the four dirs resolve to `/srv/pixabay/config`, `/srv/pixabay/data`, `/srv/pixabay/state`, and `/srv/pixabay/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "pixabay": {
      "command": "pixabay-pp-mcp",
      "env": {
        "PIXABAY_HOME": "/srv/pixabay"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `PIXABAY_DATA_DIR` overrides an explicit `--home` for that kind. Use `PIXABAY_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `PIXABAY_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `pixabay-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### images

Search Pixabay photos, illustrations, and vectors

- **`pixabay-pp-cli images get`** - Get image(s) by ID
- **`pixabay-pp-cli images search`** - Search images by query and filters

### videos

Search Pixabay videos (film and animation)

- **`pixabay-pp-cli videos get`** - Get video(s) by ID
- **`pixabay-pp-cli videos search`** - Search videos by query and filters


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
pixabay-pp-cli images get mock-value

# JSON for scripting and agents
pixabay-pp-cli images get mock-value --json

# Filter to specific fields
pixabay-pp-cli images get mock-value --json --select id,name,status

# Dry run — show the request without sending
pixabay-pp-cli images get mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
pixabay-pp-cli images get mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
pixabay-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `pixabay-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/pixabay-pp-cli/config.toml`; `--home`, `PIXABAY_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `PIXABAY_API_KEY` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `pixabay-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `pixabay-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $PIXABAY_API_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **HTTP 400 "Invalid or missing API key"** — Set PIXABAY_API_KEY=<your-key> or run `pixabay-pp-cli auth set-token <key>`; get a free key at pixabay.com/api/docs.
- **Search errors when paging deep / per_page high** — Pixabay caps results at 500 per query (page × per_page must be ≤ 500). Narrow your query with filters (category, colors, image-type) instead of paging past the cap.
- **Downloaded image URLs 404 a day later** — Pixabay image URLs expire after 24h. Re-run `sync` or use `pull`, which re-resolves expired URLs by id before downloading.
- **Batch job dies on HTTP 429** — Run `pixabay-pp-cli quota` to see remaining requests; pace large `harvest`/`pull` runs under the limit.

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**gopixabay**](https://github.com/netbrain/gopixabay) — Go (30 stars)
- [**python-pixabay**](https://github.com/momozor/python-pixabay) — Python (30 stars)
- [**pixabay-mcp**](https://github.com/zym9863/pixabay-mcp) — TypeScript (7 stars)
- [**pixabayjs**](https://github.com/leonardseymore/pixabayjs) — JavaScript (5 stars)
- [**pixabay-cli-downloader**](https://github.com/AlexusBlack/pixabay-cli-downloader) — Python (3 stars)

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
