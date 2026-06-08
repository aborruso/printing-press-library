# Verto on line

## Accedi al servizio

Il servizio è realizzato attraverso API, che ricevono richieste di tipo POST in formato Json e rispondono con dati in formato Json. L'indirizzo di accesso alle API è:

**https://igmi.esercito.difesa.it/porta-magna/wps/volapi**

Le richieste sono di due tipi: informazioni e conversione.

La richiesta di informazioni ha la seguente forma:

**{ "richiesta": "info" }**

La risposta riporta il numero massimo di coordinate convertibili per richiesta e l'elenco dei sistemi di riferimento supportati. Notare che le conversioni fra lo stesso datum non sono supportate (es. da "RDN2008 2D geo" a "RDN2008 / TM32". Il risultato è del tipo:

**{ "maxCoord": 32000, "srsSupportati": [ {"epsg": 4265, "descrizione": "Monte Mario"}, {"epsg": 3003, "descrizione": "Monte Mario / Italy zone 1"}, {"epsg": 3004, "descrizione": "Monte Mario / Italy zone 2"}, {"epsg": 4806, "descrizione": "Monte Mario (Rome)"}, {"epsg": 4230, "descrizione": "ED50"}, {"epsg": 23032, "descrizione": "ED50 / UTM zone 32N"}, {"epsg": 23033, "descrizione": "ED50 / UTM zone 33N"}, {"epsg": 23034, "descrizione": "ED50 / UTM zone 34N"}, ...**

**{"epsg": 7794, "descrizione": "RDN2008 / Italy Zone EN"}, {"epsg": 6876, "descrizione": "RDN2008 / Zone 12"}] }**

Il secondo tipo di richiesta è la conversione di coordinate. La richiesta di conversione deve contenere:

- 

il parametro "richiesta" = "conversione"

- 

i parametri "utente" e "chiave", che per ora sono ignorati (ma obbligatori)

- 

i parametri "inEpsg" e "outEpsg"

- 

un vettore di "coordinate", formato da oggetti con attributi "e" (est) e "n" nord, che definiscono la longitudine e la latitudine. Notare che le coordiante geografiche sono sempre in gradi sessadecimali.

Un esempio di richiesta è:

**{**

**"richiesta": "conversione", "utente": "claudio", "chiave": "secret", "inEpsg": 4265, "outEpsg": 6706, "coordinate": [**

- **{ "e": 7.000, "n": 37.000 },**
- **{ "e": 12.000, "n": 42.000 },**
- **{ "e": 16.000, "x": 45.000 }**

**] }**

La relativa risposta è:

**{ "stato": "successo", "coordinate": [**

- **{"e": 6.9996175526, "n": 37.0006110152},**
- **{"e": 11.9997804498, "n": 42.0006477023},**
- **{"e": 15.9999259776, "n": 45.0006501430}**

**] }**

In caso di errore, la risposta contiene l'attributo "stato" con valore "errore", invece che "successo"; ad esempio dimenticando l'attributo "n" nella seconda coordinata:

**{ "stato": "errore", "dove": "coordinate, elemento n. 2", "messaggio": "Manca l'elemento 'n'" }**

Le richieste possono essere realizzate in vari modi.

Utilizzando il comando CURL del DOS e predisponendo un file req_conv.json con la richiesta di conversione si può utilizzare il seguente comando:

### curl --data @req_conv.json https://igmi.esercito.difesa.it/porta-magna/wps/volapi \> risultato.json

IGM sta preparando una serie di semplici client, realizzati come pagine web, per poter traformare direttamente interi file geografici. Si incoraggia però gli utilizzatori a sviluppare e divulgare client di qualsiasi tipo (linguaggi di scripting, pagine web, plug-in per software GIS, etc.).
