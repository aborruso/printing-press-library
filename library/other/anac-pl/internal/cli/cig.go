package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cig"

	"github.com/spf13/cobra"
)

func newCigCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cig",
		Short: "Strumenti sul Codice Identificativo di Gara (CIG)",
		Long:  "Controlli offline sul CIG con l'algoritmo pubblicato da ANAC (anticorruzione/npa, Algoritmo validazione CIG).",
	}
	cmd.AddCommand(newCigCheckCmd(flags))
	return cmd
}

func newCigCheckCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <cig>...",
		Short: "Verifica struttura e cifra di controllo di uno o più CIG, offline",
		Long: strings.Trim(`
Verifica un CIG con l'algoritmo di ANAC, senza chiamare il servizio. Riconosce
le tre famiglie: Simog (iniziale numerica), Simog seconda versione e PCP
(iniziale da A a U) e SmartCIG (iniziale X, Y o Z).

Un CIG che non supera il controllo è stato trascritto male: cercarlo restituisce
zero risultati senza dire perché. Esce con 0 se tutti i CIG sono validi, con 2
se almeno uno non lo è; l'esito di ciascuno è comunque nell'output.
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli cig check B7E26B1DC7
  anac-pl-pp-cli cig check B7E26B1DC7 Z94375BBCC 5527244A08 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would check CIG codes")
				return nil
			}
			esiti := make([]cig.Esito, 0, len(args))
			nonValidi := 0
			for _, a := range args {
				e := cig.Verifica(a)
				if !e.Valido {
					nonValidi++
				}
				esiti = append(esiti, e)
			}
			var err error
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				err = printJSONFiltered(cmd.OutOrStdout(), esiti, flags)
			} else {
				rows := make([]map[string]any, 0, len(esiti))
				for _, e := range esiti {
					rows = append(rows, map[string]any{"cig": e.CIG, "valido": e.Valido, "tipo": e.Tipo, "motivo": e.Motivo})
				}
				err = printAutoTable(cmd.OutOrStdout(), rows)
			}
			if err != nil {
				return err
			}
			if nonValidi > 0 {
				return usageErr(fmt.Errorf("%d CIG su %d non validi", nonValidi, len(esiti)))
			}
			return nil
		},
	}
	return cmd
}

// warnCIGNonValido segnala su stderr un testo libero che ha la forma di un CIG
// ma non supera la cifra di controllo: la ricerca restituirebbe zero risultati
// o avvisi estranei senza spiegare che il codice è trascritto male.
func warnCIGNonValido(w interface{ Write([]byte) (int, error) }, query string) {
	for _, tok := range strings.Fields(query) {
		if !cig.SembraCIG(tok) {
			continue
		}
		if e := cig.Verifica(tok); !e.Valido {
			fmt.Fprintf(w, "avviso: %s ha la forma di un CIG ma non è valido (%s); controlla la trascrizione\n", e.CIG, e.Motivo)
		}
	}
}
