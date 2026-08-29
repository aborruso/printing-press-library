package icaroclient

import (
	"fmt"
	"strings"
	"time"
)

// I parametri che possono portare un range di date AAMMGG/AAMMGG. `anno` c'è
// perché su ddl non esiste un campo anno: --anno è compilato in un range DATPRE
// 1 gen - 31 dic (vedi normalizeParams), quindi non è l'alternativa sicura a
// --data che sembra — è la stessa cosa, con estremi fissi.
var chiaviRange = []string{"data", "anno"}

// chiaveRange trova il parametro che porta un range di date spezzabile.
// Ritorna la chiave e i due estremi.
func chiaveRange(params map[string]string) (chiave, lo, hi string, ok bool) {
	for _, k := range chiaviRange {
		v := strings.TrimSpace(params[k])
		a, b, isRange := strings.Cut(v, "/")
		if !isRange || !isAAMMGGRange(a, b) {
			continue
		}
		return k, a, b, true
	}
	return "", "", "", false
}

func isAAMMGGRange(a, b string) bool {
	return len(a) == 6 && len(b) == 6 && soloCifre(a) && soloCifre(b) && a < b
}

func soloCifre(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Il secolo è già perduto a monte: `aammgg` in normalizeParams tiene solo le
// ultime due cifre dell'anno, e il portale indicizza così. Qui serve solo per
// fare aritmetica sui giorni e viene riassorbito in uscita, quindi la scelta
// del 2000 non aggiunge un'assunzione che non ci fosse già.
const secoloAAMMGG = 2000

func daAAMMGG(s string) (time.Time, error) {
	t, err := time.Parse("060102", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("data AAMMGG non valida %q: %w", s, err)
	}
	return t, nil
}

func aAAMMGG(t time.Time) string { return t.Format("060102") }

// spezzaPerAnno taglia il range sui confini di anno solare. Ritorna le fette
// dalla più recente alla più vecchia: le fette sono blocchi cronologici, quindi
// concatenarle un ordine glielo impone comunque, e questo è quello che gli altri
// comandi della CLI già danno (il più recente prima).
//
// Ritorna nil se il range sta dentro un anno solo: lì per tagliare serve
// spezzaAMeta.
func spezzaPerAnno(lo, hi string) []string {
	a, err := daAAMMGG(lo)
	if err != nil {
		return nil
	}
	b, err := daAAMMGG(hi)
	if err != nil || !a.Before(b) || a.Year() == b.Year() {
		return nil
	}
	var fette []string
	for anno := a.Year(); anno <= b.Year(); anno++ {
		inizio := time.Date(anno, 1, 1, 0, 0, 0, 0, time.UTC)
		fine := time.Date(anno, 12, 31, 0, 0, 0, 0, time.UTC)
		if inizio.Before(a) {
			inizio = a
		}
		if fine.After(b) {
			fine = b
		}
		fette = append(fette, aAAMMGG(inizio)+"/"+aAAMMGG(fine))
	}
	return inverti(fette)
}

// spezzaAMeta taglia il range in due sul giorno di mezzo. Serve quando una fetta
// annuale cede ancora: un anno solare non è una garanzia, il motore cede sul
// NUMERO di documenti e la densità cambia da archivio ad archivio.
//
// Ritorna nil sui range di un giorno solo, dove non c'è più niente da tagliare.
func spezzaAMeta(lo, hi string) []string {
	a, err := daAAMMGG(lo)
	if err != nil {
		return nil
	}
	b, err := daAAMMGG(hi)
	if err != nil || !a.Before(b) {
		return nil
	}
	mezzo := a.Add(b.Sub(a) / 2)
	if !mezzo.Before(b) || mezzo.Before(a) {
		return nil
	}
	return []string{
		aAAMMGG(mezzo.AddDate(0, 0, 1)) + "/" + aAAMMGG(b),
		aAAMMGG(a) + "/" + aAAMMGG(mezzo),
	}
}

func inverti(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}
