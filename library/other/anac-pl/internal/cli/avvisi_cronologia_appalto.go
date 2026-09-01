// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/client"
)

// resolveIdAppalto ricava l'idAppalto di un avviso dal suo dettaglio.
//
// Da agosto 2026 il servizio risponde 404 a /avvisi/{id}/cronologia se manca
// il parametro idAppalto: a giugno era facoltativo e serviva solo a
// disambiguare. Verificato il 01/09/2026 su dieci avvisi presi dalla ricerca,
// tutti 404 senza il parametro e 200 con. Chiedere all'utente un secondo UUID
// che il servizio conosce gia' non ha senso, quindi lo si risolve qui.
func resolveIdAppalto(ctx context.Context, c *client.Client, idAvviso string) (string, error) {
	raw, err := c.Get(ctx, "/avvisi/"+idAvviso, nil)
	if err != nil {
		return "", err
	}
	var detail struct {
		IdAppalto string `json:"idAppalto"`
	}
	if err := json.Unmarshal(raw, &detail); err != nil {
		return "", fmt.Errorf("lettura del dettaglio dell'avviso: %w", err)
	}
	if detail.IdAppalto == "" {
		return "", fmt.Errorf("l'avviso %s non dichiara un idAppalto: la cronologia non e' interrogabile", idAvviso)
	}
	return detail.IdAppalto, nil
}
