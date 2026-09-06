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
		param   string
		in      string
		out     string
		rimossi []string
		perche  string
	}{
		{"iter", "Approvato dall'Assemblea", "Approvato dall Assemblea", []string{"'"}, "lo stato scritto dal portale stesso"},
		{"firmatario", "D'Agostino", "D Agostino", []string{"'"}, "cognome siciliano: rifiutato oggi su --firmatario"},
		{"materia", "sanita'", "sanita", []string{"'"}, "apostrofo finale"},
		{"testo", "COVID-19", "COVID 19", []string{"-"}, "il trattino e' fra i caratteri rifiutati"},
		{"data", "260101/261231", "260101/261231", nil, "range DATPRE costruito dalla CLI: sola struttura, intatto"},
		{"data", "2026-07-01:2026-07-31", "2026-07-01:2026-07-31", nil, "data ISO non normalizzata: intatta, cosi' il portale la rifiuta invece di rispondere a una domanda diversa"},
		{"isbn", "978-88-7524-166-7", "978 88 7524 166 7", []string{"-"}, "un ISBN e' fatto di cifre e trattini ma non e' una data: va ripulito, o il portale lo rifiuta (QR999)"},
		{"numero", "1173", "1173", nil, "numero"},
		{"testo", "(aree E idonee)", "(aree E idonee)", nil, "espressione dell'utente: le parentesi sono la via d'uscita convenzionale"},
		{"iter", "Assembl$", "Assembl$", nil, "troncamento ISIS: verificato, torna righe"},
		{"iter", "Approvato dall Assemblea", "Approvato dall Assemblea", nil, "gia' pulito, nessun avviso"},
		{"testo", "l'art. 5, comma 3", "l art 5 comma 3", []string{"'", ".", ","}, "piu' caratteri, nominati una volta ciascuno"},
	}
	for _, c := range casi {
		got, rimossi := ValoreRipulito(c.param, c.in)
		if got != c.out {
			t.Errorf("ValoreRipulito(%q, %q) = %q, atteso %q (%s)", c.param, c.in, got, c.out, c.perche)
		}
		if !reflect.DeepEqual(rimossi, c.rimossi) {
			t.Errorf("ValoreRipulito(%q, %q) rimossi = %v, atteso %v", c.param, c.in, rimossi, c.rimossi)
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

// Le forme esentate si nominano una per una. La prima versione del guardiano
// esentava qualunque valore di cifre e separatori, e ci passava dentro un
// ISBN-13.
func TestEsenzioneLegataAlParametroNonAllaForma(t *testing.T) {
	// Sui due parametri che portano una data costruita dalla CLI il valore
	// passa intatto: li' barra e trattino sono sintassi.
	for _, v := range []string{"18", "260101/261231", "2026-07-01", "2026-07-01:2026-07-31"} {
		if got, rimossi := ValoreRipulito("data", v); got != v || rimossi != nil {
			t.Errorf("--data %q doveva restare intatto, ottenuto %q", v, got)
		}
	}
	// La stessa forma dentro un campo di testo NON e' una data: e' un
	// documento che cita quella data, e va ripulito come tutto il resto.
	if got, rimossi := ValoreRipulito("testo", "2026-07-01"); got != "2026 07 01" || len(rimossi) != 1 {
		t.Errorf("--testo \"2026-07-01\" = %q rimossi=%v, atteso «2026 07 01»", got, rimossi)
	}
	if got, _ := ValoreRipulito("isbn", "978-88-7524-166-7"); got != "978 88 7524 166 7" {
		t.Errorf("un ISBN non e' una data: %q", got)
	}
	// Un --data in una forma che nessuno riconosce non viene ne' ripulito di
	// nascosto ne' spedito come se fosse valido: resta com'e' e il portale lo
	// rifiuta rumorosamente.
	if got, _ := ValoreRipulito("data", "2026-7-1"); got != "2026 7 1" {
		t.Errorf("forma di data non riconosciuta: attesa la ripulitura, ottenuto %q", got)
	}
	if !campoData("data") || !campoData("anno") || campoData("testo") || campoData("isbn") {
		t.Error("l'esenzione deve valere per --data e --anno e per nessun altro")
	}
}

// Sul campo ISBN la fonte tiene due grafie e nessuna riscrittura da sola le
// copre entrambe. Misurato sull'archivio 205 il 2026-09-06: il record
// 9788875241667 esce solo dalla forma unita, il record scritto "978 88
// 98231-25-6" solo da quella coi separatori resi spazio.
func TestEspressioneIdentificativo(t *testing.T) {
	if got := EspressioneIdentificativo("978 88 7524 166 7"); got != "(9788875241667 O (978 88 7524 166 7))" {
		t.Errorf("EspressioneIdentificativo = %q", got)
	}
	// Nessun separatore: una grafia sola, niente OR da costruire.
	if got := EspressioneIdentificativo("9788875241667"); got != "9788875241667" {
		t.Errorf("valore gia' unito riscritto: %q", got)
	}
	if !CampoIdentificativo("isbn") || CampoIdentificativo("dewey") {
		t.Error("l'eccezione vale per isbn e non per dewey: sono stati misurati al contrario")
	}
}

func TestBuildQueryISBNMandaEntrambeLeGrafie(t *testing.T) {
	arc := Archive{ID: "205", Slug: "biblioteca", FieldMap: map[string]string{"isbn": "ISBN"}}
	got := BuildQuery(arc, map[string]string{"isbn": "978-88-7524-166-7"}, "")
	want := "((9788875241667 O (978 88 7524 166 7)).ISBN)"
	if got != want {
		t.Errorf("BuildQuery = %q, atteso %q", got, want)
	}
}

// Il portale rifiuta per sintassi in due modi, e il secondo porta un codice:
// guardare solo la presenza del codice lo confonderebbe con la soglia.
func TestQueryNonCostruibileRiconosceEntrambiIModi(t *testing.T) {
	casi := []struct {
		body string
		want bool
	}{
		{`<div class="message ko"> Impossibile creare la Query  QRY0 ()`, true},
		{`<div class="message ko"> Operando con crt non validi/bl (QR999)  QRY0 ()`, true},
		{`<div class="message ko"> (QR997)  QRY0 ()`, false},
	}
	for _, c := range casi {
		if got := QueryNonCostruibile(c.body); got != c.want {
			t.Errorf("QueryNonCostruibile(%.50q) = %v, atteso %v", c.body, got, c.want)
		}
	}
}
