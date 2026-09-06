VERDICT: CHANGES_REQUESTED

## Sicherheitsbericht

### Scanner-Abdeckung
Für dieses Projekt wurde kein Security-Scanner (`gosec`, `govulncheck`, `semgrep`) ausgeführt. Das Fehlen von Scanner-Ergebnissen ist kein Nachweis für die Abwesenheit von Schwachstellen. Die folgende Bewertung basiert auf manueller Codeanalyse.

---

### 1. Fehlende Authentifizierung und Autorisierung
**Schweregrad:** Mittel (hoch, falls der Dienst außerhalb eines vertrauenswürdigen Netzes betrieben wird)

**Betroffene Stellen:**
- `main.go` – gesamte Routenregistrierung
- `internal/api/create.go`, `internal/api/update.go`, `internal/api/delete.go`, `internal/api/get.go`, `internal/api/list.go`, `internal/api/evaluate.go`

**Befund:**  
Alle Endpunkte außer `/healthz` sind ohne jede Zugriffskontrolle erreichbar. Jeder, der den Dienst erreichen kann, darf Flags anlegen, ändern, löschen, auslesen und Rollout-Entscheidungen abfragen. Für einen Feature-Flag-Dienst bedeutet das, dass eine unbefugte Person Rollouts manipulieren oder Feature-Konfigurationen ausspähen kann.

**Konkreter Fix:**  
Eine Authentifizierungsschicht in die Middleware einführen, z. B.:
- Bearer-Token aus `Authorization`-Header oder API-Key aus Umgebungsvariable (`FLAG_API_KEY`) prüfen.
- Middleware vor allen `/flags`-Routen registrieren; `/healthz` kann offen bleiben.
- Schreibende Routen zusätzlich mit einer Schreibberechtigung schützen.

---

### 2. Unverschlüsselter Transport (nur HTTP, kein TLS)
**Schweregrad:** Mittel

**Betroffene Stelle:**
- `main.go` – `srv.ListenAndServe()` statt `srv.ListenAndServeTLS(...)`

**Befund:**  
Der Server kommuniziert ausschließlich im Klartext über HTTP. Beim Evaluate-Endpunkt wird die Nutzer-ID (`user`) als Query-Parameter übertragen. Außerhalb einer vertrauenswürdigen Umgebung können Nutzer-IDs (potenziell personenbezogene Daten) und Flag-Konfigurationen mitgelesen oder verändert werden.

**Konkreter Fix:**  
- TLS direkt im Dienst aktivieren (`ListenAndServeTLS` mit Zertifikat/Schlüssel) **oder**
- verbindlich einen TLS-terminierenden Reverse-Proxy vorschalten und dokumentieren, dass der Dienst nicht direkt im Klartext exponiert werden darf.
- Optional HTTP-zu-HTTPS-Weiterleitung erzwingen.

---

### 3. Unvollständige JSON-Validierung: Trailing-Daten werden akzeptiert
**Schweregrad:** Niedrig

**Betroffene Stellen:**
- `internal/api/create.go` – `json.NewDecoder(r.Body).Decode(&req)`
- `internal/api/update.go` – `json.NewDecoder(r.Body).Decode(&req)`

**Befund:**  
Nach dem ersten gültigen JSON-Objekt werden weitere Daten im Request-Body stillschweigend ignoriert. Ein Body wie `{"key":"x"} unerwartet` wird akzeptiert. Das unterläuft eine strikte Eingabevalidierung und kann zu unerwartetem Verhalten führen.

**Konkreter Fix:**  
Nach dem ersten `Decode` prüfen, dass der Stream zu Ende ist:
```go
dec := json.NewDecoder(r.Body)
if err := dec.Decode(&req); err != nil {
    WriteError(w, http.StatusBadRequest, "invalid request body")
    return
}
if err := dec.Decode(&struct{}{}); err != io.EOF {
    WriteError(w, http.StatusBadRequest, "invalid request body")
    return
}
```
Alternativ `dec.More()` nach dem ersten `Decode` auswerten.

---

### 4. Interne Fehlerdetails in Fehlerantwort möglich
**Schweregrad:** Niedrig

**Betroffene Stelle:**
- `internal/api/create.go` – im Fehlerzweig:
  ```go
  WriteError(w, http.StatusInternalServerError, err.Error())
  ```

**Befund:**  
Aktuell liefert der Store im Create-Pfad nur `ErrConflict`. Sollte sich die Store-Implementierung künftig ändern, könnten interne Fehlermeldungen nach außen gelangen. Das widerspräche den Datenschutz-Vorgaben (AC-13/AC-16), die ausschließlich definierte Fehlermeldungen ohne Implementierungsdetails fordern.

**Konkreter Fix:**  
Statische Meldung verwenden:
```go
WriteError(w, http.StatusInternalServerError, "internal error")
```
Zusätzlich `errors.Is(err, store.ErrConflict)` statt `err == store.ErrConflict` verwenden, um auch gewrappte Fehler robust zu erkennen.

---

### 5. Unkontrolliertes Speicherwachstum des In-Memory-Stores
**Schweregrad:** Niedrig

**Betroffene Stellen:**
- `internal/api/create.go` – Validierung von `key`/`description`
- `internal/store/store.go` – unbegrenzte Map-Größe

**Befund:**  
Die 1-MiB-Body-Grenze begrenzt nur die Größe eines einzelnen Requests, nicht die Anzahl der gespeicherten Flags oder die Länge eines einzelnen `key`/`description`. Wiederholte POST-Anfragen können den Speicher des Dienstes erschöpfen (DoS). Ein einzelner Schlüssel darf theoretisch bis zu ~1 MiB groß sein.

**Konkreter Fix:**  
- Fachliche Längengrenzen einführen, z. B. `key` max. 128 Zeichen, `description` max. 4096 Zeichen.
- Optional erlaubte Zeichen für `key` einschränken (z. B. `[A-Za-z0-9._-]+`).
- Optional eine maximale Anzahl von Flags im Store festlegen.

---

### 6. Log-Injection über URL-Pfad
**Schweregrad:** Niedrig

**Betroffene Stelle:**
- `internal/middleware/middleware.go` – Logging-Middleware:
  ```go
  log.Printf("%s %s %d", r.Method, r.URL.Path, sw.status)
  ```

**Befund:**  
Der Pfad wird ungefiltert in das Protokoll geschrieben. Enthält ein Request-Pfad Steuerzeichen wie `%0A` (Newline), kann ein Angreifer Protokollzeilen spalten oder fingierte Log-Einträge erzeugen. Das untergräbt die forensische Auswertbarkeit.

**Konkreter Fix:**  
Pfad vor dem Loggen bereinigen oder strukturiert loggen:
```go
log.Printf("%s %q %d", r.Method, r.URL.Path, sw.status)
```
oder `strconv.Quote(r.URL.Path)` verwenden. Alternativ auf strukturiertes Logging (JSON) umstellen, das Steuerzeichen escaped.

---

### Fazit
Es wurden keine harten Geheimnisse, injizierbaren Code oder bekannten verwundbaren Drittabhängigkeiten gefunden. Die Kernlogik (Mutex-Store, Rollout-Hash, Body-Limit, Timeouts, Query-freie Logs) ist solide umgesetzt. Die Hauptrisiken liegen im fehlenden Zugriffsschutz und im fehlenden Transportverschlüsselung. Daher wird eine Überarbeitung vor der Auslieferung empfohlen.