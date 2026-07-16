#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${VEC_SMOKE_BASE_URL:-${BOLSA_SMOKE_BASE_URL:-http://127.0.0.1:18082}}"
ARTIFACT_DIR="${VEC_SMOKE_ARTIFACT_DIR:-${BOLSA_SMOKE_ARTIFACT_DIR:-"$ROOT_DIR/var/smoke"}}"
MANAGED="${VEC_SMOKE_MANAGED:-${BOLSA_SMOKE_MANAGED:-0}}"
DATA_DIR="${VEC_SMOKE_DATA_DIR:-${BOLSA_SMOKE_DATA_DIR:-"$ROOT_DIR/var/bolsa"}}"
SERVER_PID=""

mkdir -p "$ARTIFACT_DIR" "$DATA_DIR"

cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

json_assert() {
  local file="$1"
  local expr="$2"
  jq -e "$expr" "$file" >/dev/null
}

wait_health() {
  local base="$1"
  for _ in $(seq 1 50); do
    if curl -fsS "$base/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "server did not become healthy at $base" >&2
  return 1
}

start_managed_server() {
  local addr="${BASE_URL#http://}"
  local bin="$ARTIFACT_DIR/vec-server-smoke"
  (cd "$ROOT_DIR" && go build -buildvcs=false -o "$bin" ./cmd/vec-server)
  VEC_HTTP_ADDR="$addr" \
    VEC_BOLSA_STORAGE_MODE=local_durable \
    VEC_BOLSA_DATA_DIR="$DATA_DIR" \
    VEC_AUTH_MODE=trusted_headers \
    VEC_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128 \
    "$bin" >"$ARTIFACT_DIR/server.log" 2>&1 &
  SERVER_PID="$!"
  wait_health "$BASE_URL"
}

restart_managed_server() {
  if [[ -z "$SERVER_PID" ]]; then
    return 0
  fi
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
  SERVER_PID=""
  start_managed_server
}

require_cmd curl
require_cmd jq
require_cmd go
require_cmd node
require_cmd python3

if [[ "${VEC_SMOKE_SKIP_TESTS:-${BOLSA_SMOKE_SKIP_TESTS:-0}}" != "1" ]]; then
  (cd "$ROOT_DIR" && go test -count=1 ./...)
  (cd "$ROOT_DIR" && node --input-type=module --check < web/static/app.js)
  (cd "$ROOT_DIR" && node --test web/static/modulos/cronos/resumen.test.mjs)
  (cd "$ROOT_DIR" && python3 -m json.tool locales/es.json >/dev/null)
fi

if [[ "$MANAGED" == "1" ]]; then
  start_managed_server
else
  wait_health "$BASE_URL"
fi

run_ref="smoke-$(date +%Y%m%d%H%M%S)"
candidate_id="cand-$run_ref"
solicitud_id="sol-$run_ref"

staff_headers=(
  -H "Content-Type: application/json"
  -H "X-Auth-Mechanism: kerberos_ad"
  -H "X-Auth-Subject: staff"
  -H "X-Auth-Roles: tecnico_rrhh"
  -H "X-VEC-Auth-Mechanism: kerberos_ad"
  -H "X-VEC-Subject: staff"
  -H "X-VEC-Roles: validator_l1"
)
candidate_headers=(
  -H "Content-Type: application/json"
  -H "X-Auth-Mechanism: clave"
  -H "X-Auth-Subject: $candidate_id"
  -H "X-Auth-Roles: ciudadano"
  -H "X-VEC-Auth-Mechanism: clave"
  -H "X-VEC-Subject: $candidate_id"
  -H "X-VEC-Roles: candidate"
)

curl -fsS "$BASE_URL/healthz" >"$ARTIFACT_DIR/health.json"
curl -fsS "$BASE_URL/api/vec/modules" "${staff_headers[@]}" >"$ARTIFACT_DIR/vec_modules.json"
json_assert "$ARTIFACT_DIR/vec_modules.json" '(.data.modules|map(.id)|index("vec.module.bolsa")) != null'
curl -fsS "$BASE_URL/api/vec/workspace" "${staff_headers[@]}" >"$ARTIFACT_DIR/vec_workspace.json"
json_assert "$ARTIFACT_DIR/vec_workspace.json" '(.data.modules|length) >= 4'
curl -fsS "$BASE_URL/api/vec/menu" "${staff_headers[@]}" >"$ARTIFACT_DIR/vec_menu.json"
json_assert "$ARTIFACT_DIR/vec_menu.json" '(.data.menu|length) > 0'
curl -fsS "$BASE_URL/api/modules/bolsa" "${staff_headers[@]}" >"$ARTIFACT_DIR/bolsa_manifest.json"
json_assert "$ARTIFACT_DIR/bolsa_manifest.json" '.data.module_ref=="vec.module.bolsa"'
json_assert "$ARTIFACT_DIR/bolsa_manifest.json" '(.data.http_routes|map(.route)|index("/api/candidates/{id}/documents")) != null'
curl -fsS "$BASE_URL/api/admin/status" "${staff_headers[@]}" >"$ARTIFACT_DIR/admin_status.json"
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.status=="operational"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.auth_mode=="trusted_headers"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.persistence_mode=="file"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.legal_production_ready==false'

curl -fsS -X POST "$BASE_URL/api/candidates" "${candidate_headers[@]}" \
  -d "{\"id\":\"$candidate_id\",\"dni\":\"12345678A\",\"nombre\":\"Ana Perez\",\"email\":\"ana@example.test\"}" \
  >"$ARTIFACT_DIR/candidate.json"

curl -fsS -X POST "$BASE_URL/api/candidates/$candidate_id/merits" "${candidate_headers[@]}" \
  -d '{"id":"merit-smoke","tipo":"formacion_curso","datos":{"horas":20},"estado":"Validado"}' \
  >"$ARTIFACT_DIR/merit.json"

curl -fsS -X POST "$BASE_URL/api/candidates/$candidate_id/baremo" "${candidate_headers[@]}" \
  >"$ARTIFACT_DIR/baremo.json"
json_assert "$ARTIFACT_DIR/baremo.json" '.data.total_points==1'

curl -fsS -X POST "$BASE_URL/api/candidates/$candidate_id/documents" "${candidate_headers[@]}" \
  -d "{\"id\":\"doc-smoke\",\"solicitud_id\":\"$solicitud_id\",\"procedure_id\":\"proc-smoke\",\"purpose\":\"Alegacion\",\"csv\":\"CSV-SMOKE-DOC\",\"digest_sha256\":\"abc123\",\"storage_object_ref\":\"obj-smoke\",\"document_type\":\"alegacion\",\"title\":\"Titulo smoke\",\"format\":\"application/pdf\",\"signature_ref\":\"sig-smoke\"}" \
  >"$ARTIFACT_DIR/document.json"

curl -fsS -X POST "$BASE_URL/api/candidates/$candidate_id/claims" "${candidate_headers[@]}" \
  -d "{\"id\":\"claim-smoke\",\"solicitud_id\":\"$solicitud_id\",\"text\":\"Revisar baremo\",\"receipt_csv\":\"CSV-SMOKE-CLAIM\"}" \
  >"$ARTIFACT_DIR/claim.json"

curl -fsS -X POST "$BASE_URL/api/notifications" "${staff_headers[@]}" \
  -d "{\"candidate_id\":\"$candidate_id\",\"id\":\"note-smoke\",\"solicitud_id\":\"$solicitud_id\",\"type\":\"subsanacion\",\"subject\":\"Aportar titulo\",\"body\":\"Revise evidencia\"}" \
  >"$ARTIFACT_DIR/notification.json"

curl -fsS -X POST "$BASE_URL/api/notifications/note-smoke/send" "${staff_headers[@]}" \
  -d "{\"notification_id\":\"note-smoke\",\"csv\":\"CSV-SMOKE-SEND\",\"recipient_id\":\"$candidate_id\",\"channel\":\"sede\"}" \
  >"$ARTIFACT_DIR/notification_send.json"

curl -fsS -X POST "$BASE_URL/api/notifications/note-smoke/read" "${staff_headers[@]}" \
  -d "{\"notification_id\":\"note-smoke\",\"csv\":\"CSV-SMOKE-READ\",\"recipient_id\":\"$candidate_id\",\"channel\":\"sede\"}" \
  >"$ARTIFACT_DIR/notification_read.json"

curl -fsS "$BASE_URL/api/audit?candidate_id=$candidate_id" "${staff_headers[@]}" \
  >"$ARTIFACT_DIR/audit.json"
json_assert "$ARTIFACT_DIR/audit.json" '(.data|length) >= 5'
json_assert "$ARTIFACT_DIR/audit.json" '.data[-1].action=="candidate.notification.read"'

if [[ "$MANAGED" == "1" ]]; then
  restart_managed_server
  curl -fsS -X POST "$BASE_URL/api/candidates/$candidate_id/baremo" "${candidate_headers[@]}" \
    >"$ARTIFACT_DIR/restart_baremo.json"
  json_assert "$ARTIFACT_DIR/restart_baremo.json" '.data.total_points==1'
  curl -fsS "$BASE_URL/api/audit?candidate_id=$candidate_id" "${staff_headers[@]}" \
    >"$ARTIFACT_DIR/restart_audit.json"
  json_assert "$ARTIFACT_DIR/restart_audit.json" '(.data|length) >= 5'
  json_assert "$ARTIFACT_DIR/restart_audit.json" '.data[-1].action=="candidate.notification.read"'
fi

echo "vec_bolsa_smoke_ok base=$BASE_URL candidate=$candidate_id artifacts=$ARTIFACT_DIR managed=$MANAGED"
