// Package giurisdizione classifica un fornitore (aggiudicatario di un
// affidamento) per giurisdizione: IT, UE o EXTRA-UE. La giurisdizione si
// determina sulla nazionalità del fornitore (e del gruppo di appartenenza),
// non sulla collocazione del dato: un fornitore soggetto a giurisdizione
// extra-UE (es. CLOUD Act USA) espone i dati anche se ospitati in UE.
//
// L'elenco è un punto di partenza estendibile, focalizzato sui provider
// email/cloud/collaboration più diffusi nella PA italiana.
package giurisdizione

import "strings"

// Esito della classificazione.
const (
	IT      = "IT"
	UE      = "UE"
	ExtraUE = "EXTRA-UE"
	Ignota  = "?"
)

type provider struct {
	pattern      string // sottostringa normalizzata cercata nella denominazione
	jurisdiction string
	gruppo       string // gruppo/casa madre (per resellers e controllate)
}

// providers: l'ordine conta solo per leggibilità; il match è per sottostringa.
// I pattern vanno scritti normalizzati (minuscolo, senza punteggiatura).
var providers = []provider{
	// EXTRA-UE (prevalentemente USA + alcuni altri)
	{"microsoft", ExtraUE, "Microsoft (USA)"},
	{"google", ExtraUE, "Google/Alphabet (USA)"},
	{"alphabet", ExtraUE, "Google/Alphabet (USA)"},
	{"amazon", ExtraUE, "Amazon/AWS (USA)"},
	{"aws", ExtraUE, "Amazon/AWS (USA)"},
	{"oracle", ExtraUE, "Oracle (USA)"},
	{"salesforce", ExtraUE, "Salesforce (USA)"},
	{"ibm", ExtraUE, "IBM (USA)"},
	{"cisco", ExtraUE, "Cisco (USA)"},
	{"zoom", ExtraUE, "Zoom (USA)"},
	{"dropbox", ExtraUE, "Dropbox (USA)"},
	{"adobe", ExtraUE, "Adobe (USA)"},
	{"vmware", ExtraUE, "VMware/Broadcom (USA)"},
	{"broadcom", ExtraUE, "Broadcom (USA)"},
	{"cloudflare", ExtraUE, "Cloudflare (USA)"},
	{"akamai", ExtraUE, "Akamai (USA)"},
	{"atlassian", ExtraUE, "Atlassian (AUS)"},
	{"zoho", ExtraUE, "Zoho (India)"},
	{"openai", ExtraUE, "OpenAI (USA)"},

	// UE (non Italia)
	{"ovh", UE, "OVHcloud (FR)"},
	{"scaleway", UE, "Scaleway (FR)"},
	{"gandi", UE, "Gandi (FR)"},
	{"hetzner", UE, "Hetzner (DE)"},
	{"sap", UE, "SAP (DE)"},
	{"deutsche telekom", UE, "Deutsche Telekom (DE)"},
	{"ionos", UE, "IONOS (DE)"},

	// IT
	{"aruba", IT, "Aruba (IT)"},
	{"register.it", IT, "Register.it (IT)"},
	{"register it", IT, "Register.it (IT)"},
	{"infocert", IT, "InfoCert (IT)"},
	{"namirial", IT, "Namirial (IT)"},
	{"seeweb", IT, "Seeweb (IT)"},
	{"netsons", IT, "Netsons (IT)"},
	{"poste italiane", IT, "Poste Italiane (IT)"},
	{"postecom", IT, "Poste Italiane (IT)"},
	{"sogei", IT, "Sogei (IT)"},
	{"almaviva", IT, "Almaviva (IT)"},
	{"engineering ingegneria", IT, "Engineering (IT)"},
	{"tim ", IT, "TIM (IT)"},
	{"telecom italia", IT, "TIM (IT)"},
	{"fastweb", IT, "Fastweb (IT)"},
	{"vodafone italia", IT, "Vodafone Italia (IT)"},
	{"poste italiane", IT, "Poste Italiane (IT)"},
	{"kolibri", IT, "Kolibri/Libra (IT)"},
	{"westpole", IT, "WestPole (IT)"},
	{"lepida", IT, "Lepida (IT)"},
}

// normalize abbassa a minuscolo, rimuove punteggiatura comune e comprime spazi.
func normalize(s string) string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(
		".", " ", ",", " ", "'", " ", "\"", " ", "-", " ",
		"(", " ", ")", " ", "/", " ", "&", " ",
	)
	s = repl.Replace(s)
	return " " + strings.Join(strings.Fields(s), " ") + " "
}

// Result è l'esito della classificazione.
type Result struct {
	Giurisdizione string `json:"giurisdizione"`
	Gruppo        string `json:"gruppo,omitempty"`
}

// Classify determina la giurisdizione del fornitore dalla sua denominazione.
// Restituisce Ignota quando nessun provider noto corrisponde.
func Classify(denominazione string) Result {
	hay := normalize(denominazione)
	for _, p := range providers {
		needle := " " + strings.Trim(normalize(p.pattern), " ") + " "
		// match anche come prefisso/sottostringa interna alla denominazione
		if strings.Contains(hay, strings.TrimSpace(needle)) {
			return Result{Giurisdizione: p.jurisdiction, Gruppo: p.gruppo}
		}
	}
	return Result{Giurisdizione: Ignota}
}
