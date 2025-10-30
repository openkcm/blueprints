#!/usr/bin/env bash
set -euo pipefail

# init-unseal.sh - Initialize and unseal a local OpenBao instance using mTLS.
# Required env vars (with defaults if exported before invoking):
#   OPENBAO_ADDR           - URL to Bao server (https://127.0.0.1:8200)
#   OPENBAO_INIT_SHARES    - Number of shares (default 1)
#   OPENBAO_INIT_THRESHOLD - Threshold (default 1)
#   OPENBAO_CA_CERT        - Path to CA cert
#   OPENBAO_CLIENT_CERT    - Path to client cert
#   OPENBAO_CLIENT_KEY     - Path to client key
# Output: JSON written to openbao-init-output.json containing root token & unseal key.

ADDR=${OPENBAO_ADDR:-https://127.0.0.1:8200}
SHARES=${OPENBAO_INIT_SHARES:-1}
THRESHOLD=${OPENBAO_INIT_THRESHOLD:-1}
CA=${OPENBAO_CA_CERT:?OPENBAO_CA_CERT required}
CERT=${OPENBAO_CLIENT_CERT:?OPENBAO_CLIENT_CERT required}
KEY=${OPENBAO_CLIENT_KEY:?OPENBAO_CLIENT_KEY required}
OUT_FILE=${OPENBAO_TOKEN_FILE:-openbao-init-output.json}

curl_opts=(--silent --show-error --fail --cacert "$CA" --cert "$CERT" --key "$KEY")

# Check initialization status
status_json=$(curl "${curl_opts[@]}" -X GET "$ADDR/v1/sys/init" || true)
initialized=$(echo "$status_json" | jq -r '.initialized // .Initialized // empty')

if [[ "$initialized" == "false" || -z "$initialized" ]]; then
  echo "[init-unseal] Performing initialization (shares=$SHARES threshold=$THRESHOLD)" >&2
  init_payload=$(jq -n --argjson shares "$SHARES" --argjson threshold "$THRESHOLD" '{secret_shares: $shares, secret_threshold: $threshold}')
  init_resp=$(curl "${curl_opts[@]}" -X POST -d "$init_payload" "$ADDR/v1/sys/init")
  echo "$init_resp" > "$OUT_FILE"
  unseal_key=$(echo "$init_resp" | jq -r '.keys[0] // .unseal_keys_b64[0]')
  root_token=$(echo "$init_resp" | jq -r '.root_token')
else
  echo "[init-unseal] Already initialized; retrieving unseal key requires prior storage (skipping)" >&2
  echo '{}' > "$OUT_FILE"
  # Attempt to load previously stored init output if exists
fi

# Attempt unseal if sealed
seal_status=$(curl "${curl_opts[@]}" -X GET "$ADDR/v1/sys/seal-status")
sealed=$(echo "$seal_status" | jq -r '.sealed // .Sealed')
if [[ "$sealed" == "true" ]]; then
  if [[ -z "${unseal_key:-}" ]]; then
    echo "[init-unseal] ERROR: instance sealed but no unseal key available" >&2
    exit 1
  fi
  echo "[init-unseal] Unsealing" >&2
  unseal_payload=$(jq -n --arg key "$unseal_key" '{key: $key}')
  curl "${curl_opts[@]}" -X POST -d "$unseal_payload" "$ADDR/v1/sys/unseal" >/dev/null
fi

# Final health check
health=$(curl "${curl_opts[@]}" -X GET "$ADDR/v1/sys/health" || true)
status_code=$(curl -o /dev/null -w '%{http_code}' "${curl_opts[@]}" "$ADDR/v1/sys/health" || true)

echo "[init-unseal] Health status code: $status_code" >&2
if [[ $status_code -ge 500 ]]; then
  echo "[init-unseal] WARNING: health endpoint returned $status_code" >&2
fi

echo "[init-unseal] Done" >&2
