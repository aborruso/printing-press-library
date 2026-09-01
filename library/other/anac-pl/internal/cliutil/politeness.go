// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MaxRequestsPerSecond e' il tetto invalicabile di chiamate al servizio ANAC.
// Non e' una preferenza dell'utente: e' una proprieta' del programma. La
// piattaforma di pubblicita' legale e' un servizio pubblico senza quota
// dichiarata, e una CLI che puo' saturarlo e' una CLI che prima o poi lo
// satura. Il tetto vale per l'intero processo, e il lock di istanza singola
// (vedi AcquireSingleInstance) fa si' che il processo sia uno solo.
const MaxRequestsPerSecond = 1.0

// minInterval e' la distanza minima fra due richieste consecutive.
const minInterval = time.Duration(float64(time.Second) / MaxRequestsPerSecond)

var pacer struct {
	mu   sync.Mutex
	last time.Time
	file *os.File
}

// Pace blocca finche' non e' passato almeno minInterval dall'ultima richiesta.
// Va chiamata immediatamente prima di ogni chiamata HTTP al servizio.
//
// Il ritmo e' condiviso fra processi: l'ultimo istante di chiamata sta in un
// file in ~/.cache, protetto da un lock esclusivo di sistema tenuto per tutta
// la durata dell'attesa. Cosi' il tetto vale anche quando la CLI e il server
// MCP lavorano nello stesso momento, che e' l'unico modo in cui questo
// programma puo' trovarsi a girare due volte. Il mutex interno serializza a
// sua volta i fan-out concorrenti dentro il singolo processo.
//
// Se il file non e' accessibile (home non determinabile, filesystem in sola
// lettura) si ripiega sul ritmo del solo processo: peggio del previsto, mai
// piu' veloce del previsto.
func Pace() {
	pacer.mu.Lock()
	defer pacer.mu.Unlock()

	f := paceFile()
	if f == nil {
		paceInProcess()
		return
	}
	if err := lockExclusiveBlocking(f); err != nil {
		paceInProcess()
		return
	}
	defer unlockFile(f)

	last := readPaceStamp(f)
	if !last.IsZero() {
		if wait := minInterval - time.Since(last); wait > 0 {
			time.Sleep(wait)
		}
	}
	now := time.Now()
	writePaceStamp(f, now)
	pacer.last = now
}

// paceInProcess e' il ripiego quando il file condiviso non e' utilizzabile.
func paceInProcess() {
	if !pacer.last.IsZero() {
		if wait := minInterval - time.Since(pacer.last); wait > 0 {
			time.Sleep(wait)
		}
	}
	pacer.last = time.Now()
}

// paceFile apre (una volta sola) il file che porta l'istante dell'ultima
// chiamata. Restituisce nil se non e' apribile.
func paceFile() *os.File {
	if pacer.file != nil {
		return pacer.file
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".cache", "anac-pl-pp-cli", "pace.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil
	}
	pacer.file = f
	return f
}

func readPaceStamp(f *os.File) time.Time {
	buf := make([]byte, 32)
	n, err := f.ReadAt(buf, 0)
	if n == 0 || (err != nil && err != io.EOF) {
		return time.Time{}
	}
	nanos, err := strconv.ParseInt(strings.TrimSpace(string(buf[:n])), 10, 64)
	if err != nil || nanos <= 0 {
		return time.Time{}
	}
	stamp := time.Unix(0, nanos)
	// Un orologio spostato all'indietro renderebbe l'attesa infinita:
	// un timestamp nel futuro si tratta come assente.
	if stamp.After(time.Now()) {
		return time.Time{}
	}
	return stamp
}

func writePaceStamp(f *os.File, t time.Time) {
	if err := f.Truncate(0); err != nil {
		return
	}
	if _, err := f.WriteAt([]byte(strconv.FormatInt(t.UnixNano(), 10)), 0); err != nil {
		return
	}
	_ = f.Sync()
}

// ClampRate riporta dentro il tetto qualsiasi valore chiesto da riga di comando.
// Un valore assente o non valido diventa il tetto stesso: il rate limiting non
// si puo' disattivare.
func ClampRate(requested float64) float64 {
	if requested <= 0 || requested > MaxRequestsPerSecond {
		return MaxRequestsPerSecond
	}
	return requested
}

var (
	instanceMu   sync.Mutex
	instanceFile *os.File
)

// ErrInstanceRunning segnala che un'altra istanza sta gia' parlando con il
// servizio.
type ErrInstanceRunning struct {
	Path string
}

func (e *ErrInstanceRunning) Error() string {
	return fmt.Sprintf("un'altra istanza di anac-pl-pp-cli sta gia' interrogando il servizio (lock: %s).\n"+
		"Per non superare %g chiamata/s verso ANAC ne gira una sola per volta: attendi che finisca, oppure interrompila.", e.Path, MaxRequestsPerSecond)
}

// AcquireSingleInstance prende un lock esclusivo di sistema, valido fra
// processi diversi, e lo tiene fino all'uscita del programma. Se il lock e'
// gia' preso restituisce *ErrInstanceRunning senza attendere: due istanze in
// parallelo raddoppierebbero il carico sul servizio, e il tetto per processo
// non basterebbe piu' a garantirlo.
//
// Chiamate ripetute nello stesso processo sono un no-op: il lock e' del
// processo, non del comando.
func AcquireSingleInstance() error {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if instanceFile != nil {
		return nil
	}
	path, err := instanceLockPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creazione della cartella del lock: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("apertura del lock di istanza: %w", err)
	}
	if err := lockExclusiveNonBlocking(f); err != nil {
		f.Close()
		if isLockBusy(err) {
			return &ErrInstanceRunning{Path: path}
		}
		return fmt.Errorf("lock di istanza su %s: %w", path, err)
	}
	// Il lock viene rilasciato dal sistema operativo alla chiusura del
	// processo, anche in caso di kill: nessun file di stato da ripulire.
	instanceFile = f
	return nil
}

func instanceLockPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cartella home non determinabile: %w", err)
	}
	return filepath.Join(home, ".cache", "anac-pl-pp-cli", "instance.lock"), nil
}
