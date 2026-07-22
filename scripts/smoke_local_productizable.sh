#!/usr/bin/env bash
set -euo pipefail

# Smoke heredado exclusivamente local. El nombre del fichero se conserva por
# compatibilidad, pero esta prueba no acredita un despliegue productivo ni usa
# cabeceras ambientales como identidad.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${VEC_SMOKE_BASE_URL:-${BOLSA_SMOKE_BASE_URL:-http://127.0.0.1:18082}}"
ARTIFACT_DIR="${VEC_SMOKE_ARTIFACT_DIR:-${BOLSA_SMOKE_ARTIFACT_DIR:-"$ROOT_DIR/var/smoke"}}"
MANAGED="${VEC_SMOKE_MANAGED:-${BOLSA_SMOKE_MANAGED:-0}}"
DATA_DIR="${VEC_SMOKE_DATA_DIR:-${BOLSA_SMOKE_DATA_DIR:-"$ROOT_DIR/var/bolsa"}}"
SERVER_PID=""
RUN_REF="smoke-$(date +%Y%m%d%H%M%S)"
CANDIDATE_ID="cand-$RUN_REF"
STAFF_TOKEN="SMOKE_LOCAL_RRHH_TOKEN_SIN_VALOR_PRODUCTIVO_0001"
CANDIDATE_TOKEN="SMOKE_LOCAL_ASPIRANTE_TOKEN_SIN_VALOR_PRODUCTIVO_0002"
ADMIN_TOKEN="SMOKE_LOCAL_ADMIN_TOKEN_SIN_VALOR_PRODUCTIVO_0003"
FAKE_CREDENTIALS_FILE="$ARTIFACT_DIR/credenciales_fake_smoke.json"

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

wait_liveness() {
  local base="$1"
  for _ in $(seq 1 50); do
    if curl -fsS "$base/livez" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "server did not become healthy at $base" >&2
  return 1
}

write_fake_credentials() {
  python3 - "$FAKE_CREDENTIALS_FILE" "$STAFF_TOKEN" "$CANDIDATE_TOKEN" "$ADMIN_TOKEN" "$CANDIDATE_ID" <<'PY'
import hashlib
import json
import os
import sys

path, staff_token, candidate_token, admin_token, candidate_id = sys.argv[1:]
content = {
    "version": 1,
    "credentials": [
        {
            "token_sha256": hashlib.sha256(staff_token.encode("ascii")).hexdigest(),
            "subject": "smoke-rrhh",
            "display_name": "RRHH smoke local",
            "roles": ["tecnico_rrhh"],
            "mechanism": "kerberos_ad",
            "assurance": "alto",
            "legacy_role": "validator_l2",
        },
        {
            "token_sha256": hashlib.sha256(candidate_token.encode("ascii")).hexdigest(),
            "subject": candidate_id,
            "display_name": "Aspirante smoke local",
            "roles": ["ciudadano"],
            "mechanism": "clave",
            "assurance": "sustancial",
            "legacy_role": "candidate",
        },
        {
            "token_sha256": hashlib.sha256(admin_token.encode("ascii")).hexdigest(),
            "subject": "smoke-administracion",
            "display_name": "Administracion tecnica smoke local",
            "roles": ["administrador"],
            "mechanism": "kerberos_ad",
            "assurance": "alto",
            "legacy_role": "system_admin",
        },
    ],
}
temporary = path + ".tmp"
try:
    os.unlink(temporary)
except FileNotFoundError:
    pass
descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
    json.dump(content, stream, ensure_ascii=False, separators=(",", ":"))
    stream.flush()
    os.fsync(stream.fileno())
os.replace(temporary, path)
os.chmod(path, 0o600)
PY
}

start_managed_server() {
  local addr="${BASE_URL#http://}"
  local bin="$ARTIFACT_DIR/vec-server-smoke"
  (cd "$ROOT_DIR" && go build -buildvcs=false -o "$bin" ./cmd/vec-server)
  VEC_HTTP_ADDR="$addr" \
    VEC_BOLSA_STORAGE_MODE=local_durable \
    VEC_BOLSA_DATA_DIR="$DATA_DIR" \
    VEC_AUTH_MODE=fake \
    VEC_FAKE_CREDENTIALS_FILE="$FAKE_CREDENTIALS_FILE" \
    "$bin" >"$ARTIFACT_DIR/server.log" 2>&1 &
  SERVER_PID="$!"
  wait_liveness "$BASE_URL"
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
  write_fake_credentials
  start_managed_server
else
  wait_liveness "$BASE_URL"
fi

solicitud_id="sol-$RUN_REF"

staff_headers=(
  -H "Content-Type: application/json"
  -H "Authorization: Bearer $STAFF_TOKEN"
)
candidate_headers=(
  -H "Content-Type: application/json"
  -H "Authorization: Bearer $CANDIDATE_TOKEN"
)
admin_headers=(
  -H "Content-Type: application/json"
  -H "Authorization: Bearer $ADMIN_TOKEN"
)

curl -fsS "$BASE_URL/livez" >"$ARTIFACT_DIR/health.json"
curl -fsS "$BASE_URL/api/vec/modules" "${staff_headers[@]}" >"$ARTIFACT_DIR/vec_modules.json"
json_assert "$ARTIFACT_DIR/vec_modules.json" '(.data.modules|map(.id)|index("vec.module.bolsa")) != null'
workspace_status="$(curl -sS -o "$ARTIFACT_DIR/vec_workspace.json" -w '%{http_code}' \
  "$BASE_URL/api/vec/workspace" "${staff_headers[@]}")"
if [[ "$workspace_status" != "403" ]]; then
  echo "workspace sintetico inesperadamente accesible: HTTP $workspace_status" >&2
  exit 1
fi
curl -fsS "$BASE_URL/api/vec/menu" "${staff_headers[@]}" >"$ARTIFACT_DIR/vec_menu.json"
json_assert "$ARTIFACT_DIR/vec_menu.json" '(.data.menu|length) == 0'
curl -fsS "$BASE_URL/api/modules/bolsa" "${staff_headers[@]}" >"$ARTIFACT_DIR/bolsa_manifest.json"
json_assert "$ARTIFACT_DIR/bolsa_manifest.json" '.data.module_ref=="vec.module.bolsa"'
json_assert "$ARTIFACT_DIR/bolsa_manifest.json" '(.data.http_routes|map(.route)|index("/api/candidates/{id}/documents")) != null'
curl -fsS "$BASE_URL/api/admin/status" "${admin_headers[@]}" >"$ARTIFACT_DIR/admin_status.json"
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.status=="operational"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.auth_mode=="fake"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.persistence_mode=="file"'
json_assert "$ARTIFACT_DIR/admin_status.json" '.data.legal_production_ready==false'

curl -fsS -X POST "$BASE_URL/api/candidates" "${candidate_headers[@]}" \
  -d "{\"id\":\"$CANDIDATE_ID\",\"dni\":\"12345678Z\",\"nombre\":\"Ana Perez\",\"email\":\"ana@example.test\",\"call_id\":\"convocatoria-demostracion\"}" \
  >"$ARTIFACT_DIR/candidate.json"

curl -fsS -X POST "$BASE_URL/api/candidates/$CANDIDATE_ID/merits" "${candidate_headers[@]}" \
  -d '{"id":"merit-smoke","tipo":"formacion_curso","datos":{"horas":20}}' \
  >"$ARTIFACT_DIR/merit.json"

curl -fsS -X POST "$BASE_URL/api/candidates/$CANDIDATE_ID/baremo" "${candidate_headers[@]}" \
  >"$ARTIFACT_DIR/baremo.json"
json_assert "$ARTIFACT_DIR/baremo.json" '.data.total_points==1'

document_status="$(curl -sS -o "$ARTIFACT_DIR/document.json" -w '%{http_code}' \
  -X POST "$BASE_URL/api/candidates/$CANDIDATE_ID/documents" "${candidate_headers[@]}" \
  -d "{\"id\":\"doc-smoke\",\"solicitud_id\":\"$solicitud_id\",\"procedure_id\":\"proc-smoke\",\"purpose\":\"Alegacion\",\"csv\":\"CSV-SMOKE-DOC\",\"digest_sha256\":\"abc123\",\"storage_object_ref\":\"obj-smoke\",\"document_type\":\"alegacion\",\"title\":\"Titulo smoke\",\"format\":\"application/pdf\",\"signature_ref\":\"sig-smoke\"}" \
  )"
if [[ "$document_status" != "503" ]]; then
  echo "alta documental heredada no permanecio cerrada: HTTP $document_status" >&2
  exit 1
fi

claim_status="$(curl -sS -o "$ARTIFACT_DIR/claim.json" -w '%{http_code}' \
  -X POST "$BASE_URL/api/candidates/$CANDIDATE_ID/claims" "${candidate_headers[@]}" \
  -d "{\"id\":\"claim-smoke\",\"solicitud_id\":\"$solicitud_id\",\"text\":\"Revisar baremo\",\"receipt_csv\":\"CSV-SMOKE-CLAIM\"}" \
  )"
if [[ "$claim_status" != "503" ]]; then
  echo "alegacion probatoria heredada no permanecio cerrada: HTTP $claim_status" >&2
  exit 1
fi

curl -fsS -X POST "$BASE_URL/api/notifications" "${staff_headers[@]}" \
  -d "{\"candidate_id\":\"$CANDIDATE_ID\",\"id\":\"note-smoke\",\"solicitud_id\":\"$solicitud_id\",\"type\":\"subsanacion\",\"subject\":\"Aportar titulo\",\"body\":\"Revise evidencia\"}" \
  >"$ARTIFACT_DIR/notification.json"

send_status="$(curl -sS -o "$ARTIFACT_DIR/notification_send.json" -w '%{http_code}' \
  -X POST "$BASE_URL/api/notifications/note-smoke/send" "${staff_headers[@]}" \
  -d "{\"notification_id\":\"note-smoke\",\"csv\":\"CSV-SMOKE-SEND\",\"recipient_id\":\"$CANDIDATE_ID\",\"channel\":\"sede\"}" \
  )"
if [[ "$send_status" != "503" ]]; then
  echo "envio probatorio heredado no permanecio cerrado: HTTP $send_status" >&2
  exit 1
fi

read_status="$(curl -sS -o "$ARTIFACT_DIR/notification_read.json" -w '%{http_code}' \
  -X POST "$BASE_URL/api/notifications/note-smoke/read" "${staff_headers[@]}" \
  -d "{\"notification_id\":\"note-smoke\",\"csv\":\"CSV-SMOKE-READ\",\"recipient_id\":\"$CANDIDATE_ID\",\"channel\":\"sede\"}" \
  )"
if [[ "$read_status" != "503" ]]; then
  echo "lectura probatoria heredada no permanecio cerrada: HTTP $read_status" >&2
  exit 1
fi

curl -fsS "$BASE_URL/api/audit?candidate_id=$CANDIDATE_ID" "${staff_headers[@]}" \
  >"$ARTIFACT_DIR/audit.json"
json_assert "$ARTIFACT_DIR/audit.json" '(.data|length) >= 1'
json_assert "$ARTIFACT_DIR/audit.json" '.data[-1].action=="candidate.notification.created"'

if [[ "$MANAGED" == "1" ]]; then
  restart_managed_server
  curl -fsS -X POST "$BASE_URL/api/candidates/$CANDIDATE_ID/baremo" "${candidate_headers[@]}" \
    >"$ARTIFACT_DIR/restart_baremo.json"
  json_assert "$ARTIFACT_DIR/restart_baremo.json" '.data.total_points==1'
  curl -fsS "$BASE_URL/api/audit?candidate_id=$CANDIDATE_ID" "${staff_headers[@]}" \
    >"$ARTIFACT_DIR/restart_audit.json"
  json_assert "$ARTIFACT_DIR/restart_audit.json" '(.data|length) >= 1'
  json_assert "$ARTIFACT_DIR/restart_audit.json" '.data[-1].action=="candidate.notification.created"'
fi

echo "vec_bolsa_smoke_local_ok base=$BASE_URL candidate=$CANDIDATE_ID artifacts=$ARTIFACT_DIR managed=$MANAGED"
