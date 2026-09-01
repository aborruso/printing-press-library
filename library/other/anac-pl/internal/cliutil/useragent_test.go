// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"strings"
	"testing"
)

func TestUserAgent(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	tests := []struct {
		name      string
		version   string
		component string
		want      string
	}{
		{
			name:      "versione iniettata al build",
			version:   "2026.9.5",
			component: "anac-pl-pp-cli",
			want:      "anac-pl-pp-cli/2026.9.5 (+" + RepoURL + ")",
		},
		{
			name:      "il server MCP si nomina da se'",
			version:   "2026.9.5",
			component: "anac-pl-pp-mcp",
			want:      "anac-pl-pp-mcp/2026.9.5 (+" + RepoURL + ")",
		},
		{
			name:      "senza componente si ripiega sulla CLI",
			version:   "2026.9.5",
			component: "",
			want:      "anac-pl-pp-cli/2026.9.5 (+" + RepoURL + ")",
		},
		{
			name:      "senza ldflags la versione non resta vuota",
			version:   "",
			component: "anac-pl-pp-cli",
			want:      "anac-pl-pp-cli/dev (+" + RepoURL + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			if got := UserAgent(tt.component); got != tt.want {
				t.Errorf("UserAgent(%q) = %q, want %q", tt.component, got, tt.want)
			}
		})
	}
}

// L'intestazione deve sempre portare un contatto: e' l'unica via che ha chi
// amministra il servizio per scrivere invece di bloccare.
func TestUserAgentPortaSempreIlRepository(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	for _, v := range []string{"", "dev", "2026.9.5"} {
		Version = v
		if ua := UserAgent(""); !strings.Contains(ua, RepoURL) {
			t.Errorf("UserAgent con Version=%q non dichiara il repository: %q", v, ua)
		}
	}
}
