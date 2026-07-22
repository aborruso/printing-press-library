package icaroclient

// Client per il backend nuovo /bd/ del portale ARS. A differenza del motore
// Icaro (GET default.jsp + shortList.jsp), i 3 archivi delle sedute
// (sommari 230, resoconti 217, convocazioni 229) sono stati migrati a un
// backend che risponde a POST /bd/<archivio> con HTML paginato. L'indice Icaro
// di questi 3 è congelato (sommari a giu 2025, resoconti a feb 2026), mentre
// /bd/ è corrente: per questo Search instrada qui gli archivi migrati.
//
// Vedi docs/bd-migration/API_DOCUMENTATION.md per il reverse-engineering.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// bdSpec descrive come parlare a un archivio /bd/: il path, la mappa dei filtri
// friendly (chiavi Params) verso i nomi campo POST, e i campi statici sempre
// impostati (i selettori di modalità `$S$...`, che valgono "all" = tutte le parole).
type bdSpec struct {
	path   string
	fields map[string]string // friendly key -> POST field name
	static map[string]string // campi sempre inviati (selettori modalità)
	// speakerField, se valorizzato, è il campo <select multiple> degli oratori
	// (es. "$Ispeakers"): il filtro --oratore viene risolto da nome a ID leggendo
	// le <option> del form, poi inviato su questo campo con modalità "$S..."="or".
	speakerField string
}

// bdArchives elenca gli archivi serviti dal backend /bd/. Gli altri restano su
// Icaro. Si aggiunge un archivio alla volta man mano che è verificato end-to-end.
var bdArchives = map[string]bdSpec{
	"sommari": {
		path: "sommari",
		fields: map[string]string{
			"legisl": "$Ilegislatura",
			"anno":   "anno",
			"numero": "$Iseduta_numero",
			"testo":  "$TTEXT",
		},
		static: map[string]string{"$S$TTEXT": "all", "$S$Todg": "all"},
	},
	"resoconti": {
		path: "resoconti",
		fields: map[string]string{
			"legisl": "$Ilegislatura",
			"anno":   "anno",
			"numero": "$Inrosed",
			"testo":  "$TTEXT",
		},
		static:       map[string]string{"$S$TTEXT": "all"},
		speakerField: "$Ispeakers", // --oratore risolto nome->ID dalle <option> del form
	},
}

// isBDArchive segnala se lo slug è servito dal backend /bd/.
func isBDArchive(slug string) bool {
	_, ok := bdArchives[slug]
	return ok
}

// bdSpeaker è un oratore estratto dal <select> del form: id numerico, nome e le
// legislature in cui è attivo (attributo data-legs, es. "18,17,16").
type bdSpeaker struct {
	ID   string
	Name string
	Legs string
}

// reSpeakerOption matcha le <option> del select oratori:
// `<option  value="971" data-legs="18">Abbate Ignazio</option>` (due spazi dopo
// <option nel markup del portale). Le option di legislatura/anno non hanno
// data-legs, quindi restano escluse.
var reSpeakerOption = regexp.MustCompile(`<option\s+value="(\d+)" data-legs="([^"]*)">([^<]+)</option>`)

// parseBDSpeakers estrae l'elenco oratori dal form (nome->ID + legislature).
func parseBDSpeakers(body string) []bdSpeaker {
	ms := reSpeakerOption.FindAllStringSubmatch(body, -1)
	out := make([]bdSpeaker, 0, len(ms))
	for _, m := range ms {
		out = append(out, bdSpeaker{ID: m[1], Legs: m[2], Name: unescapeMini(strings.TrimSpace(m[3]))})
	}
	return out
}

// resolveSpeakerIDs cerca gli oratori il cui nome contiene la query (case-insensitive)
// e, se legisl è dato, sono attivi in quella legislatura. Ritorna i loro ID.
func resolveSpeakerIDs(speakers []bdSpeaker, query, legisl string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var ids []string
	for _, s := range speakers {
		if !strings.Contains(strings.ToLower(s.Name), q) {
			continue
		}
		if legisl != "" && !legsContains(s.Legs, legisl) {
			continue
		}
		ids = append(ids, s.ID)
	}
	return ids
}

// legsContains riporta se legs ("18,17,16") include la legislatura leg.
func legsContains(legs, leg string) bool {
	for _, l := range strings.Split(legs, ",") {
		if strings.TrimSpace(l) == leg {
			return true
		}
	}
	return false
}

// unescapeMini decodifica le poche entità che compaiono nei nomi (apostrofo, &).
func unescapeMini(s string) string {
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// searchBD esegue una ricerca sul backend /bd/: GET di sessione, poi POST
// paginati, parsando l'HTML in Record. Onora Limit/MaxPages/Truncated come
// Search. Il filtro --data non ha un campo server (il portale filtra per
// `anno`): si deriva l'anno per il server e si filtra client-side sulla data.
func (c *Client) searchBD(ctx context.Context, arc Archive, opts SearchOptions) ([]Record, error) {
	spec := bdArchives[arc.Slug]
	bdURL := c.BaseURL + "/bd/" + spec.path

	// Sessione (cookie JSESSIONID nel jar del Client). La risposta contiene anche
	// il form, incluso il <select> degli oratori: la teniamo per risolvere --oratore.
	sessionHTML, err := c.get(ctx, bdURL)
	if err != nil {
		return nil, fmt.Errorf("bd session (%s): %w", arc.Slug, err)
	}

	form := url.Values{}
	for k, v := range spec.static {
		form.Set(k, v)
	}
	var keepDate func(rowDate string) bool
	for k, v := range opts.Params {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if k == "data" {
			anno, keep := bdDateFilter(v)
			if anno != "" {
				form.Set("anno", anno)
			}
			keepDate = keep
			continue
		}
		if k == "oratore" && spec.speakerField != "" {
			continue // risolto sotto, serve l'HTML del form
		}
		if field, ok := spec.fields[k]; ok {
			form.Set(field, v)
		}
	}

	// Filtro --oratore: risolve il nome negli ID del <select> oratori del form
	// (con le legislature in cui l'oratore è attivo) e li invia in modalità "or".
	if spec.speakerField != "" {
		if orat := strings.TrimSpace(opts.Params["oratore"]); orat != "" {
			ids := resolveSpeakerIDs(parseBDSpeakers(sessionHTML), orat, strings.TrimSpace(opts.Params["legisl"]))
			if len(ids) == 0 {
				return nil, nil // nessun oratore corrisponde: risultato vuoto
			}
			form[spec.speakerField] = ids
			form.Set("$S"+spec.speakerField, "or")
		}
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}

	var all []Record
	lastPage, lastTotal := 0, 0
	droppedByLimit := false
	for page := 1; ; page++ {
		form.Set("page", strconv.Itoa(page))
		body, err := c.post(ctx, bdURL, form)
		if err != nil {
			return all, err
		}
		rows, total, err := parseBDList(body, arc)
		if err != nil {
			return all, err
		}
		if keepDate != nil {
			kept := rows[:0]
			for _, r := range rows {
				if keepDate(r.Fields["Data"]) {
					kept = append(kept, r)
				}
			}
			rows = kept
		}
		all = append(all, rows...)
		lastPage, lastTotal = page, total

		if opts.Limit > 0 && len(all) >= opts.Limit {
			droppedByLimit = len(all) > opts.Limit
			all = all[:opts.Limit]
			break
		}
		if page >= total {
			break
		}
		// Con un filtro data attivo scorriamo tutte le pagine dell'anno (bounded),
		// così il filtro client-side vede l'intero anno; altrimenti rispettiamo
		// MaxPages come nel flusso Icaro.
		if keepDate == nil && page >= maxPages {
			break
		}
	}
	if opts.Truncated != nil {
		*opts.Truncated = droppedByLimit || (keepDate == nil && lastPage < lastTotal)
	}
	return all, nil
}

// bdDateFilter traduce un valore --data (già normalizzato da normalizeParams in
// AAMMGG, oppure ancora in YYYY-MM-DD) in: l'anno per il filtro server e una
// funzione che tiene solo le righe la cui data (dd/mm/yyyy) cade nell'intervallo.
// Ritorna keep=nil se il valore non è interpretabile (nessun filtro client).
func bdDateFilter(v string) (anno string, keep func(rowDate string) bool) {
	lo, hi, ok := parseDateBounds(v)
	if !ok {
		return "", nil
	}
	anno = lo[:4]
	return anno, func(rowDate string) bool {
		d := ddmmyyyyToISO(rowDate)
		return d != "" && d >= lo && d <= hi
	}
}

// parseDateBounds normalizza un valore data (o range) in due estremi yyyymmdd.
// Accetta: YYYY-MM-DD, YYYY-MM-DD:YYYY-MM-DD, AAMMGG, AAMMGG/AAMMGG.
func parseDateBounds(v string) (lo, hi string, ok bool) {
	split := func(s, sep string) (string, string, bool) {
		if a, b, found := strings.Cut(s, sep); found {
			return a, b, true
		}
		return s, s, false
	}
	toISO := func(s string) string {
		s = strings.TrimSpace(s)
		// YYYY-MM-DD
		if len(s) == 10 && s[4] == '-' && s[7] == '-' {
			iso := s[:4] + s[5:7] + s[8:10]
			if isDigits(iso) {
				return iso
			}
		}
		// AAMMGG
		if len(s) == 6 && isDigits(s) {
			return "20" + s
		}
		return ""
	}
	// range con ':' (ISO) o '/' (AAMMGG)
	a, b, isRange := split(v, ":")
	if !isRange {
		a, b, isRange = split(v, "/")
	}
	loISO, hiISO := toISO(a), toISO(b)
	if loISO == "" || hiISO == "" {
		return "", "", false
	}
	if loISO > hiISO {
		loISO, hiISO = hiISO, loISO
	}
	return loISO, hiISO, true
}

// isDigits riporta se s è non vuoto e composto solo da cifre.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ddmmyyyyToISO converte "14/07/2026" in "20260714". "" se non riconosciuto.
func ddmmyyyyToISO(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 10 && s[2] == '/' && s[5] == '/' {
		iso := s[6:10] + s[3:5] + s[0:2]
		if isDigits(iso) {
			return iso
		}
	}
	return ""
}

// parseBDList estrae le righe da una risposta HTML /bd/ e il numero di pagine.
// Struttura: <ul class="tabella"> con <li class="intestazione"> (header, saltato)
// e <li> per riga; ogni colonna è <div class="intesta"> con <span class="simobile">
// etichetta + <p> valore; l'ultima colonna ha <h3><a>denominazione</a></h3><p>testo</p>.
func parseBDList(body string, arc Archive) ([]Record, int, error) {
	root, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("parsing /bd/ HTML: %w", err)
	}
	totalPages := extractTotalPages(root)

	var ul *html.Node
	walk(root, func(n *html.Node) {
		if ul == nil && n.Type == html.ElementNode && n.Data == "ul" && hasClass(n, "tabella") {
			ul = n
		}
	})
	if ul == nil {
		return nil, totalPages, nil
	}
	var rows []Record
	for li := ul.FirstChild; li != nil; li = li.NextSibling {
		if li.Type != html.ElementNode || li.Data != "li" {
			continue
		}
		if hasClass(li, "intestazione") {
			continue
		}
		rec := parseBDRow(li)
		if rec.Title != "" || len(rec.Fields) > 0 {
			rows = append(rows, rec)
		}
	}
	return rows, totalPages, nil
}

// parseBDRow legge una singola <li> di risultato in un Record. Le etichette
// "N. Seduta"/"N. Foglio" vengono normalizzate anche su "Numero" così il resto
// della CLI (flatten/emit/sync) trova la chiave attesa.
func parseBDRow(li *html.Node) Record {
	rec := Record{Fields: map[string]string{}}
	for div := li.FirstChild; div != nil; div = div.NextSibling {
		if div.Type != html.ElementNode || div.Data != "div" || !hasClass(div, "intesta") {
			continue
		}
		label := strings.TrimSpace(findSimobileLabel(div))
		// La denominazione (commissione) sta in <h3><a>...; il testo/OdG nel <p>.
		if h3 := firstTextOfTag(div, "h3"); strings.TrimSpace(h3) != "" {
			rec.Title = collapseSpaces(h3)
			if p := nthPText(div, 0); strings.TrimSpace(p) != "" {
				rec.Excerpt = collapseSpaces(p)
			}
			continue
		}
		val := collapseSpaces(stripSimobileLabel(textContent(div), label))
		if val == "" || label == "" {
			continue
		}
		rec.Fields[label] = val
		switch label {
		case "N. Seduta", "N. Foglio":
			rec.Fields["Numero"] = val
		}
	}
	return rec
}

// post esegue una POST x-www-form-urlencoded usando il jar/limiter del Client.
func (c *Client) post(ctx context.Context, rawURL string, form url.Values) (string, error) {
	c.limiter.Wait()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", rawURL)
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		c.limiter.OnRateLimit()
		return "", &HTTPRateLimitError{URL: rawURL}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	c.limiter.OnSuccess()
	return string(raw), nil
}
