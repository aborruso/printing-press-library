// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateExtras runs after the generated store migrations and before the
// schema-version stamp. It is the canonical place for novel-feature auxiliary
// tables that need to live in the local store.
//
// Edit this file when adding tables for novel commands. Keep migrations
// idempotent with CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS so
// every store open can safely re-run them.
func (s *Store) migrateExtras(ctx context.Context, conn *sql.Conn) error {
	migrations := []string{
		// Novel-feature tables for the Pixabay CLI. Created on every
		// write-capable store open so commands that read them via a
		// read-only open (e.g. `pull --from-collection`) never hit a
		// missing-table error.
		`CREATE TABLE IF NOT EXISTS pp_collections (
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			item_id TEXT NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (name, kind, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS pp_stat_snapshots (
			kind TEXT NOT NULL,
			item_id TEXT NOT NULL,
			tags TEXT,
			views INTEGER,
			downloads INTEGER,
			likes INTEGER,
			comments INTEGER,
			snapshot_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pp_snap_item ON pp_stat_snapshots(kind, item_id, snapshot_at)`,
	}
	for _, m := range migrations {
		if _, err := conn.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("extra migration failed: %w", err)
		}
	}
	return nil
}
