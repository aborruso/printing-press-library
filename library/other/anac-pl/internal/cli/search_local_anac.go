package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/store"

	"github.com/spf13/cobra"
)

// newAnacSearchLocalCmd replaces the generated stub with a real offline search
// over the local SQLite mirror populated by `sync`. It uses the store's
// full-text index when a query is given, or lists synced avvisi otherwise.
func newAnacSearchLocalCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var dbPath string
	cmd := &cobra.Command{
		Use:   "search-local [testo]",
		Short: "Cerca tra gli avvisi già sincronizzati in locale, senza rete.",
		Long: "Ricerca offline (full-text) sugli avvisi salvati nel database locale da 'sync'.\n" +
			"Esegui prima: anac-pl-pp-cli sync --resources avvisi --param keywords=<tema>",
		Example: strings.Trim(`
  anac-pl-pp-cli sync --resources avvisi --param keywords=microsoft
  anac-pl-pp-cli search-local microsoft
  anac-pl-pp-cli search-local "licenze software" --json --limit 20
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would search the local store")
				return nil
			}
			if dbPath == "" {
				dbPath = defaultDBPath("anac-pl-pp-cli")
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				fmt.Fprintf(cmd.ErrOrStderr(), "nessun mirror locale in %s\nesegui: anac-pl-pp-cli sync --resources avvisi --param keywords=<tema> --db %s\n", dbPath, dbPath)
				if flags.asJSON || flags.agent {
					fmt.Fprintln(cmd.OutOrStdout(), "[]")
				}
				return nil
			}
			db, err := store.OpenReadOnly(dbPath)
			if err != nil {
				return fmt.Errorf("apertura database: %w", err)
			}
			defer db.Close()

			query := strings.TrimSpace(strings.Join(args, " "))
			var rows []json.RawMessage
			if query == "" {
				rows, err = db.List("avvisi", limit)
			} else {
				rows, err = db.Search(query, limit)
			}
			if err != nil {
				return fmt.Errorf("ricerca locale: %w", err)
			}
			if rows == nil {
				rows = []json.RawMessage{}
			}
			// Una ricerca senza corrispondenze esce 3 (come il resto della CLI
			// sulle risorse non trovate): l'uscita su stdout resta quella
			// normale, cosi' chi legge il JSON non trova sorprese, ma chi
			// concatena comandi distingue «nessun risultato» da «trovato».
			if query != "" && len(rows) == 0 {
				printLocalSearchOutput(cmd, rows, flags)
				return notFoundErr(fmt.Errorf("nessun avviso locale corrisponde a %q (store: %s)", query, dbPath))
			}
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
			}
			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "nessun risultato locale (hai eseguito 'sync'?)")
				return nil
			}
			items := make([]map[string]any, 0, len(rows))
			for _, r := range rows {
				var m map[string]any
				if json.Unmarshal(r, &m) == nil {
					items = append(items, m)
				}
			}
			return printAutoTable(cmd.OutOrStdout(), items)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "numero massimo di risultati")
	cmd.Flags().StringVar(&dbPath, "db", "", "Percorso del database (default: ~/.local/share/anac-pl-pp-cli/data.db)")
	return cmd
}

// printLocalSearchOutput emette l'uscita normale di search-local quando non ci
// sono righe, prima che il comando esca con codice 3. Serve a non cambiare cio'
// che legge chi consuma stdout: il codice di uscita e' l'unica novita'.
func printLocalSearchOutput(cmd *cobra.Command, rows []json.RawMessage, flags *rootFlags) {
	if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		_ = printJSONFiltered(cmd.OutOrStdout(), rows, flags)
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), "nessun risultato locale (hai eseguito 'sync'?)")
}
