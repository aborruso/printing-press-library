package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// tipologiaInfo describes one ANAC avviso tipologia. The `Template` value is
// exactly what the API expects in the `codiceScheda` query parameter
// (counterintuitively, the search filter keys on the template id, not on a
// scheda code). Slugs are friendly aliases accepted by `cerca --tipologia`.
type tipologiaInfo struct {
	Template  string   `json:"template"`
	Categoria string   `json:"categoria"`
	Tipologia string   `json:"tipologia"`
	Slugs     []string `json:"slugs"`
	// Tipo è il valore del campo `tipologia` nei record restituiti dall'API.
	// Serve a filtrare lato client la ricerca avanzata, che non accetta
	// codiceScheda. Valori verificati interrogando l'API per ogni template.
	Tipo string `json:"tipo"`
}

// tipologie is the full taxonomy, derived from the platform's /map endpoint and
// the advanced-search dropdown. Stable reference data, embedded as code.
var tipologie = []tipologiaInfo{
	{"1", "ALTRI_AVVISI", "Avvisi di pre-informazione informativi", []string{"pre-informazione-informativi"}, "AVVISI_DI_PRE-INFORMAZIONE_INFORMATIVI"},
	{"2", "BANDI_DI_GARA", "Avvisi di pre-informazione indittivi", []string{"pre-informazione-indittivi"}, "AVVISI_DI_PRE-INFORMAZIONE_INDITTIVI"},
	{"3", "ALTRI_AVVISI", "Sistemi di qualificazione", []string{"sistemi-qualificazione"}, "SISTEMI_DI_QUALIFICAZIONE"},
	{"4", "BANDI_DI_GARA", "Bandi", []string{"bandi"}, "BANDI"},
	{"5a", "ALTRI_AVVISI", "Indagini di mercato pari o sopra soglia", []string{"indagini-sopra-soglia", "indagini-sopra"}, "INDAGINI_DI_MERCATO_PARI_O_SOPRA_SOGLIA"},
	{"5b", "ALTRI_AVVISI", "Indagini di mercato sotto soglia", []string{"indagini-sotto-soglia", "indagini-sotto"}, "INDAGINI_DI_MERCATO_SOTTO_SOGLIA"},
	{"6", "ALTRI_AVVISI", "Elenchi operatori economici", []string{"elenchi-operatori", "elenchi"}, "ELENCHI_OPERATORI_ECONOMICI"},
	{"7", "ESITI_DI_GARA", "Risultati (esiti di gara)", []string{"esiti", "risultati"}, "RISULTATI"},
	{"8a", "ESITI_DI_GARA", "Affidamenti diretti sotto soglia", []string{"affidamenti-diretti"}, "AFFIDAMENTI_DIRETTI_SOTTO_SOGLIA"},
	{"8b", "ALTRI_AVVISI", "Affidamenti in house", []string{"affidamenti-in-house", "in-house"}, "AFFIDAMENTI_IN_HOUSE"},
	{"9", "ESITI_DI_GARA", "Preavvisi di aggiudicazione diretta", []string{"preavvisi-aggiudicazione", "preavvisi"}, "PREAVVISI_DI_AGGIUDICAZIONE_DIRETTA"},
	{"10", "ALTRI_AVVISI", "Modifiche contrattuali", []string{"modifiche-contrattuali", "modifiche"}, "MODIFICHE_CONTRATTUALI"},
}

// resolveTipologiaTipo mappa l'input utente al valore del campo `tipologia`
// dei record (es. "esiti" -> "RISULTATI"), usato per filtrare lato client i
// risultati della ricerca avanzata, che non ha un filtro per tipologia.
func resolveTipologiaTipo(input string) (string, bool) {
	tpl, ok := resolveTipologia(input)
	if !ok {
		return "", false
	}
	for _, t := range tipologie {
		if t.Template == tpl {
			return t.Tipo, true
		}
	}
	return "", false
}

// resolveTipologia maps a user-supplied value (template id, slug, or exact
// tipologia label) to the API `codiceScheda` template value. Returns ("", false)
// if unrecognized.
func resolveTipologia(input string) (string, bool) {
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return "", false
	}
	for _, t := range tipologie {
		if strings.ToLower(t.Template) == in || strings.ToLower(t.Tipologia) == in {
			return t.Template, true
		}
		for _, s := range t.Slugs {
			if s == in {
				return t.Template, true
			}
		}
	}
	return "", false
}

func newTipologieListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "Elenca le tipologie di avviso con il valore di filtro da usare nella ricerca",
		Long:        "Mostra la tassonomia completa delle tipologie di avviso ANAC: numero template (valore da passare a --tipologia / --scheda), categoria, e nome leggibile.",
		Example:     "  anac-pl-pp-cli tipologie list\n  anac-pl-pp-cli tipologie list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would list tipologie")
				return nil
			}
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), tipologie, flags)
			}
			rows := make([]map[string]any, 0, len(tipologie))
			for _, t := range tipologie {
				rows = append(rows, map[string]any{
					"template":  t.Template,
					"categoria": t.Categoria,
					"tipologia": t.Tipologia,
					"slug":      strings.Join(t.Slugs, ", "),
				})
			}
			return printAutoTable(cmd.OutOrStdout(), rows)
		},
	}
	return cmd
}
