VERDICT: BUGS_FOUND

**Bug 1**
- **Titel**: TestEndpointsWired schlägt fehl – 404 für unbekannte Keys wird fälschlich als „Route nicht verdrahtet“ interpretiert
- **Symptom**: `go test ./...` scheitert im Paket `featureflags` (Root). Der Test `TestEndpointsWired` erwartet für die Pfade `GET /flags/some-key`, `DELETE /flags/some-key` und `GET /flags/some-key/evaluate?user=alice` keinen Status 404, obwohl die Spezifikation für unbekannte Keys genau 404 vorsieht. Die Testsuite ist damit rot; AC-10 („go test führt alle Handler- und Rollout-Tests erfolgreich aus“) ist verletzt.
- **Repro**: `go test ./...` ausführen.
- **Evidence**:
  ```
  --- FAIL: TestEndpointsWired (0.00s)
      --- FAIL: TestEndpointsWired/GET_/flags/some-key (0.00s)
          flags_api_test.go:138: GET /flags/some-key -> 404 (route not wired)
      --- FAIL: TestEndpointsWired/DELETE_/flags/some-key (0.00s)
          flags_api_test.go:138: DELETE /flags/some-key -> 404 (route not wired)
      --- FAIL: TestEndpointsWired/GET_/flags/some-key/evaluate?user=alice (0.00s)
          flags_api_test.go:138: GET /flags/some-key/evaluate?user=alice -> 404 (route not wired)
  ```
- **Suspected file(s)**: `flags_api_test.go` – die Prüfung in `TestEndpointsWired` ist falsch formuliert: Sie verbietet Status 404, obwohl die Endpunkte laut AC-04, AC-06 und AC-07 bei unbekanntem Key korrekt 404 liefern.
- **Severity**: high

**Bug 2**
- **Titel**: TestEvaluateRolloutBoundaries schlägt fehl – Create mit rollout 100 liefert 409 statt 201
- **Symptom**: `go test ./...` scheitert zusätzlich im Root-Paket. Der Test `TestEvaluateRolloutBoundaries` versucht ein Flag mit Rollout 100 anzulegen, erhält aber Status 409 (Conflict). Dadurch ist die Rollout-Grenzwert-Prüfung nicht abschließbar und AC-10 bleibt verletzt. Die Ursache ist sehr wahrscheinlich eine Schlüsselkollision: `uniqueKey()` in `flags_api_test.go` verwendet nur `time.Now().UnixNano()`; mehrere Aufrufe innerhalb derselben Nanosekunde erzeugen denselben Key, sodass der zweite Create-Versuch als Duplikat abgelehnt wird.
- **Repro**: `go test ./...` ausführen (bei schneller Testausführung bzw. mehreren Aufrufen in derselben Nanosekunde).
- **Evidence**:
  ```
  --- FAIL: TestEvaluateRolloutBoundaries (0.00s)
      flags_api_test.go:326: create rollout 100 -> 409
  ```
- **Suspected file(s)**: `flags_api_test.go` – `uniqueKey()` ist nicht garantiert kollisionsfrei; zusätzlich könnte der Test selbst einen bereits existierenden Key verwenden. Der Produktcode verhält sich gemäß Logs korrekt (409 bei Duplikat), aber die Testisolation ist unzureichend.
- **Severity**: high