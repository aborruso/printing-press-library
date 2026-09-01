// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cpvdata

import "testing"

func TestVocabolarioCaricato(t *testing.T) {
	if Count() < 9000 {
		t.Fatalf("vocabolario CPV incompleto: %d voci", Count())
	}
	e, ok := Get("30213000")
	if !ok {
		t.Fatal("codice 30213000 assente dal vocabolario")
	}
	if e.Description == "" {
		t.Error("la voce deve portare la descrizione italiana")
	}
}

func TestNormalizeCPV(t *testing.T) {
	cases := []struct {
		name     string
		in       any
		wantCode string
		wantOK   bool
	}{
		{"codice come stringa", "30213000", "30213000", true},
		{"codice con cifra di controllo", "30213000-5", "30213000", true},
		{"oggetto con codice", map[string]any{"codice": "30213000", "descrizione": ""}, "30213000", true},
		{"stringa vuota", "", "", false},
		{"tipo non gestito", 42, "", false},
	}
	for _, c := range cases {
		code, _, ok := NormalizeCPV(c.in)
		if code != c.wantCode || ok != c.wantOK {
			t.Errorf("%s: NormalizeCPV(%v) = (%q, %v), atteso (%q, %v)", c.name, c.in, code, ok, c.wantCode, c.wantOK)
		}
	}
}

func TestNormalizeCPVRisaleAlCodiceDallaDescrizione(t *testing.T) {
	e, ok := Get("30213000")
	if !ok {
		t.Skip("codice di riferimento assente")
	}
	code, _, ok := NormalizeCPV(e.Description)
	if !ok || code != "30213000" {
		t.Errorf("dalla descrizione %q atteso il codice 30213000, ottenuto %q (ok=%v)", e.Description, code, ok)
	}
}

func TestSearchTrovaPerParola(t *testing.T) {
	res := Search("posta elettronica", 10)
	if len(res) == 0 {
		t.Fatal("nessun risultato per \"posta elettronica\"")
	}
	if len(res) > 10 {
		t.Errorf("limit non rispettato: %d risultati", len(res))
	}
}
