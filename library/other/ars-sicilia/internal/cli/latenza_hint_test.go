package cli

import (
	"strings"
	"testing"
	"time"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
)

// Un cerca vuoto con --data a ridosso di oggi deve dire che può essere
// latenza della fonte; uno che finisce mesi fa no, e con risultati mai.
func TestLatenzaHint(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		recs []icaro.Record
		data string
		want bool
	}{
		{"range recente vuoto", nil, "2026-08-01:2026-09-04", true},
		{"data singola recente vuota", nil, "2026-09-02", true},
		{"range vecchio vuoto", nil, "2020-11-01:2020-12-31", false},
		{"al limite dei 45 giorni", nil, "2026-07-21", true},
		{"oltre i 45 giorni", nil, "2026-07-20", false},
		{"senza --data", nil, "", false},
		{"con risultati", []icaro.Record{{}}, "2026-09-02", false},
		{"data non ISO", nil, "260902", false},
		{"finestra tutta nel futuro", nil, "2027-01-01:2027-12-31", false},
		{"data singola futura", nil, "2026-09-10", false},
		{"a cavallo di oggi: copre giorni recenti", nil, "2026-08-15:2026-09-30", true},
		{"inizio vecchio, fine futura: si valuta su oggi", nil, "2026-07-01:2026-12-31", true},
	}
	for _, c := range cases {
		got := latenzaHint(c.recs, "interrogazioni", map[string]string{"data": c.data}, now)
		if (got != "") != c.want {
			t.Errorf("%s: got %q, want hint=%v", c.name, got, c.want)
		}
		if got != "" && !strings.Contains(got, "--resources interrogazioni") {
			t.Errorf("%s: l'hint non nomina l'archivio: %q", c.name, got)
		}
	}
}
