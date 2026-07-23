package icaroclient

import "testing"

const bdSommariFixture = `
<html><body>
<ul class="tabella">
  <li class="intestazione">
    <div class="intesta intesta_10"><p>Legisl.</p></div>
    <div class="intesta intesta_10"><p>Data</p></div>
    <div class="intesta intesta_10"><p>N. Seduta</p></div>
    <div class="intesta intesta_40"><p>Commissione e Ordine del giorno</p></div>
  </li>
  <li>
    <div class="intesta intesta_10"><strong><span class="simobile">Legisl.</span></strong><p> XVIII </p></div>
    <div class="intesta intesta_10"><strong><span class="simobile">Data</span></strong><p> 14/07/2026 </p></div>
    <div class="intesta intesta_10"><strong><span class="simobile">N. Seduta</span></strong><p> 271 </p></div>
    <div class="intesta intesta_40"><strong><span class="simobile">Commissione e Ordine del giorno</span></strong>
      <h3><a href="javascript: openRisultati('18','116','271')"> I - Affari Istituzionali </a></h3>
      <p> 1) Esame del DEFR &quot;2027-2029&quot; </p></div>
  </li>
</ul>
<div class="pagination"><span class="pagina_di">Pagina 1 di 23</span></div>
</body></html>`

func TestParseBDList(t *testing.T) {
	rows, pages, err := parseBDList(bdSommariFixture, Archive{Slug: "sommari"}, "https://dati.ars.sicilia.it")
	if err != nil {
		t.Fatalf("parseBDList: %v", err)
	}
	if pages != 23 {
		t.Errorf("pages = %d, want 23", pages)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 (header must be skipped)", len(rows))
	}
	r := rows[0]
	if r.Fields["Legisl."] != "18" { // "XVIII" normalizzato in arabo
		t.Errorf("Legisl. = %q, want 18", r.Fields["Legisl."])
	}
	if r.Fields["Data"] != "14/07/2026" {
		t.Errorf("Data = %q", r.Fields["Data"])
	}
	if r.Fields["Numero"] != "271" { // "N. Seduta" normalizzato su "Numero"
		t.Errorf("Numero = %q", r.Fields["Numero"])
	}
	if r.Title != "I - Affari Istituzionali" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Excerpt != `1) Esame del DEFR "2027-2029"` { // entità &quot; decodificata
		t.Errorf("Excerpt = %q", r.Excerpt)
	}
	if r.URL != "https://dati.ars.sicilia.it/bd/sommari/scheda/18/116/271" { // openRisultati('18','116','271')
		t.Errorf("URL = %q", r.URL)
	}
}

// TestParseBDList_Resoconti copre la forma resoconti: colonna "Numero" (non
// "N. Seduta") e ultima colonna "Titolo" con <h3><a> ma SENZA <p> (excerpt vuoto).
func TestParseBDList_Resoconti(t *testing.T) {
	const fixture = `<ul class="tabella">
  <li class="intestazione"><div class="intesta"><p>Legisl.</p></div></li>
  <li>
    <div class="intesta intesta_10"><strong><span class="simobile">Legisl.</span></strong><p> XVIII </p></div>
    <div class="intesta intesta_16"><strong><span class="simobile">Numero</span></strong><p> 264 </p></div>
    <div class="intesta intesta_16"><strong><span class="simobile">Data</span></strong><p> 14/07/2026 </p></div>
    <div class="intesta intesta_50"><strong><span class="simobile">Titolo</span></strong>
      <h3><a href="javascript: openRisultati('18','264')"> Resoconto d'Aula della Seduta n. 264 </a></h3></div>
  </li>
</ul><span class="pagina_di">Pagina 1 di 5</span>`
	rows, pages, err := parseBDList(fixture, Archive{Slug: "resoconti"}, "https://dati.ars.sicilia.it")
	if err != nil {
		t.Fatalf("parseBDList: %v", err)
	}
	if pages != 5 || len(rows) != 1 {
		t.Fatalf("pages=%d rows=%d, want 5/1", pages, len(rows))
	}
	r := rows[0]
	if r.Fields["Numero"] != "264" {
		t.Errorf("Numero = %q", r.Fields["Numero"])
	}
	if r.Fields["Data"] != "14/07/2026" {
		t.Errorf("Data = %q", r.Fields["Data"])
	}
	if r.Title != "Resoconto d'Aula della Seduta n. 264" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Excerpt != "" {
		t.Errorf("Excerpt = %q, want empty (nessun <p>)", r.Excerpt)
	}
	if r.URL != "https://dati.ars.sicilia.it/bd/resoconti/scheda/18/264" { // openRisultati('18','264')
		t.Errorf("URL = %q", r.URL)
	}
}

// TestParseBDList_Convocazioni copre la forma a 5 colonne: "Commissione" è una
// colonna propria (<p> semplice), "N. Foglio" -> "Numero", l'OdG è l'<h3>.
func TestParseBDList_Convocazioni(t *testing.T) {
	const fixture = `<ul class="tabella">
  <li class="intestazione"><div class="intesta"><p>Legisl.</p></div></li>
  <li>
    <div class="intesta intesta_10"><strong><span class="simobile">Legisl.</span></strong><p> XVIII </p></div>
    <div class="intesta intesta_10"><strong><span class="simobile">Data</span></strong><p> 22/07/2026 </p></div>
    <div class="intesta intesta_10"><strong><span class="simobile">N. Foglio</span></strong><p> 287 </p></div>
    <div class="intesta intesta_20"><strong><span class="simobile">Commissione</span></strong><p> I - Affari Istituzionali </p></div>
    <div class="intesta intesta_40"><strong><span class="simobile">Ordine del giorno</span></strong>
      <h3><a href="javascript: openRisultati('uuid')"> 1) Esame del ddl 779 </a></h3></div>
  </li>
</ul><span class="pagina_di">Pagina 1 di 28</span>`
	rows, pages, err := parseBDList(fixture, Archive{Slug: "convocazioni"}, "https://dati.ars.sicilia.it")
	if err != nil {
		t.Fatalf("parseBDList: %v", err)
	}
	if pages != 28 || len(rows) != 1 {
		t.Fatalf("pages=%d rows=%d, want 28/1", pages, len(rows))
	}
	r := rows[0]
	if r.Fields["Commissione"] != "I - Affari Istituzionali" {
		t.Errorf("Commissione = %q", r.Fields["Commissione"])
	}
	if r.Fields["Numero"] != "287" { // da "N. Foglio"
		t.Errorf("Numero = %q", r.Fields["Numero"])
	}
	if r.Title != "1) Esame del ddl 779" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.URL != "https://dati.ars.sicilia.it/bd/convocazioni/results/uuid" { // openRisultati('uuid')
		t.Errorf("URL = %q", r.URL)
	}
}

func TestBDDateFilter(t *testing.T) {
	cases := []struct {
		in       string
		wantAnno string
		date     string // riga dd/mm/yyyy da testare
		keep     bool
	}{
		{"260714", "2026", "14/07/2026", true},                // AAMMGG esatto
		{"260714", "2026", "13/07/2026", false},               // altra data stesso anno
		{"2026-07-14", "2026", "14/07/2026", true},            // ISO esatto
		{"260701/260731", "2026", "14/07/2026", true},         // range AAMMGG, dentro
		{"260701/260731", "2026", "01/08/2026", false},        // range AAMMGG, fuori
		{"2026-07-01:2026-07-31", "2026", "14/07/2026", true}, // range ISO, dentro
	}
	for _, c := range cases {
		anno, keep := bdDateFilter(c.in)
		if anno != c.wantAnno {
			t.Errorf("bdDateFilter(%q) anno = %q, want %q", c.in, anno, c.wantAnno)
		}
		if keep == nil {
			t.Fatalf("bdDateFilter(%q) keep = nil", c.in)
		}
		if got := keep(c.date); got != c.keep {
			t.Errorf("bdDateFilter(%q).keep(%q) = %v, want %v", c.in, c.date, got, c.keep)
		}
	}
	// valore non interpretabile -> nessun filtro
	if _, keep := bdDateFilter("garbage"); keep != nil {
		t.Errorf("bdDateFilter(garbage) keep should be nil")
	}
}

func TestBDSpeakers(t *testing.T) {
	// due spazi dopo <option, come nel markup del portale
	const form = `<select id="$Ispeakers" name="$Ispeakers" multiple="multiple">
<option  value="971" data-legs="18">Abbate Ignazio</option>
<option  value="32" data-legs="18,17,16,15,14,13">Cracolici Antonino</option>
<option  value="428" data-legs="13">Acanto Giuseppe</option>
<option  value="">Tutte</option>
</select>
<option selected value="18" >XVIII</option>`
	sp := parseSelectOptions(form, "$Ispeakers")
	if len(sp) != 3 { // "Tutte" e la legislatura (senza data-legs) esclusi
		t.Fatalf("parseSelectOptions = %d oratori, want 3: %+v", len(sp), sp)
	}
	cases := []struct {
		q, legisl string
		want      []string
	}{
		{"cracolici", "18", []string{"32"}}, // match + attivo in 18
		{"abbate", "18", []string{"971"}},   // case-insensitive
		{"acanto", "18", nil},               // Acanto è solo leg 13
		{"acanto", "", []string{"428"}},     // senza filtro legislatura
		{"nessuno", "18", nil},              // nessun match
	}
	for _, c := range cases {
		got := resolveOptionIDs(sp, c.q, c.legisl)
		if len(got) != len(c.want) || (len(got) == 1 && got[0] != c.want[0]) {
			t.Errorf("resolveOptionIDs(%q, legisl=%q) = %v, want %v", c.q, c.legisl, got, c.want)
		}
	}
	if !legsContains("18,17,16", "18") || legsContains("13", "18") {
		t.Error("legsContains errato")
	}
}

func TestResolveCommissioneIDs(t *testing.T) {
	// id commissione per-legislatura: "I - Affari Istituzionali" = 1 (leg13), 116 (leg18)
	opts := []bdOption{
		{ID: "1", Name: "I - Affari Istituzionali", Legs: "13"},
		{ID: "116", Name: "I - Affari Istituzionali", Legs: "18"},
		{ID: "2", Name: "II - Bilancio e Programmazione", Legs: "13"},
		{ID: "117", Name: "II - Bilancio", Legs: "18"},
		{ID: "11", Name: "Antimafia", Legs: "18"},
	}
	cases := []struct {
		cod, com, legisl string
		want             []string
	}{
		{"1", "", "18", []string{"116"}},        // codcom 1 -> "I " -> leg18
		{"1", "", "13", []string{"1"}},          // stessa commissione, leg diversa
		{"2", "", "18", []string{"117"}},        // "II " non confonde con "I "
		{"", "Bilancio", "18", []string{"117"}}, // nome, substring
		{"", "Antimafia", "18", []string{"11"}}, // commissione speciale
		{"", "inesistente", "18", []string{}},   // richiesto ma nessun match -> [] non nil
		{"7", "", "18", []string{}},             // codcom fuori 1-6 -> []
	}
	for _, c := range cases {
		got := resolveCommissioneIDs(opts, c.cod, c.com, c.legisl)
		if len(got) != len(c.want) || (len(got) == 1 && got[0] != c.want[0]) {
			t.Errorf("resolveCommissioneIDs(cod=%q com=%q leg=%q) = %v, want %v", c.cod, c.com, c.legisl, got, c.want)
		}
	}
	// nessun filtro richiesto -> nil
	if resolveCommissioneIDs(opts, "", "", "18") != nil {
		t.Error("senza codcom/commissione deve tornare nil")
	}
}

func TestDdmmyyyyToISO(t *testing.T) {
	if got := ddmmyyyyToISO("14/07/2026"); got != "20260714" {
		t.Errorf("ddmmyyyyToISO = %q", got)
	}
	if got := ddmmyyyyToISO("boh"); got != "" {
		t.Errorf("ddmmyyyyToISO(boh) = %q, want empty", got)
	}
}

func TestRomanToArabic(t *testing.T) {
	cases := map[string]string{
		"XVIII": "18", "XVII": "17", "I": "1", "IV": "4", "IX": "9", "XIV": "14",
		"18":  "18", // già arabo -> invariato
		"foo": "foo",
	}
	for in, want := range cases {
		if got := romanToArabic(in); got != want {
			t.Errorf("romanToArabic(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsBDArchive(t *testing.T) {
	if !IsBDArchive("sommari") {
		t.Error("sommari deve essere un archivio /bd/")
	}
	if IsBDArchive("ddl") {
		t.Error("ddl NON deve essere /bd/ (resta su Icaro)")
	}
}
