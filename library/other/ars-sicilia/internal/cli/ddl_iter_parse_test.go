package cli

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
	"github.com/spf13/cobra"
)

// TestParseFirmatariBlock_NoLabelLeak covers the labeled "Firmatari" block:
// every "Nome (Gruppo)" entry is taken verbatim. Parsing the flattened body
// instead used to let the neighbouring "Gruppo Parlamentare" block run into
// the first name, and to drop signatories sharing a bullet segment.
func TestParseFirmatariBlock_NoLabelLeak(t *testing.T) {
	block := "Chinnici Valentina (Partito Democratico XVIII Legislatura). • Cracolici Antonino (Partito Democratico XVIII Legislatura).• Burtone Giovanni (Partito Democratico XVIII Legislatura).•"
	f := parseFirmatariBlock(block)
	if len(f) != 3 {
		t.Fatalf("want 3 firmatari, got %d: %+v", len(f), f)
	}
	if f[0].Nome != "Chinnici Valentina" {
		t.Fatalf("first name = %q, want %q (no group-label leak)", f[0].Nome, "Chinnici Valentina")
	}
	if f[1].Nome != "Cracolici Antonino" {
		t.Fatalf("cofirmatario dropped: %+v", f)
	}
}

// TestParseFirmatariBlock_SharedSegmentAndTruncatedTail covers two portal
// quirks in one block: signatories separated by nothing but a space (no
// bullet), and a trailing entry whose group parenthesis the portal cut off.
func TestParseFirmatariBlock_SharedSegmentAndTruncatedTail(t *testing.T) {
	block := "Scuvera Salvatore (Fratelli d'Italia XVIII Legislatura). • Assenza Giorgio (Fratelli d'Italia XVIII Legislatura).• Porto Alessandro (Fratelli d'Italia XVIII Legislatura) Bica Giuseppe (Fratelli d'Italia XVIII Legislatura) Galluzzo Giuseppe (Fratelli d'Italia"
	f := parseFirmatariBlock(block)
	if len(f) != 5 {
		t.Fatalf("want 5 firmatari, got %d: %+v", len(f), f)
	}
	if f[2].Nome != "Porto Alessandro" {
		t.Fatalf("signatory sharing a bullet segment dropped: %+v", f)
	}
	if f[4].Nome != "Galluzzo Giuseppe" || f[4].Gruppo != "Fratelli d'Italia" {
		t.Fatalf("truncated trailing signatory wrong: %+v", f[4])
	}
}

// TestDocFirmatari_PrefersBlock checks that the labeled block wins over the
// flattened body, which carries the same names polluted by the neighbouring
// "Gruppo Parlamentare" value.
func TestDocFirmatari_PrefersBlock(t *testing.T) {
	doc := icaro.Doc{
		Body:   "Risposta orale\n\nPartito Democratico\n\nChinnici Valentina (Partito Democratico XVIII Legislatura). • Cracolici Antonino (Partito Democratico XVIII Legislatura).•",
		Fields: map[string]string{"Firmatari": "Chinnici Valentina (Partito Democratico XVIII Legislatura). • Cracolici Antonino (Partito Democratico XVIII Legislatura).•"},
	}
	f := docFirmatari(doc)
	if len(f) != 2 || f[0].Nome != "Chinnici Valentina" {
		t.Fatalf("docFirmatari should read the block: %+v", f)
	}
}

func TestFirmatariNames(t *testing.T) {
	got := firmatariNames([]firmatario{
		{Nome: "Abbate Ignazio", Gruppo: "Democrazia Cristiana"},
		{Nome: "Pace Carmelo"},
		{Nome: ""},
	})
	if got != "Abbate Ignazio, Pace Carmelo" {
		t.Errorf("firmatariNames = %q", got)
	}
}

func TestCurrentIterState(t *testing.T) {
	// Prefers the labeled "Iter" field, cut at "Storico".
	doc := icaro.Doc{
		Fields: map[string]string{"Iter": "Attuale 08 lug 2026 Esaminato in commissione Seduta n. 270 Storico 24 set 2025 Assegnato"},
		Body:   "Attuale 01 gen 2020 Vecchio Storico (n. 1) DISEGNO",
	}
	if got := currentIterState(doc); got != "08 lug 2026 Esaminato in commissione Seduta n. 270" {
		t.Errorf("currentIterState (field) = %q", got)
	}
	// Falls back to Body when no labeled field.
	body := icaro.Doc{Body: "x Attuale 30 giu 2026 Assegnato per esame Commissione PRIMA (n. 1161) DISEGNO"}
	if got := currentIterState(body); got != "30 giu 2026 Assegnato per esame Commissione PRIMA" {
		t.Errorf("currentIterState (body) = %q", got)
	}
	// No status block → empty.
	if got := currentIterFromBody("nessun blocco di stato"); got != "" {
		t.Errorf("currentIterFromBody(no block) = %q, want empty", got)
	}
}

func TestParseDdlFirmatari_Bullet(t *testing.T) {
	body := "Parlamentare  Geraci Salvatore (Prima l'Italia - Lega Salvini premier). • Assenza Giorgio (Fratelli d'Italia XVIII Legislatura).• Pellegrino Stefano (Forza Italia all'ARS). (n. 1089/A) DISEGNO DI LEGGE"
	f := parseDdlFirmatari(body)
	if len(f) != 3 || f[0].Nome != "Geraci Salvatore" || f[0].Gruppo != "Prima l'Italia - Lega Salvini premier" {
		t.Fatalf("bullet parse wrong: %+v", f)
	}
}

func TestParseDdlFirmatari_Presentato(t *testing.T) {
	body := "Titolo. presentato dai deputati: Spada, Catanzaro, Cracolici. RELAZIONE"
	f := parseDdlFirmatari(body)
	if len(f) != 3 || f[2].Nome != "Cracolici" {
		t.Fatalf("presentato parse wrong: %+v", f)
	}
}

// TestDocIterEvents_PrefersField covers DDLs whose relazione/articolato
// quotes dates from the law they amend with no reliable end-of-status marker
// (e.g. DDL 331/2018 "Modifiche alla l.r. 9/2010", which repeats "8 aprile
// 2010" throughout and has neither "(n." nor "ASSEMBLEA REGIONALE SICILIANA").
// Text-mining Body alone used to leak those quoted dates in as fake events;
// the labeled "Iter" field has none of that text to leak from.
func TestDocIterEvents_PrefersField(t *testing.T) {
	doc := icaro.Doc{
		Body: "x Attuale 25 set 2018 Annunzio assegnazione Seduta n. 64 AULA Storico 07 ago 2018 Annunziato Seduta n. 60 AULA 24 set 2018 Assegnato per esame Commissione PRIMA " +
			"Onorevoli colleghi, il presente disegno di legge modifica la legge regionale 8 aprile 2010, n. 9. " +
			"Art. 1. Il comma 1 dell'art. 5 della l.r. 8 aprile 2010, n. 9 è così sostituito: ...",
		Fields: map[string]string{
			"Iter": "Attuale 25 set 2018 Annunzio assegnazione Seduta n. 64 AULA Storico 07 ago 2018 Annunziato Seduta n. 60 AULA 24 set 2018 Assegnato per esame Commissione PRIMA",
		},
	}
	ev := docIterEvents(doc)
	if len(ev) != 3 {
		t.Fatalf("want 3 events from the clean Iter field, got %d: %+v", len(ev), ev)
	}
	for _, e := range ev {
		if e.Data == "8 aprile 2010" || strings.Contains(e.Titolo, "così sostituito") {
			t.Fatalf("bill-text date leaked into iter events despite a labeled Iter field: %+v", ev)
		}
	}
}

// TestParseIterFromBody_CleansLrAnnotation covers the portal's raw
// "Lr <giorno> <mese> alr <anno> nlr <numero> Titolo : ..." law-registration
// event (real example from DDL 4991 -> L.R. 27/2024's Iter field, including
// the stray repeated quote characters the portal sometimes renders): it
// should collapse to a short "Promulgata legge regionale n. <numero>/<anno>"
// event classified as fase "legge", not surface the garbled title repeat.
func TestParseIterFromBody_CleansLrAnnotation(t *testing.T) {
	body := `x Attuale 20 nov 2024 Concluso Storico 18 nov 2024 Inviato Presidenza della Regione 18 nov 2024 Lr 18 novembre alr 2024 nlr 27 Titolo : * Disposizioni in materia di urbanistica"""""""""""""" ed edilizia. Modifiche di norme.20 nov 2024 Pubblicazione Gurs n. 51 del 20 novembre 2024 (n. 4991) DISEGNO`
	ev := parseIterFromBody(body)
	var found *iterEvent
	for i := range ev {
		if ev[i].Titolo == "Promulgata legge regionale n. 27/2024" {
			found = &ev[i]
		}
		if strings.Contains(ev[i].Titolo, "Lr ") || strings.Contains(ev[i].Titolo, `""""`) {
			t.Fatalf("raw Lr annotation leaked into iter events: %+v", ev)
		}
	}
	if found == nil {
		t.Fatalf("cleaned law-registration event not found: %+v", ev)
	}
	if found.Fase != "legge" {
		t.Fatalf("fase = %q, want \"legge\": %+v", found.Fase, found)
	}
}

func TestParseIterFromBody_FullHistory(t *testing.T) {
	body := "x Attuale 11 mar 2026 Respinto dall' Aula Seduta n. 236 AULA Storico 03 mar 2026 Assegnato per esame Commissione PRIMA 10 mar 2026 Esaminato in commissione Seduta n. 252 0100 Commissione (n. 1089/A) DISEGNO"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("want 3 events, got %d: %+v", len(ev), ev)
	}
	if ev[0].Titolo != "Respinto dall' Aula" || ev[1].Fase != "commissione" {
		t.Fatalf("iter parse wrong: %+v", ev)
	}
}

// TestParseIterFromBody_NoNumeroHeader covers finanziaria-style bills that
// have no "(n. ...)" header and open straight into the bill text: without a
// second cut marker, dates cited inside the articolato (here "3 luglio
// 1950") used to leak in as spurious iter events.
func TestParseIterFromBody_NoNumeroHeader(t *testing.T) {
	body := "x Attuale 09 gen 2026 Concluso Storico 06 nov 2025 Assegnato per esame Commissione SECONDA 09 gen 2026 Pubblicazione Gurs\n\nASSEMBLEA REGIONALE SICILIANA DISEGNO DI LEGGE N. 1030 LEGGE APPROVATA IL 21 DICEMBRE 2025 Art. 1. richiama la legge regionale 3 luglio 1950, n. 51"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("want 3 events, got %d: %+v", len(ev), ev)
	}
	if ev[2].Titolo != "Pubblicazione Gurs" {
		t.Fatalf("iter parse wrong: %+v", ev)
	}
	for _, e := range ev {
		if e.Titolo == "richiama la legge regionale" || e.Data == "3 luglio 1950" {
			t.Fatalf("bill-text date leaked into iter events: %+v", ev)
		}
	}
}

// Il numero di seduta veniva tagliato via insieme al resto della riga: è il
// solo posto in cui il portale dichiara in quale seduta un ddl è stato votato,
// e senza di esso la data dell'evento si confonde con la data in cui la
// notizia è stata scritta — che è quasi sempre il giorno dopo.
func TestParseIterSeduta(t *testing.T) {
	// Casi reali: "Seduta" maiuscolo (leg. XVIII) e minuscolo (leg. XVII, ddl
	// 290). Il vecchio taglio a case fisso lasciava il secondo dentro al titolo.
	body := "x Attuale 11 mar 2026 Respinto dall' Aula Seduta n. 236 AULA " +
		"Storico 07 mag 2019 Esaminato in Aula seduta n. 114 AULA " +
		"24 set 2018 Assegnato per esame Commissione PRIMA"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("eventi = %d, want 3: %+v", len(ev), ev)
	}
	if ev[0].Seduta != 236 {
		t.Errorf("seduta maiuscola: got %d, want 236", ev[0].Seduta)
	}
	if ev[1].Seduta != 114 {
		t.Errorf("seduta minuscola: got %d, want 114 (il taglio era case-sensitive)", ev[1].Seduta)
	}
	if strings.Contains(ev[1].Titolo, "114") || strings.Contains(ev[1].Titolo, "seduta") {
		t.Errorf("titolo %q: i metadati di seduta vanno nel campo, non nel testo", ev[1].Titolo)
	}
	if ev[2].Seduta != 0 {
		t.Errorf("evento senza seduta dichiarata: got %d, want 0", ev[2].Seduta)
	}
}

// Il numero di seduta È l'id della scheda del resoconto: verificato su leg.
// XVII (114 → 07/05/2019, 150 → 06/11/2019) e XVIII (267 → 28/07/2026). Una
// seduta inesistente risponde 404, quindi un URL costruito o risolve o fallisce
// in modo visibile — mai una pagina vuota spacciata per buona. È l'URL che
// finisce nel campo `url` degli eventi d'aula, al posto della scheda del ddl.
func TestResocontoSchedaURL(t *testing.T) {
	if got := resocontoSchedaURL(17, 150); !strings.HasSuffix(got, "/bd/resoconti/scheda/17/150") {
		t.Errorf("url = %q, want .../bd/resoconti/scheda/17/150", got)
	}
	// Senza numero di seduta non si inventa un link: meglio nessun URL che uno
	// che porta altrove.
	if got := resocontoSchedaURL(17, 0); got != "" {
		t.Errorf("seduta ignota: url = %q, want stringa vuota", got)
	}
	if got := resocontoSchedaURL(0, 150); got != "" {
		t.Errorf("legislatura ignota: url = %q, want stringa vuota", got)
	}
}

// L'Aula tiene una seduta per data: due numeri diversi sulla stessa data
// vogliono dire che almeno uno è sbagliato, e la fonte non dice quale. È il
// caso reale del ddl 199 della XVII, dove la votazione finale del 19 feb 2020
// è data in «Seduta n. 179» mentre la 179 è del 26 febbraio.
func TestSedutePerDataIncoerenti(t *testing.T) {
	evs := []iterEvent{
		{Data: "04 feb 2020", Seduta: 173, sedutaAula: true},
		{Data: "19 feb 2020", Seduta: 179, sedutaAula: true},
		{Data: "19 feb 2020", Seduta: 178, sedutaAula: true},
	}
	got := sedutePerDataIncoerenti(evs)
	if !got["19 feb 2020"] {
		t.Error("due sedute d'Aula sulla stessa data: la data va segnalata")
	}
	if got["04 feb 2020"] {
		t.Error("data con una sola seduta: nessuna incoerenza da segnalare")
	}
	// Lo stesso numero ripetuto sulla stessa data non è un conflitto: due
	// passaggi dell'iter nella medesima seduta sono la norma.
	ripetuta := []iterEvent{
		{Data: "22 lug 2026", Seduta: 266, sedutaAula: true},
		{Data: "22 lug 2026", Seduta: 266, sedutaAula: true},
	}
	if len(sedutePerDataIncoerenti(ripetuta)) != 0 {
		t.Error("stessa seduta due volte: nessuna incoerenza")
	}
	// Le sedute di commissione hanno una numerazione propria e non entrano nel
	// confronto: una commissione n. 25 e un'Aula n. 178 nello stesso giorno
	// sono due cose diverse, non un conflitto.
	miste := []iterEvent{
		{Data: "19 feb 2020", Seduta: 178, sedutaAula: true},
		{Data: "19 feb 2020", Seduta: 25},
	}
	if len(sedutePerDataIncoerenti(miste)) != 0 {
		t.Error("seduta di commissione: numerazione indipendente, nessun conflitto")
	}
}

// Le due numerazioni — sedute d'Aula e sedute di commissione — sono
// indipendenti, e solo il marcatore che segue il numero dice a quale serie
// appartiene. Sbagliarlo produce un link che risolve e mostra il documento
// sbagliato, che è peggio di un link rotto.
func TestSedutaDaAzione(t *testing.T) {
	cases := []struct {
		in       string
		want     int
		wantAula bool
	}{
		{"Esaminato in Aula Seduta n. 267 AULA", 267, true},
		{"Esaminato in Aula seduta n.114 AULA", 114, true},
		{"Annunziato Seduta n. 52 AULA", 52, true},
		// Fase "aula" per il testo, ma la seduta citata è di commissione:
		// è il caso che generava il link sbagliato.
		{"Esitato per Aula (epa) Seduta n. 260 0400 Commissione QUARTA", 260, false},
		{"Esaminato in commissione Seduta n. 35 0400 Commissione QUARTA", 35, false},
		// Il portale emette davvero questa riga, virgolette comprese.
		{`Abbinamento con ddl 49 Seduta"""""""""""""" n. 35 0400 Commissione QUARTA`, 35, false},
		{"Assegnato per esame Commissione PRIMA", 0, false},
	}
	for _, c := range cases {
		got, gotAula := sedutaDaAzione(c.in)
		if got != c.want || gotAula != c.wantAula {
			t.Errorf("sedutaDaAzione(%q) = (%d, %v), want (%d, %v)", c.in, got, gotAula, c.want, c.wantAula)
		}
	}
}

// Il nome della commissione la fonte lo scrive nel suffisso di seduta, che
// indiceSeduta taglia via: senza leggerlo prima del taglio la sede resta il
// ripiego del verbo — generica, col placeholder della fonte, vuota o col solo
// codice interno.
func TestSedeDaSuffissoSeduta(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Esaminato in commissione Seduta n. 184 0400 Commissione QUARTA", "Commissione QUARTA"},
		{"Parere espresso Commissione * Seduta n. 68 0500 Commissione QUINTA", "Commissione QUINTA"},
		{"Esitato per Aula (epa) Seduta n. 185 0100 Commissione PRIMA", "Commissione PRIMA"},
		// La forma canonica vince sul nome d'uso: «Bilancio» resta in titolo.
		{"Parere Commissione Bilancio Seduta n. 153 0200 Commissione SECONDA", "Commissione SECONDA"},
		{`Abbinamento con ddl 49 Seduta"""""""""""""" n. 35 0400 Commissione QUARTA`, "Commissione QUARTA"},
		// Seduta d'Aula: nessuna commissione da leggere.
		{"Esaminato in Aula Seduta n. 267 AULA", ""},
		// Qui il suffisso porta il marcatore AULA e la commissione sta solo nel
		// verbo, come codice grezzo: se ne occupa risolviCodiceCommissione.
		{"Rinviato Commissione 0400 Seduta n. 255 AULA", ""},
		{"Assegnato per esame Commissione PRIMA", ""},
		// Le commissioni speciali hanno nomi veri, non ordinali: una cattura di
		// una sola parola li taglierebbe a metà (leg. XVII, ddl 66).
		{"Esaminato in commissione Seduta n. 3 1200 Commissione riforma statuto", "Commissione riforma statuto"},
		// Il suffisso esce come lo scrive la fonte, trattino di a-capo compreso:
		// a ricomporlo è iterSedeRisolta, sul valore che finisce nel campo `sede`.
		{"Esitato per Aula (epa) Seduta n. 29 1200 Commissione sp- eciale per lo Statuto della Regione", "Commissione sp- eciale per lo Statuto della Regione"},
		// La parentesi annota l'evento, non la commissione: la sede finisce lì.
		{"Esaminato in commissione Seduta n. 12 0100 Commissione PRIMA (Articolo 3 stralciato)", "Commissione PRIMA"},
		// Testo reale del ddl 779 dal 25.03.2026: il portale scrive il codice e
		// omette l'ordinale. Senza risolverlo la sede resta «Commissione» nuda,
		// indistinguibile fra le sei — ed è la forma delle righe più recenti,
		// cioè proprio quelle su cui si parte da una notizia.
		{"Esaminato in commissione Seduta n. 255 0100 Commissione", "Commissione PRIMA"},
		{"Esaminato in commissione Seduta n. 262 0100 Commissione", "Commissione PRIMA"},
		// Il 1200 sono le speciali, che un ordinale non ce l'hanno: resta il
		// codice grezzo, come già faceva risolviCodiceCommissione. Meglio del
		// generico, e non inventa una commissione che non esiste.
		{"Esaminato in commissione Seduta n. 3 1200 Commissione", "Commissione 1200"},
		// Codice assente e nome nudo: non c'è niente da risolvere, e inventare
		// una commissione sarebbe peggio del generico.
		{"Esaminato in commissione Seduta n. 255 Commissione", "Commissione"},
	}
	for _, c := range cases {
		if got := sedeDaSuffissoSeduta(c.in); got != c.want {
			t.Errorf("sedeDaSuffissoSeduta(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRisolviCodiceCommissione(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Commissione 0400", "Commissione QUARTA"},
		{"Commissione 0100", "Commissione PRIMA"},
		{"Commissione 0600", "Commissione SESTA"},
		// Codici fuori dalle sei commissioni o non decimali: si lascia com'è,
		// meglio il codice grezzo di un ordinale inventato.
		{"Commissione 0700", "Commissione 0700"},
		{"Commissione 0450", "Commissione 0450"},
		{"Commissione QUARTA", "Commissione QUARTA"},
		{"Commissione *", "Commissione *"},
		{"", ""},
	}
	for _, c := range cases {
		if got := risolviCodiceCommissione(c.in); got != c.want {
			t.Errorf("risolviCodiceCommissione(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Le run di virgolette del portale non sono contenuto: cadono sull'azione, non
// sulla regione — una di esse spezza la data dentro il titolo di una legge
// citata, e toglierla prima creerebbe un evento inesistente.
func TestParseIterFromBody_RipuliceRunDiVirgolette(t *testing.T) {
	body := `Attuale 24 lug 2019 Abbinamento con ddl 5; ddl"""""""""""""" 229; - VEDI ddl 587 Seduta"""""""""""""" n. 123 AULA ` +
		`31 lug 2018 Esaminato in commissione Seduta n. 3 1200 Commissione riforma"""""""""""""" statuto (n. 66)`
	ev := parseIterFromBody(body)
	if len(ev) != 2 {
		t.Fatalf("eventi = %d, want 2: %+v", len(ev), ev)
	}
	if strings.Contains(ev[0].Titolo, `"`) {
		t.Errorf("titolo con virgolette residue: %q", ev[0].Titolo)
	}
	if ev[0].Seduta != 123 {
		t.Errorf("seduta = %d, want 123 (la run stava dentro «Seduta»)", ev[0].Seduta)
	}
	if want := "Commissione riforma statuto"; ev[1].Sede != want {
		t.Errorf("sede = %q, want %q", ev[1].Sede, want)
	}
}

// La data dentro il titolo di una legge citata non e' un evento dell'iter: la
// run di virgolette che la spezza va tolta dopo il taglio in eventi, non prima.
func TestParseIterFromBody_DataNelTitoloCitatoNonEUnEvento(t *testing.T) {
	body := `Attuale 12 mar 2025 Lr 12 marzo alr 2025 nlr 8 Titolo : * Modifiche alla legge regionale 31"""""""""""""" gennaio 2024, n. 3` +
		`21 mar 2025 Pubblicazione Gurs n. 14 del 21 marzo 2025 (n. 738)`
	ev := parseIterFromBody(body)
	for _, e := range ev {
		if strings.Contains(e.Data, "2024") {
			t.Errorf("evento fantasma datato 2024: %+v", e)
		}
	}
	if len(ev) == 0 || ev[0].Titolo != "Promulgata legge regionale n. 8/2025" {
		t.Errorf("primo evento = %+v, want la promulgazione classificata", ev)
	}
}

// La data che segue un «del» pendente chiude la frase dell'evento: tagliarla
// perdeva la data di pubblicazione in Gurs, l'unica cosa che quell'evento
// aggiunge alla promulgazione già registrata il giorno prima.
func TestParseIterFromBody_DataGursRiattaccata(t *testing.T) {
	body := "Attuale 19 feb 2026 Inviato Presidenza della Regione 24 feb 2026 Concluso " +
		"24 feb 2026 Pubblicazione Gurs n. 10 del 24 febbraio 2026 (n. 73813)"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("eventi = %d, want 3: %+v", len(ev), ev)
	}
	if want := "Pubblicazione Gurs n. 10 del 24 febbraio 2026"; ev[2].Titolo != want {
		t.Errorf("titolo = %q, want %q", ev[2].Titolo, want)
	}
}

// Una data che segue una parola qualsiasi apre l'evento dopo, e non va
// riattaccata: è il caso normale, ed è quello che regge tutta la cronologia.
func TestParseIterFromBody_DataNonRiattaccataSenzaPreposizione(t *testing.T) {
	body := "Attuale 06 ago 2020 Approvato dall'Assemblea Seduta n. 213 AULA " +
		"12 ago 2020 Inviato Presidenza della Regione (n. 587)"
	ev := parseIterFromBody(body)
	if len(ev) != 2 {
		t.Fatalf("eventi = %d, want 2: %+v", len(ev), ev)
	}
	if ev[1].Data != "12 ago 2020" {
		t.Errorf("secondo evento datato %q, want «12 ago 2020»", ev[1].Data)
	}
}

// Il titolo della legge, dentro l'annotazione «Lr … Titolo : …», cita le norme
// modificate con le loro date: non sono eventi dell'iter, e prima finivano in
// cima alla cronologia datate anni prima dell'atto.
func TestParseIterFromBody_DataCitataNelTitoloNonEUnEvento(t *testing.T) {
	body := "Attuale 26 mar 2025 Inviato Presidenza della Regione " +
		"01 apr 2025 Lr 01 aprile alr 2025 nlr 10 Titolo : * Riconoscimento della legittimità dei debiti " +
		"fuori bilancio ai sensi dell'articolo 73, comma 1, lettera e) del decreto legislativo 23 giugno 2011, " +
		"n. 118 e successive modificazioni. D.F.B. 2023. Mesi di novembre e dicembre" +
		"11 apr 2025 Pubblicazione Gurs n. 17 del 11 aprile 2025 (n. 700)"
	ev := parseIterFromBody(body)
	for _, e := range ev {
		if strings.Contains(e.Data, "2011") {
			t.Errorf("evento inesistente dalla norma citata: %+v", e)
		}
	}
	if len(ev) != 3 {
		t.Fatalf("eventi = %d, want 3: %+v", len(ev), ev)
	}
	// La finestra si chiude sulla prima data che non torna indietro: la
	// pubblicazione in Gurs resta.
	if want := "Pubblicazione Gurs n. 17 del 11 aprile 2025"; ev[2].Titolo != want {
		t.Errorf("ultimo evento = %q, want %q", ev[2].Titolo, want)
	}
}

// Gli eventi anteriori alla data di scheda di uno stralcio sono storia vera
// dell'atto e non vanno toccati: nessuna annotazione Lr li precede.
func TestParseIterFromBody_StoriaAnterioreDelloStralcioResta(t *testing.T) {
	body := "Attuale 04 ago 2026 Inviato Presidenza della Regione Storico " +
		"13 gen 2026 Assegnato per esame Commissione QUARTA " +
		"27 gen 2026 Esaminato in commissione Seduta n. 184 0400 Commissione QUARTA (n. 6030)"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("eventi = %d, want 3: %+v", len(ev), ev)
	}
	if ev[1].Data != "13 gen 2026" {
		t.Errorf("l'assegnazione del 13 gen è sparita: %+v", ev)
	}
}

// Il verso inverso non e' una contraddizione: la stessa seduta d'Aula su piu'
// date e' la seduta aperta, sospesa e ripresa che l'Assemblea tiene sulle
// manovre. E' il caso reale del ddl 733 della XVII (poi L.R. 9/2020), dove la
// seduta 187 compare il 28 aprile e di nuovo il 2 maggio 2020: la 187 esiste,
// l'archivio la indicizza al 28 aprile, e il voto del 2 maggio sta li' dentro.
func TestSedutePluriGiorno(t *testing.T) {
	// L'elenco e' quello vero dell'iter del ddl 733, ridotto alle righe che
	// contano: le due d'Aula sulla stessa seduta, la 187 di COMMISSIONE del 20
	// aprile (numerazione indipendente, non deve entrare nel confronto) e la
	// 198 «Esitato per Aula», evento di fase aula che pero' cita una seduta di
	// commissione.
	evs := []iterEvent{
		{Data: "20 apr 2020", Seduta: 187},
		{Data: "26 apr 2020", Seduta: 198},
		{Data: "27 apr 2020", Seduta: 186, sedutaAula: true},
		{Data: "28 apr 2020", Seduta: 187, sedutaAula: true},
		{Data: "02 mag 2020", Seduta: 187, sedutaAula: true},
	}
	got := sedutePluriGiorno(evs)
	if got[187] != "28 apr 2020" {
		t.Errorf("seduta 187 pluri-giorno, apertura = %q, want %q", got[187], "28 apr 2020")
	}
	if _, ok := got[186]; ok {
		t.Errorf("segnalate sedute di un giorno solo: %v", got)
	}
	if _, ok := got[198]; ok {
		t.Errorf("seduta di commissione nel confronto: %v", got)
	}
	if len(got) != 1 {
		t.Errorf("sedute pluri-giorno = %v, want solo la 187", got)
	}

	// La marcatura tocca solo i due eventi d'Aula sulla 187, e nessuno porta
	// `anomalia`: il numero e' uno solo, il resoconto esiste, il link si puo'
	// costruire. `data_apertura` esce solo sulla ripresa, dove la data
	// dell'evento non e' quella con cui l'archivio indicizza la seduta.
	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)
	incoerenti, avviso := marcaEventiIncoerenti(cmd, context.Background(), nil, 17, evs)
	if avviso == "" {
		t.Fatal("seduta pluri-giorno: l'avviso non deve essere vuoto")
	}
	if len(incoerenti) != 0 {
		t.Errorf("date incoerenti = %v, want nessuna: la contraddizione qui non c'e'", incoerenti)
	}
	marcati := 0
	for i, ev := range evs {
		if ev.Anomalia {
			t.Errorf("evento %d marcato anomalia: la seduta pluri-giorno non e' un'anomalia: %+v", i, ev)
		}
		if ev.SedutaPluriGiorno {
			marcati++
			if !ev.sedutaAula || ev.Seduta != 187 {
				t.Errorf("evento %d marcato a torto: %+v", i, ev)
			}
		}
	}
	if marcati != 2 {
		t.Errorf("eventi pluri-giorno = %d, want 2 (le due righe d'Aula sulla seduta 187)", marcati)
	}
	if evs[3].DataApertura != "" {
		t.Errorf("evento del giorno di apertura: data_apertura = %q, want vuoto", evs[3].DataApertura)
	}
	if evs[4].DataApertura != "28 apr 2020" {
		t.Errorf("evento di ripresa: data_apertura = %q, want %q", evs[4].DataApertura, "28 apr 2020")
	}
	// Senza client la verifica sull'archivio non parte: il marcatore resta, ma
	// l'avviso deve dire che la data d'apertura e' quella dichiarata dalla
	// fonte e non una data d'archivio.
	if !strings.Contains(avviso, "non verificata") {
		t.Errorf("verifica non eseguita: l'avviso deve dirlo, invece e' %q", avviso)
	}

	// La stessa data scritta nelle due forme della fonte e' la stessa data: una
	// seduta di un giorno solo non va marcata solo perche' il portale alterna i
	// formati.
	forme := []iterEvent{
		{Data: "28 apr 2020", Seduta: 187, sedutaAula: true},
		{Data: "28.04.20", Seduta: 187, sedutaAula: true},
	}
	if len(sedutePluriGiorno(forme)) != 0 {
		t.Error("stessa data in due formati: un giorno solo")
	}
}

// La contraddizione vera resta tale: quando la stessa data porta due numeri di
// seduta diversi, l'anomalia c'e' e il link non si costruisce. E' la guardia
// che la riclassificazione del pluri-giorno non deve avere spento.
func TestAnomaliaRestaSuStessaDataSeduteDiverse(t *testing.T) {
	evs := []iterEvent{
		{Data: "04 feb 2020", Seduta: 173, sedutaAula: true},
		{Data: "19 feb 2020", Seduta: 179, sedutaAula: true},
		{Data: "19 feb 2020", Seduta: 178, sedutaAula: true},
	}
	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)
	incoerenti, avviso := marcaEventiIncoerenti(cmd, context.Background(), nil, 17, evs)
	if !incoerenti["19 feb 2020"] {
		t.Error("la data con due sedute deve restare fra le incoerenti: e' quella che sopprime il link")
	}
	if avviso == "" {
		t.Fatal("iter incoerente: l'avviso non deve essere vuoto")
	}
	if evs[0].Anomalia {
		t.Errorf("evento sano marcato: %+v", evs[0])
	}
	for _, i := range []int{1, 2} {
		if !evs[i].Anomalia {
			t.Errorf("evento %d non marcato: %+v", i, evs[i])
		}
		if evs[i].SedutaPluriGiorno {
			t.Errorf("evento %d marcato pluri-giorno: e' una contraddizione, non una ripresa: %+v", i, evs[i])
		}
	}
}

// Una cronologia sana non deve produrre né marcature né avviso: è la condizione
// che rende il marcatore un segnale e non rumore di fondo.
func TestMarcaEventiIncoerentiIterSano(t *testing.T) {
	evs := []iterEvent{
		{Data: "07 feb 2024", Seduta: 95, sedutaAula: true},
		{Data: "20 mar 2024", Seduta: 101, sedutaAula: true},
		{Data: "20 mar 2024", Seduta: 101, sedutaAula: true},
		{Data: "21 mar 2024", Seduta: 44},
	}
	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)
	incoerenti, avviso := marcaEventiIncoerenti(cmd, context.Background(), nil, 18, evs)
	if avviso != "" {
		t.Errorf("iter coerente: avviso = %q, want vuoto", avviso)
	}
	if len(incoerenti) != 0 {
		t.Errorf("iter coerente: date incoerenti = %v", incoerenti)
	}
	for i, ev := range evs {
		if ev.Anomalia {
			t.Errorf("evento %d marcato in un iter coerente: %+v", i, ev)
		}
	}
}

// La fonte spezza il nome della commissione con un trattino e un a-capo
// (`Commissione</a> T-<br> ERZA` nell'HTML della scheda 221 del ddl 5030), e la
// sede è un campo a vocabolario chiuso: propagata così com'è, lo stesso iter
// dichiara due sedi dove ce n'è una.
func TestIterSedeRisolta_RicomponeTrattinoACapo(t *testing.T) {
	cases := []struct{ suffisso, action, want string }{
		{"Commissione T- ERZA", "", "Commissione TERZA"},
		{"Commissione sp- eciale per lo Statuto della Regione", "", "Commissione speciale per lo Statuto della Regione"},
		// Le righe sane dello stesso iter non si muovono.
		{"Commissione TERZA", "", "Commissione TERZA"},
		{"Commissione Bilancio", "", "Commissione Bilancio"},
		// Il codice grezzo continua a risolversi quando il suffisso manca.
		{"", "Rinviato Commissione 0400", "Commissione QUARTA"},
		{"", "Assegnato per esame Commissione T- ERZA", "Commissione TERZA"},
	}
	for _, c := range cases {
		if got := iterSedeRisolta(c.suffisso, c.action); got != c.want {
			t.Errorf("iterSedeRisolta(%q, %q) = %q, want %q", c.suffisso, c.action, got, c.want)
		}
	}
}

// Il trattino con lo spazio da entrambi i lati separa due parole invece di
// spezzarne una: non si tocca.
func TestRicomponiTrattinoACapo_LasciaISeparatori(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Commissione T- ERZA", "Commissione TERZA"},
		{"Commissione PRIMA - SECONDA", "Commissione PRIMA - SECONDA"},
		{"Commissione anti-mafia", "Commissione anti-mafia"},
		{"", ""},
	}
	for _, c := range cases {
		if got := ricomponiTrattinoACapo(c.in); got != c.want {
			t.Errorf("ricomponiTrattinoACapo(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Il verbo nomina la destinazione, la seduta dichiara il luogo: dove le due si
// contraddicono vince la seduta, in tutte e due le direzioni. «Esitato per
// Aula» su una seduta di commissione e' un evento di commissione; «Rinviato
// Commissione» su una seduta d'Aula e' un evento d'Aula.
func TestFaseIterConSeduta(t *testing.T) {
	cases := []struct {
		action, sedeSuffisso string
		sedutaAula           bool
		want                 string
	}{
		{"Esitato per Aula (epa)", "Commissione TERZA", false, "commissione"},
		{"Esitato per Aula", "Commissione PRIMA", false, "commissione"},
		// Le righe d'Aula portano il marcatore AULA al posto della commissione,
		// quindi il suffisso è vuoto e la fase resta quella del verbo.
		{"Esaminato in Aula", "", true, "aula"},
		{"Approvato dall'Assemblea", "", true, "aula"},
		// Il verso speculare: il rinvio in commissione deciso in Aula. E' la
		// riga del 10 giu 2026 sul ddl 18/6030, seduta 255 d'Aula.
		{"Rinviato Commissione 0400", "", true, "aula"},
		// Senza il marcatore d'Aula la stessa riga resta di commissione: la
		// promozione nasce dalla seduta dichiarata, non dal verbo.
		{"Rinviato Commissione 0400", "", false, "commissione"},
		// Le altre fasi non si toccano, nemmeno con una commissione dichiarata.
		{"Esaminato in commissione", "Commissione SECONDA", false, "commissione"},
		{"Parere espresso Commissione *", "Commissione QUINTA", false, "commissione"},
		{"Abbinamento con ddl 49", "Commissione QUARTA", false, "iter"},
		{"Promulgata legge regionale n. 4/2025", "", true, "legge"},
	}
	for _, c := range cases {
		if got := faseIterConSeduta(c.action, c.sedeSuffisso, c.sedutaAula); got != c.want {
			t.Errorf("faseIterConSeduta(%q, %q, %v) = %q, want %q", c.action, c.sedeSuffisso, c.sedutaAula, got, c.want)
		}
	}
}

// La regola con cui si legge la risposta dell'archivio, provata senza rete. E'
// il cuore del rilievo di review: un numero riusato per errore non deve
// ricevere il marcatore di seduta pluri-giorno.
func TestEsitoVerificaSeduta(t *testing.T) {
	cases := []struct {
		nome, apertura, dataArchivio string
		err                          error
		want                         esitoSeduta
	}{
		// La 187 della L.R. 9/2020: l'archivio la apre il giorno che l'iter
		// dichiara per primo.
		{"apertura combaciante", "28 apr 2020", "28.04.20", nil, sedutaConfermata},
		// Stessa data, forma diversa: e' la stessa data.
		{"forme diverse della stessa data", "28 apr 2020", "28/04/2020", nil, sedutaConfermata},
		// Il caso che la verifica esiste per prendere: numero che nell'archivio
		// non c'e'. Non e' una risposta mancante, e' una risposta negativa.
		{"seduta inesistente", "28 apr 2020", "", nil, sedutaSmentita},
		// Esiste ma si apre altrove: la coppia non torna.
		{"apertura diversa", "28 apr 2020", "12.05.20", nil, sedutaSmentita},
		// L'archivio non ha risposto: non si conclude nulla, e non si torna ad
		// anomalia — un marcatore che cambia con la rete e' peggio.
		{"archivio irraggiungibile", "28 apr 2020", "", errors.New("dial tcp: timeout"), sedutaNonVerificata},
	}
	for _, c := range cases {
		if got := esitoVerificaSeduta(c.apertura, c.dataArchivio, c.err); got != c.want {
			t.Errorf("%s: esitoVerificaSeduta(%q, %q, %v) = %v, want %v", c.nome, c.apertura, c.dataArchivio, c.err, got, c.want)
		}
	}
}
