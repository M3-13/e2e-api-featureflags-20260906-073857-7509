VERDICT: BUGS_FOUND

- **Titel:** Root-Test `TestEndpointsWired` wertet 404 für unbekannten Key als „Route nicht verdrahtet“
- **Symptom:** `go test ./...` endet mit Exit-Code 1. Drei Untertests von `TestEndpointsWired` schlagen fehl, weil sie den nicht existierenden Key `some-key` verwenden und einen 404-Response als nicht registrierte Route interpretieren. Die Spezifikation verlangt für unbekannte Keys jedoch genau 404 (AC-04, AC-06); der Test ist zu streng und blockiert die grüne Test-Suite.
- **Repro:** `go test ./...`
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
- **Suspected file(s):** `flags_api_test.go` (Testfunktion `TestEndpointsWired`). Die Routen selbst sind in `main.go` korrekt registriert; der Test müsste einen zuvor angelegten Key verwenden.
- **Severity:** high (AC-10 verletzt, CI rot)

- **Titel:** Root-Test `TestEvaluateRolloutBoundaries` erhält 409 beim Anlegen eines Flags mit Rollout 100
- **Symptom:** Beim Erzeugen des Flags für den Boundary-Test liefert `POST /flags` 409 Conflict statt 201 Created. Dadurch bricht der Test ab und die gesamte Test-Suite bleibt rot.
- **Repro:** `go test ./...` (führt `TestEvaluateRolloutBoundaries` aus)
- **Evidence:**
  ```
  --- FAIL: TestEvaluateRolloutBoundaries (0.00s)
      flags_api_test.go:326: create rollout 100 -> 409
  ```
- **Suspected file(s):** `flags_api_test.go` (Testfunktion `TestEvaluateRolloutBoundaries`). Vermutlich wird ein fester oder bereits verwendeter Key genutzt, oder `uniqueKey()` kollidiert; der Test muss einen frischen Key sicherstellen.
- **Severity:** high (AC-10 verletzt, CI rot)