# ANAC Pubblicità a Valore Legale CLI

**Cerca bandi, esiti e avvisi di gara della piattaforma ANAC dalla riga di comando, con dettaglio JSON, cronologia, store locale e ricerca offline.**

anac-pl espone l'API pubblica della Piattaforma di Pubblicità a Valore Legale di ANAC come CLI agent-native: ricerca full-text con filtri (data, importo, CPV, tipologia), dettaglio JSON completo degli esiti, cronologia delle rettifiche, e un database SQLite locale per ricerca offline ed export CSV/JSON. Nessuna autenticazione richiesta.

Printed by [@aborruso](https://github.com/aborruso) (aborruso).

## Install

### Dal catalogo Printing Press

Una volta che questa CLI e' nel catalogo, l'installer fa tutto in un comando, binario piu' skill per gli agent:

```bash
npx -y @mvanhorn/printing-press-library install anac-pl
```

Solo il binario, senza skill:

```bash
npx -y @mvanhorn/printing-press-library install anac-pl --cli-only
```

Senza Node, con Go 1.26.6 o superiore:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/anac-pl/cmd/anac-pl-pp-cli@latest
```

### Binario precompilato dal repo dell'autore

Scarica l'archivio per la tua piattaforma dall'[ultima release](https://github.com/aborruso/anac-pl-pp-cli/releases/latest), scompattalo e metti il binario in una cartella del `PATH`.

```bash
# Linux x86-64
curl -sL https://github.com/aborruso/anac-pl-pp-cli/releases/latest/download/anac-pl-pp-cli_linux_amd64.tar.gz | tar xz
chmod +x anac-pl-pp-cli
./anac-pl-pp-cli doctor
```

Su macOS, la prima volta va tolta la quarantena di Gatekeeper: `xattr -d com.apple.quarantine anac-pl-pp-cli`.

### Dai sorgenti (richiede Go 1.26.6 o superiore)

```bash
git clone https://github.com/aborruso/anac-pl-pp-cli.git
cd anac-pl-pp-cli
go build -o anac-pl-pp-cli ./cmd/anac-pl-pp-cli
```

Nessuna configurazione, nessuna chiave: l'API di ANAC e' pubblica e di sola lettura.

## Uso con Claude Desktop e altri client MCP

Il repo contiene anche un server MCP, `anac-pl-pp-mcp`, che espone la ricerca avvisi agli assistenti che parlano quel protocollo.

```bash
go build -o anac-pl-pp-mcp ./cmd/anac-pl-pp-mcp
```

Poi nella configurazione del client (per Claude Desktop `~/Library/Application Support/Claude/claude_desktop_config.json`, su Windows `%APPDATA%\Claude\claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "anac-pl": {
      "command": "/percorso/assoluto/anac-pl-pp-mcp"
    }
  }
}
```

Per gli agenti che leggono le skill (Claude Code, Codex, Cursor, ...) il repo contiene `SKILL.md`: copialo nella cartella delle skill del tuo agente.

## Quick Start

```bash
# Verifica raggiungibilita' dell'API
anac-pl-pp-cli doctor

# Ultimi bandi pubblicati, dal piu' recente
anac-pl-pp-cli cerca --tipologia bandi --size 10 --sort-field dataPubblicazione --sort-dir DESC

# Trova il codice CPV partendo dalle parole
anac-pl-pp-cli cpv search "posta elettronica"

# Filtro CPV che seleziona davvero per codice (ricerca avanzata del portale)
anac-pl-pp-cli cerca-avanzata --cpv 30213000

# Tabella committente -> aggiudicatario -> importo -> CIG -> CPV -> giurisdizione
anac-pl-pp-cli affidamenti --cpv-code 72412000 -t "" --pages 3 --from-search --csv

# Tipologie di avviso e valore da usare come filtro
anac-pl-pp-cli tipologie list
```

## Un limite messo apposta: una chiamata al secondo, una istanza per volta

La piattaforma di ANAC e' un servizio pubblico che non dichiara alcuna quota, e non c'e' un modo per chiedere piu' banda. Il tetto quindi e' scritto nel programma e non e' configurabile: al massimo una richiesta al secondo.

Il ritmo e' condiviso fra processi, non solo dentro il processo: l'istante dell'ultima chiamata sta in `~/.cache/anac-pl-pp-cli/pace.lock`, protetto da un lock esclusivo di sistema. Anche facendo lavorare insieme la CLI e il server MCP, la somma resta una chiamata al secondo.

Due copie della CLI non girano in parallelo: la seconda aspetta il proprio turno, dicendolo su stderr, fino a cinque minuti; scaduti quelli esce con codice 7. In modalita' non interattiva (`--no-input`, e quindi `--agent`) non attende affatto ed esce 7 subito, perche' uno script preferisce un errore immediato a un comando che tace. Il lock e' `~/.cache/anac-pl-pp-cli/instance.lock`, rilasciato dal sistema operativo alla fine del processo, quindi non resta mai appeso.

Il lock non e' pero' cio' che garantisce il tetto - quello lo tiene `pace.lock`, che vale per un numero qualunque di processi. Serve come rete per il caso in cui quel file non sia utilizzabile e il ritmo torni a valere per il solo processo. Per questo `doctor` lo salta: un controllo di salute deve rispondere anche mentre un `sync` lavora.

Ogni richiesta si presenta con un `User-Agent` che dichiara nome, versione e questo repository, cosi' che chi amministra la piattaforma possa riconoscere il traffico e, se desse fastidio, scrivere invece di bloccare.

`--rate-limit` esiste ancora, ma serve solo a rallentare ulteriormente: `--rate-limit 0.2` scende a una chiamata ogni cinque secondi, `--rate-limit 100` non alza nulla.

La conseguenza pratica e' che le scansioni lunghe sono lente per costruzione: `sync` di molte pagine va lanciato e lasciato lavorare. Per le analisi ripetute conviene sincronizzare una volta e poi interrogare lo store locale con `search-local` ed `export`, che non toccano la rete.

## Due avvertenze sui dati

Il campo CPV di `cerca` non e' un filtro sul codice ma un match testuale: restituisce anche avvisi con CPV estranei. Per selezionare davvero per codice serve `cerca-avanzata`, che usa l'endpoint della ricerca avanzata rilasciata in beta a luglio 2026. La CLI lo segnala su stderr quando usi `cerca --cpv`.

Il numero di risultati dichiarato dal servizio, sugli aggregati, e' una stima progressiva: cambia mentre sfogli le pagine e fra chiamate identiche. Va usato come ordine di grandezza, non come totale. I codici CPV completi a 8 cifre sono invece stabili.

## Unique Features

These capabilities aren't available in any other tool for this API.

### Stato locale che si accumula
- **`sync`** — Scarica e conserva gli avvisi in un database SQLite locale per analisi offline.

  _Permette analisi ripetute e aggregazioni che l'API paginata non offre._

  ```bash
  anac-pl-pp-cli sync --resources avvisi --param keywords=microsoft
  ```
- **`search-local`** — Cerca tra gli avvisi gia' sincronizzati in locale, senza rete.

  _Risposte istantanee e componibili con jq/SQL._

  ```bash
  anac-pl-pp-cli search-local microsoft
  ```
- **`export`** — Esporta gli avvisi sincronizzati in CSV o JSON per fogli di calcolo e pipeline dati.

  _Porta i dati ANAC direttamente in strumenti di analisi._

  ```bash
  anac-pl-pp-cli export avvisi
  ```

## Recipes


### Esiti recenti per parola chiave

```bash
anac-pl-pp-cli avvisi search --query 'servizi informatici' --scheda P7_1_1 --size 20
```

Filtra i risultati di gara per oggetto.

### Dettaglio JSON di un esito

```bash
anac-pl-pp-cli avvisi get c5bfcc8d-ebed-4b6b-ab5f-661d78fa88e2 --json
```

Recupera il JSON completo del detail page.

### Estrai solo campi chiave

```bash
anac-pl-pp-cli avvisi search --query microsoft --agent --select idAvviso,codiceScheda,dataPubblicazione,score
```

Output compatto per agenti su risposte voluminose.

## Usage

Run `anac-pl-pp-cli --help` for the full command reference and flag list.

## Commands

### avvisi

Ricerca e consultazione di bandi, esiti e avvisi pubblicati sulla Piattaforma di Pubblicità a Valore Legale ANAC

- **`anac-pl-pp-cli avvisi cronologia`** - Cronologia delle versioni/rettifiche di un avviso nel tempo
- **`anac-pl-pp-cli avvisi get`** - Dettaglio completo di un avviso/esito in formato JSON, incluse sezioni e committente
- **`anac-pl-pp-cli avvisi search`** - Ricerca full-text di avvisi (bandi, esiti, altri avvisi) con ranking di rilevanza e filtri

### news

Avvisi e comunicazioni della piattaforma

- **`anac-pl-pp-cli news`** - Ultime news pubblicate sulla piattaforma

### tipologie

Tassonomia delle tipologie di avviso (categorie, tipologie e codici scheda)

- **`anac-pl-pp-cli tipologie`** - Mappa di categorie, tipologie e codici scheda usati per filtrare la ricerca


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
anac-pl-pp-cli avvisi get mock-value

# JSON for scripting and agents
anac-pl-pp-cli avvisi get mock-value --json

# Filter to specific fields
anac-pl-pp-cli avvisi get mock-value --json --select id,name,status

# Dry run — show the request without sending
anac-pl-pp-cli avvisi get mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
anac-pl-pp-cli avvisi get mock-value --agent
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

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
anac-pl-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/anac-pl-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

- **`search-local` esce 3 quando la ricerca non trova nulla**: e' la risposta, non un guasto. Lo stdout resta quello normale (`[]` in JSON), cambia solo il codice di uscita, cosi' uno script distingue «nessun avviso» da «trovati».

### API-specific
- **HTTP 500 sui filtri data** — Usa il formato GG/MM/AAAA per --published-from e --published-to
- **ricerca archivio vuota** — Con --archive specifica un intervallo date inferiore a 6 mesi
