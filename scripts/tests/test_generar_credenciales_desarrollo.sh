#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

DIRECTORIO_TEST=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_TEST/../.." && pwd -P)
GENERADOR="$RAIZ_REPOSITORIO/scripts/generar_credenciales_desarrollo.sh"
TEMPORAL=$(mktemp -d)
trap 'rm -rf -- "$TEMPORAL"' EXIT HUP INT TERM

DESTINO="$TEMPORAL/estado/credenciales"
SALIDA_GENERACION="$TEMPORAL/salida-generacion.txt"
"$GENERADOR" "$DESTINO" >"$SALIDA_GENERACION"

[[ -f "$DESTINO/ca/ca.crt" ]]
[[ -f "$DESTINO/tls/servidor.key" ]]
[[ -f "$DESTINO/mtls/cliente.key" ]]
[[ -f "$DESTINO/mtls/cliente.p12" ]]
[[ -f "$DESTINO/mtls/cliente.p12.password" ]]
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
[[ $(stat -c '%a' "$DESTINO/mtls/cliente.p12") == 600 ]]
[[ $(stat -c '%a' "$DESTINO/mtls/cliente.p12.password") == 600 ]]
[[ $(stat -c '%a' "$DESTINO") == 700 ]]

CONTRASENA_PKCS12=$(<"$DESTINO/mtls/cliente.p12.password")
[[ "$CONTRASENA_PKCS12" =~ ^[0-9a-f]{64}$ ]]
if grep -Fq -- "$CONTRASENA_PKCS12" "$SALIDA_GENERACION"; then
  printf 'el generador mostro la contrasena PKCS#12 en su salida\n' >&2
  exit 1
fi
grep -Fq "$DESTINO/ca/ca.crt" "$SALIDA_GENERACION"
grep -Fq "$DESTINO/mtls/cliente.p12" "$SALIDA_GENERACION"
grep -Fq "$DESTINO/mtls/cliente.p12.password" "$SALIDA_GENERACION"

openssl pkcs12 -in "$DESTINO/mtls/cliente.p12" \
  -passin "file:$DESTINO/mtls/cliente.p12.password" -noout 2>/dev/null
HUELLA_CLIENTE_PEM=$(openssl x509 -in "$DESTINO/mtls/cliente.crt" -outform DER |
  sha256sum | awk '{print $1}')
HUELLA_CLIENTE_PKCS12=$(openssl pkcs12 -in "$DESTINO/mtls/cliente.p12" \
  -passin "file:$DESTINO/mtls/cliente.p12.password" -clcerts -nokeys 2>/dev/null |
  openssl x509 -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
[[ "$HUELLA_CLIENTE_PKCS12" == "$HUELLA_CLIENTE_PEM" ]]
grep -Fq "\"huella_cliente_sha256\":\"$HUELLA_CLIENTE_PKCS12\"" "$DESTINO/manifiesto.json"

HUELLA_CLAVE_PEM=$(openssl pkey -in "$DESTINO/mtls/cliente.key" -pubout -outform DER 2>/dev/null |
  sha256sum | awk '{print $1}')
HUELLA_CLAVE_PKCS12=$(openssl pkcs12 -in "$DESTINO/mtls/cliente.p12" \
  -passin "file:$DESTINO/mtls/cliente.p12.password" -nocerts -nodes 2>/dev/null |
  openssl pkey -pubout -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
[[ "$HUELLA_CLAVE_PKCS12" == "$HUELLA_CLAVE_PEM" ]]

HUELLA_CA_PEM=$(openssl x509 -in "$DESTINO/ca/ca.crt" -outform DER |
  sha256sum | awk '{print $1}')
HUELLA_CA_PKCS12=$(openssl pkcs12 -in "$DESTINO/mtls/cliente.p12" \
  -passin "file:$DESTINO/mtls/cliente.p12.password" -cacerts -nokeys 2>/dev/null |
  openssl x509 -outform DER 2>/dev/null | sha256sum | awk '{print $1}')
[[ "$HUELLA_CA_PKCS12" == "$HUELLA_CA_PEM" ]]

DESTINO_DOS="$TEMPORAL/estado-dos/credenciales"
"$GENERADOR" "$DESTINO_DOS" >/dev/null
CONTRASENA_PKCS12_DOS=$(<"$DESTINO_DOS/mtls/cliente.p12.password")
[[ "$CONTRASENA_PKCS12_DOS" =~ ^[0-9a-f]{64}$ ]]
[[ "$CONTRASENA_PKCS12_DOS" != "$CONTRASENA_PKCS12" ]]

PADRE_CONCURRENTE="$TEMPORAL/concurrente"
DESTINO_CONCURRENTE="$PADRE_CONCURRENTE/credenciales"
BIN_CONCURRENTE="$TEMPORAL/bin-concurrente"
BARRERA_MV="$TEMPORAL/barrera-mv"
SALIDA_UNO="$TEMPORAL/concurrente-uno.out"
SALIDA_DOS="$TEMPORAL/concurrente-dos.out"
ERROR_UNO="$TEMPORAL/concurrente-uno.err"
ERROR_DOS="$TEMPORAL/concurrente-dos.err"
MV_REAL=$(command -v mv)
mkdir -p -- "$BIN_CONCURRENTE" "$BARRERA_MV"
cat >"$BIN_CONCURRENTE/mv" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
: "${VEC_TEST_MV_REAL:?}"
: "${VEC_TEST_MV_BARRERA:?}"
printf '' >"$VEC_TEST_MV_BARRERA/listo.$BASHPID"
limite=$((SECONDS + 30))
while [[ $(find "$VEC_TEST_MV_BARRERA" -maxdepth 1 -type f -name 'listo.*' | wc -l) -lt 2 ]]; do
  (( SECONDS < limite )) || exit 70
  sleep 0.01
done
exec "$VEC_TEST_MV_REAL" "$@"
SH
chmod 700 "$BIN_CONCURRENTE/mv"

PATH="$BIN_CONCURRENTE:$PATH" VEC_TEST_MV_REAL="$MV_REAL" VEC_TEST_MV_BARRERA="$BARRERA_MV" \
  "$GENERADOR" "$DESTINO_CONCURRENTE" >"$SALIDA_UNO" 2>"$ERROR_UNO" &
PID_UNO=$!
PATH="$BIN_CONCURRENTE:$PATH" VEC_TEST_MV_REAL="$MV_REAL" VEC_TEST_MV_BARRERA="$BARRERA_MV" \
  "$GENERADOR" "$DESTINO_CONCURRENTE" >"$SALIDA_DOS" 2>"$ERROR_DOS" &
PID_DOS=$!
ESTADO_UNO=0
ESTADO_DOS=0
wait "$PID_UNO" || ESTADO_UNO=$?
wait "$PID_DOS" || ESTADO_DOS=$?
if (( ESTADO_UNO != 0 || ESTADO_DOS != 0 )); then
  printf 'fallo en generadores concurrentes: uno=%d dos=%d\n' "$ESTADO_UNO" "$ESTADO_DOS" >&2
  exit 1
fi
[[ $(awk 'index($0, "Credenciales de desarrollo generadas fuera de Git:") {n++} END {print n+0}' \
  "$SALIDA_UNO" "$SALIDA_DOS") -eq 1 ]]
[[ $(awk 'index($0, "Otro proceso publico credenciales de desarrollo validas:") {n++} END {print n+0}' \
  "$SALIDA_UNO" "$SALIDA_DOS") -eq 1 ]]
[[ -z "$(find "$PADRE_CONCURRENTE" -type d -name '.vec-desarrollo.*' -print -quit)" ]]
[[ $(find "$DESTINO_CONCURRENTE" -type f -name 'cliente.key' | wc -l) -eq 1 ]]
[[ $(find "$DESTINO_CONCURRENTE" -type f -name 'cliente.p12' | wc -l) -eq 1 ]]
[[ $(find "$DESTINO_CONCURRENTE" -type f -name 'cliente.p12.password' | wc -l) -eq 1 ]]
"$GENERADOR" "$DESTINO_CONCURRENTE" >/dev/null

HUELLA_ANTES=$(find "$DESTINO" -type f -print0 | sort -z | xargs -0 sha256sum)
"$GENERADOR" "$DESTINO" >/dev/null
HUELLA_DESPUES=$(find "$DESTINO" -type f -print0 | sort -z | xargs -0 sha256sum)
[[ "$HUELLA_ANTES" == "$HUELLA_DESPUES" ]]

PAQUETE_ORIGINAL="$TEMPORAL/cliente.p12.original"
cp -- "$DESTINO/mtls/cliente.p12" "$PAQUETE_ORIGINAL"
printf 'X' | dd of="$DESTINO/mtls/cliente.p12" bs=1 count=1 conv=notrunc status=none
if "$GENERADOR" "$DESTINO" >/dev/null 2>&1; then
  printf 'el generador acepto un paquete PKCS#12 alterado\n' >&2
  exit 1
fi
mv -- "$PAQUETE_ORIGINAL" "$DESTINO/mtls/cliente.p12"
"$GENERADOR" "$DESTINO" >/dev/null

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
