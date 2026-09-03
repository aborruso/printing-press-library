// Copyright 2026 Vincent Colombo and contributors. Licensed under Apache-2.0. See LICENSE.
//
// Pixabay API Terms of Service compliance: Pixabay must be credited with a
// visible link whenever search results are displayed. This file centralizes
// that credit so every result-displaying command emits it uniformly. Wired from
// root.go's PersistentPostRunE. Hand-authored; survives `generate --force`.

package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// pixabayCreditLine is the visible attribution shown after Pixabay results. It
// names Pixabay and carries a clickable link, satisfying the ToS requirement.
const pixabayCreditLine = "— Results from Pixabay (https://pixabay.com). Per Pixabay's API Terms, credit Pixabay with a visible link wherever you display these results."

// pixabayResultCommandPaths are the command paths (binary name stripped) whose
// output displays Pixabay search results or Pixabay-derived data and therefore
// require visible attribution. Pure-local management commands (collection,
// quota, doctor, auth, profile, version) are excluded — they do not display
// Pixabay content.
var pixabayResultCommandPaths = map[string]bool{
	"images search": true,
	"images get":    true,
	"videos search": true,
	"videos get":    true,
	"media search":  true,
	"similar":       true,
	"trends":        true,
	"contributors":  true,
	"search":        true,
	"sync":          true,
}

// emitPixabayCredit prints the Pixabay attribution line to stderr for
// human-facing, result-displaying commands. It is intentionally quiet for
// machine output (--json/--agent/--quiet/--csv/--plain) and piped stdout: those
// consumers receive per-result pageURL links in the data, and a stderr footer
// must never corrupt a JSON stream. Writing to stderr keeps stdout clean while
// remaining visible in an interactive terminal.
func emitPixabayCredit(cmd *cobra.Command, flags *rootFlags) {
	if cmd == nil || flags == nil {
		return
	}
	if flags.asJSON || flags.agent || flags.quiet || flags.csv || flags.plain {
		return
	}
	if !commandDisplaysPixabayResults(cmd) {
		return
	}
	// Only credit when a human is actually looking at the output.
	if !isTerminal(cmd.OutOrStdout()) {
		return
	}
	out := cmd.ErrOrStderr()
	if !isTerminal(out) {
		out = cmd.OutOrStdout()
	}
	// Direct write — fmt via the shared helper keeps the dependency surface small.
	_, _ = out.Write([]byte(pixabayCreditLine + "\n"))
}

// commandDisplaysPixabayResults reports whether the executed command's path is
// one that shows Pixabay results.
func commandDisplaysPixabayResults(cmd *cobra.Command) bool {
	path := cmd.CommandPath() // e.g. "pixabay-pp-cli images search"
	if i := strings.IndexByte(path, ' '); i >= 0 {
		path = strings.TrimSpace(path[i+1:]) // drop the binary name
	} else {
		return false
	}
	return pixabayResultCommandPaths[path]
}
