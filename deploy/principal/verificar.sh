#!/usr/bin/env bash
set -Eeuo pipefail

: "${VEC_VERIFY_BASE_URL:?Falta VEC_VERIFY_BASE_URL}"
: "${VEC_VERIFY_CLIENT_CERT:?Falta VEC_VERIFY_CLIENT_CERT}"
: "${VEC_VERIFY_CLIENT_KEY:?Falta VEC_VERIFY_CLIENT_KEY}"
: "${VEC_VERIFY_CA_CERT:?Falta VEC_VERIFY_CA_CERT}"
: "${VEC_VERIFY_EXPEDIENTE_REF:?Falta VEC_VERIFY_EXPEDIENTE_REF}"
: "${VEC_VERIFY_SEGUIMIENTO_REF:?Falta VEC_VERIFY_SEGUIMIENTO_REF}"
: "${VEC_VERIFY_ANOTACION_CLAVE:?Falta VEC_VERIFY_ANOTACION_CLAVE}"
: "${VEC_VERIFY_BOLSA_REF:?Falta VEC_VERIFY_BOLSA_REF}"

base=${VEC_VERIFY_BASE_URL%/}
curl_comun=(
  --fail-with-body --silent --show-error --location
  --connect-timeout 5 --max-time 20
  --cert "$VEC_VERIFY_CLIENT_CERT" --key "$VEC_VERIFY_CLIENT_KEY"
  --cacert "$VEC_VERIFY_CA_CERT" -H 'Accept: application/json'
)

comprobar() {
  local nombre=$1 ruta=$2 temporal estado
  temporal=$(mktemp)
  trap 'rm -f "$temporal"' RETURN
  estado=$(curl "${curl_comun[@]}" --output "$temporal" --write-out '%{http_code}' "$base$ruta")
  if [[ $estado != 200 ]]; then
    printf '%s: HTTP %s\n' "$nombre" "$estado" >&2
    return 1
  fi
  printf '%s: 200\n' "$nombre"
}

comprobar_b6() {
  local temporal
  temporal=$(mktemp)
  trap 'rm -f "$temporal"' RETURN
  curl "${curl_comun[@]}" --output "$temporal" "$base/api/vec/bolsa/bolsas/$bolsa/candidatos?limite=50"
  jq -e '.data.bolsa.politica_orden.provisional == true
    and .data.bolsa.politica_orden.rotulo == "Provisional, pendiente de RRHH (dudas 13–14)"
    and ([.data.candidatos[] | has("orden") and has("orden_acta") and has("razon_orden")] | all)' "$temporal" >/dev/null
  printf 'B6: política y orden calculado verificados\n'
}

expediente=$(printf '%s' "$VEC_VERIFY_EXPEDIENTE_REF" | jq -sRr @uri)
seguimiento=$(printf '%s' "$VEC_VERIFY_SEGUIMIENTO_REF" | jq -sRr @uri)
anotacion=$(printf '%s' "$VEC_VERIFY_ANOTACION_CLAVE" | jq -sRr @uri)
bolsa=$(printf '%s' "$VEC_VERIFY_BOLSA_REF" | jq -sRr @uri | sed 's/%3A/:/g')

comprobar incorporacion "/api/vec/contratacion-temporal/incorporaciones-ejercicio?expediente_ref=$expediente"
comprobar ficha-ginpix "/api/vec/contratacion-temporal/incorporaciones-ejercicio/ficha-ginpix?expediente_ref=$expediente"
comprobar anotacion "/api/vec/contratacion-temporal/expedientes/anotaciones-administrativas/recuperacion?expediente_ref=$expediente&clave_idempotencia=$anotacion"
comprobar cierre "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese/preparacion?expediente_ref=$expediente&seguimiento_ref=$seguimiento"
comprobar seguimiento "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=$expediente"
comprobar B12 "/api/vec/bolsa/bolsas"
comprobar B5 "/api/vec/bolsa/bolsas/$bolsa/candidatos?limite=50"
comprobar_b6
comprobar B10 "/api/publico/bolsa/bolsas"
