// Copyright 2026 Rick van de Laar and contributors. Licensed under Apache-2.0. See LICENSE.

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
		// credits_snapshots powers `credits runway`: one row per observation of
		// the account balance, appended on each runway invocation.
		`CREATE TABLE IF NOT EXISTS credits_snapshots (
			taken_at DATETIME NOT NULL PRIMARY KEY,
			total_credits REAL NOT NULL,
			total_usage REAL NOT NULL
		)`,
		// models_catalog_snapshots powers `models churn`: a point-in-time copy
		// of (model id, name, pricing) taken on each churn invocation, keyed by
		// snapshot timestamp so later runs can diff against an older baseline.
		`CREATE TABLE IF NOT EXISTS models_catalog_snapshots (
			snap_at DATETIME NOT NULL,
			model_id TEXT NOT NULL,
			name TEXT,
			prompt_price TEXT,
			completion_price TEXT,
			PRIMARY KEY (snap_at, model_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_models_catalog_snapshots_snap
			ON models_catalog_snapshots (snap_at)`,
	}
	for _, m := range migrations {
		if _, err := conn.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("extra migration failed: %w", err)
		}
	}
	return nil
}
