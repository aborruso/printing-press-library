// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: live rate-limit awareness. Surfaces the X-RateLimit-* headers
// Pixabay returns but no other wrapper exposes, persists them, and projects the
// cost of a planned batch. Hand-authored; survives `generate --force`.
//
// pp:data-source live

package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/config"
)

type quotaReport struct {
	Limit         int    `json:"limit"`
	Remaining     int    `json:"remaining"`
	ResetSeconds  int    `json:"reset_seconds"`
	CheckedAt     string `json:"checked_at"`
	PlanRequests  int    `json:"plan_requests,omitempty"`
	AfterPlan     *int   `json:"remaining_after_plan,omitempty"`
	WouldThrottle bool   `json:"would_throttle,omitempty"`
	Note          string `json:"note,omitempty"`
}

func newNovelQuotaCmd(flags *rootFlags) *cobra.Command {
	var planRequests int
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "Show remaining Pixabay rate-limit budget and project a planned batch",
		Long: strings.TrimSpace(`
Make one lightweight request and read the X-RateLimit-Limit/Remaining/Reset
headers Pixabay returns on every response (which no other Pixabay tool exposes).
Use --plan N to project whether a batch of N requests would throttle. Run this
before a large 'pull' so you can pace within the rate limit.`),
		Example:     "  pixabay-pp-cli quota --plan 200 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if cliutil.IsVerifyEnv() {
				rep := quotaReport{Note: "verify mode: no live request made"}
				return printJSONFiltered(cmd.OutOrStdout(), rep, flags)
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			key := strings.TrimSpace(cfg.PixabayApiKey)
			if key == "" {
				key = strings.TrimSpace(cfg.AuthHeader())
			}
			if key == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "no Pixabay API key configured; set PIXABAY_API_KEY or run 'pixabay-pp-cli auth set-token <key>'")
				return printJSONFiltered(cmd.OutOrStdout(), quotaReport{Note: "no API key configured"}, flags)
			}
			base := strings.TrimRight(cfg.BaseURL, "/")
			if base == "" {
				base = "https://pixabay.com"
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			u := base + "/api/"
			qv := url.Values{}
			qv.Set("key", key)
			qv.Set("per_page", "3")
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+qv.Encode(), nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				// Transport errors are *url.Error and embed the full request
				// URL, which carries key=<secret> in the query string. Scrub
				// the key before it reaches classifyAPIError (which does not
				// mask credentials) so it cannot leak to stderr or logs.
				return classifyAPIError(scrubSecret(err, key), flags)
			}
			defer resp.Body.Close()

			// Non-2xx responses (401/403/429/5xx) carry no rate-limit headers;
			// don't fall through to a misleading -1/-1 report — surface the
			// status as an actionable error so a bad key reads as an auth
			// failure, not "no headers".
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return classifyAPIError(fmt.Errorf("Pixabay returned HTTP %d (check that PIXABAY_API_KEY is valid)", resp.StatusCode), flags)
			}

			rep := quotaReport{
				Limit:        headerInt(resp.Header, "X-RateLimit-Limit", -1),
				Remaining:    headerInt(resp.Header, "X-RateLimit-Remaining", -1),
				ResetSeconds: headerInt(resp.Header, "X-RateLimit-Reset", -1),
				CheckedAt:    time.Now().UTC().Format(time.RFC3339),
			}
			if rep.Limit < 0 && rep.Remaining < 0 {
				rep.Note = fmt.Sprintf("server returned HTTP %d without rate-limit headers", resp.StatusCode)
			}
			if planRequests > 0 && rep.Remaining >= 0 {
				after := rep.Remaining - planRequests
				rep.PlanRequests = planRequests
				rep.AfterPlan = &after
				rep.WouldThrottle = after < 0
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), rep, flags)
			}
			if rep.Note != "" {
				fmt.Fprintln(cmd.OutOrStdout(), rep.Note)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rate limit: %s remaining of %s (resets in %ds)\n",
				numOrDash(rep.Remaining), numOrDash(rep.Limit), rep.ResetSeconds)
			if rep.AfterPlan != nil {
				verdict := "OK"
				if rep.WouldThrottle {
					verdict = "WOULD THROTTLE"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Plan of %d request(s) -> %d remaining after [%s]\n", rep.PlanRequests, *rep.AfterPlan, verdict)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&planRequests, "plan", 0, "Project remaining budget after a batch of this many requests")
	return cmd
}

// scrubSecret rewrites an error message so the API key cannot leak (transport
// errors embed the full request URL, key included).
func scrubSecret(err error, secret string) error {
	if err == nil || secret == "" {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), secret, "***"))
}

func headerInt(h http.Header, key string, fallback int) int {
	v := strings.TrimSpace(h.Get(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func numOrDash(n int) string {
	if n < 0 {
		return "?"
	}
	return strconv.Itoa(n)
}
