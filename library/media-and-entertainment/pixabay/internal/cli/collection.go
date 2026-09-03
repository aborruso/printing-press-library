// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Novel command: named offline collections of synced hits. Hand-authored;
// survives `generate --force` as a whole unit.
//
// pp:data-source local

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/pixabay/internal/store"
)

func newNovelCollectionCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collection",
		Short: "Group synced hits into named local collections that feed pull, similar, and trends",
		Long: strings.TrimSpace(`
Manage named local collections of Pixabay hit IDs. Collections are pure local
state that downstream commands consume: 'pull --from-collection', 'similar', and
'trends' all read collection members. Add IDs you find via 'images search' or
'media search', then materialize or analyze them later without re-querying.`),
		Annotations: map[string]string{"mcp:read-only": "false"},
	}
	cmd.AddCommand(newCollectionAddCmd(flags))
	cmd.AddCommand(newCollectionListCmd(flags))
	cmd.AddCommand(newCollectionShowCmd(flags))
	cmd.AddCommand(newCollectionRemoveCmd(flags))
	return cmd
}

func collectionKind(k string) (string, error) {
	switch strings.ToLower(k) {
	case "", "images", "image":
		return "images", nil
	case "videos", "video":
		return "videos", nil
	default:
		return "", usageErr(fmt.Errorf("--kind must be images or videos, got %q", k))
	}
}

func newCollectionAddCmd(flags *rootFlags) *cobra.Command {
	var kind, dbPath string
	cmd := &cobra.Command{
		Use:   "add <name> <id> [id...]",
		Short: "Add one or more hit IDs to a named collection",
		Example: strings.Trim(`
  pixabay-pp-cli collection add winter 195893,1850181
  pixabay-pp-cli collection add nature 1234 5678 --kind videos`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 2 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("collection add requires a name and at least one ID"))
			}
			k, err := collectionKind(kind)
			if err != nil {
				return err
			}
			name := args[0]
			ids := splitIDs(args[1:])
			if len(ids) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("no valid IDs provided"))
			}
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			added := 0
			for _, id := range ids {
				res, err := db.DB().ExecContext(cmd.Context(),
					`INSERT OR IGNORE INTO pp_collections (name, kind, item_id) VALUES (?, ?, ?)`,
					name, k, id)
				if err != nil {
					return fmt.Errorf("adding to collection: %w", err)
				}
				if n, _ := res.RowsAffected(); n > 0 {
					added++
				}
			}
			out := map[string]any{"collection": name, "kind": k, "added": added, "ids": ids}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %d ID(s) to collection %q (%s)\n", added, name, k)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "images", "Media kind: images or videos")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newCollectionListCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List named collections with member counts",
		Example:     "  pixabay-pp-cli collection list --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				return emptyCollectionResult(cmd, flags)
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			rows, err := db.DB().QueryContext(cmd.Context(),
				`SELECT name, kind, COUNT(*) FROM pp_collections GROUP BY name, kind ORDER BY name, kind`)
			if err != nil {
				return fmt.Errorf("listing collections: %w", err)
			}
			defer rows.Close()
			type collRow struct {
				Name  string `json:"collection"`
				Kind  string `json:"kind"`
				Count int    `json:"count"`
			}
			result := make([]collRow, 0)
			for rows.Next() {
				var r collRow
				if err := rows.Scan(&r.Name, &r.Kind, &r.Count); err != nil {
					continue
				}
				result = append(result, r)
			}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			if len(result) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No collections yet. Add one with: pixabay-pp-cli collection add <name> <id>")
				return nil
			}
			for _, r := range result {
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-7s %d item(s)\n", r.Name, r.Kind, r.Count)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newCollectionShowCmd(flags *rootFlags) *cobra.Command {
	var kind, dbPath string
	cmd := &cobra.Command{
		Use:         "show <name>",
		Short:       "Show the member IDs of a collection",
		Example:     "  pixabay-pp-cli collection show winter --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("collection show requires a collection name"))
			}
			name := args[0]
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				return emptyCollectionResult(cmd, flags)
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			members, err := collectionMembers(cmd, db, name, kind)
			if err != nil {
				return err
			}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), members, flags)
			}
			if len(members) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Collection %q is empty or does not exist.\n", name)
				return nil
			}
			for _, m := range members {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", m["kind"], m["id"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by media kind: images or videos")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newCollectionRemoveCmd(flags *rootFlags) *cobra.Command {
	var kind, dbPath string
	var all bool
	cmd := &cobra.Command{
		Use:         "remove <name> [id...]",
		Short:       "Remove IDs from a collection, or the whole collection with --all",
		Example:     "  pixabay-pp-cli collection remove winter 195893\n  pixabay-pp-cli collection remove winter --all",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("collection remove requires a collection name"))
			}
			name := args[0]
			ids := splitIDs(args[1:])
			if !all && len(ids) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("provide IDs to remove, or --all to delete the whole collection"))
			}
			if dbPath == "" {
				dbPath = defaultDBPath(pixabayCLIName)
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			_ = kind
			var removed int64
			if all {
				res, err := db.DB().ExecContext(cmd.Context(), `DELETE FROM pp_collections WHERE name = ?`, name)
				if err != nil {
					return fmt.Errorf("removing collection: %w", err)
				}
				removed, _ = res.RowsAffected()
			} else {
				for _, id := range ids {
					res, err := db.DB().ExecContext(cmd.Context(),
						`DELETE FROM pp_collections WHERE name = ? AND item_id = ?`, name, id)
					if err != nil {
						return fmt.Errorf("removing from collection: %w", err)
					}
					n, _ := res.RowsAffected()
					removed += n
				}
			}
			out := map[string]any{"collection": name, "removed": removed}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %d item(s) from collection %q\n", removed, name)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by media kind")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().BoolVar(&all, "all", false, "Remove the entire collection")
	return cmd
}

// collectionMembers returns [{id, kind}] for a collection, optionally filtered
// by kind.
func collectionMembers(cmd *cobra.Command, db *store.Store, name, kind string) ([]map[string]string, error) {
	q := `SELECT item_id, kind FROM pp_collections WHERE name = ?`
	args := []any{name}
	if kind != "" {
		k, err := collectionKind(kind)
		if err != nil {
			return nil, err
		}
		q += ` AND kind = ?`
		args = append(args, k)
	}
	q += ` ORDER BY added_at`
	rows, err := db.DB().QueryContext(cmd.Context(), q, args...)
	if err != nil {
		if isNoSuchTableErr(err) {
			// A read-only open of a store that was never write-opened has no
			// pp_collections table yet — treat as an empty collection set.
			return []map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading collection: %w", err)
	}
	defer rows.Close()
	members := make([]map[string]string, 0)
	for rows.Next() {
		var id, k string
		if err := rows.Scan(&id, &k); err != nil {
			continue
		}
		members = append(members, map[string]string{"id": id, "kind": k})
	}
	return members, nil
}

func emptyCollectionResult(cmd *cobra.Command, flags *rootFlags) error {
	if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
		fmt.Fprintln(cmd.OutOrStdout(), "[]")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "No collections yet. Add one with: pixabay-pp-cli collection add <name> <id>")
	return nil
}

// splitIDs flattens positional args that may be comma-separated lists, deduped.
func splitIDs(args []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, a := range args {
		for _, part := range strings.Split(a, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			out = append(out, part)
		}
	}
	return out
}
