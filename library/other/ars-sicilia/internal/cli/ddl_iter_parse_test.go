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
