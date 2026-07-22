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
	rows, pages, err := parseBDList(bdSommariFixture, Archive{Slug: "sommari"})
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
	if r.Fields["Legisl."] != "XVIII" {
		t.Errorf("Legisl. = %q", r.Fields["Legisl."])
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
	rows, pages, err := parseBDList(fixture, Archive{Slug: "resoconti"})
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
	sp := parseBDSpeakers(form)
	if len(sp) != 3 { // "Tutte" e la legislatura (senza data-legs) esclusi
		t.Fatalf("parseBDSpeakers = %d oratori, want 3: %+v", len(sp), sp)
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
		got := resolveSpeakerIDs(sp, c.q, c.legisl)
		if len(got) != len(c.want) || (len(got) == 1 && got[0] != c.want[0]) {
			t.Errorf("resolveSpeakerIDs(%q, legisl=%q) = %v, want %v", c.q, c.legisl, got, c.want)
		}
	}
	if !legsContains("18,17,16", "18") || legsContains("13", "18") {
		t.Error("legsContains errato")
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

func TestIsBDArchive(t *testing.T) {
	if !isBDArchive("sommari") {
		t.Error("sommari deve essere un archivio /bd/")
	}
	if isBDArchive("ddl") {
		t.Error("ddl NON deve essere /bd/ (resta su Icaro)")
	}
}
