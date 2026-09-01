package cli

// ANAC-specific primary-key overrides. The Pubblicità a Valore Legale API
// keys avvisi by `idAvviso` (UUID), not the generic `id` the runtime fallback
// looks for. Without this override sync/search-local cannot cache rows, which
// breaks the offline store and export features. Kept in a standalone file so it
// survives `generate --force` regeneration.
func init() {
	resourceIDFieldOverrides["avvisi"] = "idAvviso"
}
