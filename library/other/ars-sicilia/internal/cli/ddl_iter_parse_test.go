package cli

import "testing"

func TestParseDdlFirmatari_Bullet(t *testing.T) {
	body := "Parlamentare  Geraci Salvatore (Prima l'Italia - Lega Salvini premier). • Assenza Giorgio (Fratelli d'Italia XVIII Legislatura).• Pellegrino Stefano (Forza Italia all'ARS). (n. 1089/A) DISEGNO DI LEGGE"
	f := parseDdlFirmatari(body)
	if len(f) != 3 || f[0].Nome != "Geraci Salvatore" || f[0].Gruppo != "Prima l'Italia - Lega Salvini premier" {
		t.Fatalf("bullet parse wrong: %+v", f)
	}
}

func TestParseDdlFirmatari_Presentato(t *testing.T) {
	body := "Titolo. presentato dai deputati: Spada, Catanzaro, Cracolici. RELAZIONE"
	f := parseDdlFirmatari(body)
	if len(f) != 3 || f[2].Nome != "Cracolici" {
		t.Fatalf("presentato parse wrong: %+v", f)
	}
}

func TestParseIterFromBody_FullHistory(t *testing.T) {
	body := "x Attuale 11 mar 2026 Respinto dall' Aula Seduta n. 236 AULA Storico 03 mar 2026 Assegnato per esame Commissione PRIMA 10 mar 2026 Esaminato in commissione Seduta n. 252 0100 Commissione (n. 1089/A) DISEGNO"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("want 3 events, got %d: %+v", len(ev), ev)
	}
	if ev[0].Titolo != "Respinto dall' Aula" || ev[1].Fase != "commissione" {
		t.Fatalf("iter parse wrong: %+v", ev)
	}
}

// TestParseIterFromBody_NoNumeroHeader covers finanziaria-style bills that
// have no "(n. ...)" header and open straight into the bill text: without a
// second cut marker, dates cited inside the articolato (here "3 luglio
// 1950") used to leak in as spurious iter events.
func TestParseIterFromBody_NoNumeroHeader(t *testing.T) {
	body := "x Attuale 09 gen 2026 Concluso Storico 06 nov 2025 Assegnato per esame Commissione SECONDA 09 gen 2026 Pubblicazione Gurs\n\nASSEMBLEA REGIONALE SICILIANA DISEGNO DI LEGGE N. 1030 LEGGE APPROVATA IL 21 DICEMBRE 2025 Art. 1. richiama la legge regionale 3 luglio 1950, n. 51"
	ev := parseIterFromBody(body)
	if len(ev) != 3 {
		t.Fatalf("want 3 events, got %d: %+v", len(ev), ev)
	}
	if ev[2].Titolo != "Pubblicazione Gurs" {
		t.Fatalf("iter parse wrong: %+v", ev)
	}
	for _, e := range ev {
		if e.Titolo == "richiama la legge regionale" || e.Data == "3 luglio 1950" {
			t.Fatalf("bill-text date leaked into iter events: %+v", ev)
		}
	}
}
