# Migrazione endpoint IPA: da .php a /ws/.../api/

Fonte autorevole: `docs/ipa-reference/WS00_INDICE_DEI_SERVIZI.pdf` (v2.0, 31/03/2023) + email Gestore IPA del 2026-05-18.

## Diagnosi

Il problema descritto come "HTTP 500 server-side bug" nei manuscripts NON era un bug del server: era causato dal nostro client che chiama endpoint `.php` deprecati per WS >= 18. Il PDF WS00 §3.1 chiarisce:

- WS01–WS16: disponibili sia in formato `public-ws/<WS>.php` sia in formato `ws/<Bundle>Services/api/<WS>` — entrambi validi.
- WS18, WS19, WS20, WS21, WS22, WS23: disponibili SOLO in formato `ws/<Bundle>Services/api/<WS>` — la forma `.php` non esiste lato server.

Base URL canonica: `https://www.indicepa.gov.it:443/`. Il nostro `BaseURL` attuale (`https://indicepa.gov.it/public-ws`) include il prefisso `public-ws` ed è incompleto per i nuovi path.

## Tabella mapping (estratta da WS00 §2)

| WS | Path nuovo REST | In codice oggi? |
|----|-----------------|-----------------|
| WS01 | `/ws/WS01SFECFServices/api/WS01_SFE_CF` | sì (cf.go, doctor) |
| WS02 | `/ws/WS02AOOServices/api/WS02_AOO` | sì |
| WS03 | `/ws/WS03OUServices/api/WS03_OU` | sì |
| WS04 | `/ws/WS04SFEServices/api/WS04_SFE` | sì |
| WS05 | `/ws/WS05AMMServices/api/WS05_AMM` | sì |
| WS06 | `/ws/WS06OUCODUNIServices/api/WS06_OU_COD_UNI` | sì — **attenzione: nome con underscore diverso da .php** (`WS06_OU_COD_UNI` vs `WS06_OU_CODUNI.php`) |
| WS07 | `/ws/WS07EMAILServices/api/WS07_EMAIL` | sì |
| WS08 | `/ws/WS08AOOCServices/api/WS08_AOOC` | sì |
| WS09 | `/ws/WS09DOMDIGAOOServices/api/WS09_DOMDIGAOO` | sì — **nome diverso**: `WS09_DOMDIGAOO` (senza underscore tra DOM/DIG/AOO) vs `WS09_DOM_DIG_AOO.php` |
| WS10 | `/ws/WS10DOMDIGOUServices/api/WS10_DOM_DIG_OU` | sì |
| WS11 | `/ws/WS11DOMDIGSTORAOOServices/api/WS11_DOM_DIG_STOR_AOO` | sì |
| WS12 | `/ws/WS12DOMDIGSTOROUServices/api/WS12_DOM_DIG_STOR_OU` | sì |
| WS13 | `/ws/WS13DOMDIGServices/api/WS13_DOM_DIG` | sì |
| WS14 | `/ws/WS14NSOCFServices/api/WS14_NSO_CF` | sì (cf.go, doctor) |
| WS15 | `/ws/WS15NSOServices/api/WS15_NSO` | sì |
| WS16 | `/ws/WS16DESAMMServices/api/WS16_DES_AMM` | sì |
| **WS18** | `/ws/WS18AOOServices/api/WS18_AOO` | sì — `aoo_cerca.go:50` (oggi `.php`, **da migrare**) |
| WS19 | `/ws/WS19AOOServices/api/WS19_AOO` | no (stub) |
| WS20 | `/ws/WS20PECServices/api/WS20_PEC` | no (stub) |
| WS21 | `/ws/WS21PECENTESTORServices/api/WS21_PEC_ENTE_STOR` | no (stub) |
| WS22 | `/ws/WS22PECSTORServices/api/WS22_PEC_STOR` | no (stub) |
| **WS23** | `/ws/WS23DOMDIGCFServices/api/WS23_DOM_DIG_CF` | sì — `domicilio_cf.go:35`, `cf.go:111` (oggi `.php`, **da migrare**) |

## Passaggio parametri

Il PDF non lo dice testualmente, ma l'email IPA elenca 3 modi accettati per il nuovo formato:
- `application/json`: AUTH_ID come Header, parametri nel body JSON
- `multipart/form-data`: AUTH_ID + parametri nella form
- `application/x-www-form-urlencoded`: AUTH_ID + parametri nel body

Il nostro `client.go` (riga 221 `buildFormBody`) già usa `application/x-www-form-urlencoded` con AUTH_ID nel form body → compatibile, non serve cambiare il content-type né il modo di passare AUTH_ID. Resta valido anche per gli endpoint pre-WS18 (che accettano lo stesso schema).

## Cambiamenti minimi proposti

### Fase A — Migrazione critica (WS18 + WS23)

A1. Aggiornare `internal/config/config.go` cambiando il default `BaseURL` da `https://indicepa.gov.it/public-ws` a `https://www.indicepa.gov.it`. Tutti i path dovranno diventare assoluti includendo il prefisso `/public-ws/` o `/ws/`.

A2. In `internal/cli/aoo_cerca.go:50` cambiare il path da `/WS18_AOO.php` a `/ws/WS18AOOServices/api/WS18_AOO`. Marcare con `// PATCH(...)`.

A3. In `internal/cli/domicilio_cf.go:35` e `internal/cli/cf.go:111` cambiare il path da `/WS23_DOM_DIG_CF.php` a `/ws/WS23DOMDIGCFServices/api/WS23_DOM_DIG_CF`. Aggiornare anche l'annotation `pp:path` del Cobra command. Marcare con `// PATCH(...)`.

A4. Aggiornare i path di TUTTI gli altri WS01–WS16 da `/WS<N>_<NAME>.php` a `/public-ws/WS<N>_<NAME>.php`. Solo prefisso, nessun cambio di nome o sintassi. Questo perché il BaseURL passa da `…/public-ws` a `…` (senza `/public-ws`).

A5. Registrare ogni patch in `.printing-press-patches.json` con `id`, `summary`, `reason`, `files`, `upstream_issue` (link da decidere).

### Fase B — Verifica live

B1. Con AUTH_ID valido del profilo configurato:
   - `openipa-pp-cli aoo cerca agid_aoo` → atteso HTTP 200, dati AOO AGID
   - `openipa-pp-cli domicilio cf --cf <CF noto>` → atteso 200
   - `openipa-pp-cli cf <CF noto>` (aggregato WS01+WS14+WS23) → atteso `dom_status` valido, non più `"error"`
   - Almeno una chiamata pre-WS18 di regressione: `openipa-pp-cli enti get <cod_amm>` (WS05) per confermare che il nuovo BaseURL non ha rotto i path legacy.

B2. Aggiornare `dogfood-results.json` con i nuovi PASS.

B3. Eseguire `go build ./... && go vet ./... && govulncheck ./... && go test ./...` dalla root della CLI.

### Fase C — Docs

C1. Riscrivere le 2 modifiche pendenti in `SKILL.md` / `README.md` (note HTTP 500) con la verità: "WS23 ora utilizza endpoint REST aggiornato secondo WS00_INDICE_DEI_SERVIZI v2.0". Citare il PDF in `docs/ipa-reference/`.

C2. Eliminare/correggere i passaggi nei manuscripts che dichiarano "HTTP 500 = bug server IPA". I manuscripts sono storico immutabile per definizione — meglio aggiungere una nota a margine in `docs/ipa-reference/HOW_WE_GOT_HERE.md` che chiarisca la correzione.

### Fase D (opzionale, da decidere)

D1. Implementare i comandi PEC: `openipa-pp-cli pec ente <cod_amm>` (WS20), `pec storico <cod_amm>` (WS21), `pec cerca <indirizzo>` (WS22). Richiede 3 nuovi file in `internal/cli/`. Decidere insieme se farlo ora o in una PR successiva.

D2. Implementare `openipa-pp-cli aoo storico <cod_amm>` (WS19) — restituisce AOO cessate e non cessate.

## Domande aperte (per check-in con utente)

1. **Naming dei nuovi comandi PEC**: il manuscript propone `pec ente`/`pec storico`/`pec cerca`. Mantenere? In alternativa: `pec ente`/`pec storico ente`/`pec storico pec` per simmetria con `domicilio storico-aoo`. Ti va bene la prima?

2. **WS06 e WS09 hanno nomi leggermente diversi nel nuovo formato**:
   - WS06: `_OU_COD_UNI` (nuovo) vs `_OU_CODUNI` (PHP)
   - WS09: `_DOMDIGAOO` (nuovo) vs `_DOM_DIG_AOO` (PHP)
   
   La migrazione mantiene i due path distinti per ogni WS (legacy + nuovo) o passa subito al solo formato nuovo? **Proposta**: passare al solo formato nuovo per tutti i WS, è più semplice e l'IPA stessa indica il REST come canonico.

3. **Cache HTTP**: la cache locale (`~/.cache/openipa-pp-cli/http/`) si invalida da sola? No — le key sono basate su path. Cambiando i path le entry vecchie diventano garbage ma non rotte. Lascio così oppure aggiungo un'invalidazione one-shot al primo run post-aggiornamento?

4. **Versione printing-press**: il client.go ha header "DO NOT EDIT" perché generato. Modifico comunque a mano e registro in `.printing-press-patches.json` come previsto da AGENTS.md? (Sì, è il flusso canonico per fix di CLI già pubblicata.)

5. **Fase D va inclusa in questa PR o in una successiva?** Mia opinione: includere ora, perché siamo già nel cuore del codice IPA e i comandi PEC erano stub frustranti.

## Mio piano d'esecuzione proposto

1. Aspetto le tue risposte alle 5 domande sopra.
2. Faccio le modifiche delle Fasi A + B + C.
3. (Se sì alla domanda 5) faccio anche Fase D nello stesso branch.
4. Test live con auth tua se preferisci farli tu, oppure tu mi passi un AUTH_ID di test.
5. Quando tutto verde, lanciamo `/printing-press-publish openipa` per la PR di follow-up al catalogo.
