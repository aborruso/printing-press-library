package cli

import (
	"strings"
	"testing"
)

// L'avviso sulla punteggiatura descrive una riscrittura che avviene dentro
// BuildQuery. Gli archivi /bd/ BuildQuery non lo attraversano: il valore parte
// intatto nella POST del form e torna righe. Dirlo li' sarebbe un avviso che
// afferma il falso.
func TestPunteggiaturaHintMutoSugliArchiviBD(t *testing.T) {
	params := map[string]string{"testo": "dell'ambiente"}
	if h := punteggiaturaHint("resoconti", params); h != "" {
		t.Errorf("atteso muto su /bd/, ottenuto: %s", h)
	}
	if h := punteggiaturaHint("sommari", params); h != "" {
		t.Errorf("atteso muto su /bd/, ottenuto: %s", h)
	}
	h := punteggiaturaHint("ddl", params)
	if !strings.Contains(h, "dell ambiente") {
		t.Errorf("su un archivio ISIS l'avviso deve dire cosa e' partito: %q", h)
	}
}

// Su un campo identificativo partono due grafie, non una: annunciarne una sola
// descriverebbe meta' della query.
func TestPunteggiaturaHintSuIdentificativoNominaEntrambeLeGrafie(t *testing.T) {
	h := punteggiaturaHint("biblioteca", map[string]string{"isbn": "978-88-7524-166-7"})
	if !strings.Contains(h, "9788875241667") || !strings.Contains(h, "978 88 7524 166 7") {
		t.Errorf("l'avviso deve nominare entrambe le grafie: %q", h)
	}
}
