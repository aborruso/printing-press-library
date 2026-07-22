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
