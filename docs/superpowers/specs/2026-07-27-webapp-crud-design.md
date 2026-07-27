# Webapp + dbapi CRUD Design

**Date:** 2026-07-27  
**Status:** Implemented  
**Visual style:** Night Circuit (dark garage, lime/amber accents)

## Goal

Build a Go **webapp** that serves HTML pages for car inventory (list, add, edit, delete) in a colorful Night Circuit design with a **CarInfo** headline. The webapp talks only to **dbapi** (not MariaDB directly). Extend dbapi with JSON CRUD endpoints for cars. Ship container and deploy manifests for webapp.

## Architecture

```
Browser → frontend (existing reverse proxy) → webapp (:8080)
                                              │
                                              │ HTTP JSON via DBAPI_URL
                                              ▼
                                            dbapi (:8080) → MariaDB
```

Keep existing dbapi `POST /query` behavior unchanged.

## dbapi API additions

| Method | Path | Body / notes | Response |
|--------|------|--------------|----------|
| GET | `/cars` | — | JSON array of cars |
| GET | `/cars/{id}` | — | Single car JSON; 404 if missing |
| POST | `/cars` | JSON car fields (no id) | Created car JSON (201) |
| PUT | `/cars/{id}` | JSON car fields | Updated car JSON |
| DELETE | `/cars/{id}` | — | 204 No Content |

### Car JSON shape

```json
{
  "id": 1,
  "name": "nency",
  "year": 1983,
  "condition": "new",
  "reason": "out of the factory",
  "module": "B5252S",
  "manufacture": "Volvo"
}
```

Mapped from tables `cars` + `cars_vendors` (join on `vendor_id`). Create/update resolve or create vendor by `manufacture` name as needed so inserts remain practical.

Use parameterized SQL (no string-concat queries for new endpoints).

## webapp

### Layout

Replace stub content under `src/webapp/` with:

| Path | Role |
|------|------|
| `main.go` | HTTP server, routes, template load |
| `handlers.go` | Page + form handlers; call dbapi client |
| `dbapi_client.go` | Thin HTTP client for dbapi CRUD |
| `go.mod` | Module `carinfo` (or `carinfo/webapp`), Go 1.16+ |
| `html/layout.html` | Shell, CarInfo brand, CSS link |
| `html/cars.html` | Table + actions |
| `html/car_form.html` | Shared add/edit form |
| `html/static/style.css` | Night Circuit theme |

Remove or overwrite empty legacy `templates/` stubs as part of cleanup (prefer `html/` as the skeleton directory).

### Routes

| Method | Path | Behavior |
|--------|------|----------|
| GET | `/` | List cars table |
| GET | `/cars/new` | Empty add form |
| POST | `/cars` | Create via dbapi; redirect `/` |
| GET | `/cars/{id}/edit` | Prefill edit form |
| POST | `/cars/{id}` | Update via dbapi; redirect `/` |
| POST | `/cars/{id}/delete` | Delete via dbapi; redirect `/` |
| GET | `/static/*` | Serve `html/static` |

### Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | Listen port |
| `DBAPI_URL` | `http://dbapi:8080` | dbapi base URL |

### UI (Night Circuit)

- Dark background with subtle green/purple radial accents
- **CarInfo** as hero-level lime→amber gradient headline
- Table with lime header accents; condition shown in amber/red/green tones
- Add / Edit / Delete actions on the list page
- Forms match the same theme
- Desktop and mobile readable (simple responsive table)

### Errors

- dbapi unreachable or non-2xx: show error banner in HTML layout; log details server-side
- Invalid form input: re-render form with message

## Containers & deploy

- `Containerfile.webapp` — same UBI go-toolset pattern as other services; binary `webapp`; embed/copy `html/`
- Image: `quay.io/ooichman/carinfo/webapp:latest`
- `Deploy/webapp/` — Deployment, Service (no public Route required if frontend is the entry; optional Route for direct access)
- Env on Deployment: `PORT`, `DBAPI_URL=http://dbapi:8080`
- Rebuild/redeploy **dbapi** image so new `/cars` endpoints are available

Frontend already defaults `WEB_URL` to `http://webapp:8080` — no frontend code change required unless service name differs.

## Out of scope

- Auth / multi-user
- Changing MariaDB schema beyond what seed data already provides
- SPA / heavy JavaScript frameworks
- Helm chart overhaul (optional later)

## Success criteria

- `GET /` via webapp shows CarInfo headline and a populated table from dbapi `GET /cars`
- Add / edit / delete round-trip through dbapi and refresh the table
- Night Circuit styling applied from `html/static/style.css`
- `Containerfile.webapp` builds; Deploy manifests reference `webapp` image
- Existing `POST /query` on dbapi still works
