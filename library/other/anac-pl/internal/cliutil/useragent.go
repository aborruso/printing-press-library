// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cliutil

import "fmt"

// Version e' la versione del programma, iniettata al build con
// -X anac-pl-pp-cli/internal/cliutil.Version=<versione>. Sta qui e non in
// internal/cli perche' anche il server MCP deve poterla dichiarare, e
// internal/mcp non puo' importare internal/cli.
var Version = "dev"

// RepoURL e' il posto dove chi amministra il servizio puo' vedere cosa fa
// questo programma e a chi scrivere.
const RepoURL = "https://github.com/aborruso/anac-pl-pp-cli"

// UserAgent compone l'intestazione con cui i due binari si presentano ad ANAC.
// La piattaforma e' un servizio pubblico senza quota dichiarata e senza un
// canale per concordarne una: un client che non si nomina lascia a chi lo
// riceve una sola mossa possibile, il blocco. Con nome, versione e URL del
// repository, invece, un traffico che desse fastidio si puo' riconoscere e
// segnalare a chi lo genera.
func UserAgent(component string) string {
	if component == "" {
		component = "anac-pl-pp-cli"
	}
	version := Version
	if version == "" {
		version = "dev"
	}
	return fmt.Sprintf("%s/%s (+%s)", component, version, RepoURL)
}
