package icaroclient

import (
	"reflect"
	"testing"
)

// I casi sono quelli misurati sul portale il 2026-09-06: la punteggiatura fa
// rifiutare la query, e la riscrittura in spazio e' fedele a come l'indice
// separa le parole.
func TestValoreRipulito(t *testing.T) {
	casi := []struct {
		in      string
		out     string
		rimossi []string
		perche  string
	}{
		{"Approvato dall'Assemblea", "Approvato dall Assemblea", []string{"'"}, "lo stato scritto dal portale stesso"},
		{"D'Agostino", "D Agostino", []string{"'"}, "cognome siciliano: rifiutato oggi su --firmatario"},
		{"sanita'", "sanita", []string{"'"}, "apostrofo finale"},
		{"COVID-19", "COVID 19", []string{"-"}, "il trattino e' fra i caratteri rifiutati"},
		{"260101/261231", "260101/261231", nil, "range DATPRE costruito dalla CLI: sola struttura, intatto"},
		{"1173", "1173", nil, "numero"},
		{"(aree E idonee)", "(aree E idonee)", nil, "espressione dell'utente: le parentesi sono la via d'uscita convenzionale"},
		{"Assembl$", "Assembl$", nil, "troncamento ISIS: verificato, torna righe"},
		{"Approvato dall Assemblea", "Approvato dall Assemblea", nil, "gia' pulito, nessun avviso"},
		{"l'art. 5, comma 3", "l art 5 comma 3", []string{"'", ".", ","}, "piu' caratteri, nominati una volta ciascuno"},
	}
	for _, c := range casi {
		got, rimossi := ValoreRipulito(c.in)
		if got != c.out {
			t.Errorf("ValoreRipulito(%q) = %q, atteso %q (%s)", c.in, got, c.out, c.perche)
		}
		if !reflect.DeepEqual(rimossi, c.rimossi) {
			t.Errorf("ValoreRipulito(%q) rimossi = %v, atteso %v", c.in, rimossi, c.rimossi)
		}
	}
}

// La riscrittura deve arrivare fino all'espressione: un valore di campo con
// spazio e' adiacenza, ed e' quello che si voleva.
func TestBuildQueryRipulisceIValori(t *testing.T) {
	arc := Archive{ID: "221", Slug: "ddl", FieldMap: map[string]string{"iter": "ITERST", "legisl": "LEGISL", "data": "DATPRE"}}
	got := BuildQuery(arc, map[string]string{
		"iter":   "Approvato dall'Assemblea",
		"legisl": "18",
		"data":   "260101/261231",
	}, "")
	want := "(260101/261231.DATPRE E (Approvato dall Assemblea).ITERST E 18.LEGISL)"
	if got != want {
		t.Errorf("BuildQuery = %q, atteso %q", got, want)
	}
}

// --isis-query resta la via d'uscita: passa verbatim, apostrofo compreso.
func TestBuildQueryISISRawIntatta(t *testing.T) {
	arc := Archive{ID: "221", Slug: "ddl", FieldMap: map[string]string{"iter": "ITERST"}}
	raw := "dall'Assemblea.ITERST E 18.LEGISL"
	if got := BuildQuery(arc, map[string]string{"iter": "x"}, raw); got != raw {
		t.Errorf("BuildQuery con ISISRaw = %q, atteso %q", got, raw)
	}
}

func TestValoriRipuliti(t *testing.T) {
	rimossi, presenti := ValoriRipuliti(map[string]string{"iter": "Approvato dall'Assemblea", "legisl": "18"})
	if !presenti {
		t.Fatal("atteso presenti=true")
	}
	if len(rimossi) != 1 || !reflect.DeepEqual(rimossi["iter"], []string{"'"}) {
		t.Errorf("rimossi = %v", rimossi)
	}
	if _, presenti := ValoriRipuliti(map[string]string{"legisl": "18"}); presenti {
		t.Error("nessuna punteggiatura: atteso presenti=false")
	}
}

// I due rifiuti del portale, come li ha risposti il 2026-09-06.
func TestQueryNonCostruibile(t *testing.T) {
	sintassi := `<div class="message ko"> Impossibile creare la Query  QRY0 ()`
	soglia := `<div class="message ko"> (QR997)  QRY0 ()`
	if !QueryNonCostruibile(sintassi) {
		t.Error("pagina di sintassi non riconosciuta")
	}
	if QueryNonCostruibile(soglia) {
		t.Error("la soglia non e' un errore di sintassi")
	}
}
