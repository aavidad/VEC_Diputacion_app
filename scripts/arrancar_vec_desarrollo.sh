#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'
umask 077

fallar() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

mostrar_ayuda() {
  cat <<'AYUDA'
Uso:
  scripts/arrancar_vec_desarrollo.sh [--puerto PUERTO] [--directorio-material RUTA]

Arranca cmd/vec-server en primer plano para desarrollo local seguro. Genera o
verifica las credenciales con scripts/generar_credenciales_desarrollo.sh,
construye un binario temporal y publica exclusivamente en 127.0.0.1 con
TLS 1.3, mTLS y la doble llave de desarrollo vigente.

Opciones:
  --puerto PUERTO             Puerto TCP local (predeterminado: 8443).
  --directorio-material RUTA  Directorio absoluto de credenciales fuera de Git.
  -h, --help                  Muestra esta ayuda.

El proceso permanece supervisado por este lanzador. Ctrl-C, SIGHUP o SIGTERM
terminan el servidor y eliminan el binario temporal. No habilita producción,
despliegue, cabeceras de identidad ni autenticación bearer/fake.
AYUDA
}

DIRECTORIO_SCRIPT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_SCRIPT/.." && pwd -P)
GENERADOR="$DIRECTORIO_SCRIPT/generar_credenciales_desarrollo.sh"
PUERTO=8443
DIRECTORIO_MATERIAL=''
PUERTO_INDICADO=false
MATERIAL_INDICADO=false
SERVIDOR_PID=''
TEMPORAL_BUILD=''

# Invocada indirectamente por trap.
# shellcheck disable=SC2329
limpiar() {
  local estado=$?
  trap - EXIT HUP INT TERM
  if [[ -n "$SERVIDOR_PID" ]] && kill -0 "$SERVIDOR_PID" 2>/dev/null; then
    kill -TERM "$SERVIDOR_PID" 2>/dev/null || true
    wait "$SERVIDOR_PID" 2>/dev/null || true
  fi
  if [[ -n "$TEMPORAL_BUILD" ]]; then
    case "$TEMPORAL_BUILD" in
      /tmp/vec-arranque-desarrollo.*) rm -rf -- "$TEMPORAL_BUILD" ;;
      *) printf 'ERROR: temporal de build inesperado: %s\n' "$TEMPORAL_BUILD" >&2 ;;
    esac
  fi
  exit "$estado"
}

trap limpiar EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

while (( $# > 0 )); do
  case "$1" in
    --puerto)
      [[ "$PUERTO_INDICADO" == false ]] || fallar '--puerto solo puede indicarse una vez'
      (( $# >= 2 )) || fallar 'falta el valor de --puerto'
      PUERTO=$2
      PUERTO_INDICADO=true
      shift 2
      ;;
    --directorio-material)
      [[ "$MATERIAL_INDICADO" == false ]] || fallar '--directorio-material solo puede indicarse una vez'
      (( $# >= 2 )) || fallar 'falta el valor de --directorio-material'
      DIRECTORIO_MATERIAL=$2
      MATERIAL_INDICADO=true
      shift 2
      ;;
    -h|--help)
      mostrar_ayuda
      exit 0
      ;;
    *)
      fallar "argumento desconocido: $1"
      ;;
  esac
done

[[ "$PUERTO" =~ ^[0-9]+$ ]] || fallar 'el puerto debe ser un entero entre 1 y 65535'
(( ${#PUERTO} <= 5 )) || fallar 'el puerto debe estar entre 1 y 65535'
PUERTO=$((10#$PUERTO))
(( PUERTO >= 1 && PUERTO <= 65535 )) || fallar 'el puerto debe estar entre 1 y 65535'

if [[ -z "$DIRECTORIO_MATERIAL" ]]; then
  RAIZ_ESTADO=${XDG_STATE_HOME:-${HOME:?HOME no definida}/.local/state}
  DIRECTORIO_MATERIAL="$RAIZ_ESTADO/vec-diputacion/desarrollo"
fi
[[ "$DIRECTORIO_MATERIAL" == /* ]] || fallar 'el directorio de material debe ser absoluto'

for ORDEN in go curl python3 realpath mktemp rm; do
  command -v "$ORDEN" >/dev/null 2>&1 || fallar "falta la herramienta obligatoria: $ORDEN"
done
[[ -x "$GENERADOR" ]] || fallar "generador no ejecutable: $GENERADOR"
DIRECTORIO_MATERIAL=$(realpath -m -- "$DIRECTORIO_MATERIAL")

comprobar_puerto_libre() {
  local detalle
  if ! detalle=$(python3 - "$PUERTO" 2>&1 <<'PY'
import socket
import sys

puerto = int(sys.argv[1])
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
try:
    sock.bind(("127.0.0.1", puerto))
except OSError as exc:
    print(exc.strerror or str(exc))
    raise SystemExit(1)
finally:
    sock.close()
PY
  ); then
    fallar "el puerto 127.0.0.1:$PUERTO no está disponible: $detalle"
  fi
}

comprobar_puerto_libre
"$GENERADOR" "$DIRECTORIO_MATERIAL"

# No se carga desarrollo.env como código. El generador ya ha verificado el
# material y estas son exactamente las guardas que consume config.Config.
export VEC_EXECUTION_PROFILE=desarrollo
export VEC_AUTH_MODE=desarrollo
export VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
export VEC_DEVELOPMENT_MATERIAL_DIR="$DIRECTORIO_MATERIAL"
export VEC_TLS_CERT_FILE="$DIRECTORIO_MATERIAL/tls/servidor.crt"
export VEC_TLS_KEY_FILE="$DIRECTORIO_MATERIAL/tls/servidor.key"
export VEC_HTTP_ADDR="127.0.0.1:$PUERTO"

TEMPORAL_BUILD=$(mktemp -d /tmp/vec-arranque-desarrollo.XXXXXX)
BINARIO="$TEMPORAL_BUILD/vec-server"
(
  cd "$RAIZ_REPOSITORIO"
  go build -buildvcs=false -o "$BINARIO" ./cmd/vec-server
)
comprobar_puerto_libre

(
  cd "$RAIZ_REPOSITORIO"
  exec "$BINARIO"
) &
SERVIDOR_PID=$!
URL_BASE="https://localhost:$PUERTO"
LISTO=false
for (( INTENTO = 0; INTENTO < 200; INTENTO++ )); do
  if ! kill -0 "$SERVIDOR_PID" 2>/dev/null; then
    if wait "$SERVIDOR_PID"; then
      ESTADO_SERVIDOR=0
    else
      ESTADO_SERVIDOR=$?
    fi
    SERVIDOR_PID=''
    fallar "vec-server terminó antes de estar disponible (estado $ESTADO_SERVIDOR)"
  fi
  if curl --silent --show-error --fail \
    --cacert "$DIRECTORIO_MATERIAL/ca/ca.crt" \
    --cert "$DIRECTORIO_MATERIAL/mtls/cliente.crt" \
    --key "$DIRECTORIO_MATERIAL/mtls/cliente.key" \
    "$URL_BASE/livez" >/dev/null 2>&1; then
    LISTO=true
    break
  fi
  sleep 0.05
done
[[ "$LISTO" == true ]] || fallar "vec-server no respondió en $URL_BASE/livez"

printf '\nVEC local de desarrollo disponible. No es un despliegue productivo.\n'
printf 'Portal: %s/portal-empleado/\n' "$URL_BASE"
printf 'Salud:\n  curl --cacert %q --cert %q --key %q %q\n' \
  "$DIRECTORIO_MATERIAL/ca/ca.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.key" \
  "$URL_BASE/livez"
printf 'Cuadro de contratación temporal:\n  curl --cacert %q --cert %q --key %q -H %q -H %q --data-binary %q %q\n' \
  "$DIRECTORIO_MATERIAL/ca/ca.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.key" \
  'Content-Type: application/json' \
  'Accept: application/json' \
  '{"filtros":{"texto":"2026/CT","estado_clave":"en_curso","fase_clave":"analisis"},"paginacion":{"limite":50,"cursor":""}}' \
  "$URL_BASE/api/vec/contratacion-temporal/cuadro/consultas"
printf 'Detalle de contratación temporal:\n  curl --cacert %q --cert %q --key %q -H %q -H %q --data-binary %q %q\n' \
  "$DIRECTORIO_MATERIAL/ca/ca.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.crt" \
  "$DIRECTORIO_MATERIAL/mtls/cliente.key" \
  'Content-Type: application/json' \
  'Accept: application/json' \
  '{"expediente_ref":"expediente:ct:demo:0001","version_observada":3}' \
  "$URL_BASE/api/vec/contratacion-temporal/expedientes/consultas"
printf 'Pulse Ctrl-C para detener VEC y limpiar el binario temporal.\n\n'

if wait "$SERVIDOR_PID"; then
  ESTADO_SERVIDOR=0
else
  ESTADO_SERVIDOR=$?
fi
SERVIDOR_PID=''
exit "$ESTADO_SERVIDOR"
