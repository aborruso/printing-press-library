// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package icaroclient

import "testing"

// TestParseShortList_LastColumnHoldsTitle covers the shortList row parsing:
// the last column's Fields entry (named after the column label, e.g.
// "Titolo") must hold the title itself, not the raw div text with the title
// stripped out — that leftover is the excerpt, and it used to masquerade as
// the title in JSON/CSV output and in biblioteca's sync ID derivation.
func TestParseShortList_LastColumnHoldsTitle(t *testing.T) {
	arc := Archive{
		ID:   "221",
		Slug: "ddl",
		FieldMap: map[string]string{
			"legisl": "LEGISL",
			"numero": "NUMDDL",
		},
		Columns: []string{"Legisl.", "Numero", "Data", "Firmatari", "Titolo"},
	}
	body := `<html><body><ul id="shortListTable">
		<li class="intestazione">header</li>
		<li href="javascript: showDoc(2)">
			<div>18</div>
			<div>6381</div>
			<div>9.01.24</div>
			<div>Rossi Mario</div>
			<div><h3>Disposizioni varie e finanziarie</h3><p>ddl n. 638/A Stralcio del 09 gennaio 2024 XVIII Legislatura</p></div>
		</li>
	</ul></body></html>`

	recs, _, err := ParseShortList(body, arc, "https://dati.ars.sicilia.it")
	if err != nil {
		t.Fatalf("ParseShortList error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d: %+v", len(recs), recs)
	}
	r := recs[0]
	if r.Title != "Disposizioni varie e finanziarie" {
		t.Fatalf("Title = %q, want %q", r.Title, "Disposizioni varie e finanziarie")
	}
	if r.Excerpt != "ddl n. 638/A Stralcio del 09 gennaio 2024 XVIII Legislatura" {
		t.Fatalf("Excerpt = %q, want the portal designation", r.Excerpt)
	}
	if got := r.Fields["Titolo"]; got != r.Title {
		t.Fatalf(`Fields["Titolo"] = %q, want it to match Title %q (not the excerpt)`, got, r.Title)
	}
}
