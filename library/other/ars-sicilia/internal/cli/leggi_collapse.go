// Novel behaviour — l'archivio leggi (201) è indicizzato PER ARTICOLO: una
// legge di venti articoli occupa venti righe, tutte con lo stesso titolo.
//
// Con la resa riga-per-riga il `--limit` veniva consumato dagli articoli della
// prima legge: `leggi cerca --legisl 18 --anno 2025 --limit 10` rispondeva
// "L.R. 1" e basta, mentre nel 2025 le leggi sono almeno quattro. Non era un
// problema di leggibilità: era una risposta sbagliata data in silenzio, perché
// nulla nell'output diceva che le altre leggi esistevano.
//
// Qui le righe si aggregano per legge, così `--limit` conta leggi. Gli articoli
// restano raggiungibili con `--articoli`.

package cli

import (
	"strings"

	icaro "github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/icaroclient"
)

// leggeAggregata è una legge, con gli articoli che la ricerca ha agganciato.
//
// ArticoliTrovati conta gli articoli **restituiti da questa ricerca**, non gli
// articoli della legge: con `--testo rifiuti` sono gli articoli che citano
// "rifiuti", e con una ricerca per anno sono quelli entrati nella finestra
// scaricata. Il nome dice "trovati" per non far leggere un totale dove non c'è.
type leggeAggregata struct {
	Legisl          string   `json:"legisl,omitempty"`
	Atto            string   `json:"atto,omitempty"`
	Data            string   `json:"data,omitempty"`
	Titolo          string   `json:"titolo,omitempty"`
	ArticoliTrovati int      `json:"articoli_trovati"`
	Articoli        []string `json:"articoli,omitempty"`
	URL             string   `json:"url,omitempty"`
}

// collapseLeggi aggrega le righe-articolo per legge, conservando l'ordine di
// arrivo dal portale (già ordinato per data decrescente).
//
// La chiave è legislatura + atto + data, non il solo atto: "L.R. 1" si ripete
// ogni anno, e senza la data due leggi omonime di anni diversi collasserebbero
// in una.
func collapseLeggi(recs []icaro.Record) []leggeAggregata {
	idx := map[string]int{}
	out := []leggeAggregata{}
	for _, r := range recs {
		atto := strings.TrimSpace(r.Fields["Atto"])
		data := strings.TrimSpace(r.Fields["Data"])
		legisl := strings.TrimSpace(r.Fields["Legisl."])
		if atto == "" && data == "" {
			continue
		}
		chiave := legisl + "|" + atto + "|" + data
		i, visto := idx[chiave]
		if !visto {
			idx[chiave] = len(out)
			out = append(out, leggeAggregata{
				Legisl: legisl,
				Atto:   atto,
				Data:   data,
				Titolo: r.Title,
				URL:    r.URL,
			})
			i = len(out) - 1
		}
		out[i].ArticoliTrovati++
		if art := strings.TrimSpace(r.Fields["Docum."]); art != "" {
			out[i].Articoli = append(out[i].Articoli, art)
		}
	}
	return out
}

// leggiRawLimit traduce un limite espresso in leggi nel numero di righe da
// scaricare. Una legge occupa una riga per articolo, quindi servono molte più
// righe di quante leggi si vogliano: il fattore è generoso perché una
// finanziaria da sola vale decine di articoli.
//
// Il tetto è solo una rete di sicurezza contro un --limit assurdo, non un
// freno: deve restare alto abbastanza da non rendere inefficace l'unico rimedio
// che l'utente ha quando l'output è troncato, cioè alzare --limit. Il portale è
// limitato a 2 richieste al secondo e Icaro pagina a 10 righe, quindi il costo
// è circa mezzo secondo per legge chiesta.
func leggiRawLimit(limitLeggi int) int {
	if limitLeggi <= 0 {
		limitLeggi = 10
	}
	raw := limitLeggi * 10
	if raw > 500 {
		raw = 500
	}
	return raw
}
