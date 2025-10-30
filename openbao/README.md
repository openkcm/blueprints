# OpenBao Tool

A lightweight Go utility and REST API server for interacting with an OpenBao cluster using mutual TLS (mTLS). It supports:

- Health checks
- Namespace create/delete/list (falls back to root-only if namespaces unsupported)
- Transit key lifecycle (list/create/rotate/delete) with automatic mount enable
- KV v2 secret lifecycle (enable mount, write/read/delete)
- Automatic key configuration (`deletion_allowed=true`) so keys can be deleted
- REST API server mode (simple `http.ServeMux` handlers)
- Kubernetes manifests (two deployments using distinct client certs: namespace-admin & key-admin)
- Makefile targets for local development AND in-cluster port-forwarded tests

## Contents
- `main.go` – CLI + REST server implementation
- `openbao-dev.hcl` – Local server config (verified client certs)
- `init-unseal.sh` – Automates init & unseal for local dev
- `Dockerfile` – Multi-stage build producing a distroless static image
- `deploy.yaml` – Namespace + two Deployments + Services (namespace vs key admin)
- `Makefile` – Local & Kubernetes test automation

## Building Locally
```sh
cd openbao
make build
```
Binary produced: `openbao-tool`.

## Generating Dev Certificates & Running Local Server
```sh
make bao-cert-gen            # CA/server/client certs
make bao-server              # Starts OpenBao (foreground)
make bao-init                # Init & unseal (stores root token JSON)
make bao-health              # Health via tool
make bao-curl-health         # Health via curl with mTLS
make bao-test-all            # Full ephemeral workflow (auto cleanup)
```
Artifacts created:
- `certs/ca.pem`, `certs/server.crt`, `certs/client.crt` etc.
- `openbao-init-output.json` (contains root token & unseal key)

## CLI Operations
All operations share flags for mTLS and token authentication.

Important flags:
- `--addr` (default `https://localhost:8200`)
- `--ca`, `--cert`, `--key` (mTLS paths)
- `--token` (root/client token)
- `--op` – Operation selector
- `--namespace` – Target namespace (may be empty for root)
- `--transit` – Transit mount path (default `transit`)
- `--key-name`, `--key-type` – Transit key identifiers
- KV flags: `--secret-mount` (default `secret`), `--secret-name`, `--secret-data`

Operations:
```
health | create-namespace | delete-namespace | list-namespaces |
list-keys | create-key | delete-key | rotate-key |
ensure-key-all-namespaces | enable-kv | put-secret | read-secret | delete-secret
```
Example create key in namespace:
```sh
./openbao-tool --op=create-key --namespace=team-a --key-name=demo \
  --addr=https://localhost:8200 --ca=certs/ca.pem --cert=certs/client.crt --key=certs/client.key --token=$(jq -r .root_token openbao-init-output.json)
```

## REST API Server Mode
Enable with `--server` (or env `OPENBAO_SERVER_MODE=1`). Listen address via `--listen` (default `:8080`). Endpoints:

| Endpoint | Method(s) | Description |
|----------|-----------|-------------|
| `/health` | GET | Health probe (Bao sys/health) |
| `/namespaces` | GET, POST | List namespaces or create one |
| `/namespaces/{name}` | DELETE | Delete namespace |
| `/keys` | GET, POST | List transit keys (all or by ?namespace=) / create key (supports `allow_delete`) |
| `/keys/{namespace}/{name}` | GET, DELETE | Get key metadata or delete key |
| `/keys/{namespace}/{name}/rotate` | POST | Rotate a transit key |
| `/ensure-key-all-namespaces` | POST | Ensure a key exists in all namespaces (auto-mount transit) |
| `/secrets` | GET, POST | KV v2 secret read (query params) / write (JSON body) |
| `/secrets/{namespace}/{name}` | DELETE | Delete KV secret metadata |

Notes:
- If namespaces unsupported, listing returns `[""]` (root only) and operations continue.
- Transit mount auto-enabled by `ensureTransitMounted` helper.
- Key creation sets `deletion_allowed=true` and can override via `allow_delete` field.

### Sample REST Calls (local)
```sh
TOKEN=$(jq -r .root_token openbao-init-output.json)
PORT=8080
# Start server
./openbao-tool --server --listen=:8080 --addr=https://localhost:8200 --ca=certs/ca.pem --cert=certs/client.crt --key=certs/client.key --token=$TOKEN &
# Create namespace
curl -s -X POST -H 'Content-Type: application/json' -d '{"name":"dev-ns"}' http://localhost:$PORT/namespaces
# Create key with deletion
curl -s -X POST -H 'Content-Type: application/json' -d '{"namespace":"dev-ns","name":"demo","allow_delete":true}' http://localhost:$PORT/keys
# List keys
curl -s "http://localhost:$PORT/keys?namespace=dev-ns"
# Rotate
curl -s -X POST http://localhost:$PORT/keys/dev-ns/demo/rotate
# Delete key
curl -s -X DELETE http://localhost:$PORT/keys/dev-ns/demo
# Delete namespace
curl -s -X DELETE http://localhost:$PORT/namespaces/dev-ns
```

## Kubernetes Deployment
`deploy.yaml` provisions:
- Namespace `openbao`
- Two Deployments / Services:
  - `openbao-namespace-admin` (client cert secret: `namespace-admin-client-cert`)
  - `openbao-key-admin` (client cert secret: `key-admin-client-cert`)
- Both run server mode on port 8080 exposed via Service port 80.

Apply:
```sh
kubectl apply -f openbao/deploy.yaml
kubectl -n openbao get pods
```

### In-Cluster Tests (Port-forward)
Makefile targets (require working cluster + secrets):
- `make bao-k8s-ns-test` – Namespace create/list/delete.
- `make bao-k8s-key-test` – Root namespace key lifecycle.
- `make bao-k8s-ns-key-test` – Combined test: create namespace via namespace-admin, run key lifecycle via key-admin, then delete namespace.

Combined test example output shows each JSON body plus status; unsupported features (like namespaces or transit list) are gracefully skipped.

## Environment Variables (Server / CLI)
| Variable | Purpose | Default |
|----------|---------|---------|
| `OPENBAO_ADDR` | OpenBao server address | `https://localhost:8200` |
| `OPENBAO_CA_CERT` | CA cert path | `/certs/ca.pem` |
| `OPENBAO_CLIENT_CERT` | Client cert path | `/certs/client.crt` |
| `OPENBAO_CLIENT_KEY` | Client key path | `/certs/client.key` |
| `OPENBAO_TOKEN` | Auth token | (empty) |
| `OPENBAO_SERVER_MODE` | Run REST server if set | unset |
| `OPENBAO_LISTEN` | Server listen address | `:8080` |
| `OPENBAO_TRANSIT_PATH` | Transit mount path | `transit` |
| `OPENBAO_SECRET_MOUNT` | KV v2 mount path | `secret` |

## Key Deletion
Transit key deletion requires `deletion_allowed=true`. The tool:
1. Sends it in the initial create body.
2. Calls `/keys/<name>/config` to enforce it if the create request ignored it.
You can override by sending `{"allow_delete": false}` in the create POST body to disable deletion.

## Limitations / Technical Debt
- Uses `client.RawRequest` (deprecated in API) – future refactor to logical client methods planned.
- No listing of secrets (only single secret read/write/delete). Could add `/secrets/list` endpoint.
- No auth policy differentiation logic (handled outside of this code via cert issuance).
- Basic error mapping: many backend failures map to 502; could standardize error codes.

## Development Tips
- Ensure Go version matches module `go.mod` requirement.
- Regenerate binary after edits: `make build`.
- Rebuild container when changing code: `docker build -t <tag> openbao/` (or rely on CI workflow).

## CI/CD
A GitHub Actions workflow (outside this folder) builds and pushes the image to GHCR (`ghcr.io/openkcm/openbao-tool:<sha>` + `latest`). Update `deploy.yaml` image tag accordingly.

## Troubleshooting
| Symptom | Cause | Fix |
|---------|-------|-----|
| 404 unsupported path for `/v1/sys/namespaces` | Namespaces disabled | Tool falls back to root namespace automatically |
| 404 listing transit keys | Transit engine not mounted | Auto-mount via `ensureTransitMounted`; redeploy if persistent |
| Key delete fails | `deletion_allowed` not set | Ensure recent binary; check key config endpoint response |
| jq parse errors in tests | Control characters / non-JSON body | Debug raw output printed; inspect server logs |

## License
See top-level `LICENSE` and `REUSE.toml`.

---
Generated README capturing current capabilities. Update as features evolve.
