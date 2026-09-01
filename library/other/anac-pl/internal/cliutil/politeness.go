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
	// Waited e' quanto si e' atteso prima di arrendersi.
	Waited time.Duration
}

func (e *ErrInstanceRunning) Error() string {
	if e.Waited <= 0 {
		return fmt.Sprintf("un'altra istanza di anac-pl-pp-cli sta gia' interrogando il servizio (lock: %s).\n"+
			"Ne gira una sola per volta: attendi che finisca, oppure interrompila.", e.Path)
	}
	return fmt.Sprintf("un'altra istanza di anac-pl-pp-cli sta ancora interrogando il servizio dopo %s di attesa (lock: %s).\n"+
		"Ne gira una sola per volta: attendi che finisca, oppure interrompila.", e.Waited, e.Path)
}

// InstanceWaitTimeout e' quanto AcquireSingleInstance attende il proprio turno
// prima di arrendersi. Un sync lungo puo' tenere il lock per parecchi minuti,
// e un comando che esce subito costringerebbe a riprovare a mano; un'attesa
// senza fine, pero', sarebbe indistinguibile da un blocco.
const InstanceWaitTimeout = 5 * time.Minute

// instanceRetryInterval e' la distanza fra due tentativi di presa del lock.
const instanceRetryInterval = 200 * time.Millisecond

// InstanceWaitNotice, se valorizzata, riceve una riga da mostrare all'utente
// quando l'attesa comincia davvero. Sta qui e non in un fmt.Fprintf perche'
// cliutil non deve decidere dove scrive un comando.
var InstanceWaitNotice func(string)

// AcquireSingleInstance prende un lock esclusivo di sistema, valido fra
// processi diversi, e lo tiene fino all'uscita del programma.
//
// Se il lock e' gia' preso **attende** fino a InstanceWaitTimeout, e solo
// allora restituisce *ErrInstanceRunning. Non e' questo lock a garantire il
// tetto di chiamate al secondo — quello lo garantisce Pace(), che condivide
// l'istante dell'ultima chiamata fra processi diversi tramite pace.lock, e
// regge quindi con un numero qualunque di processi. Il lock di istanza serve
// come rete per il ramo degradato, quando il file condiviso non e'
// utilizzabile e Pace() ripiega sul ritmo del solo processo: li' due istanze
// in parallelo raddoppierebbero il carico. Rifiutare invece di attendere
// costava un errore a chi lancia un comando mentre un sync sta lavorando,
// senza togliere una sola chiamata al servizio.
//
// Chiamate ripetute nello stesso processo sono un no-op: il lock e' del
// processo, non del comando.
func AcquireSingleInstance() error {
	return AcquireSingleInstanceWithin(InstanceWaitTimeout)
}

// AcquireSingleInstanceWithin e' AcquireSingleInstance con un'attesa massima
// scelta dal chiamante. Un timeout nullo o negativo significa "non attendere":
// e' cio' che serve in modalita' non interattiva, dove un comando che si ferma
// cinque minuti e' peggio di un errore immediato.
func AcquireSingleInstanceWithin(wait time.Duration) error {
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

	deadline := time.Now().Add(wait)
	notified := false
	for {
		lockErr := lockExclusiveNonBlocking(f)
		if lockErr == nil {
			break
		}
		if !isLockBusy(lockErr) {
			f.Close()
			return fmt.Errorf("lock di istanza su %s: %w", path, lockErr)
		}
		if wait <= 0 || time.Now().After(deadline) {
			f.Close()
			return &ErrInstanceRunning{Path: path, Waited: max(wait, 0)}
		}
		if !notified && InstanceWaitNotice != nil {
			InstanceWaitNotice(fmt.Sprintf("un'altra istanza di anac-pl-pp-cli sta interrogando il servizio: attendo il mio turno (fino a %s).", wait))
			notified = true
		}
		time.Sleep(instanceRetryInterval)
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
