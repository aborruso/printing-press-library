package cli

import "testing"

func TestValidateCPVFilter(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", false},
		{"30213000", false},
		{"302", false},
		{"45", false},
		{"30213000-5", false},           // forma degli atti ufficiali
		{"30213000-9", false},           // il servizio ignora la cifra, giusta o sbagliata
		{"30213000-5, 42120000", false}, // più valori, con e senza cifra
		{"302-5", true},                 // prefisso + cifra: il servizio non trova nulla
		{"30213000-", true},             // trattino senza cifra
		{"30213000-55", true},           // due cifre dopo il trattino
		{"3021300-5", true},             // base di 7 cifre
		{"Computer personali", true},    // descrizione
		{"3", true},                     // meno di 2 cifre
		{"302130001", true},             // più di 8 cifre senza trattino
	}
	for _, c := range cases {
		err := validateCPVFilter(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("validateCPVFilter(%q) err=%v; wantErr=%v", c.in, err, c.wantErr)
		}
	}
}
