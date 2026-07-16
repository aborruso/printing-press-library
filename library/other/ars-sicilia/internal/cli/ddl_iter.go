// pp:data-source live
// pp:client-call
// Novel feature — ricostruisce la cronologia completa di un DDL.
// Combina ricerche su più archivi (DDL 221, sommari commissione 230,
// resoconti d'aula 217) usando direttamente l'icaroclient.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
	"github.com/spf13/cobra"
)

func newNovelDdlIterCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "iter <legisl> <numero>",
		Short:   "Ricostruisce la cronologia completa di un disegno di legge: presentazione, commissione, aula, eventuale legge.",
		Example: "  ars-sicilia-pp-cli ddl iter 18 1500 --json",
		Args:    cobra.MaximumNArgs(2),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				if dryRunOK(flags) || cliIsVerify() {
					return cmd.Help()
				}
				return usageErr(fmt.Errorf("richiesti 2 argomenti: <legisl> e <numero>"))
			}
			if dryRunOK(flags) {
				return nil
			}
			legisl, err := atoiArg(args[0], "legisl")
			if err != nil {
				return err
			}
			numero, err := atoiArg(args[1], "numero")
			if err != nil {
				return err
			}
			return runDdlIter(cmd, flags, legisl, numero)
		},
	}
	return cmd
}

type iterEvent struct {
	Fase      string `json:"fase"`
	Data      string `json:"data,omitempty"`
	Sede      string `json:"sede,omitempty"`
	Titolo    string `json:"titolo,omitempty"`
	Oratori   string `json:"oratori,omitempty"`
	URL       string `json:"url,omitempty"`
	ArchiveID string `json:"archive_id,omitempty"`
	DocID     int    `json:"doc_id,omitempty"`
}

type iterReport struct {
	Legisl int         `json:"legisl"`
	Numero int         `json:"numero"`
	Titolo string      `json:"titolo,omitempty"`
	Eventi []iterEvent `json:"eventi"`
	Note   string      `json:"note,omitempty"`
}

func runDdlIter(cmd *cobra.Command, flags *rootFlags, legisl, numero int) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	c, err := icaro.New(nil)
	if err != nil {
		return err
	}
	report := iterReport{Legisl: legisl, Numero: numero, Eventi: []iterEvent{}}

	// 1. DDL stesso (archivio 221): presentazione + apertura del documento per
	// leggere l'iter reale (sezione "Attuale … Storico" nel corpo del doc).
	arc := icaro.BySlug("ddl")
	if arc == nil {
		report.Note = "archivio ddl non disponibile"
		return emitIter(cmd, flags, report)
	}
	recs, err := c.Search(ctx, *arc, icaro.SearchOptions{
		Params: map[string]string{"legisl": itoa(legisl), "numero": itoa(numero)},
		Limit:  1,
	})
	if err != nil {
		return fmt.Errorf("ricerca ddl: %w", err)
	}
	if len(recs) == 0 {
		report.Note = fmt.Sprintf("DDL %d non trovato nell'archivio della legislatura %d. Verifica legisl e numero con `ars-sicilia-pp-cli ddl cerca`.", numero, legisl)
		return emitIter(cmd, flags, report)
	}
	report.Titolo = recs[0].Title
	report.Eventi = append(report.Eventi, iterEvent{
		Fase:      "presentazione",
		Data:      recs[0].Fields["Data"],
		Sede:      "Assemblea (presentazione DDL)",
		Titolo:    recs[0].Title,
		URL:       recs[0].URL,
		ArchiveID: arc.ID,
		DocID:     recs[0].DocID,
	})

	// 2. Iter reale dal corpo del documento: il portale elenca i passaggi
	// ("Assegnato per esame Commissione QUARTA", "Approvato", "Promulgata legge
	// regionale n. …") tra i marcatori "Attuale" e "Storico". È l'unica fonte
	// affidabile: i sommari/resoconti NON citano il DDL con la stringa "ddl N".
	if doc, derr := c.GetDoc(ctx, *arc, recs[0].DocID); derr == nil {
		for _, ev := range parseIterFromBody(doc.Body) {
			ev.URL = recs[0].URL
			ev.ArchiveID = arc.ID
			ev.DocID = recs[0].DocID
			report.Eventi = append(report.Eventi, ev)
		}
	}

	// Sort events by ISO date when parseable.
	sort.SliceStable(report.Eventi, func(i, j int) bool {
		return iterDateKey(report.Eventi[i].Data) < iterDateKey(report.Eventi[j].Data)
	})
	return emitIter(cmd, flags, report)
}

func emitIter(cmd *cobra.Command, flags *rootFlags, report iterReport) error {
	out := cmd.OutOrStdout()
	if flags.asJSON || !isTerminal(out) {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	fmt.Fprintf(out, "DDL %d/%d — %s\n", report.Legisl, report.Numero, report.Titolo)
	if report.Note != "" {
		fmt.Fprintf(out, "  %s\n", report.Note)
	}
	for _, e := range report.Eventi {
		fmt.Fprintf(out, "  [%s] %s — %s\n", e.Fase, e.Data, strings.TrimSpace(e.Sede+" "+e.Titolo))
	}
	return nil
}

// reIterDate matches an Italian short date "<DD> <mese> <YYYY>" used to anchor
// each iter step in the document status block.
var reIterDate = regexp.MustCompile(`(\d{1,2})\s+([a-zàèéìòù]{3,})\s+(\d{4})`)

var itaMonths = map[string]string{
	"gen": "01", "feb": "02", "mar": "03", "apr": "04", "mag": "05", "giu": "06",
	"lug": "07", "ago": "08", "set": "09", "ott": "10", "nov": "11", "dic": "12",
}

// parseIterFromBody extracts iter events (committee assignment, aula passage,
// approval/rejection, promulgation) from the status block the portal renders at
// the top of a DDL document body. The block runs from "Attuale" up to the bill
// header "(n. …)" and contains both the current status and, after the "Storico"
// label, the full chronological history. Each step is "<date> <action> [Seduta
// n. N …]"; we cut the action at "Seduta" to drop the sitting metadata (and its
// stray digits). Returns nil when no status block is present.
func parseIterFromBody(body string) []iterEvent {
	if body == "" {
		return nil
	}
	start := strings.Index(body, "Attuale")
	if start < 0 {
		return nil
	}
	region := body[start+len("Attuale"):]
	// The bill text proper begins either with the "(n. <numero>)" header or,
	// for records lacking that header (e.g. the finanziaria and other
	// governativi that open straight into the masthead), with the fixed
	// "ASSEMBLEA REGIONALE SICILIANA" line. Cut the region at whichever comes
	// first; everything after it is document content, not iter. Without this,
	// long articolati leak into the region and dates cited inside the bill
	// text (e.g. "3 luglio 1950, n. 51") get parsed as iter events.
	cutAt := -1
	for _, marker := range []string{"(n.", "ASSEMBLEA REGIONALE SICILIANA"} {
		if i := strings.Index(region, marker); i >= 0 && (cutAt < 0 || i < cutAt) {
			cutAt = i
		}
	}
	if cutAt >= 0 {
		region = region[:cutAt]
	}
	// "Storico" is a section label between current status and history, not an event.
	region = strings.ReplaceAll(region, "Storico", " ")

	locs := reIterDate.FindAllStringIndex(region, -1)
	subs := reIterDate.FindAllStringSubmatch(region, -1)
	var events []iterEvent
	for i, loc := range locs {
		dd, mon, yyyy := subs[i][1], strings.ToLower(subs[i][2]), subs[i][3]
		actEnd := len(region)
		if i+1 < len(locs) {
			actEnd = locs[i+1][0]
		}
		action := region[loc[1]:actEnd]
		if s := strings.Index(action, "Seduta"); s >= 0 {
			action = action[:s]
		}
		action = strings.Join(strings.Fields(action), " ")
		if action == "" {
			continue
		}
		events = append(events, iterEvent{
			Fase:   classifyIterFase(action),
			Data:   fmt.Sprintf("%s %s %s", dd, mon, yyyy),
			Sede:   iterSede(action),
			Titolo: action,
		})
	}
	return events
}

type firmatario struct {
	Nome   string `json:"nome"`
	Gruppo string `json:"gruppo,omitempty"`
}

// reFirmEntry matches a "Nome Cognome (Gruppo)" firmatario entry. The name is a
// run of capitalised words; the group is the parenthesised text.
var reFirmEntry = regexp.MustCompile(`([A-ZÀ-Ù][\p{L}'.\-]*(?:\s+[A-ZÀ-Ù][\p{L}'.\-]*){0,3})\s*\(([^)]{2,90})\)`)

// firmLabelWords are non-name capitalised words that can sit right before a
// firmatario in the flattened body (section labels / iniziativa values).
var firmLabelWords = map[string]bool{
	"Parlamentare": true, "Governativa": true, "Popolare": true, "Iniziativa": true,
	"Gruppo": true, "Firmatari": true, "Argomenti": true, "Premier": true, "ARS": true,
}

// parseDdlFirmatari extracts the bill signatories from the document body. It
// handles the structured sidebar form ("Nome (Gruppo). • Nome (Gruppo).", with
// party groups) and the relazione form ("presentato dai deputati: A, B, C",
// names only). Returns nil when none is found (e.g. some governativi).
func parseDdlFirmatari(body string) []firmatario {
	if body == "" {
		return nil
	}
	// Form A: bullet-separated "Nome (Gruppo)" entries.
	if strings.Contains(body, "•") {
		var out []firmatario
		seen := map[string]bool{}
		for _, seg := range strings.Split(body, "•") {
			ms := reFirmEntry.FindAllStringSubmatch(seg, -1)
			if len(ms) == 0 {
				continue
			}
			m := ms[len(ms)-1] // firmatario sits at the end of the segment
			nome := cleanFirmatarioName(m[1])
			grp := strings.Join(strings.Fields(m[2]), " ")
			if nome == "" || seen[nome+"|"+grp] {
				continue
			}
			seen[nome+"|"+grp] = true
			out = append(out, firmatario{Nome: nome, Gruppo: grp})
		}
		if len(out) > 0 {
			return out
		}
	}
	// Form B: "presentato dai deputati: A, B, C" (no groups).
	for _, marker := range []string{"presentato dai deputati", "presentato dal deputato", "presentato dalla deputata"} {
		if i := strings.Index(strings.ToLower(body), marker); i >= 0 {
			rest := body[i+len(marker):]
			rest = strings.TrimLeft(rest, ": ")
			if e := strings.IndexAny(rest, ".\n"); e >= 0 {
				rest = rest[:e]
			}
			var out []firmatario
			for _, n := range strings.Split(rest, ",") {
				if n = strings.Join(strings.Fields(n), " "); n != "" {
					out = append(out, firmatario{Nome: n})
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

// cleanFirmatarioName strips leading section-label words and caps the name to
// its trailing capitalised tokens.
func cleanFirmatarioName(s string) string {
	toks := strings.Fields(s)
	for len(toks) > 0 && firmLabelWords[toks[0]] {
		toks = toks[1:]
	}
	if len(toks) > 4 {
		toks = toks[len(toks)-4:]
	}
	return strings.Join(toks, " ")
}

func classifyIterFase(action string) string {
	a := strings.ToLower(action)
	switch {
	case strings.Contains(a, "promulgat") || strings.Contains(a, "legge regionale") || strings.Contains(a, "l.r."):
		return "legge"
	case strings.Contains(a, "aula") || strings.Contains(a, "assemblea") || strings.Contains(a, "approvat"):
		return "aula"
	case strings.Contains(a, "commissione") || strings.Contains(a, "esame") || strings.Contains(a, "parere"):
		return "commissione"
	default:
		return "iter"
	}
}

// iterSede returns the committee name when the action references one.
func iterSede(action string) string {
	if i := strings.Index(strings.ToLower(action), "commissione"); i >= 0 {
		return strings.TrimSpace(action[i:])
	}
	return ""
}

// iterDateKey returns a sortable "YYYY-MM-DD" key for both the short-list date
// form (DD.MM.YY) and the document status form ("DD mese YYYY").
func iterDateKey(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, ".") {
		return parseICaroDate(s)
	}
	parts := strings.Fields(s)
	if len(parts) == 3 {
		if mm, ok := itaMonths[strings.ToLower(parts[1])[:min3(len(parts[1]))]]; ok {
			dd := parts[0]
			if len(dd) == 1 {
				dd = "0" + dd
			}
			return parts[2] + "-" + mm + "-" + dd
		}
	}
	return s
}

func min3(n int) int {
	if n < 3 {
		return n
	}
	return 3
}

// parseICaroDate converts DD.MM.YYYY (or DD.MM.YY) into a sortable string
// "YYYY-MM-DD"; returns the input as-is when the format isn't recognized.
func parseICaroDate(s string) string {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return s
	}
	dd, mm, yy := parts[0], parts[1], parts[2]
	if len(yy) == 2 {
		// Crude century pivot — atti precedenti al 2000 sono rari nel sito.
		if yy[0] >= '0' && yy[0] <= '4' {
			yy = "20" + yy
		} else {
			yy = "19" + yy
		}
	}
	if len(yy) != 4 || len(mm) > 2 || len(dd) > 2 {
		return s
	}
	if len(mm) == 1 {
		mm = "0" + mm
	}
	if len(dd) == 1 {
		dd = "0" + dd
	}
	return yy + "-" + mm + "-" + dd
}
