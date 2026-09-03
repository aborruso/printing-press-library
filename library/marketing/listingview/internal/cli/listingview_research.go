// Hand-authored, not generated. Shared helpers for ListingView novel commands:
// proxy-endpoint calls + envelope unwrapping, safe field extraction, and a
// local snapshot store that gives drift/opportunities the change-over-time
// history the ListingView API itself does not expose (it returns only
// point-in-time estimates).
package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/client"
	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/store"
)

const lvProxyPrefix = "/api/proxy/api/integration/etsy/"

// lvEnvelope is the consistent ListingView response wrapper.
type lvEnvelope struct {
	StatusCode int             `json:"statusCode"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

// callProxyPOST POSTs body to a ListingView proxy operation and returns the
// unwrapped `data` object as a raw-message map.
func callProxyPOST(ctx context.Context, c *client.Client, op string, body map[string]any) (map[string]json.RawMessage, error) {
	raw, _, err := c.Post(ctx, lvProxyPrefix+op, body)
	if err != nil {
		return nil, err
	}
	return unwrapData(raw)
}

// callProxyGET GETs a proxy operation with query params and returns the
// unwrapped `data` object.
func callProxyGET(ctx context.Context, c *client.Client, op string, params map[string]string) (map[string]json.RawMessage, error) {
	raw, err := c.Get(ctx, lvProxyPrefix+op, params)
	if err != nil {
		return nil, err
	}
	return unwrapData(raw)
}

func unwrapData(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var env lvEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Data) > 0 {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(env.Data, &m); err == nil {
			return m, nil
		}
	}
	// Fall back to treating the whole body as the data object.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return m, nil
}

// listOf extracts an array of objects from a data field (e.g. "listings",
// "keywords", "tags").
func listOf(data map[string]json.RawMessage, key string) []map[string]json.RawMessage {
	raw, ok := data[key]
	if !ok {
		return nil
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	out := make([]map[string]json.RawMessage, 0, len(arr))
	for _, item := range arr {
		var m map[string]json.RawMessage
		if json.Unmarshal(item, &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

// stringsOf extracts a []string from a data field. Handles arrays of plain
// strings and arrays of objects carrying a "tag"/"name"/"value" field.
func stringsOf(data map[string]json.RawMessage, key string) []string {
	raw, ok := data[key]
	if !ok {
		return nil
	}
	var plain []string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}
	// Fall back to objects carrying a tag/name/value field. Use a fresh slice:
	// a failed []string unmarshal above can leave `plain` partially populated.
	var out []string
	for _, item := range listOf(data, key) {
		for _, f := range []string{"tag", "name", "value", "title"} {
			if s := strOf(item, f); s != "" {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// numOf returns a numeric field as float64 (0 when missing), tolerating values
// encoded as JSON strings.
func numOf(m map[string]json.RawMessage, key string) float64 {
	if v, ok := cliutil.ExtractNumber(m, key); ok {
		return v
	}
	return 0
}

// strOf returns a string field ("" when missing).
func strOf(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

// --- Local snapshot store (lazy table; survives regen as a hand-authored file) ---

func openLVStore(dbPath string) (*store.Store, error) {
	if dbPath == "" {
		dbPath = defaultDBPath("listingview-pp-cli")
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening local store: %w", err)
	}
	if err := ensureSnapshotTable(s.DB()); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// listingViewSyncRequiredParams returns query params a resource's list endpoint
// requires but the generic syncer does not supply. watchlist/list-favourite
// rejects requests without a `type` discriminator; default to syncing the
// listing watchlist.
func listingViewSyncRequiredParams(resource string) map[string]string {
	switch resource {
	case "watchlist":
		return map[string]string{"type": "listing"}
	}
	return nil
}

// tryOpenStore opens the local snapshot store, returning nil on any error.
// Snapshotting is best-effort and must never block a live command.
func tryOpenStore(dbPath string) *store.Store {
	s, err := openLVStore(dbPath)
	if err != nil {
		return nil
	}
	return s
}

func ensureSnapshotTable(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS lv_snapshots (
		kind TEXT NOT NULL,
		key TEXT NOT NULL,
		fetched_at INTEGER NOT NULL,
		metrics TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("creating lv_snapshots: %w", err)
	}
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_lv_snap ON lv_snapshots(kind, key, fetched_at)`)
	return nil
}

// saveSnapshot records a metrics snapshot for a (kind,key) at the current time.
// Best-effort: snapshot failures never break the user-facing command.
func saveSnapshot(db *sql.DB, kind, key string, metrics map[string]float64) {
	if db == nil || key == "" || len(metrics) == 0 {
		return
	}
	b, err := json.Marshal(metrics)
	if err != nil {
		return
	}
	_, _ = db.Exec(`INSERT INTO lv_snapshots(kind, key, fetched_at, metrics) VALUES(?,?,?,?)`,
		kind, key, time.Now().Unix(), string(b))
}

type snapRecord struct {
	Kind      string             `json:"kind"`
	Key       string             `json:"key"`
	FetchedAt int64              `json:"fetched_at"`
	Metrics   map[string]float64 `json:"metrics"`
}

// latestPerKey returns the most recent snapshot for every (kind,key).
func latestPerKey(db *sql.DB) ([]snapRecord, error) {
	rows, err := db.Query(`SELECT s.kind, s.key, s.fetched_at, s.metrics
		FROM lv_snapshots s
		JOIN (SELECT kind, key, MAX(fetched_at) AS mx FROM lv_snapshots GROUP BY kind, key) m
		ON s.kind=m.kind AND s.key=m.key AND s.fetched_at=m.mx`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	all, err := scanSnaps(rows)
	if err != nil {
		return nil, err
	}
	// Same-second snapshots tie on MAX(fetched_at); keep one row per (kind,key).
	seen := map[string]bool{}
	out := all[:0]
	for _, r := range all {
		k := r.Kind + "\x00" + r.Key
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, r)
	}
	return out, nil
}

// historyForKey returns all snapshots for a (kind,key) ordered newest-first.
func historyForKey(db *sql.DB, kind, key string) ([]snapRecord, error) {
	rows, err := db.Query(`SELECT kind, key, fetched_at, metrics FROM lv_snapshots
		WHERE kind=? AND key=? ORDER BY fetched_at DESC`, kind, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSnaps(rows)
}

// distinctKeys returns the (kind,key) pairs that have at least minCount snapshots.
func distinctKeys(db *sql.DB, minCount int) ([]snapRecord, error) {
	rows, err := db.Query(`SELECT kind, key, COUNT(*) FROM lv_snapshots GROUP BY kind, key HAVING COUNT(*) >= ?`, minCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []snapRecord
	for rows.Next() {
		var r snapRecord
		var n int
		if err := rows.Scan(&r.Kind, &r.Key, &n); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanSnaps(rows *sql.Rows) ([]snapRecord, error) {
	var out []snapRecord
	for rows.Next() {
		var r snapRecord
		var metrics sql.NullString
		if err := rows.Scan(&r.Kind, &r.Key, &r.FetchedAt, &metrics); err != nil {
			continue
		}
		r.Metrics = map[string]float64{}
		if metrics.Valid {
			_ = json.Unmarshal([]byte(metrics.String), &r.Metrics)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
