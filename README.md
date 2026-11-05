# anystatus

A tiny Go service that answers any request to `/http?code=NNN` with the HTTP
status `NNN`. Built as a deployment-test stub: point your probes, health
checks, alerting rules or retry logic at it and get exactly the status code
you want to exercise — no auth, no state, no surprises.

## Run

```bash
go run .                 # listens on :8080
PORT=9000 go run .       # or: go run . -port 9000
```

## Use

```bash
curl -i 'http://localhost:8080/http?code=503'   # -> HTTP/1.1 503 Service Unavailable
curl -i 'http://localhost:8080/http?code=418'   # -> HTTP/1.1 418 I'm a teapot
curl -i 'http://localhost:8080/http'            # -> 200 OK (code defaults to 200)
curl -i 'http://localhost:8080/http?code=nope'  # -> 400 Bad Request
```

Rules:

- `code` must be an integer in **200–599**; anything else → `400 Bad Request`.
- Missing or empty `code` → `200 OK`.
- `204`, `205` and `304` are sent without a body (RFC 9110).
- Any HTTP method is accepted (`GET`, `POST`, `HEAD`, …); the answer is the same.
- Any other path returns a one-line usage hint with status 200.

## Docker

```bash
docker build -t anystatus .
docker run -p 8080:8080 anystatus
```

Multi-stage build; the final image is distroless static (~2 MB), runs as
non-root, listens on `$PORT` (default 8080).

## Configuration

| Knob | Default | Meaning |
|---|---|---|
| `-port` flag | `$PORT` env, else `8080` | listen port |

## Development

```bash
go vet ./... && go test ./...
```
