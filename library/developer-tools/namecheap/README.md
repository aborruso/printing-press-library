# Namecheap CLI

Curated OpenAPI description for Namecheap's XML API. The real API uses a single
endpoint (`/xml.response`) with a `Command` query parameter plus Namecheap's
query-string authentication parameters (`ApiUser`, `ApiKey`, `UserName`, `ClientIp`).
Generation uses command-shaped pseudo paths that are normalized back to `/xml.response`
by the Namecheap printed CLI patch layer.

## Install

### Binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/namecheap-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

### Go

```
go install github.com/mvanhorn/printing-press-library/library/developer-tools/namecheap/cmd/namecheap-pp-cli@latest
```

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Set Up Credentials

Get your API key from your API provider's developer portal. The key typically looks like a long alphanumeric string.

```bash
export NAMECHEAP_API_KEY="<paste-your-key>"
```

You can also persist this in your config file at `~/.config/namecheap-pp-cli/config.toml`.

### 3. Verify Setup

```bash
namecheap-pp-cli doctor
```

This checks your configuration and credentials.

### 4. Try Your First Command

```bash
namecheap-pp-cli xml-response domains-check
```


### Namecheap authentication

Namecheap’s official XML API requires credentials as query parameters on every request: `ApiUser`, `ApiKey`, `UserName`, and `ClientIp`. The CLI maps these from:

```bash
export NAMECHEAP_USERNAME="<namecheap-user>"
export NAMECHEAP_API_KEY="<api-key>"
export NAMECHEAP_CLIENT_IP="<whitelisted-public-ip>"  # optional; auto-detected when omitted
export NAMECHEAP_SANDBOX=1                            # optional sandbox endpoint
```

Enable API access and whitelist the client IP in Namecheap before live use. Use `--dry-run` to inspect paid or mutating requests before sending them.

## Usage

Run `namecheap-pp-cli --help` for the full command reference and flag list.

## Commands

### xml-response

Manage xml response

- **`namecheap-pp-cli xml-response domains-check`** - Check domain availability for one or more domains.
- **`namecheap-pp-cli xml-response domains-create`** - Runs `namecheap.domains.create`. This is a mutating paid operation; use dry-run unless intentionally registering.
- **`namecheap-pp-cli xml-response domains-dns-get-email-forwarding`** - Runs `namecheap.domains.dns.getEmailForwarding`.
- **`namecheap-pp-cli xml-response domains-dns-get-hosts`** - Runs `namecheap.domains.dns.getHosts`.
- **`namecheap-pp-cli xml-response domains-dns-get-list`** - Get DNS nameserver type and nameservers.
- **`namecheap-pp-cli xml-response domains-dns-set-custom`** - Runs `namecheap.domains.dns.setCustom`.
- **`namecheap-pp-cli xml-response domains-dns-set-default`** - Switch a domain to Namecheap default DNS.
- **`namecheap-pp-cli xml-response domains-dns-set-hosts`** - Runs `namecheap.domains.dns.setHosts`; HostName1/RecordType1/Address1/TTL1 style parameters can be passed via --param-json in the patched CLI.
- **`namecheap-pp-cli xml-response domains-get-contacts`** - Runs `namecheap.domains.getContacts`.
- **`namecheap-pp-cli xml-response domains-get-info`** - Runs `namecheap.domains.getInfo` for a domain.
- **`namecheap-pp-cli xml-response domains-get-list`** - Runs `namecheap.domains.getList` with paging and optional filters.
- **`namecheap-pp-cli xml-response domains-get-registrar-lock`** - Runs `namecheap.domains.getRegistrarLock`.
- **`namecheap-pp-cli xml-response domains-get-tld-list`** - Runs `namecheap.domains.getTldList`.
- **`namecheap-pp-cli xml-response domains-renew`** - Runs `namecheap.domains.renew`. Mutating paid operation.
- **`namecheap-pp-cli xml-response domains-set-registrar-lock`** - Runs `namecheap.domains.setRegistrarLock`.
- **`namecheap-pp-cli xml-response ssl-get-info`** - Get SSL certificate information.
- **`namecheap-pp-cli xml-response ssl-get-list`** - Runs `namecheap.ssl.getList`.
- **`namecheap-pp-cli xml-response ssl-parse-csr`** - Parse a certificate signing request.
- **`namecheap-pp-cli xml-response users-address-get-info`** - Runs `namecheap.users.address.getInfo`.
- **`namecheap-pp-cli xml-response users-address-get-list`** - Runs `namecheap.users.address.getList`.
- **`namecheap-pp-cli xml-response users-get-balances`** - Runs `namecheap.users.getBalances`.
- **`namecheap-pp-cli xml-response users-get-pricing`** - Runs `namecheap.users.getPricing`.


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
namecheap-pp-cli xml-response domains-check

# JSON for scripting and agents
namecheap-pp-cli xml-response domains-check --json

# Filter to specific fields
namecheap-pp-cli xml-response domains-check --json --select id,name,status

# Dry run — show the request without sending
namecheap-pp-cli xml-response domains-check --dry-run

# Agent mode — JSON + compact + no prompts in one flag
namecheap-pp-cli xml-response domains-check --agent
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

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-namecheap -g
```

Then invoke `/pp-namecheap <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:

```bash
go install github.com/mvanhorn/printing-press-library/library/developer-tools/namecheap/cmd/namecheap-pp-mcp@latest
```

Then register it:

```bash
claude mcp add namecheap namecheap-pp-mcp -e NAMECHEAP_API_KEY=<your-key>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/namecheap-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `NAMECHEAP_API_KEY` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.

```bash
go install github.com/mvanhorn/printing-press-library/library/developer-tools/namecheap/cmd/namecheap-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "namecheap": {
      "command": "namecheap-pp-mcp",
      "env": {
        "NAMECHEAP_API_KEY": "<your-key>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
namecheap-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/namecheap-pp-cli/config.toml`

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `NAMECHEAP_API_KEY` | per_call | Yes | Set to your API credential. |

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `namecheap-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $NAMECHEAP_API_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
