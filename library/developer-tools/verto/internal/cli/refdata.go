// Copyright 2026 aborruso and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

// refdata is a curated static reference table for the 20 reference systems the
// IGM Verto Online service supports. It powers `inspect`, `targets`, and
// `detect` with domain knowledge the API does not expose: datum family (which
// determines valid conversion targets, since same-datum conversions are
// rejected by the service), axis kind, units, zone, and false easting.
//
// pp:novel-static-reference

// datum family identifiers. Same-family conversions are rejected by the
// service ("Le conversioni fra sistemi con lo stesso datum non sono ammesse").
const (
	familyRoma40  = "Roma40 / Monte Mario"
	familyED50    = "ED50"
	familyETRS89  = "ETRS89 / IGM95 (ETRF89)"
	familyRDN2008 = "RDN2008 (ETRF2000)"
)

// sysRef is the static metadata for one EPSG.
type sysRef struct {
	Epsg         int    `json:"epsg"`
	Name         string `json:"name"`
	Family       string `json:"datum_family"`
	Kind         string `json:"kind"`          // "geographic" or "projected"
	Unit         string `json:"unit"`          // "degree" or "metre"
	Zone         string `json:"zone"`          // human zone label, "" if N/A
	FalseEasting string `json:"false_easting"` // descriptive, "" if N/A
	Notes        string `json:"notes"`
}

// refTable is keyed by EPSG. Descriptions mirror the IGM `info` response.
var refTable = map[int]sysRef{
	// Roma40 / Monte Mario family
	4265: {4265, "Monte Mario", familyRoma40, "geographic", "degree", "", "", "Roma40 geographic (Greenwich); e=lon, n=lat in decimal degrees."},
	4806: {4806, "Monte Mario (Rome)", familyRoma40, "geographic", "degree", "", "", "Roma40 geographic referred to the Rome (Monte Mario) prime meridian."},
	3003: {3003, "Monte Mario / Italy zone 1", familyRoma40, "projected", "metre", "Gauss-Boaga fuso Ovest (zone 1)", "1 500 000 m", "Gauss-Boaga West belt; covers Italy west of ~12°E."},
	3004: {3004, "Monte Mario / Italy zone 2", familyRoma40, "projected", "metre", "Gauss-Boaga fuso Est (zone 2)", "2 520 000 m", "Gauss-Boaga East belt; covers Italy east of ~12°E."},

	// ED50 family
	4230:  {4230, "ED50", familyED50, "geographic", "degree", "", "", "European Datum 1950 geographic; e=lon, n=lat in decimal degrees."},
	23032: {23032, "ED50 / UTM zone 32N", familyED50, "projected", "metre", "UTM zone 32N", "500 000 m", "ED50 UTM; 6°E–12°E."},
	23033: {23033, "ED50 / UTM zone 33N", familyED50, "projected", "metre", "UTM zone 33N", "500 000 m", "ED50 UTM; 12°E–18°E."},
	23034: {23034, "ED50 / UTM zone 34N", familyED50, "projected", "metre", "UTM zone 34N", "500 000 m", "ED50 UTM; 18°E–24°E (eastern Italy / Salento)."},

	// ETRS89 / IGM95 family (ETRF89 realization + pan-European ETRS89 grids)
	4670: {4670, "IGM95", familyETRS89, "geographic", "degree", "", "", "ETRS89 (ETRF89) geographic; the IGM95 national network."},
	3064: {3064, "IGM95 / UTM zone 32N", familyETRS89, "projected", "metre", "UTM zone 32N", "500 000 m", "ETRS89 UTM; 6°E–12°E."},
	3065: {3065, "IGM95 / UTM zone 33N", familyETRS89, "projected", "metre", "UTM zone 33N", "500 000 m", "ETRS89 UTM; 12°E–18°E."},
	9716: {9716, "IGM95 / UTM zone 34N", familyETRS89, "projected", "metre", "UTM zone 34N", "500 000 m", "ETRS89 UTM; eastern Italy."},
	3035: {3035, "ETRS89 / ETRS-LAEA", familyETRS89, "projected", "metre", "pan-European LAEA", "4 321 000 m (x)", "Lambert Azimuthal Equal Area; pan-European statistical grid."},
	3034: {3034, "ETRS89 / ETRS-LCC", familyETRS89, "projected", "metre", "pan-European LCC", "4 000 000 m (x)", "Lambert Conformal Conic; pan-European mapping."},

	// RDN2008 family (ETRF2000 realization — the current Italian official datum)
	6706: {6706, "RDN2008 2D geo", familyRDN2008, "geographic", "degree", "", "", "RDN2008 (ETRF2000) geographic; the current official Italian datum."},
	6707: {6707, "RDN2008 / TM32", familyRDN2008, "projected", "metre", "Transverse Mercator zone 32 (CM 9°E)", "500 000 m", "RDN2008 UTM-style zone 32; easting ~500 000 at the central meridian."},
	6708: {6708, "RDN2008 / TM33", familyRDN2008, "projected", "metre", "Transverse Mercator zone 33 (CM 15°E)", "500 000 m", "RDN2008 UTM-style zone 33; easting ~500 000 at the central meridian."},
	6709: {6709, "RDN2008 / TM34", familyRDN2008, "projected", "metre", "Transverse Mercator zone 34 (CM 21°E)", "500 000 m", "RDN2008 UTM-style zone 34; eastern Italy."},
	7794: {7794, "RDN2008 / Italy Zone (E-N)", familyRDN2008, "projected", "metre", "single national zone", "~7 000 000 m (zone-prefixed)", "RDN2008 single national zone; very large zone-prefixed easting (~6.7–7.1M)."},
	6876: {6876, "RDN2008 / Zone 12", familyRDN2008, "projected", "metre", "zone 12", "~3 000 000 m (zone-prefixed)", "RDN2008 zone-12 system; zone-prefixed easting (~2.7–3.1M)."},
}

// refByEpsg returns the static metadata for an EPSG, if known.
func refByEpsg(epsg int) (sysRef, bool) {
	r, ok := refTable[epsg]
	return r, ok
}

// datumFamily returns the datum family for an EPSG, or "" if unknown.
func datumFamily(epsg int) string {
	if r, ok := refTable[epsg]; ok {
		return r.Family
	}
	return ""
}
