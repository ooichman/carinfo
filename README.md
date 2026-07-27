# CarInfo

Car inventory application with a public frontend gateway, HTML web UI, JSON API, and MariaDB-backed dbapi.

## Architecture

```
Browser
  │
  ▼
frontend (:8080)          ← only public Route / Ingress
  ├─ /api/*  → app-api   (JSON API, strips /api)
  └─ /*      → webapp    (HTML UI)
       │
       ▼
     dbapi → MariaDB
```

| Component | Image | Role |
|-----------|-------|------|
| frontend | `quay.io/ooichman/carinfo/frontend:latest` | Reverse proxy entry point |
| app-api | `quay.io/ooichman/carinfo/app-api:latest` | JSON API gateway to dbapi (HTTP 8080 + HTTPS 8443) |
| webapp | `quay.io/ooichman/carinfo/webapp:latest` | HTML CRUD UI (HTTP 8080 + HTTPS 8443) |
| dbapi | `quay.io/ooichman/carinfo/dbapi:latest` | Database API |
| mariadb | `mariadb:latest` | Database |

All application traffic goes through **frontend**. Do not expose app-api or webapp Routes publicly.

## Build images

From the repository root:

```bash
podman build -f Containerfile.frontend -t quay.io/ooichman/carinfo/frontend:latest .
podman build -f Containerfile.app-api   -t quay.io/ooichman/carinfo/app-api:latest .
podman build -f Containerfile.webapp    -t quay.io/ooichman/carinfo/webapp:latest .
podman build -f Containerfile.dbapi     -t quay.io/ooichman/carinfo/dbapi:latest .

podman push quay.io/ooichman/carinfo/frontend:latest
podman push quay.io/ooichman/carinfo/app-api:latest
podman push quay.io/ooichman/carinfo/webapp:latest
podman push quay.io/ooichman/carinfo/dbapi:latest
```

Images under `quay.io/ooichman/carinfo/*` must be **public** (or the cluster needs a pull secret).

## Deploy with Kustomize

Manifests live under `Deploy/`:

```text
Deploy/
  base/                 # shared Deployments, Services, PVC
  overlays/
    openshift/          # base + frontend Route (+ serving certs for TLS)
    kubernetes/         # base + Ingress to frontend
```

### OpenShift

```bash
oc new-project carinfo
oc apply -k Deploy/overlays/openshift

oc get pods,svc,route -n carinfo
oc get route frontend -n carinfo -o jsonpath='{.spec.host}{"\n"}'
```

Example URL:

```text
https://frontend-carinfo.apps.<cluster-domain>
```

OpenShift automatically creates TLS secrets for `app-api` and `webapp` (`app-api-tls`, `webapp-tls`) via service serving certificates. Set `USE_TLS=true` (default in Deploy manifests) plus `TLS_CERT_FILE` / `TLS_KEY_FILE` to enable HTTPS on port 8443. Frontend still talks to them over HTTP on port 8080 inside the cluster.

### Kubernetes

```bash
kubectl create namespace carinfo
kubectl apply -k Deploy/overlays/kubernetes

kubectl get pods,svc,ingress -n carinfo
```

Without OpenShift serving certs, app-api/webapp run HTTP-only unless you mount your own TLS secrets named `app-api-tls` / `webapp-tls`.

### Rollout after image updates

```bash
oc rollout restart deploy/frontend deploy/app-api deploy/webapp deploy/dbapi -n carinfo
# or
kubectl rollout restart deploy/frontend deploy/app-api deploy/webapp deploy/dbapi -n carinfo
```

## Web UI

Open the frontend URL in a browser:

- Fleet table at `/`
- Add car at `/cars/new`
- Edit / delete from the table actions

## API CRUD with curl

All API calls go through the **frontend** `/api` prefix (proxied to app-api `/v1/...`).

Set your frontend base URL:

```bash
export FE="https://frontend-carinfo.apps.<cluster-domain>"
# local / port-forward example:
# export FE="http://localhost:8080"
```

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

### List cars

```bash
curl -sk "$FE/api/v1/cars"
```

### Get one car

```bash
curl -sk "$FE/api/v1/cars/1"
```

### Create a car

```bash
curl -sk -X POST "$FE/api/v1/cars" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "TestRacer",
    "year": 2024,
    "condition": "new",
    "reason": "demo create",
    "module": "V8",
    "manufacture": "Ferrari"
  }'
```

Returns `201` with the created car (includes `id`).

### Update a car

```bash
curl -sk -X PUT "$FE/api/v1/cars/1" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "nency",
    "year": 1983,
    "condition": "mid condition",
    "reason": "updated via api",
    "module": "B5252S",
    "manufacture": "Volvo"
  }'
```

### Delete a car

```bash
curl -sk -X DELETE "$FE/api/v1/cars/1" -w "\nHTTP %{http_code}\n"
```

Returns `204` on success.

### Legacy query (optional)

Manufacturer query still available:

```bash
curl -sk -X POST "$FE/api/v1" \
  -H 'Content-Type: application/json' \
  -d '{"module":"B5252S","manufacture":"Volvo"}'
```

## Quick health checks

```bash
# UI
curl -sk -o /dev/null -w "%{http_code}\n" "$FE/"

# API list
curl -sk -o /dev/null -w "%{http_code}\n" "$FE/api/v1/cars"

# Pods
oc get pods -n carinfo
```

## License

See [LICENSE](LICENSE).
