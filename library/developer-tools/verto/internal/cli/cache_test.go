// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNovelCacheCommand(t *testing.T) {
	cmd := newNovelCacheCmd(&rootFlags{})
	if !strings.HasPrefix(cmd.Use, "cache") {
		t.Errorf("Use = %q, want prefix 'cache'", cmd.Use)
	}
	if cmd.Flags().Lookup("clear") == nil {
		t.Error("cache must expose --clear")
	}
}

func TestCacheStatsOffline(t *testing.T) {
	// `cache --stats` touches only the local cache file, never the network.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var buf bytes.Buffer
	flags := &rootFlags{asJSON: true}
	cmd := newNovelCacheCmd(flags)
	cmd.SetOut(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cache stats: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v (%s)", err, buf.String())
	}
	if _, ok := result["entries"]; !ok {
		t.Errorf("stats output missing 'entries': %s", buf.String())
	}
}
