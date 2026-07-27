# Frontend Reverse Proxy Design

**Date:** 2026-07-27  
**Status:** Implemented

## Goal

Add a new Go service under `src/frontend` that acts as the public entry point for carinfo. It reverse-proxies requests to either the API backend (`app-api`) or the web UI backend (`webapp`) based on path prefix.

Image name: `frontend` (`quay.io/ooichman/carinfo/frontend:latest`).

## Routing

| Condition | Backend | Path rewrite |
|-----------|---------|--------------|
| Path is `/api` or starts with `/api/` | `APP_API_URL` (app-api) | Strip `/api` prefix (`/api/v1` → `/v1`; `/api` or `/api/` → `/`) |
| Everything else | `WEB_URL` (webapp) | Unchanged |

Exact match: only paths equal to `/api` or with prefix `/api/` count as API. A path like `/apiv2` is **not** treated as API and goes to webapp.

This is a reverse proxy (request is forwarded and the upstream response is returned). It is not an HTTP 302/307 redirect.

## Architecture

```
Client
  │
  ▼
frontend (:8080)
  ├─ /api/*  → APP_API_URL  (default http://app-api:8080), strip /api
  └─ /*      → WEB_URL      (default http://webapp:8080)
```

## Components

### `src/frontend/`

- `go.mod` — module consistent with other services (e.g. `carinfo` / Go 1.16+)
- `main.go` — HTTP server and reverse-proxy routing
  - Use `net/http/httputil.ReverseProxy`
  - Single catch-all handler that chooses upstream by path
  - Env helpers matching the style used in `app-api`

### Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | Listen port |
| `APP_API_URL` | `http://app-api:8080` | Upstream app-api base URL |
| `WEB_URL` | `http://webapp:8080` | Upstream webapp base URL |

### `Containerfile.frontend`

Same pattern as `Containerfile.app-api`:

- Build stage: UBI9 go-toolset, build from `src/frontend`, output `/tmp/frontend`
- Runtime stage: UBI9 minimal, binary at `/usr/bin/frontend`, user `1001`, expose `8080`

### `Deploy/frontend/`

- `deployment.yaml` — image `quay.io/ooichman/carinfo/frontend:latest`, env for `PORT`, `APP_API_URL`, `WEB_URL`
- `service.yaml` — ClusterIP on 8080
- `route.yaml` — OpenShift Route targeting the frontend service (public entry)

## Error handling

- Upstream unreachable or proxy error → respond `502 Bad Gateway`; log the error
- Invalid / unparseable `APP_API_URL` or `WEB_URL` at startup → fatal exit

## Out of scope

- Implementing or containerizing the webapp itself
- Authentication, TLS termination inside the Go process, rate limiting, caching
- Helm chart updates beyond what’s needed if filenames/values already exist for a prior frontend

## Success criteria

- Building `Containerfile.frontend` produces a runnable `frontend` image
- Requests to `/api/v1` are proxied to app-api as `/v1`
- Requests to `/` (and other non-`/api` paths) are proxied to webapp unchanged
- Deploy manifests reference image `quay.io/ooichman/carinfo/frontend:latest`
