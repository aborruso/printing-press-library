// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package giurisdizione

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		denominazione string
		want          string
	}{
		{"Microsoft Ireland Operations Limited", ExtraUE},
		{"MICROSOFT SRL", ExtraUE},
		{"Google Cloud Italy S.r.l.", ExtraUE},
		{"Aruba S.p.A.", IT},
		{"Ditta Rossi Informatica di Rossi Mario", Ignota},
		{"", Ignota},
	}
	for _, c := range cases {
		got := Classify(c.denominazione)
		if got.Giurisdizione != c.want {
			t.Errorf("Classify(%q).Giurisdizione = %q, atteso %q", c.denominazione, got.Giurisdizione, c.want)
		}
	}
}

func TestClassifyValorizzaIlGruppoSoloQuandoRiconosce(t *testing.T) {
	if got := Classify("Microsoft Ireland Operations Limited"); got.Gruppo == "" {
		t.Error("un fornitore riconosciuto deve portare il gruppo di appartenenza")
	}
	if got := Classify("Ditta Rossi Informatica"); got.Gruppo != "" {
		t.Errorf("un fornitore non riconosciuto non deve dichiarare un gruppo, trovato %q", got.Gruppo)
	}
}

func TestNormalizeIgnoraMaiuscoleEPunteggiatura(t *testing.T) {
	// La normalizzazione trasforma la punteggiatura in spazi, quindi il match
	// regge su maiuscole e su punteggiatura attorno al nome, non dentro:
	// un acronimo puntato ("A.R.U.B.A.") diventa lettere separate e non
	// corrisponde. Nessuna denominazione reale dei provider censiti lo usa.
	cases := []string{"MICROSOFT S.R.L.", "microsoft srl", "Microsoft (Ireland) Operations Ltd."}
	for _, d := range cases {
		if got := Classify(d); got.Giurisdizione != ExtraUE {
			t.Errorf("Classify(%q).Giurisdizione = %q, atteso %q", d, got.Giurisdizione, ExtraUE)
		}
	}
}
