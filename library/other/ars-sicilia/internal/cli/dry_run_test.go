package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
	"github.com/spf13/cobra"
)

// dryRunOut esegue un'anteprima su un comando finto e ne rilegge il JSON.
func dryRunOut(t *testing.T, fn func(*cobra.Command) error) map[string]any {
	t.Helper()
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := fn(cmd); err != nil {
		t.Fatalf("anteprima fallita: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output non JSON (%q): %v", buf.String(), err)
	}
	return got
}

func requestsOf(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	raw, ok := out["requests"].([]any)
	if !ok {
		t.Fatalf("manca `requests` in %v", out)
	}
	reqs := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			t.Fatalf("richiesta non è un oggetto: %v", r)
		}
		reqs = append(reqs, m)
	}
	return reqs
}

// Il difetto: gli archivi delle sedute sono serviti dal backend /bd/, ma
// l'anteprima li descriveva come query Icaro, annunciando un URL che il
// comando non interroga.
func TestDryRunTargetUsaIlBackendGiusto(t *testing.T) {
	casi := []struct {
		slug        string
		vuoleBE     string
		vuoleChiave string
	}{
		{"resoconti", "bd", "would_post"},
		{"sommari", "bd", "would_post"},
		{"convocazioni", "bd", "would_post"},
		{"ddl", "icaro", "would_fetch"},
		{"leggi", "icaro", "would_fetch"},
		{"pareri", "icaro", "would_fetch"},
	}
	for _, c := range casi {
		arc := icaro.BySlug(c.slug)
		if arc == nil {
			t.Fatalf("archivio %s non registrato", c.slug)
		}
		got := dryRunTarget(*arc, map[string]string{"legisl": "18"}, "")
		if got["backend"] != c.vuoleBE {
			t.Errorf("%s: backend = %v, voluto %s", c.slug, got["backend"], c.vuoleBE)
		}
		url, _ := got[c.vuoleChiave].(string)
		if url == "" {
			t.Errorf("%s: manca %s in %v", c.slug, c.vuoleChiave, got)
			continue
		}
		if c.vuoleBE == "bd" && !strings.Contains(url, "/bd/"+c.slug) {
			t.Errorf("%s: would_post = %q, atteso il path /bd/", c.slug, url)
		}
		if c.vuoleBE == "icaro" && !strings.Contains(url, "icaDB="+arc.ID) {
			t.Errorf("%s: would_fetch = %q, atteso icaDB=%s", c.slug, url, arc.ID)
		}
		if _, ha := got["isis_query"]; ha != (c.vuoleBE == "icaro") {
			t.Errorf("%s: isis_query presente=%v, backend=%s", c.slug, ha, c.vuoleBE)
		}
	}
}

// Il difetto: --dry-run era accettato e scartato, uscita vuota ed exit 0.
func TestLeggeCronologiaDryRunAnteprimaLaLegge(t *testing.T) {
	out := dryRunOut(t, func(cmd *cobra.Command) error {
		return emitLeggeCronologiaDryRun(cmd, 18, 1, 2025)
	})
	if out["dry_run"] != true {
		t.Errorf("manca dry_run: %v", out)
	}
	reqs := requestsOf(t, out)
	if len(reqs) != 1 {
		t.Fatalf("richieste = %d, voluta 1 (la legge; il ddl d'origine dipende da questa risposta)", len(reqs))
	}
	q, _ := reqs[0]["isis_query"].(string)
	for _, atteso := range []string{"18.LEGISL", "1.LEGNUM", "2025.LEGANN"} {
		if !strings.Contains(q, atteso) {
			t.Errorf("isis_query = %q, manca %s", q, atteso)
		}
	}
	nota, _ := out["note"].(string)
	if !strings.Contains(nota, "P010/P012") {
		t.Errorf("la nota deve dichiarare il passo non anteprimabile, invece: %q", nota)
	}
}

// Senza --anno l'anteprima deve dirlo: la cronologia può uscire coerente e
// riferita all'atto sbagliato.
func TestLeggeCronologiaDryRunSenzaAnnoAvverte(t *testing.T) {
	out := dryRunOut(t, func(cmd *cobra.Command) error {
		return emitLeggeCronologiaDryRun(cmd, 18, 26, 0)
	})
	nota, _ := out["note"].(string)
	if !strings.Contains(nota, "--anno") {
		t.Errorf("nota senza avviso su --anno: %q", nota)
	}
	q, _ := requestsOf(t, out)[0]["isis_query"].(string)
	if strings.Contains(q, "LEGANN") {
		t.Errorf("senza --anno la query non deve vincolare l'anno: %q", q)
	}
}

// L'anteprima del dossier vale se mostra la traduzione dell'argomento: `codcom`
// grezzo verso /bd/, ordinale a lettere verso l'ISIS. Normalizzare i parametri
// qui (come fa `*/cerca`) annuncerebbe `commissione: SESTA` anche su /bd/,
// cioè un parametro diverso da quello che parte.
func TestCommissioneDossierDryRunNonNormalizzaCodcom(t *testing.T) {
	out := dryRunOut(t, func(cmd *cobra.Command) error {
		return emitCommissioneDossierDryRun(cmd, "6", 18)
	})
	reqs := requestsOf(t, out)
	if len(reqs) != 4 {
		t.Fatalf("richieste = %d, volute 4 (convocazioni, sommari, pareri, ddl)", len(reqs))
	}
	perArchivio := map[string]map[string]any{}
	for _, r := range reqs {
		slug, _ := r["archive"].(string)
		perArchivio[slug] = r
	}
	bd, ok := perArchivio["convocazioni"]
	if !ok {
		t.Fatalf("manca la sezione convocazioni: %v", perArchivio)
	}
	params, _ := bd["params"].(map[string]any)
	if params["codcom"] != "6" {
		t.Errorf("convocazioni: params = %v, atteso codcom=6 come lo manda runCommissioneDossier", params)
	}
	pareri, ok := perArchivio["pareri"]
	if !ok {
		t.Fatalf("manca la sezione pareri: %v", perArchivio)
	}
	if q, _ := pareri["isis_query"].(string); !strings.Contains(q, "SESTA") {
		t.Errorf("pareri: isis_query = %q, atteso l'ordinale a lettere", q)
	}
}

// Il profilo interroga sei archivi come firmatario più i resoconti come testo
// libero: l'anteprima deve elencarli tutti, ed è quella differenza a spiegare
// perché lo stesso nome renda su un archivio e non sull'altro.
func TestDeputatoProfiloDryRunElencaTuttiGliArchivi(t *testing.T) {
	out := dryRunOut(t, func(cmd *cobra.Command) error {
		return emitDeputatoProfiloDryRun(cmd, "Cracolici", 18, "2024-01-01:2024-12-31")
	})
	reqs := requestsOf(t, out)
	if len(reqs) != len(profiloFirmaArchives)+1 {
		t.Fatalf("richieste = %d, volute %d", len(reqs), len(profiloFirmaArchives)+1)
	}
	for i, slug := range profiloFirmaArchives {
		if reqs[i]["archive"] != slug {
			t.Errorf("richiesta %d = %v, voluto %s", i, reqs[i]["archive"], slug)
		}
		q, _ := reqs[i]["isis_query"].(string)
		if !strings.Contains(q, "Cracolici.FIRMAT") {
			t.Errorf("%s: isis_query = %q, atteso il nome come firmatario", slug, q)
		}
		// --data deve arrivare già in formato ISIS, come a runtime.
		if !strings.Contains(q, "240101/241231.DATPRE") {
			t.Errorf("%s: isis_query = %q, --data non normalizzata come a runtime", slug, q)
		}
	}
	ultima := reqs[len(reqs)-1]
	if ultima["archive"] != "resoconti" {
		t.Fatalf("ultima richiesta = %v, voluti i resoconti", ultima["archive"])
	}
	if ultima["backend"] != "bd" {
		t.Errorf("resoconti: backend = %v, voluto bd", ultima["backend"])
	}
	params, _ := ultima["params"].(map[string]any)
	if params["testo"] != "Cracolici" {
		t.Errorf("resoconti: params = %v, atteso il nome come testo libero", params)
	}
}

// La matrice di collaudo sonda i comandi scritti a mano con argomenti
// segnaposto (`mock-value`). Prima delle anteprime il ramo dry-run usciva 0
// senza guardarli; dopo, un argomento non numerico li faceva fallire — una
// sonda che passava trasformata in errore. Il ripiego e' l'help, come sul ramo
// degli argomenti mancanti.
func TestDryRunConArgomentiSegnapostoNonFallisce(t *testing.T) {
	flags := &rootFlags{dryRun: true}
	casi := []struct {
		nome string
		cmd  *cobra.Command
		args []string
	}{
		{"legge cronologia", newNovelLeggeCronologiaCmd(flags), []string{"mock-value", "mock-value"}},
		{"ddl iter", newNovelDdlIterCmd(flags), []string{"mock-value", "mock-value"}},
	}
	for _, c := range casi {
		var buf bytes.Buffer
		c.cmd.SetOut(&buf)
		c.cmd.SetErr(&buf)
		c.cmd.SetArgs(c.args)
		if err := c.cmd.Execute(); err != nil {
			t.Errorf("%s: --dry-run con argomenti segnaposto ha fallito: %v", c.nome, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%s: uscita muta; una sonda --dry-run deve stampare qualcosa", c.nome)
		}
	}
}

// Senza --dry-run un argomento non numerico resta un errore: il ripiego vale
// solo per le sonde, non nasconde un uso sbagliato.
func TestArgomentoNonNumericoRestaErroreSenzaDryRun(t *testing.T) {
	for _, c := range []struct {
		nome string
		cmd  *cobra.Command
	}{
		{"legge cronologia", newNovelLeggeCronologiaCmd(&rootFlags{})},
		{"ddl iter", newNovelDdlIterCmd(&rootFlags{})},
	} {
		var buf bytes.Buffer
		c.cmd.SetOut(&buf)
		c.cmd.SetErr(&buf)
		c.cmd.SilenceUsage = true
		c.cmd.SilenceErrors = true
		c.cmd.SetArgs([]string{"mock-value", "mock-value"})
		if err := c.cmd.Execute(); err == nil {
			t.Errorf("%s: atteso errore su argomento non numerico senza --dry-run", c.nome)
		}
	}
}
