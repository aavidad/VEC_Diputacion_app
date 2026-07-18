#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

DIRECTORIO_TEST=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_TEST/../.." && pwd -P)
GENERADOR="$RAIZ_REPOSITORIO/scripts/generar_credenciales_desarrollo.sh"
TEMPORAL=$(mktemp -d)
trap 'rm -rf -- "$TEMPORAL"' EXIT HUP INT TERM

DESTINO="$TEMPORAL/estado/credenciales"
"$GENERADOR" "$DESTINO" >/dev/null

[[ -f "$DESTINO/ca/ca.crt" ]]
[[ -f "$DESTINO/tls/servidor.key" ]]
[[ -f "$DESTINO/mtls/cliente.key" ]]
[[ -f "$DESTINO/kms/atestacion-ed25519.key" ]]
[[ -f "$DESTINO/kms/atestacion-ed25519.pub" ]]
[[ -f "$DESTINO/kms/revalidacion-ed25519.key" ]]
[[ -f "$DESTINO/kms/revalidacion-ed25519.pub" ]]
[[ -f "$DESTINO/idempotencia/configuracion.json" ]]
[[ -f "$DESTINO/idempotencia/g2-localizador.bin" ]]
[[ -f "$DESTINO/idempotencia/g2-huella-solicitud.bin" ]]
[[ -f "$DESTINO/idempotencia/g1-localizador.bin" ]]
[[ -f "$DESTINO/idempotencia/g1-huella-solicitud.bin" ]]
[[ "$(openssl pkey -in "$DESTINO/kms/atestacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" != \
  "$(openssl pkey -in "$DESTINO/kms/revalidacion-ed25519.key" -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')" ]]
[[ $(stat -c '%s' "$DESTINO/kms/clave-maestra.bin") -eq 32 ]]
[[ $(stat -c '%s' "$DESTINO/tsa/clave-hmac.bin") -eq 32 ]]
for CLAVE_HMAC in "$DESTINO"/idempotencia/g{2,1}-{localizador,huella-solicitud}.bin; do
  [[ $(stat -c '%s' "$CLAVE_HMAC") -eq 32 ]]
  [[ $(stat -c '%a' "$CLAVE_HMAC") == 600 ]]
done
[[ $(sha256sum \
  "$DESTINO/kms/clave-maestra.bin" \
  "$DESTINO/tsa/clave-hmac.bin" \
  "$DESTINO"/idempotencia/g{2,1}-{localizador,huella-solicitud}.bin |
  awk '{print $1}' | sort -u | wc -l) -eq 6 ]]
grep -Fq '"version":3' "$DESTINO/manifiesto.json"
grep -Fq '"idempotencia_hmac":"idempotencia-hmac-fichero-local-v1"' "$DESTINO/manifiesto.json"
grep -Fq '"generacion":2' "$DESTINO/idempotencia/configuracion.json"
grep -Fq '"generacion":1' "$DESTINO/idempotencia/configuracion.json"
[[ $(stat -c '%a' "$DESTINO/tls/servidor.key") == 600 ]]
[[ $(stat -c '%a' "$DESTINO") == 700 ]]

HUELLA_ANTES=$(find "$DESTINO" -type f -print0 | sort -z | xargs -0 sha256sum)
"$GENERADOR" "$DESTINO" >/dev/null
HUELLA_DESPUES=$(find "$DESTINO" -type f -print0 | sort -z | xargs -0 sha256sum)
[[ "$HUELLA_ANTES" == "$HUELLA_DESPUES" ]]

MANIFIESTO_ORIGINAL=$(<"$DESTINO/manifiesto.json")
printf '{}\n' >"$DESTINO/manifiesto.json"
if "$GENERADOR" "$DESTINO" >/dev/null 2>&1; then
  printf 'el generador acepto un manifiesto alterado\n' >&2
  exit 1
fi
printf '%s\n' "$MANIFIESTO_ORIGINAL" >"$DESTINO/manifiesto.json"

printf '{"version":1}\n' >"$DESTINO/idempotencia/configuracion.json"
if "$GENERADOR" "$DESTINO" >/dev/null 2>&1; then
  printf 'el generador acepto configuracion HMAC alterada\n' >&2
  exit 1
fi

PARCIAL="$TEMPORAL/parcial"
mkdir -m 700 "$PARCIAL"
if "$GENERADOR" "$PARCIAL" >/dev/null 2>&1; then
  printf 'el generador acepto un directorio parcial\n' >&2
  exit 1
fi

if "$GENERADOR" "$RAIZ_REPOSITORIO/.vec-desarrollo/prueba" >/dev/null 2>&1; then
  printf 'el generador escribio dentro del repositorio\n' >&2
  exit 1
fi

printf 'generador de credenciales desarrollo: OK\n'
