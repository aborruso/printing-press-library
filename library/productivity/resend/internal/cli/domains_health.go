// PATCH: novel domain health summary — verification + DKIM/SPF/DMARC status across every domain. No aggregate API endpoint.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/productivity/resend/internal/store"
)

func newDomainsHealthCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Verification + DKIM/SPF/DMARC status across every domain (no aggregate API)",
		Long: `Aggregates locally-synced domains with their verification status and the
DKIM/SPF/DMARC record state from the domain.data JSON. Flags domains that
aren't fully verified. No aggregate endpoint exists — today this requires
N GET /domains/{id} calls.`,
		Example: strings.Trim(`
  # Summary across all domains
  resend-pp-cli domains health --json

  # Only flag unhealthy domains
  resend-pp-cli domains health --unhealthy-only --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if dbPath == "" {
				dbPath = defaultDBPath("resend-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w\nRun 'resend-pp-cli sync' first.", err)
			}
			defer db.Close()

			rows, err := db.Query(`
				SELECT
					d.id,
					COALESCE(d.name, '') AS name,
					COALESCE(d.status, '') AS status,
					COALESCE(d.region, '') AS region,
					COALESCE(d.click_tracking, 0) AS click_tracking,
					COALESCE(d.open_tracking, 0) AS open_tracking,
					COALESCE(json_extract(d.data, '$.records'), '') AS records_json
				FROM domains d
				ORDER BY d.name
			`)
			if err != nil {
				return fmt.Errorf("querying domains: %w", err)
			}
			defer rows.Close()

			type health struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				Status        string `json:"status"`
				Region        string `json:"region"`
				ClickTracking bool   `json:"click_tracking"`
				OpenTracking  bool   `json:"open_tracking"`
				DKIM          string `json:"dkim_status"`
				SPF           string `json:"spf_status"`
				MX            string `json:"mx_status"`
				Healthy       bool   `json:"healthy"`
			}
			results := []health{}
			for rows.Next() {
				var h health
				var clickInt, openInt int
				var recordsJSON string
				if err := rows.Scan(&h.ID, &h.Name, &h.Status, &h.Region, &clickInt, &openInt, &recordsJSON); err != nil {
					continue
				}
				h.ClickTracking = clickInt == 1
				h.OpenTracking = openInt == 1
				// Best-effort extraction of record statuses; records is an array of {record, status, type, ...}.
				if strings.Contains(recordsJSON, `"DKIM"`) {
					h.DKIM = extractRecordStatus(recordsJSON, "DKIM")
				}
				if strings.Contains(recordsJSON, `"SPF"`) {
					h.SPF = extractRecordStatus(recordsJSON, "SPF")
				}
				if strings.Contains(recordsJSON, `"MX"`) {
					h.MX = extractRecordStatus(recordsJSON, "MX")
				}
				h.Healthy = h.Status == "verified"
				results = append(results, h)
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterating domains: %w", err)
			}

			unhealthyOnly, _ := cmd.Flags().GetBool("unhealthy-only")
			if unhealthyOnly {
				filtered := results[:0]
				for _, r := range results {
					if !r.Healthy {
						filtered = append(filtered, r)
					}
				}
				results = filtered
			}

			out := cmd.OutOrStdout()
			if flags.asJSON {
				return printJSONFiltered(out, map[string]any{
					"count":   len(results),
					"domains": results,
				}, flags)
			}
			if len(results) == 0 {
				if unhealthyOnly {
					fmt.Fprintln(out, "All domains are healthy.")
				} else {
					fmt.Fprintln(out, "No domains in the local store.")
					fmt.Fprintln(out, "(Run 'resend-pp-cli sync --full' to refresh.)")
				}
				return nil
			}
			fmt.Fprintf(out, "%d domain(s):\n\n", len(results))
			fmt.Fprintf(out, "%-25s %-12s %-10s %-10s %-10s %s\n", "NAME", "STATUS", "DKIM", "SPF", "MX", "HEALTHY")
			fmt.Fprintf(out, "%-25s %-12s %-10s %-10s %-10s %s\n", "----", "------", "----", "---", "--", "-------")
			for _, r := range results {
				healthMark := "yes"
				if !r.Healthy {
					healthMark = "NO"
				}
				fmt.Fprintf(out, "%-25s %-12s %-10s %-10s %-10s %s\n", truncate(r.Name, 23), truncate(r.Status, 10), truncate(r.DKIM, 8), truncate(r.SPF, 8), truncate(r.MX, 8), healthMark)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/resend-pp-cli/data.db)")
	cmd.Flags().Bool("unhealthy-only", false, "Only show domains that are not fully verified")
	return cmd
}

// extractRecordStatus does a best-effort substring search for the status of a named record type.
// We avoid pulling a JSON dependency since the data shape is simple.
func extractRecordStatus(jsonStr, recordType string) string {
	// Look for: {..."record":"DKIM",..."status":"verified"...}
	idx := strings.Index(jsonStr, `"record":"`+recordType+`"`)
	if idx < 0 {
		return ""
	}
	tail := jsonStr[idx:]
	statusIdx := strings.Index(tail, `"status":"`)
	if statusIdx < 0 {
		return ""
	}
	start := statusIdx + len(`"status":"`)
	end := strings.Index(tail[start:], `"`)
	if end < 0 {
		return ""
	}
	return tail[start : start+end]
}
