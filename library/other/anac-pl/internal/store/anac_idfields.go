package store

// ANAC primary-key override for the store layer. UpsertBatch/ExtractResourceID
// consult this map before the generic id-field fallbacks; the avvisi resource
// is keyed by `idAvviso` (UUID), which the generic list does not include.
// Standalone file so it survives generate --force regeneration.
func init() {
	resourceIDFieldOverrides["avvisi"] = "idAvviso"
}
