package cli

import (
	"strings"
	"testing"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
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
