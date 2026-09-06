VERDICT: APPROVED

## Sicherheitsprüfung

**Scanner-Hinweis:** Für diesen Projekttyp (`go-backend`) wurden keine anwendbaren Security-Scanner ausgeliefert. Die folgende Bewertung beruht auf manueller Analyse des sichtbaren Quellcodes. Das Fehlen von Scanner-Output ist selbst kein Befund.

### Gesamturteil
Es wurden keine ausnutzbaren Schwachstellen mit hohem oder kritischem Risiko gefunden. Die geforderten Sicherheitsmaßnahmen (Body-Limit, Server-Timeouts, JSON-Fehler ohne interne Details, Datenschutz für den `user`-Parameter) sind korrekt umgesetzt. Es verbleiben einige niedrige Härtungsempfehlungen.

### Einzelbefunde

#### 1. Health-Endpoint gibt Versionsdetails preis
- **Schweregrad:** niedrig
- **Ort:** `internal/api/health.go:13-20`
- **Beschreibung:** `GET /healthz` ist öffentlich und liefert `go_version` sowie `module_version`. Diese Informationen sind für den Betrieb des Feature-Flag-Dienstes nicht erforderlich und können bei der Aufklärung durch Angreifer helfen (Reconnaissance). AC-01 verlangt lediglich ein Status-JSON.
- **Konkrete Korrektur:** Health-Antwort auf `{"status":"ok"}` reduzieren oder die Versionsfelder nur bei authentifizierten Requests zurückgeben.

#### 2. API-Key-Vergleich nicht in konstanter Zeit
- **Schweregrad:** niedrig
- **Ort:** `internal/middleware/authn.go:29-35`
- **Beschreibung:** Der API-Key wird per normalem Stringvergleich (`==`, `strings.TrimPrefix`) geprüft. Dadurch sind theoretisch Timing-Angriffe auf die Länge bzw. den Wert des Keys möglich. In der Praxis ist das über Netzwerke schwer auszunutzen, sollte für einen statischen API-Key aber vermieden werden.
- **Konkrete Korrektur:** `crypto/subtle.ConstantTimeCompare` für den Vergleich von Header-Wert und erwartetem Key verwenden. Beispiel:
  ```go
  if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Key")), []byte(key)) == 1 { ... }
  ```
  Analog für den Bearer-Token aus dem `Authorization`-Header.

#### 3. Offener Modus bei leerem API-Key und nicht-loopback-Bindung
- **Schweregrad:** niedrig
- **Ort:** `main.go:118-151`
- **Beschreibung:** Wenn `FLAG_API_KEY` leer ist, bleibt der Dienst unauthentifiziert. Die Standardbindung ist zwar Loopback (`127.0.0.1`), und bei Wildcard-Adressen ohne TLS wird erzwungen auf Loopback ausgewichen. Wird jedoch eine konkrete nicht-loopback-IP als `LISTEN_ADDR` gesetzt (z. B. `192.168.1.10:8080`) und kein TLS/API-Key konfiguriert, startet der Dienst offen auf dieser Adresse.
- **Konkrete Korrektur:** Optional: Start verweigern bzw. auf Loopback zwingen, wenn `FLAG_API_KEY` leer und `LISTEN_ADDR` nicht Loopback ist. Für bewusst offene interne Deployments einen expliziten Override (z. B. `ALLOW_INSECURE_AUTH_LESS=true`) vorsehen.

#### 4. Unbegrenzte Speicherbelegung durch viele Flags
- **Schweregrad:** niedrig
- **Ort:** `internal/store/store.go:36-59`, `internal/api/create.go`
- **Beschreibung:** Der In-Memory-Store hat keine Obergrenze für die Anzahl der Flags. Zwar ist jeder einzelne Request-Body auf 1 MiB begrenzt, aber viele erfolgreiche `POST /flags` können den Speicher erschöpfen. Ein Rate-Limit oder eine feste Maximalkonfiguration wäre eine sinnvolle Härtung.
- **Konkrete Korrektur:** Maximale Flag-Anzahl im Store einführen (z. B. konfigurierbare `maxFlags`) und bei Überschreitung mit `429` oder `503` antworten. Alternativ eine Rate-Limit-Middleware vor `CreateFlag`/`UpdateFlag` schalten.

### Positiv geprüfte Aspekte

- **Request-Body-Limit:** `LimitBody(1 << 20)` wird auf `POST /flags` und `PUT /flags/{key}` angewendet und puffert nur maximal `1 MiB + 1 Byte`.
- **Server-Timeouts:** `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` und `IdleTimeout` sind mit Werten zwischen 10 und 30 Sekunden gesetzt.
- **Fehlerantworten:** Alle Fehlerpfade nutzen das definierte JSON-Format `{"error":"..."}`. Interna, Stacktraces oder Pfade werden nicht ausgegeben.
- **Datenschutz `user`:** Der `user`-Parameter des Evaluate-Endpunkts wird ausschließlich für die Hash-Berechnung verwendet und weder gespeichert noch geloggt. Die Logging-Middleware protokolliert nur `Methode`, `URL.Path` (`%q`, escape-sicher) und `Statuscode`.
- **Strikte Key-Validierung:** Der Key-RegExp erlaubt nur `[a-zA-Z0-9._-]{1,128}`; leere und übergroße Keys werden abgewiesen.
- **Thread-Sicherheit:** Der In-Memory-Store nutzt `sync.RWMutex` korrekt für alle CRUD-Operationen.
- **Transport-Härtung:** Ohne TLS wird die Bindung an `0.0.0.0` / `[::]` automatisch auf Loopback zurückgesetzt.
- **Abhängigkeiten:** Es sind ausschließlich Standardbibliotheks-Pakete und projektinterne Pakete sichtbar; keine externen Dependency-CVEs erkennbar.