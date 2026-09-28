#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

# Único punto de entrada. Todos los procesos y secretos son efímeros. Nunca
# hereda conexiones VEC/PG del operador y solo acepta localhost.
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd -P)
SCRIPT_DIR="$ROOT/scripts/rrhh_e2e_aislado"
MODE=${1:---full}
case "$MODE" in --smoke|--preflight|--full) ;; *) printf 'Uso: %s [--smoke|--preflight|--full]\n' "$0" >&2; exit 2 ;; esac

for NAME in $(compgen -e); do
  case "$NAME" in VEC_*|PG*) unset "$NAME" ;; esac
done
# Fijar el daemon local antes de cualquier llamada Docker.
unset DOCKER_CONTEXT DOCKER_HOST DOCKER_TLS_VERIFY DOCKER_CERT_PATH DOCKER_CONFIG
[[ -S /var/run/docker.sock ]] || { echo 'Falta socket Docker Unix local' >&2; exit 2; }
export DOCKER_HOST=unix:///var/run/docker.sock
for TOOL in docker python3 curl openssl go mktemp sha256sum; do
  command -v "$TOOL" >/dev/null 2>&1 || { printf 'Falta %s\n' "$TOOL" >&2; exit 2; }
done
[[ -d /dev/shm && -w /dev/shm ]] || { echo '/dev/shm no disponible' >&2; exit 2; }
VEC_E2E_ROOT="$ROOT"
VEC_E2E_WORK=$(mktemp -d /dev/shm/vec-e2e-rrhh.XXXXXXXX)
export VEC_E2E_ROOT VEC_E2E_WORK
DOCKER_CONFIG="$VEC_E2E_WORK/docker-config"
mkdir -m 0700 -- "$DOCKER_CONFIG"
export DOCKER_CONFIG
VEC_E2E_MATERIAL="$VEC_E2E_WORK/material"
export VEC_E2E_MATERIAL
VEC_E2E_HTTP_PORT=$(python3 - <<'PY'
import socket
s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()
PY
)
export VEC_E2E_HTTP_PORT
APP_PID=''

cleanup() {
  local rc=$?
  trap - EXIT INT TERM HUP
  if [[ -n "$APP_PID" ]]; then kill "$APP_PID" 2>/dev/null || true; wait "$APP_PID" 2>/dev/null || true; fi
  if declare -F rrhh_e2e_detener_pg >/dev/null; then rrhh_e2e_detener_pg || true; fi
  # Mantener únicamente evidencia redactada si falla. Los secretos nunca salen
  # de memoria compartida y desaparecen al terminar.
  if [[ -f "$VEC_E2E_WORK/evidencia.json" ]]; then
    cp -- "$VEC_E2E_WORK/evidencia.json" "$SCRIPT_DIR/ultima_evidencia.json"
  fi
  rm -rf -- "$VEC_E2E_WORK"
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

# shellcheck source=infra.sh
source "$SCRIPT_DIR/infra.sh"
# shellcheck source=fixtures.sh
source "$SCRIPT_DIR/fixtures.sh"
# shellcheck source=roles.sh
source "$SCRIPT_DIR/roles.sh"

if [[ "$MODE" != --smoke ]]; then
  rrhh_e2e_iniciar_pg
  rrhh_e2e_instalar_sql
  rrhh_e2e_preparar_fixtures
  if [[ "$MODE" == --preflight ]]; then
    printf 'Preflight PG18 sintético completado; no se afirma E2E HTTP.\n'
    exit 0
  fi
  rrhh_e2e_exportar_dsns
fi

"$ROOT/scripts/generar_credenciales_desarrollo.sh" "$VEC_E2E_MATERIAL" >"$VEC_E2E_WORK/credenciales.log"
export VEC_EXECUTION_PROFILE=desarrollo
export VEC_AUTH_MODE=desarrollo
export VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
export VEC_DEVELOPMENT_MATERIAL_DIR="$VEC_E2E_MATERIAL"
export VEC_TLS_CERT_FILE="$VEC_E2E_MATERIAL/tls/servidor.crt"
export VEC_TLS_KEY_FILE="$VEC_E2E_MATERIAL/tls/servidor.key"
export VEC_HTTP_ADDR="127.0.0.1:$VEC_E2E_HTTP_PORT"
export TMPDIR="$VEC_E2E_WORK" GOCACHE="$VEC_E2E_WORK/gocache"
export GOTOOLCHAIN=local GOENV=off GOPROXY=off GOSUMDB=off
GO_LOCAL=$("$ROOT/scripts/seleccionar_toolchain_go_local.sh")
(cd "$ROOT" && "$GO_LOCAL" build -buildvcs=false -o "$VEC_E2E_WORK/vec-server" ./cmd/vec-server)
(cd "$ROOT" && exec "$VEC_E2E_WORK/vec-server") >"$VEC_E2E_WORK/app.log" 2>&1 &
APP_PID=$!

URL="https://localhost:$VEC_E2E_HTTP_PORT"
ready=false
for ((i=0; i<200; i++)); do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo 'vec-server terminó antes de livez; últimas líneas redactadas:' >&2
    tail -n 12 "$VEC_E2E_WORK/app.log" | sed -E 's#postgres(ql)?://[^[:space:]" ]+#<DSN-redactado>#g' >&2
    exit 1
  fi
  if curl --silent --show-error --fail --cacert "$VEC_E2E_MATERIAL/ca/ca.crt" \
      --cert "$VEC_E2E_MATERIAL/mtls/cliente.crt" --key "$VEC_E2E_MATERIAL/mtls/cliente.key" \
      "$URL/livez" >/dev/null 2>&1; then ready=true; break; fi
  sleep 0.1
done
[[ "$ready" == true ]] || { echo 'Sin livez mTLS' >&2; exit 1; }
grep -q 'vec server listening' "$VEC_E2E_WORK/app.log" || { echo 'Falta señal vec server listening' >&2; exit 1; }

if [[ "$MODE" == --full ]]; then
  export VEC_AUDITORIA_E2E_URL="$URL"
  (cd "$ROOT" && GOMAXPROCS=2 "$GO_LOCAL" test ./internal/app/bootstrap \
    -run '^TestAuditoriaConsultaRRHHHTTPPostgreSQL18$' -count=1)
fi

browser_args=(--base-url "$URL" --material "$VEC_E2E_MATERIAL" --salida "$VEC_E2E_WORK/evidencia.json")
if [[ -n "${VEC_E2E_BOLSA_REF:-}" ]]; then browser_args+=(--bolsa-ref "$VEC_E2E_BOLSA_REF"); fi
python3 "$SCRIPT_DIR/navegador.py" "${browser_args[@]}"
if [[ "$MODE" == --smoke ]]; then echo 'Smoke mTLS/Chrome completado sin PostgreSQL ni E2E de negocio.'; fi
printf 'Evidencia HTTP/Chrome redactada: %s\n' "$SCRIPT_DIR/ultima_evidencia.json"
