VERDICT: BLOCKED

## Zusammenfassung

Der Feature-Flag-Service erfüllt viele technische Anforderungen aus dem Sprint sauber: Body-Limit, Server-Timeouts, JSON-Fehlerobjekte ohne Stacktraces, kein Speichern des `user`-Parameters, kein Logging des Query-Strings durch die eigene Middleware. Die Tests decken die Kernlogik gut ab.

Trotzdem ist der aktuelle Auslieferungszustand nicht marktreif: Die API startet per Default als unverschlüsselter HTTP-Server auf allen Interfaces, verarbeitet personenbezogene Nutzerkennungen (`user=<id>`) im Query-String und besitzt keinerlei Zugriffskontrolle für mutierende Endpunkte. Das ist ein klarer Verstoß gegen die Sicherheitsanforderungen der DSGVO (Art. 32 DSGVO) und die Security-by-Default-Vorgaben des Cyber Resilience Act. Deshalb: **BLOCKED**, bis Transportverschlüsselung und Zugriffskontrolle verbindlich umgesetzt sind.

## Einschlägigkeit der Prüfbereiche

| Bereich | Einschlägig? | Begründung |
|---|---|---|
| DSGVO | Ja | Verarbeitung personenbezogener/pseudonymer Nutzerkennungen; Speicherung frei formulierbarer Beschreibungen; Logging |
| EU Cyber Resilience Act (CRA) | Ja | Produkt mit digitalen Elementen; REST-API mit Standardbibliothek |
| EU AI Act | Nein | Kein KI-/ML-Feature sichtbar; rein deterministische Hash-Entscheidung |
| Impressum/Datenschutzerklärung/Cookie-Banner | Nein | Kein öffentliches Endnutzer-UI |
| Barrierefreiheit / WCAG / BITV / EAA | Nein | Kein öffentliches Web-UI |

---

## 1. DSGVO

### 1.1 Kritisch: Unverschlüsselte Übertragung personenbezogener Nutzerkennungen

**Fundstelle:**  
- `main.go`: `srv := &http.Server{Addr: ":" + port, ...}` + `srv.ListenAndServe()`
- `internal/api/evaluate.go`: `user := r.URL.Query().Get("user")`

**Befund:**  
Der `user`-Parameter wird als Query-String über unverschlüsseltes HTTP übertragen. Nutzerkennungen sind je nach Kontext personenbezogen oder pseudonym und unterfallen damit der DSGVO. Der Service bindet per Default an `:8080` und damit an alle Netzwerkinterfaces. Ein Netzwerkbeobachter, vorgelagerter Proxy oder Zugriffs-Log eines vorgeschalteten Gateways kann die Nutzerkennung im Klartext mitlesen. Das verletzt Art. 32 DSGVO und ist als Klartext-Exposition personenbezogener Daten ein Blockierungsgrund.

**Abhilfe:**
- In `main.go` TLS verbindlich machen:
  - Entweder `srv.ListenAndServeTLS(tlsCert, tlsKey)` verwenden, wenn `TLS_CERT_FILE` und `TLS_KEY_FILE` gesetzt sind.
  - Oder den Dienst bei fehlendem TLS nicht öffentlich binden, z. B. Default-Adresse auf `127.0.0.1:8080` ändern und nur bei expliziter `LISTEN_ADDR` und vorhandenem TLS öffentlich lauschen.
- Alternativ TLS-Terminierung an einem vorgelagerten Reverse-Proxy erzwingen und im Repo dokumentieren. Reine Dokumentation reicht aber nicht, solange der Standard-Start des Produkts weiterhin unverschlüsselt lauscht.
- Optional langfristig: Den `user`-Parameter aus dem Query-String in einen Request-Body oder Header verlagern. Das würde aber die aktuelle API-Spezifikation (`GET /flags/{key}/evaluate?user={id}`) brechen und ist daher **keine Pflicht**, solange TLS verbindlich ist.

### 1.2 Hoch: Keine Zugriffskontrolle für mutierende Endpunkte

**Fundstelle:**  
- `main.go`, `newHandler`: Routen für `POST /flags`, `PUT /flags/{key}`, `DELETE /flags/{key}` sind ohne Authentifizierungs-/Autorisierungsmiddleware registriert.

**Befund:**  
Jeder, der den Port erreichen kann, kann Flags anlegen, ändern und löschen. Das ist ein erhebliches Sicherheitsrisiko. Wenn Beschreibungstexte personenbezogene Daten enthalten, steigt das DSGVO-Risiko zusätzlich, weil unberechtigte Dritte darauf zugreifen oder sie verändern können.

**Abhilfe:**
- Neue Middleware `internal/middleware/authn.go` ergänzen, z. B. `Authorization: Bearer <token>` oder `X-API-Key`.
- Key/Token aus `os.Getenv` lesen; bei leerem Token Start abbrechen oder nur `127.0.0.1` erlauben.
- Auf alle Routen außer `/healthz` anwenden.
- Zugriffsschutz so implementieren, dass Tests mit gesetztem Testtoken weiter funktionieren.

### 1.3 Mittel: Unkontrollierte Speicherung frei formulierbarer `description`-Felder

**Fundstelle:**  
- `internal/api/create.go`: akzeptiert `Description string` ohne Längen- oder Inhaltsbegrenzung.
- `internal/store/store.go`: speichert `Description` dauerhaft im In-Memory-Store, bis der Prozess endet oder das Flag gelöscht wird.

**Befund:**  
Die `description` ist ein Freitextfeld. Es ist nicht ausgeschlossen, dass Betreiber dort personenbezogene Daten ablegen. Es gibt dann keine gesonderte Lösch-, Änderungs- oder Exportfunktion außer `DELETE /flags/{key}` und `PUT /flags/{key}`. Auch keine technische Durchsetzung der Datenminimierung.

**Abhilfe:**
- Im API-Vertrag festschreiben: `description` darf keine personenbezogenen Daten enthalten.
- Technische Begrenzung in `internal/api/create.go`: z. B. `len(req.Description) <= 1024`, Zeichenbegrenzung/Whitespace-Beschränkung für `key` und `description`.
- Falls PII in `description` nicht ausgeschlossen werden kann: DSGVO-konforme Lösch- und Exportfunktionalität ergänzen oder das Feld aus dem Produkt entfernen.

### 1.4 Mittel: Log-Injection über beliebige Flag-Keys möglich

**Fundstelle:**  
- `internal/api/create.go`: validiert nur `key != ""`.
- `internal/middleware/middleware.go`: `log.Printf("%s %s %d", r.Method, r.URL.Path, sw.status)`

**Befund:**  
Wenn ein Flag-Key Steuerzeichen wie `\n` enthält, kann ein Angreifer mit einem präparierten Key die Logzeile aufspalten oder manipulieren. Enthält ein Key personenbezogene Daten, würden diese über den Pfad im Log landen.

**Abhilfe:**
- In `internal/api/create.go` einen erlaubten Zeichenraum erzwingen, z. B. `^[a-zA-Z0-9._-]{1,128}$`.
- Alternativ in der Middleware `r.URL.Path` vor dem Logging sanitieren oder Escaping anwenden.
- Tests für ungültige Keys ergänzen.

### 1.5 Positiv: `user` wird nicht gespeichert und nicht geloggt

**Fundstellen:**  
- `internal/api/evaluate.go`: `user` wird ausschließlich an `evaluate.Decide` übergeben.
- `internal/store/store.go`: kein `user`-Feld.
- `internal/middleware/middleware.go`: loggt nur `r.URL.Path`, nicht `r.URL.RawQuery`.

**Bewertung:**  
Erfüllt AC-14 und AC-15. Die eigene Anwendung leakt den Query-String nicht. Der verbleibende Klartexttransport ist unter 1.1 und 1.2 adressiert.

---

## 2. EU Cyber Resilience Act (CRA)

### 2.1 Hoch: Kein sichtbares SBOM / kein dokumentierter Update- und Patch-Prozess

**Fundstelle:**  
- Kein SBOM-Dokument sichtbar.
- `go.mod` ist nur in der Dateiliste vorhanden, es gibt keine SBOM-Erzeugung, keine Versionsinformation, keine dokumentierte Update-Policy.

**Befund:**  
Der CRA verlangt für Produkte mit digitalen Elementen unter anderem eine Software-Stückliste (SBOM), dokumentierte Sicherheitseigenschaften und die Fähigkeit, Sicherheitsupdates einzuspielen. Der Service nutzt zwar nur die Standardbibliothek, ein SBOM ist dennoch erforderlich.

**Abhilfe:**
- In CI/CD SBOM erzeugen, z. B. CycloneDX oder SPDX: `go list -deps -json` + Generator oder `syft`.
- In `internal/api/health.go` Versions- und Build-Information ergänzen (`runtime/debug.ReadBuildInfo`, Version, Commit).
- `README.md` um Abschnitt „Sicherheit, Updates, Support-Zeitraum“ ergänzen.
- Dependency-Scan aktivieren, auch bei reiner Go-Standardbibliothek.

### 2.2 Mittel: Security by Default nicht vollständig

**Fundstelle:**  
- `main.go`: keine TLS-Konfiguration, kein Authentifizierungsmechanismus.

**Befund:**  
Security by Design/Default verlangt sichere Grundeinstellungen. Der aktuelle Default ist ein offener, unverschlüsselter Dienst.

**Abhilfe:**  
Wie unter 1.1 und 1.2 beschrieben: TLS erzwingen, Default-Bindung absichern, optionale Authentifizierung oder explizite Allowlist/Netzwerkpolicy.

### 2.3 Positiv: Wichtige technische Sicherheitsmaßnahmen sind vorhanden

- Body-Limit: `internal/middleware/middleware.go` (`LimitBody(1 << 20)` in `main.go`)
- Timeouts: `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` in `main.go`
- Keine Stacktraces in Fehlerantworten: `internal/api/errors.go`
- Thread-sicherer Store: `sync.RWMutex` in `internal/store/store.go`

---

## 3. EU AI Act

**Bewertung:**  
Nicht einschlägig. Kein KI-/ML-Modell, kein automatisiertes Entscheidungssystem mit KI-Bezug. Die Evaluierung ist ein deterministischer FNV-1a-Hash. Keine Transparenz-/Kennzeichnungspflichten nach AI Act erkennbar.

---

## 4. Pflichttexte und UI

**Bewertung:**  
Nicht einschlägig. Kein öffentliches Web-UI, keine Cookies, kein Verkaufsvorgang mit Widerrufsbelehrung, keine Impressumspflicht für eine reine Backend-API.

Falls später ein Administrations-UI ergänzt wird, wären Datenschutzerklärung, Impressum und ggf. Cookie-Einwilligung erforderlich.

---

## 5. Barrierefreiheit

**Bewertung:**  
Nicht einschlägig. Kein öffentliches Web-UI. Falls später ein UI ergänzt wird, gelten WCAG/BITV/EAA.

---

## Abhilfe-Reihenfolge

1. **Blocker sofort beheben:** TLS verbindlich machen oder öffentliches Binden ohne TLS verhindern.
2. **Blocker sofort beheben:** Zugriffskontrolle auf allen nicht-öffentlichen Routen ergänzen.
3. **Hoch:** SBOM, Versions-/Build-Info, Update-Policy ergänzen.
4. **Mittel:** Key-Validierung einführen, Beschreibungstexte fachlich beschränken und Längenlimits setzen.

## Reconcile-Hinweis

Die vorgeschlagenen TLS- und Authentifizierungsmaßnahmen dürfen den regulären Betrieb nicht brechen. Konkret bedeutet das:

- TLS muss so umgesetzt werden, dass lokale Tests und `httptest` weiterhin ohne Zertifikat funktionieren. Die Handler selbst bleiben transportneutral.
- Die Authentifizierung muss konfigurierbar sein. Ohne konfiguriertes Token darf der Dienst nur lokal lauschen oder den Start abbrechen.
- Die `GET /flags/{key}/evaluate?user={id}`-Route bleibt gemäß AC-07 unverändert erhalten; zusätzliche Header sind abwärtskompatibel, solange Clients sie mitsenden.