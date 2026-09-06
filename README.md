# Feature-Flag-Service

Ein eigenständiger Feature-Flag-Service als REST-API, geschrieben in Go und
ausschließlich mit der Standardbibliothek (`net/http`, kein externes
Web-Framework). Flags lassen sich anlegen, auflisten, einzeln lesen, ändern und
löschen; zusätzlich liefert ein Evaluate-Endpunkt eine deterministische
Ja/Nein-Entscheidung pro Nutzer auf Basis eines stabilen Hashs gegen den
Rollout-Prozentsatz. Die Daten liegen in einem thread-sicheren In-Memory-Store.

## Tech Stack

- **Sprache**: Go (go.mod mit `go 1.22`)
- **Framework**: `net/http` (Standardbibliothek, Routing über `http.ServeMux` mit Methodenmustern)
- **Storage**: In-Memory mit `sync.RWMutex`
- **Testing**: `httptest` / `go test`

## Installation

Voraussetzung ist eine Go-Installation (≥ 1.22). Keine externen Abhängigkeiten —
das Projekt nutzt ausschließlich die Standardbibliothek, daher sind keine
`go get`-Schritte nötig.

## Starten (Entwicklung)

```sh
go run .
```

Der Service lauscht auf dem Port aus der Umgebungsvariable `PORT` (Default `8080`):

```sh
PORT=8080 go run .
```

## Build (Produktion)

```sh
go build ./...
```

## Konfiguration

| Variable | Beschreibung | Default |
|----------|--------------|---------|
| `PORT`   | Port, auf dem der Service lauscht | `8080` |

## Endpunkte

| Methode | Pfad | Beschreibung | Antwort |
|---------|------|--------------|---------|
| `GET`  | `/healthz` | Health-Check | `200 {"status":"ok"}` |
| `POST` | `/flags` | Flag anlegen (`{key, enabled, description?, rollout_percent?}`) | `201` Flag / `400`/`409`/`413` `{"error":...}` |
| `GET`  | `/flags` | Alle Flags, sortiert nach `key` | `200 [Flag]` |
| `GET`  | `/flags/{key}` | Einzelnes Flag | `200` Flag / `404` `{"error":...}` |
| `PUT`  | `/flags/{key}` | Flag ändern (`{enabled?, description?, rollout_percent?}`) | `200` Flag / `400`/`404` |
| `DELETE` | `/flags/{key}` | Flag löschen | `204` / `404` `{"error":...}` |
| `GET`  | `/flags/{key}/evaluate?user={id}` | Deterministische Entscheidung | `200 {"result":bool}` / `400`/`404` |

### Flag-Datenmodell

```json
{
  "id": 1,
  "key": "my-flag",
  "enabled": true,
  "description": "Beschreibung",
  "rollout_percent": 50
}
```

Alle Fehlerantworten haben exakt die Form `{"error": "Meldung"}`.

### Beispiel

```sh
curl -X POST http://localhost:8080/flags \
  -H "Content-Type: application/json" \
  -d '{"key":"new-feature","enabled":true,"rollout_percent":100}'

curl http://localhost:8080/flags
curl http://localhost:8080/flags/new-feature
curl "http://localhost:8080/flags/new-feature/evaluate?user=alice"
```

## Features

- CRUD-Endpunkte für Feature-Flags mit In-Memory-Store
- Deterministischer Rollout über einen Evaluate-Endpunkt
- JSON-Fehlerobjekte ohne Stacktraces oder interne Details
- Logging ausschließlich von Methode, Pfad und Statuscode (kein Query-String)
- Body-Größenbegrenzung (1 MiB) für `POST /flags` und `PUT /flags/{key}`
- Server-Timeouts gegen hängende Clients
