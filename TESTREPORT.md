VERDICT: BUGS_FOUND

Der Testlauf ist nicht sauber: `go test ./...` endet mit `exit 1`. Es schlagen zwei Tests im Root-Paket fehl.

**1. Parametrisierte Routen bestehen Verdrahtungsprüfung nicht**
- **Titel:** Verdrahtungsprüfung für parametrisierte Routen schlägt fehl
- **Symptom:** Der Test `TestEndpointsWired` meldet für `GET /flags/{key}`, `DELETE /flags/{key}` und `GET /flags/{key}/evaluate` „route not wired“. Die Log-Ausgabe (`GET /flags/some-key 404` usw.) zeigt, dass die Handler grundsätzlich erreicht werden; die 404-Antworten werden vom Test jedoch offenbar nicht als gültige JSON-Fehlerantworten der verdrahteten Routen akzeptiert.
- **Repro:** `go test ./...` → `TestEndpointsWired`
- **Evidence:**
  ```
  --- FAIL: TestEndpointsWired (0.00s)
      --- FAIL: TestEndpointsWired/GET_/flags/some-key (0.00s)
          flags_api_test.go:138: GET /flags/some-key -> 404 (route not wired)
      --- FAIL: TestEndpointsWired/DELETE_/flags/some-key (0.00s)
          flags_api_test.go:138: DELETE /flags/some-key -> 404 (route not wired)
      --- FAIL: TestEndpointsWired/GET_/flags/some-key/evaluate?user=alice (0.00s)
          flags_api_test.go:138: GET /flags/some-key/evaluate?user=alice -> 404 (route not wired)
  ```
- **Suspected files:** Gemeinsame Schicht ist die Routing-/Middleware-Verdrahtung, daher primär `main.go` (Routenregistrierung, `json405`-Middleware) sowie ggf. `internal/api/errors.go`. Nicht isoliert in den einzelnen API-Dateien, da drei verschiedene Handler betroffen sind.
- **Severity:** high

**2. Boundary-Test für rollout_percent=100 schlägt fehl**
- **Titel:** Boundary-Test für `rollout_percent=100` erhält 409 statt 201
- **Symptom:** Beim Anlegen eines Flags mit `rollout_percent=100` liefert der Server im Testlauf eine 409-Antwort, obwohl ein neuer Key angelegt werden soll. Dadurch ist die in AC-08 geforderte Zusicherung „rollout_percent=100 => immer an“ nicht erfolgreich über den End-to-End-Test verifiziert.
- **Repro:** `go test ./...` → `TestEvaluateRolloutBoundaries`
- **Evidence:**
  ```
  --- FAIL: TestEvaluateRolloutBoundaries (0.00s)
      flags_api_test.go:326: create rollout 100 -> 409
  ```
- **Suspected files:** Vermutlich `internal/store/store.go` bzw. `internal/api/create.go` oder mangelnde Isolation des Stores im Integrationstest `flags_api_test.go`. Eine eindeutige Lokalisierung ist aus dem Bericht nicht möglich.
- **Severity:** high