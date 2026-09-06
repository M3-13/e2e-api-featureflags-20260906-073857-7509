VERDICT: CHANGES_REQUESTED

## Prüfbericht

**Projekttyp:** `go-backend` — reine REST-API ohne Endnutzer-UI.  
**Geprüft:** der vorgelegte, sichtbare Code und die sichtbaren Tests. Die Dateien `README.md`, `SECURITY.md` und `COMPLIANCE.md` sind vorhanden, ihr Inhalt lag jedoch nicht vor und wurde daher nicht inhaltlich bewertet.

---

### 1. DSGVO / GDPR

**Relevante personenbezogene Daten:**  
Der Query-Parameter `user` bei `GET /flags/{key}/evaluate` ist eine Nutzerkennung und damit potenziell personenbeziehbar. Er wird in `internal/api/evaluate.go` ausschließlich für die Hash-Berechnung verwendet, nicht im Store gespeichert und nicht geloggt. Das ist datenminimierend und entspricht den sichtbaren AC-14/AC-15.

**Speicherung und Löschung:**  
Die Flags liegen in einem flüchtigen In-Memory-Store (`internal/store/store.go`). Löschung ist über `DELETE /flags/{key}` möglich; bei Prozessende sind die Daten weg. Eine überlange Speicherung ist damit nicht erkennbar.

**Protokollierung:**  
Die Logging-Middleware in `internal/middleware/middleware.go` protokolliert nur Methode, Pfad und Statuscode. Query-Strings werden nicht geloggt. Der Test `TestLoggingDoesNotLogQuery` bestätigt das. Kein PII-Leak in Logs erkennbar.

**Fehlerantworten:**  
Fehlerantworten enthalten nur das Feld `error`; keine Stacktraces, internen Pfade oder Implementierungsdetails. Das erfüllt die sichtbaren AC-13/AC-16.

**Befunde:**

- **DS-01 — mittel — Transportverschlüsselung wird nicht für jede nicht-loopback Adresse erzwungen.**  
  Datei: `main.go`, Block in `main()` bei `LISTEN_ADDR`.  
  Problem: `bindsAllInterfaces` erfasst nur `""`, `0.0.0.0` und `::`. Setzt ein Betreiber `LISTEN_ADDR=192.0.2.10:8080`, ohne `TLS_CERT_FILE`/`TLS_KEY_FILE` zu setzen, bindet der Dienst an diese konkrete externe Adresse ohne TLS. Der `user`-Parameter würde dann im Klartext übertragen. Der aktuelle Default ist zwar loopback und damit sicher, aber die Lücke ist über Konfiguration auslösbar.  
  Abhilfe: Nach Ermittlung von `addr` prüfen:
  ```go
  if !tlsEnabled && !isLoopbackHost(addr) {
      log.Fatalf("LISTEN_ADDR %q erfordert TLS_CERT_FILE und TLS_KEY_FILE", listenAddr)
  }
  ```
  Dafür einen Helper `isLoopbackHost` ergänzen, der `127.0.0.1`, `::1` und `localhost` erkennt. Alternativ nicht-loopback Adressen grundsätzlich nur mit TLS akzeptieren. Der bisherige Schutz für `0.0.0.0`/`::` bleibt dabei erhalten.

- **DS-02 — niedrig — Beschreibungsfeld kann personenbezogene Daten aufnehmen.**  
  Dateien: `internal/api/create.go`, `internal/api/update.go`.  
  Problem: `description` ist ein freies Textfeld bis 1024 Zeichen. Es ist nicht ausgeschlossen, dass Betreiber dort Namen, E-Mail-Adressen oder andere PII ablegen.  
  Abhilfe: In `README.md` und/oder `COMPLIANCE.md` dokumentieren: „Das Feld `description` darf nur sachliche Beschreibungen enthalten; personenbezogene Daten dürfen darin nicht gespeichert werden.“ Eine technische Maskierung wäre hier unverhältnismäßig.

- **DS-03 — niedrig — Der `user`-Wert wird mit FNV-1a 32 Bit gehasht.**  
  Datei: `internal/evaluate/evaluate.go`.  
  Problem: FNV-1a ist schnell und deterministisch, aber keine starke Pseudonymisierung. Der Hash wird zwar nicht gespeichert oder ausgegeben, bei sehr kleinen Nutzer-ID-Räumen kann er jedoch theoretisch rekonstruiert werden, wenn das boolesche Ergebnis bekannt ist.  
  Abhilfe: Optional auf HMAC-SHA-256 mit einem serverindividuellen Geheimnis umstellen, z. B. `FLAG_HASH_SECRET`; Ergebnis modulo 100. Falls das Verteilungsverhalten sich dadurch ändert, muss der Betreiber das bewusst abnehmen. Eine Pflicht dazu ist aus dem sichtbaren Code nicht ableitbar.

---

### 2. EU Cyber Resilience Act (CRA)

**Security by design/default:**  
Positiv sichtbar: Body-Limit von 1 MiB für `POST /flags` und `PUT /flags/{key}`, Server-Timeouts zwischen 5 und 30 Sekunden, JSON-Fehler ohne interne Details, standardmäßige Bindung an Loopback, optionale TLS-Aktivierung, optionale API-Key-Authentifizierung, keine PII in Logs.

**Abhängigkeiten:**  
Es sind keine Drittanbieter-Abhängigkeiten erkennbar; der Code nutzt die Go-Standardbibliothek. `go.mod` ist vorhanden. Dadurch ist das Lieferkettenrisiko gering.

**Befunde:**

- **CRA-01 — niedrig — API-Key-Vergleich nicht in konstanter Zeit.**  
  Datei: `internal/middleware/authn.go`, Funktion `authorized`.  
  Problem: `r.Header.Get("X-API-Key") == key` und `strings.TrimPrefix(auth, "Bearer ") == key` vergleichen Zeichenketten nicht in konstanter Zeit. Bei einem exponierten Dienst kann das theoretisch Timing-Angriffe auf den API-Key erleichtern.  
  Abhilfe: `crypto/subtle.ConstantTimeCompare` mit vorheriger Längenprüfung verwenden, z. B.:
  ```go
  if len(got) != len(key) { return false }
  return subtle.ConstantTimeCompare([]byte(got), []byte(key)) == 1
  ```
  Das gilt sowohl für `X-API-Key` als auch für das Bearer-Token.

- **CRA-02 — niedrig — Keine Recovery-Middleware.**  
  Datei: `main.go`, Funktion `newHandler`.  
  Problem: Eine unerwartete Panik in einem Handler erzeugt keine kontrollierte JSON-Fehlerantwort und kann interne Details in Server-Logs schreiben.  
  Abhilfe: Eine `Recover`-Middleware als äußerste Schicht ergänzen, die `recover()` aufruft, serverseitig ohne Query-String loggt und an den Client `api.WriteError(w, http.StatusInternalServerError, "internal error")` zurückschreibt.

- **CRA-03 — niedrig — `MaxHeaderBytes` nicht explizit gesetzt.**  
  Datei: `main.go`, `http.Server{...}`.  
  Problem: Header werden nur durch den Go-Standardwert begrenzt. Eine explizite Grenze ist sicherer dokumentierbar.  
  Abhilfe: Im `http.Server`-Literal `MaxHeaderBytes: oneMiB` ergänzen.

- **CRA-04 — niedrig — Kein Rate-Limiting / Brute-Force-Schutz erkennbar.**  
  Datei: `main.go` bzw. Middleware-Schicht.  
  Problem: Bei aktiviertem API-Key und exponierter Instanz fehlt eine Begrenzung für wiederholte Fehlversuche.  
  Abhilfe: Entweder eine einfache Limit-Middleware für `/flags` ergänzen oder in `SECURITY.md` klar dokumentieren, dass Rate-Limiting am vorgelagerten Reverse-Proxy erfolgen muss.

---

### 3. EU AI Act

**Nicht anwendbar.**  
Im sichtbaren Produkt ist keine KI-Funktion, kein ML-Modell und keine automatisierte Entscheidungsfindung mit KI-Bezug vorhanden. Der Evaluate-Endpunkt ist eine deterministische Rollout-Logik, aber kein KI-System im Sinne des AI Act.

---

### 4. Pflichttexte und UI-Pflichten

**Nicht anwendbar.**  
Reines Backend ohne Endnutzer-UI. Es bestehen keine sichtbaren Pflichten zu Impressum, Datenschutzerklärung, Cookie-Banner, Consent-Management oder Widerrufsbelehrung. Die API ist keine Verkaufsfläche.

---

### 5. Barrierefreiheit

**Nicht anwendbar.**  
Es gibt keine öffentliche Web-UI. WCAG/BITV/EAA betreffen dieses Produkt nicht unmittelbar.

---

### Gesamtbewertung

Der Service erfüllt die wesentlichen funktionalen und datenschutzbezogenen Anforderungen des Sprints. Die Nutzerkennung wird minimal verarbeitet, nicht gespeichert und nicht geloggt. Der wichtigste offene Punkt ist die mögliche unverschlüsselte Übertragung bei einer konkret konfigurierten nicht-loopback Adresse ohne TLS. Das ist ein behebbarer Mangel; er ist kein fundamentaler Blocker, sollte aber vor Marktfreigabe behoben werden.