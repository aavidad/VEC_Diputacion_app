#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

puerto="${VEC_SMOKE_PRESENTACION_PORT:-18091}"
base="http://127.0.0.1:${puerto}"
temporal="$(mktemp -d)"
pid=""

limpiar() {
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
  rm -rf "$temporal"
}
trap limpiar EXIT INT TERM

go build -trimpath -o "$temporal/vec-presentacion" ./cmd/vec-presentacion

env -i \
  PATH="$PATH" \
  HOME="${HOME:-/tmp}" \
  VEC_HTTP_ADDR="127.0.0.1:${puerto}" \
  VEC_HTTP_ALLOWED_CIDRS="127.0.0.1/32,::1/128" \
  VEC_EXECUTION_PROFILE="presentacion_rrhh" \
  VEC_RRHH_PRESENTATION_ENABLED="true" \
  VEC_RRHH_PRESENTATION_GUARD_ONE="ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO" \
  VEC_RRHH_PRESENTATION_GUARD_TWO="CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA" \
  VEC_BOLSA_STORAGE_MODE="memory" \
  VEC_PERSONAL_CATALOG_PATH="memory" \
  VEC_BOLSA_PUBLIC_SOURCE_PATH="data/demo/convocatorias_publicas.demo.json" \
  VEC_BOLSA_CATEGORIES_SOURCE_PATH="data/catalogos/categorias-profesionales/v1.demo.json" \
  "$temporal/vec-presentacion" >"$temporal/servidor.log" 2>&1 &
pid=$!

intentos=0
until curl --fail --silent --max-time 1 "$base/healthz" >/dev/null 2>&1; do
  intentos=$((intentos + 1))
  if [ "$intentos" -ge 50 ] || ! kill -0 "$pid" 2>/dev/null; then
    sed -n '1,120p' "$temporal/servidor.log" >&2
    echo "ERROR: la presentacion no alcanzo estado saludable" >&2
    exit 1
  fi
  sleep 0.2
done

curl --fail --silent --max-time 5 "$base/presentacion/" | grep -Fq 'href="/area-personal/?presentacion=rrhh"'
curl --fail --silent --head --max-time 5 "$base/presentacion/" | grep -Fiq 'X-Vec-Modo-Presentacion: aislada-sintetica-v1'
curl --fail --silent --max-time 5 "$base/area-personal/?presentacion=rrhh" | grep -Fq 'MODO DEMOSTRACIÓN'
curl --fail --silent --max-time 5 "$base/area-personal/adaptador-presentacion.js" | grep -Fq 'Adaptador efímero y exclusivo de presentación'
curl --fail --silent --max-time 5 "$base/portal-empleado/?presentacion=rrhh&perfil=administrador" | grep -Fq 'Presentación para RRHH'
curl --fail --silent --max-time 5 "$base/portal-empleado/portal-presentacion-adaptador.js" | grep -Fq 'Adaptador volátil y sustituible'
curl --fail --silent --max-time 5 "$base/api/publico/bolsa/convocatorias" | grep -Fq 'vec.bolsa.publico.convocatorias.v1'

estado_privado="$(curl --silent --output /dev/null --write-out '%{http_code}' --max-time 5 "$base/api/vec/session")"
estado_cookie="$(curl --silent --output /dev/null --write-out '%{http_code}' --max-time 5 --header 'Cookie: sesion=no-admitida' "$base/presentacion/")"
if [ "$estado_privado" != "404" ] || [ "$estado_cookie" != "400" ]; then
  echo "ERROR: fronteras inesperadas: API privada=$estado_privado cookie=$estado_cookie" >&2
  exit 1
fi

echo "Smoke del artefacto de presentacion RRHH superado."
