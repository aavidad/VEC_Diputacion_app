#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

DIRECTORIO_TEST=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_TEST/../.." && pwd -P)
LANZADOR="$RAIZ_REPOSITORIO/scripts/arrancar_vec_desarrollo.sh"
TEMPORAL=$(mktemp -d)
LANZADOR_PID=''
BLOQUEADOR_PID=''
SERVIDOR_PRUEBA_PID=''

# Invocada indirectamente por trap.
# shellcheck disable=SC2329
limpiar() {
  if [[ -n "$LANZADOR_PID" ]] && kill -0 "$LANZADOR_PID" 2>/dev/null; then
    kill -TERM "$LANZADOR_PID" 2>/dev/null || true
    wait "$LANZADOR_PID" 2>/dev/null || true
  fi
  if [[ -n "$BLOQUEADOR_PID" ]] && kill -0 "$BLOQUEADOR_PID" 2>/dev/null; then
    kill -TERM "$BLOQUEADOR_PID" 2>/dev/null || true
    wait "$BLOQUEADOR_PID" 2>/dev/null || true
  fi
  if [[ -n "$SERVIDOR_PRUEBA_PID" ]] && kill -0 "$SERVIDOR_PRUEBA_PID" 2>/dev/null; then
    kill -TERM "$SERVIDOR_PRUEBA_PID" 2>/dev/null || true
    wait "$SERVIDOR_PRUEBA_PID" 2>/dev/null || true
  fi
  rm -rf -- "$TEMPORAL"
}
trap limpiar EXIT HUP INT TERM

"$LANZADOR" --help >"$TEMPORAL/ayuda"
grep -Fq -- '--puerto PUERTO' "$TEMPORAL/ayuda"
grep -Fq -- '--directorio-material RUTA' "$TEMPORAL/ayuda"
grep -Fq 'TLS 1.3, mTLS' "$TEMPORAL/ayuda"

if "$LANZADOR" --puerto 0 >"$TEMPORAL/puerto-invalido" 2>&1; then
  printf 'el lanzador aceptó el puerto cero\n' >&2
  exit 1
fi
grep -Fq 'el puerto debe estar entre 1 y 65535' "$TEMPORAL/puerto-invalido"

if "$LANZADOR" --directorio-material relativo >"$TEMPORAL/material-relativo" 2>&1; then
  printf 'el lanzador aceptó un directorio relativo\n' >&2
  exit 1
fi
grep -Fq 'el directorio de material debe ser absoluto' "$TEMPORAL/material-relativo"

python3 - "$TEMPORAL/puerto-bloqueado" <<'PY' &
import signal
import socket
import sys

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.bind(("127.0.0.1", 0))
sock.listen()
with open(sys.argv[1], "w", encoding="ascii") as salida:
    salida.write(str(sock.getsockname()[1]))
    salida.flush()
signal.pause()
PY
BLOQUEADOR_PID=$!
for (( INTENTO = 0; INTENTO < 100; INTENTO++ )); do
  [[ ! -s "$TEMPORAL/puerto-bloqueado" ]] || break
  sleep 0.02
done
[[ -s "$TEMPORAL/puerto-bloqueado" ]] || {
  printf 'no se obtuvo el puerto bloqueado de prueba\n' >&2
  exit 1
}
PUERTO_BLOQUEADO=$(<"$TEMPORAL/puerto-bloqueado")
if "$LANZADOR" --puerto "$PUERTO_BLOQUEADO" \
  --directorio-material "$TEMPORAL/material-bloqueado" \
  >"$TEMPORAL/colision" 2>&1; then
  printf 'el lanzador aceptó un puerto ocupado\n' >&2
  exit 1
fi
grep -Fq "el puerto 127.0.0.1:$PUERTO_BLOQUEADO no está disponible" "$TEMPORAL/colision"
kill -TERM "$BLOQUEADOR_PID"
wait "$BLOQUEADOR_PID" 2>/dev/null || true
BLOQUEADOR_PID=''

FAKES="$TEMPORAL/fakes"
ESTADO="$TEMPORAL/estado"
mkdir -p "$FAKES" "$ESTADO"
cat >"$FAKES/go" <<'FAKE_GO'
#!/usr/bin/env bash
set -Eeuo pipefail
SALIDA=''
while (( $# > 0 )); do
  if [[ "$1" == -o ]]; then
    SALIDA=$2
    shift 2
  else
    shift
  fi
done
[[ -n "$SALIDA" ]]
cat >"$SALIDA" <<'BINARIO'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$$" >"$VEC_ARRANQUE_TEST_ESTADO/pid"
printf '%s\n' "$0" >"$VEC_ARRANQUE_TEST_ESTADO/binario"
DIRECTORIO_TRABAJO=$(pwd -P)
printf '%s\n' "$DIRECTORIO_TRABAJO" >"$VEC_ARRANQUE_TEST_ESTADO/cwd"
[[ "$DIRECTORIO_TRABAJO" == "$VEC_ARRANQUE_TEST_RAIZ" ]]
{
  printf 'VEC_EXECUTION_PROFILE=%s\n' "$VEC_EXECUTION_PROFILE"
  printf 'VEC_AUTH_MODE=%s\n' "$VEC_AUTH_MODE"
  printf 'VEC_DEVELOPMENT_GUARD=%s\n' "$VEC_DEVELOPMENT_GUARD"
  printf 'VEC_DEVELOPMENT_MATERIAL_DIR=%s\n' "$VEC_DEVELOPMENT_MATERIAL_DIR"
  printf 'VEC_HTTP_ADDR=%s\n' "$VEC_HTTP_ADDR"
  printf 'VEC_TLS_CERT_FILE=%s\n' "$VEC_TLS_CERT_FILE"
  printf 'VEC_TLS_KEY_FILE=%s\n' "$VEC_TLS_KEY_FILE"
} >"$VEC_ARRANQUE_TEST_ESTADO/entorno"
touch "$VEC_ARRANQUE_TEST_ESTADO/listo"
trap 'exit 0' HUP INT TERM
while :; do
  sleep 0.1
done
BINARIO
chmod 700 "$SALIDA"
FAKE_GO
cat >"$FAKES/curl" <<'FAKE_CURL'
#!/usr/bin/env bash
exit 0
FAKE_CURL
chmod 700 "$FAKES/go" "$FAKES/curl"

PUERTO_LIBRE=$(python3 - <<'PY'
import socket

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.bind(("127.0.0.1", 0))
print(sock.getsockname()[1])
sock.close()
PY
)
MATERIAL="$TEMPORAL/material"
CWD_EXTERNO="$TEMPORAL/cwd-externo"
mkdir -p -- "$CWD_EXTERNO"
(
  cd "$CWD_EXTERNO"
  export PATH="$FAKES:$PATH"
  export VEC_ARRANQUE_TEST_ESTADO="$ESTADO"
  export VEC_ARRANQUE_TEST_RAIZ="$RAIZ_REPOSITORIO"
  exec "$LANZADOR" --puerto "$PUERTO_LIBRE" --directorio-material "$MATERIAL"
) >"$TEMPORAL/salida" 2>&1 &
LANZADOR_PID=$!
for (( INTENTO = 0; INTENTO < 300; INTENTO++ )); do
  [[ ! -f "$ESTADO/listo" ]] || break
  if ! kill -0 "$LANZADOR_PID" 2>/dev/null; then
    break
  fi
  sleep 0.02
done
[[ -f "$ESTADO/listo" ]] || {
  printf 'el lanzador no inició el servidor de prueba\n' >&2
  sed -n '1,120p' "$TEMPORAL/salida" >&2
  exit 1
}

SERVIDOR_PRUEBA_PID=$(<"$ESTADO/pid")
kill -0 "$SERVIDOR_PRUEBA_PID"
grep -Fxq "$RAIZ_REPOSITORIO" "$ESTADO/cwd"
grep -Fxq 'VEC_EXECUTION_PROFILE=desarrollo' "$ESTADO/entorno"
grep -Fxq 'VEC_AUTH_MODE=desarrollo' "$ESTADO/entorno"
grep -Fxq 'VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO' "$ESTADO/entorno"
grep -Fxq "VEC_DEVELOPMENT_MATERIAL_DIR=$MATERIAL" "$ESTADO/entorno"
grep -Fxq "VEC_HTTP_ADDR=127.0.0.1:$PUERTO_LIBRE" "$ESTADO/entorno"
grep -Fxq "VEC_TLS_CERT_FILE=$MATERIAL/tls/servidor.crt" "$ESTADO/entorno"
grep -Fxq "VEC_TLS_KEY_FILE=$MATERIAL/tls/servidor.key" "$ESTADO/entorno"
for (( INTENTO = 0; INTENTO < 100; INTENTO++ )); do
  if grep -Fq "Portal: https://localhost:$PUERTO_LIBRE/portal-empleado/" "$TEMPORAL/salida"; then
    break
  fi
  sleep 0.02
done
grep -Fq "Portal: https://localhost:$PUERTO_LIBRE/portal-empleado/" "$TEMPORAL/salida"
grep -Fq 'Cuadro de contratación temporal:' "$TEMPORAL/salida"
grep -Fq 'Detalle de contratación temporal:' "$TEMPORAL/salida"
if grep -Fq 'Authorization:' "$TEMPORAL/salida"; then
  printf 'el lanzador publicó una autenticación bearer\n' >&2
  exit 1
fi

BINARIO=$(<"$ESTADO/binario")
DIRECTORIO_BUILD=$(dirname -- "$BINARIO")
kill -TERM "$LANZADOR_PID"
if wait "$LANZADOR_PID"; then
  ESTADO_LANZADOR=0
else
  ESTADO_LANZADOR=$?
fi
LANZADOR_PID=''
[[ "$ESTADO_LANZADOR" -eq 143 ]]
for (( INTENTO = 0; INTENTO < 100; INTENTO++ )); do
  [[ -e "/proc/$SERVIDOR_PRUEBA_PID" ]] || break
  sleep 0.02
done
[[ ! -e "/proc/$SERVIDOR_PRUEBA_PID" ]]
SERVIDOR_PRUEBA_PID=''
[[ ! -e "$BINARIO" ]]
[[ ! -d "$DIRECTORIO_BUILD" ]]

printf 'arranque VEC desarrollo: OK\n'
